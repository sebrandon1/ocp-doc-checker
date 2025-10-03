package parser

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// OCPDocURL represents a parsed OCP documentation URL
type OCPDocURL struct {
	BaseURL     string
	Version     string
	MajorMinor  [2]int // e.g., [4, 17] for version 4.17
	Format      string // e.g., "html-single" or "html"
	Document    string // e.g., "disconnected_environments"
	Page        string // e.g., "index" or "telco-hub-ref-design-specs"
	Anchor      string // e.g., "mirroring-image-set-full"
	OriginalURL string
}

// ParseOCPDocURL parses an OCP documentation URL and extracts its components
func ParseOCPDocURL(rawURL string) (*OCPDocURL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Validate this is an OCP documentation URL
	if !strings.Contains(parsedURL.Host, "docs.redhat.com") {
		return nil, fmt.Errorf("not a Red Hat documentation URL")
	}

	// Extract components from path
	// Expected format: /en/documentation/openshift_container_platform/VERSION/FORMAT/DOCUMENT/PAGE
	pathRegex := regexp.MustCompile(`/documentation/openshift_container_platform/(\d+\.\d+)/([^/]+)/([^/]+)/([^/?\#]+)`)
	matches := pathRegex.FindStringSubmatch(parsedURL.Path)

	if len(matches) < 5 {
		return nil, fmt.Errorf("URL does not match expected OCP documentation format")
	}

	version := matches[1]
	format := matches[2]
	document := matches[3]
	page := matches[4]

	// Parse major.minor version
	versionParts := strings.Split(version, ".")
	major, _ := strconv.Atoi(versionParts[0])
	minor, _ := strconv.Atoi(versionParts[1])

	return &OCPDocURL{
		BaseURL:     fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host),
		Version:     version,
		MajorMinor:  [2]int{major, minor},
		Format:      format,
		Document:    document,
		Page:        page,
		Anchor:      strings.TrimPrefix(parsedURL.Fragment, ""),
		OriginalURL: rawURL,
	}, nil
}

// BuildURL constructs a URL for a specific version
func (o *OCPDocURL) BuildURL(version string) string {
	url := fmt.Sprintf("%s/en/documentation/openshift_container_platform/%s/%s/%s/%s",
		o.BaseURL, version, o.Format, o.Document, o.Page)

	if o.Anchor != "" {
		url += "#" + o.Anchor
	}

	return url
}

// GetVersionFloat returns the version as a float for comparison
func (o *OCPDocURL) GetVersionFloat() float64 {
	return float64(o.MajorMinor[0]) + float64(o.MajorMinor[1])/100.0
}
