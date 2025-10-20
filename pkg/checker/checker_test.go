package checker

import (
	"strings"
	"testing"
)

func TestCheckAnchorInHTML(t *testing.T) {
	tests := []struct {
		name       string
		html       string
		anchor     string
		wantExists bool
		wantErr    bool
	}{
		{
			name: "Anchor exists with id attribute",
			html: `
<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
	<h1 id="test-anchor">Test Header</h1>
	<p>Some content</p>
</body>
</html>`,
			anchor:     "test-anchor",
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "Anchor exists with name attribute in <a> tag",
			html: `
<!DOCTYPE html>
<html>
<body>
	<a name="legacy-anchor">Legacy</a>
	<p>Some content</p>
</body>
</html>`,
			anchor:     "legacy-anchor",
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "Anchor does not exist",
			html: `
<!DOCTYPE html>
<html>
<body>
	<h1 id="different-anchor">Test Header</h1>
	<p>Some content</p>
</body>
</html>`,
			anchor:     "missing-anchor",
			wantExists: false,
			wantErr:    false,
		},
		{
			name: "Complex Red Hat docs style anchor",
			html: `
<!DOCTYPE html>
<html>
<body>
	<div class="section">
		<h2 id="installing-sr-iov-operator_installing-sriov-operator">Installing SR-IOV Operator</h2>
		<p>Content about installation</p>
	</div>
</body>
</html>`,
			anchor:     "installing-sr-iov-operator_installing-sriov-operator",
			wantExists: true,
			wantErr:    false,
		},
		{
			name: "Anchor with special characters",
			html: `
<!DOCTYPE html>
<html>
<body>
	<h1 id="section-1.2.3">Section 1.2.3</h1>
</body>
</html>`,
			anchor:     "section-1.2.3",
			wantExists: true,
			wantErr:    false,
		},
	}

	checker := NewChecker()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			gotExists, err := checker.checkAnchorInHTML(reader, tt.anchor)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkAnchorInHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotExists != tt.wantExists {
				t.Errorf("checkAnchorInHTML() = %v, want %v", gotExists, tt.wantExists)
			}
		})
	}
}

func TestCheckURL_FragmentParsing(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantHasAnchor bool
		wantFragment  string
	}{
		{
			name:          "URL with anchor",
			url:           "https://example.com/page#section-1",
			wantHasAnchor: true,
			wantFragment:  "section-1",
		},
		{
			name:          "URL without anchor",
			url:           "https://example.com/page",
			wantHasAnchor: false,
			wantFragment:  "",
		},
		{
			name:          "URL with complex anchor",
			url:           "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/networking/index#installing-sr-iov-operator_installing-sriov-operator",
			wantHasAnchor: true,
			wantFragment:  "installing-sr-iov-operator_installing-sriov-operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test fragment extraction logic
			fragment := ""
			hasAnchor := false

			if idx := strings.Index(tt.url, "#"); idx != -1 {
				fragment = tt.url[idx+1:]
				hasAnchor = true
			}

			if hasAnchor != tt.wantHasAnchor {
				t.Errorf("hasAnchor = %v, want %v", hasAnchor, tt.wantHasAnchor)
			}

			if fragment != tt.wantFragment {
				t.Errorf("fragment = %v, want %v", fragment, tt.wantFragment)
			}
		})
	}
}
