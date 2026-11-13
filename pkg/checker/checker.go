package checker

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sebrandon1/ocp-doc-checker/pkg/parser"
	"golang.org/x/net/html"
)

// VersionCheckResult represents the result of checking a version
type VersionCheckResult struct {
	Version      string
	URL          string
	Exists       bool
	AnchorExists bool // true if URL has anchor and it exists, false if URL has anchor but doesn't exist, N/A if no anchor
	HasAnchor    bool // true if URL contains a fragment/anchor
	Error        error
	CheckedAt    time.Time
}

// CheckResult represents the complete check result
type CheckResult struct {
	OriginalURL     string
	OriginalVersion string
	LatestVersion   string
	IsOutdated      bool
	NewerVersions   []VersionCheckResult
	AllResults      []VersionCheckResult
}

// DefaultUserAgent identifies the tool to documentation servers.
const DefaultUserAgent = "ocp-doc-checker (+https://github.com/sebrandon1/ocp-doc-checker)"

// Checker handles checking OCP documentation URLs
type Checker struct {
	client        *http.Client
	knownVersions []string
	maxConcurrent int
	userAgent     string
	retryDelay    time.Duration
}

// NewChecker creates a new Checker instance
func NewChecker() *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: 30 * time.Second, // Increased timeout for CI environments
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Follow redirects
				return nil
			},
		},
		// Known OCP versions to check (can be expanded)
		knownVersions: []string{
			"4.10", "4.11", "4.12", "4.13", "4.14",
			"4.15", "4.16", "4.17", "4.18", "4.19",
			"4.20", "4.21", "4.22",
			"5.0",
		},
		maxConcurrent: 5,
		userAgent:     DefaultUserAgent,
		retryDelay:    2 * time.Second,
	}
}

// SetVersions allows setting custom versions to check
func (c *Checker) SetVersions(versions []string) {
	c.knownVersions = versions
}

// SetUserAgent overrides the User-Agent header sent with requests
func (c *Checker) SetUserAgent(ua string) {
	c.userAgent = ua
}

// Check performs the URL check
func (c *Checker) Check(rawURL string) (*CheckResult, error) {
	// Parse the URL
	docURL, err := parser.ParseOCPDocURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	result := &CheckResult{
		OriginalURL:     rawURL,
		OriginalVersion: docURL.Version,
		AllResults:      []VersionCheckResult{},
	}

	// Filter versions to check (only those newer than current)
	versionsToCheck := c.getNewerVersions(docURL.Version)

	var firstErr error
	errCount := 0

	// Check each version
	for _, version := range versionsToCheck {
		checkURL := docURL.BuildURL(version)
		exists, anchorExists, hasAnchor, err := c.checkURL(checkURL)

		versionResult := VersionCheckResult{
			Version:      version,
			URL:          checkURL,
			Exists:       exists,
			AnchorExists: anchorExists,
			HasAnchor:    hasAnchor,
			Error:        err,
			CheckedAt:    time.Now(),
		}

		result.AllResults = append(result.AllResults, versionResult)

		if err != nil {
			errCount++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		// Only consider it a valid newer version if both page and anchor (if present) exist
		if exists && (!hasAnchor || anchorExists) {
			result.NewerVersions = append(result.NewerVersions, versionResult)
		}
	}

	// If no newer version could be checked at all, we cannot claim the URL is
	// up to date. Surface the failure instead of silently reporting "current".
	if errCount > 0 && errCount == len(versionsToCheck) {
		return nil, fmt.Errorf("unable to check any newer version of %s: %w", rawURL, firstErr)
	}

	// Determine if outdated and latest version
	if len(result.NewerVersions) > 0 {
		result.IsOutdated = true
		// Latest version is the last one in the list (they're sorted)
		result.LatestVersion = result.NewerVersions[len(result.NewerVersions)-1].Version
	} else {
		result.LatestVersion = docURL.Version
	}

	return result, nil
}

// HTTPStatusError reports an HTTP response that neither confirms nor denies
// that a page exists (e.g. 403 from bot protection, 429, 5xx).
type HTTPStatusError struct {
	URL        string
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	msg := fmt.Sprintf("unexpected HTTP %d %s from %s", e.StatusCode, http.StatusText(e.StatusCode), e.URL)
	if e.StatusCode == http.StatusForbidden {
		msg += " (access denied; the documentation site may be blocking automated requests)"
	}
	return msg
}

// isRetryableStatus reports whether a status code is worth retrying.
func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

func (c *Checker) do(method, rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return c.client.Do(req)
}

// checkURL checks if a URL exists and validates anchor if present
// Returns: (pageExists, anchorExists, hasAnchor, error)
//
// A 404/410 means the page does not exist and is not an error. Any other
// non-2xx/3xx status is returned as an *HTTPStatusError, because it says
// nothing about whether the page exists.
func (c *Checker) checkURL(urlString string) (bool, bool, bool, error) {
	maxRetries := 3
	var lastErr error

	// Parse URL to extract fragment/anchor
	fragment := ""
	baseURL := urlString
	if idx := strings.Index(urlString, "#"); idx != -1 {
		fragment = urlString[idx+1:]
		baseURL = urlString[:idx]
	}

	hasAnchor := fragment != ""

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Wait a bit before retrying (linear backoff)
			time.Sleep(time.Duration(attempt) * c.retryDelay)
		}

		var resp *http.Response
		var err error

		if hasAnchor {
			// If we need to check anchor, use GET to fetch the HTML
			resp, err = c.do(http.MethodGet, baseURL)
		} else {
			// No anchor, use HEAD for efficiency
			resp, err = c.do(http.MethodHead, baseURL)
			if err != nil {
				// If HEAD fails, try GET
				resp, err = c.do(http.MethodGet, baseURL)
			}
		}

		if err != nil {
			lastErr = err
			continue // Retry
		}

		pageExists, anchorExists, retry, err := c.handleResponse(resp, baseURL, fragment)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			if retry {
				continue
			}
			return false, false, hasAnchor, err
		}
		return pageExists, anchorExists, hasAnchor, nil
	}

	return false, false, hasAnchor, lastErr
}

// handleResponse interprets a single HTTP response.
// Returns: (pageExists, anchorExists, retry, error)
func (c *Checker) handleResponse(resp *http.Response, baseURL, fragment string) (bool, bool, bool, error) {
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		// Page genuinely doesn't exist for this version
		return false, false, false, nil
	case resp.StatusCode < 200 || resp.StatusCode >= 400:
		statusErr := &HTTPStatusError{URL: baseURL, StatusCode: resp.StatusCode}
		return false, false, isRetryableStatus(resp.StatusCode), statusErr
	}

	// Page exists (2xx or 3xx)
	if fragment == "" {
		return true, false, false, nil
	}

	// Validate anchor exists in HTML
	anchorExists, err := c.checkAnchorInHTML(resp.Body, fragment)
	if err != nil {
		return false, false, true, err
	}
	return true, anchorExists, false, nil
}

// checkAnchorInHTML parses HTML and checks if an anchor/fragment exists
func (c *Checker) checkAnchorInHTML(body io.Reader, anchor string) (bool, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return false, fmt.Errorf("failed to parse HTML: %w", err)
	}

	found := false
	var checkNode func(*html.Node)
	checkNode = func(n *html.Node) {
		if found {
			return
		}

		if n.Type == html.ElementNode {
			// Check for id attribute
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val == anchor {
					found = true
					return
				}
				// Also check for name attribute (older HTML anchor style)
				if n.Data == "a" && attr.Key == "name" && attr.Val == anchor {
					found = true
					return
				}
			}
		}

		// Recursively check child nodes
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			checkNode(child)
		}
	}

	checkNode(doc)
	return found, nil
}

// getNewerVersions returns versions newer than the given version
func (c *Checker) getNewerVersions(currentVersion string) []string {
	var newer []string

	currentDoc := &parser.OCPDocURL{Version: currentVersion}
	if err := parseVersionInPlace(currentDoc); err != nil {
		return newer
	}

	currentFloat := currentDoc.GetVersionFloat()

	for _, v := range c.knownVersions {
		testDoc := &parser.OCPDocURL{Version: v}
		if err := parseVersionInPlace(testDoc); err != nil {
			continue
		}

		if testDoc.GetVersionFloat() > currentFloat {
			newer = append(newer, v)
		}
	}

	// Sort versions
	sort.Slice(newer, func(i, j int) bool {
		vi := &parser.OCPDocURL{Version: newer[i]}
		vj := &parser.OCPDocURL{Version: newer[j]}
		_ = parseVersionInPlace(vi) // already validated above
		_ = parseVersionInPlace(vj)
		return vi.GetVersionFloat() < vj.GetVersionFloat()
	})

	return newer
}

// parseVersionInPlace is a helper to parse version string into MajorMinor
func parseVersionInPlace(doc *parser.OCPDocURL) error {
	var major, minor int
	_, err := fmt.Sscanf(doc.Version, "%d.%d", &major, &minor)
	if err != nil {
		return err
	}
	doc.MajorMinor = [2]int{major, minor}
	return nil
}
