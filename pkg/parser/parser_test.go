package parser

import (
	"testing"
)

func TestParseOCPDocURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantVersion string
		wantDoc     string
		wantPage    string
		wantAnchor  string
		wantErr     bool
	}{
		{
			name:        "Valid 4.17 URL with anchor (html-single)",
			url:         "https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/disconnected_environments/index#mirroring-image-set-full",
			wantVersion: "4.17",
			wantDoc:     "disconnected_environments",
			wantPage:    "index",
			wantAnchor:  "mirroring-image-set-full",
			wantErr:     false,
		},
		{
			name:        "Valid 4.19 URL (html-single)",
			url:         "https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/disconnected_environments/index#mirroring-image-set-partial",
			wantVersion: "4.19",
			wantDoc:     "disconnected_environments",
			wantPage:    "index",
			wantAnchor:  "mirroring-image-set-partial",
			wantErr:     false,
		},
		{
			name:        "URL without anchor (html-single)",
			url:         "https://docs.redhat.com/en/documentation/openshift_container_platform/4.15/html-single/authentication_and_authorization/index",
			wantVersion: "4.15",
			wantDoc:     "authentication_and_authorization",
			wantPage:    "index",
			wantAnchor:  "",
			wantErr:     false,
		},
		{
			name:        "Multi-page URL (html)",
			url:         "https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html/scalability_and_performance/telco-hub-ref-design-specs",
			wantVersion: "4.18",
			wantDoc:     "scalability_and_performance",
			wantPage:    "telco-hub-ref-design-specs",
			wantAnchor:  "",
			wantErr:     false,
		},
		{
			name:    "Invalid URL - not Red Hat",
			url:     "https://example.com/docs",
			wantErr: true,
		},
		{
			name:    "Invalid URL - wrong format",
			url:     "https://docs.redhat.com/something/else",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOCPDocURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseOCPDocURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Version != tt.wantVersion {
				t.Errorf("ParseOCPDocURL() Version = %v, want %v", got.Version, tt.wantVersion)
			}
			if got.Document != tt.wantDoc {
				t.Errorf("ParseOCPDocURL() Document = %v, want %v", got.Document, tt.wantDoc)
			}
			if got.Page != tt.wantPage {
				t.Errorf("ParseOCPDocURL() Page = %v, want %v", got.Page, tt.wantPage)
			}
			if got.Anchor != tt.wantAnchor {
				t.Errorf("ParseOCPDocURL() Anchor = %v, want %v", got.Anchor, tt.wantAnchor)
			}
		})
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name    string
		docURL  *OCPDocURL
		version string
		want    string
	}{
		{
			name: "html-single with anchor",
			docURL: &OCPDocURL{
				BaseURL:  "https://docs.redhat.com",
				Format:   "html-single",
				Document: "disconnected_environments",
				Page:     "index",
				Anchor:   "mirroring-image-set-full",
			},
			version: "4.19",
			want:    "https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-single/disconnected_environments/index#mirroring-image-set-full",
		},
		{
			name: "html multi-page without anchor",
			docURL: &OCPDocURL{
				BaseURL:  "https://docs.redhat.com",
				Format:   "html",
				Document: "scalability_and_performance",
				Page:     "telco-hub-ref-design-specs",
				Anchor:   "",
			},
			version: "4.19",
			want:    "https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html/scalability_and_performance/telco-hub-ref-design-specs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.docURL.BuildURL(tt.version)
			if got != tt.want {
				t.Errorf("BuildURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVersionFloat(t *testing.T) {
	tests := []struct {
		majorMinor [2]int
		want       float64
	}{
		{[2]int{4, 17}, 4.17},
		{[2]int{4, 9}, 4.09},
		{[2]int{5, 0}, 5.00},
	}

	for _, tt := range tests {
		docURL := &OCPDocURL{MajorMinor: tt.majorMinor}
		got := docURL.GetVersionFloat()
		if got != tt.want {
			t.Errorf("GetVersionFloat() = %v, want %v", got, tt.want)
		}
	}
}
