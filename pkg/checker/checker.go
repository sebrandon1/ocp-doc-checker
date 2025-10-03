package checker

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/sebrandon1/ocp-doc-checker/pkg/parser"
)

// VersionCheckResult represents the result of checking a version
type VersionCheckResult struct {
	Version   string
	URL       string
	Exists    bool
	Error     error
	CheckedAt time.Time
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
		exists, err := c.checkURL(checkURL)

		versionResult := VersionCheckResult{
			Version:   version,
			URL:       checkURL,
			Exists:    exists,
			Error:     err,
			CheckedAt: time.Now(),
		}

		result.AllResults = append(result.AllResults, versionResult)

		if exists {
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

// checkURL checks if a URL exists and returns 200 OK with retry logic
func (c *Checker) checkURL(url string) (bool, error) {
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Wait a bit before retrying (exponential backoff)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
		
		resp, err := c.client.Head(url)
		if err != nil {
			// If HEAD fails, try GET
			resp, err = c.client.Get(url)
			if err != nil {
				lastErr = err
				continue // Retry
			}
		}
		defer resp.Body.Close()

		// Consider 2xx and 3xx status codes as existing
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true, nil
		}
		
		// If we get a 4xx or 5xx, no point retrying
		if resp.StatusCode >= 400 {
			return false, nil
		}
	}
	
	return false, lastErr
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
