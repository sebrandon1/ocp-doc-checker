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

// Checker handles checking OCP documentation URLs
type Checker struct {
	client        *http.Client
	knownVersions []string
	maxConcurrent int
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
			"4.20",
		},
		maxConcurrent: 5,
	}
}

// SetVersions allows setting custom versions to check
func (c *Checker) SetVersions(versions []string) {
	c.knownVersions = versions
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

		// Only consider it a valid newer version if both page and anchor (if present) exist
		if exists && (!hasAnchor || anchorExists) {
			result.NewerVersions = append(result.NewerVersions, versionResult)
		}
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

// checkURL checks if a URL exists and validates anchor if present
// Returns: (pageExists, anchorExists, hasAnchor, error)
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
			// Wait a bit before retrying (exponential backoff)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		// First check if the base URL exists
		var resp *http.Response
		var err error

		if hasAnchor {
			// If we need to check anchor, use GET to fetch the HTML
			resp, err = c.client.Get(baseURL)
		} else {
			// No anchor, use HEAD for efficiency
			resp, err = c.client.Head(baseURL)
			if err != nil {
				// If HEAD fails, try GET
				resp, err = c.client.Get(baseURL)
			}
		}

		if err != nil {
			lastErr = err
			continue // Retry
		}
		defer resp.Body.Close()

		// Check if page exists
		if resp.StatusCode >= 400 {
			// 4xx or 5xx - page doesn't exist, no point retrying
			return false, false, hasAnchor, nil
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			// Unexpected status code
			continue // Retry
		}

		// Page exists (2xx or 3xx)
		pageExists := true

		// If no anchor, we're done
		if !hasAnchor {
			return pageExists, false, hasAnchor, nil
		}

		// Validate anchor exists in HTML
		anchorExists, err := c.checkAnchorInHTML(resp.Body, fragment)
		if err != nil {
			lastErr = err
			continue // Retry
		}

		return pageExists, anchorExists, hasAnchor, nil
	}

	return false, false, hasAnchor, lastErr
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
		parseVersionInPlace(vi)
		parseVersionInPlace(vj)
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
