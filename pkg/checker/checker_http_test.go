package checker

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTestChecker returns a Checker whose requests are all routed to srv,
// regardless of the host in the URL, so docs.redhat.com URLs can be tested offline.
func newTestChecker(t *testing.T, srv *httptest.Server) *Checker {
	t.Helper()
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c := NewChecker()
	c.retryDelay = 0
	c.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		return http.DefaultTransport.RoundTrip(req)
	})
	return c
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCheckURL_StatusHandling(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		headStatus   int // if set, HEAD requests get this status instead of status
		body         string
		url          string
		wantExists   bool
		wantAnchor   bool
		wantErr      bool
		wantRequests int32
	}{
		{name: "200 no anchor", status: 200, url: "/page", wantExists: true, wantRequests: 1},
		{name: "200 anchor found", status: 200, body: `<div id="sec"></div>`, url: "/page#sec", wantExists: true, wantAnchor: true, wantRequests: 1},
		{name: "200 anchor missing", status: 200, body: `<div id="other"></div>`, url: "/page#sec", wantExists: true, wantRequests: 1},
		{name: "404 not found is not an error", status: 404, url: "/page", wantRequests: 1},
		{name: "410 gone is not an error", status: 410, url: "/page", wantRequests: 1},
		{name: "403 is an error and not retried", status: 403, url: "/page", wantErr: true, wantRequests: 1},
		{name: "503 is an error after retries", status: 503, url: "/page", wantErr: true, wantRequests: 3},
		{name: "429 is an error after retries", status: 429, url: "/page", wantErr: true, wantRequests: 3},
		{name: "HEAD 405 falls back to GET", status: 200, headStatus: 405, url: "/page", wantExists: true, wantRequests: 2},
		{name: "HEAD 501 falls back to GET that 404s", status: 404, headStatus: 501, url: "/page", wantRequests: 2},
		{name: "HEAD 403 does not fall back", status: 200, headStatus: 403, url: "/page", wantErr: true, wantRequests: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			var gotUA atomic.Value
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				gotUA.Store(r.Header.Get("User-Agent"))
				if tt.headStatus != 0 && r.Method == http.MethodHead {
					w.WriteHeader(tt.headStatus)
					return
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := newTestChecker(t, srv)
			exists, anchor, _, err := c.checkURL(srv.URL + tt.url)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				// The error carries the HEAD status unless HEAD was rejected and retried as GET.
				wantStatus := tt.status
				if tt.headStatus != 0 && !headRejected(tt.headStatus) {
					wantStatus = tt.headStatus
				}
				var statusErr *HTTPStatusError
				if !errors.As(err, &statusErr) || statusErr.StatusCode != wantStatus {
					t.Errorf("err = %v, want *HTTPStatusError with status %d", err, wantStatus)
				}
			}
			if exists != tt.wantExists {
				t.Errorf("exists = %v, want %v", exists, tt.wantExists)
			}
			if anchor != tt.wantAnchor {
				t.Errorf("anchorExists = %v, want %v", anchor, tt.wantAnchor)
			}
			if got := requests.Load(); got != tt.wantRequests {
				t.Errorf("requests = %d, want %d", got, tt.wantRequests)
			}
			if ua, _ := gotUA.Load().(string); ua != DefaultUserAgent {
				t.Errorf("User-Agent = %q, want %q", ua, DefaultUserAgent)
			}
		})
	}
}

const testDocURL = "https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html-single/storage/index"

func TestCheck_AllBlockedReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	result, err := c.Check(testDocURL)
	if err == nil {
		t.Fatalf("expected error when every version is blocked, got result %+v", result)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention 403, got %v", err)
	}
}

func TestCheck_PartialFailureStillReportsFound(t *testing.T) {
	// 4.19 is blocked, 4.20 exists: result should report 4.20 and keep the 4.19 error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/4.19/") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	c.SetVersions([]string{"4.18", "4.19", "4.20"})
	result, err := c.Check(testDocURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsOutdated || result.LatestVersion != "4.20" {
		t.Errorf("got outdated=%v latest=%s, want outdated=true latest=4.20", result.IsOutdated, result.LatestVersion)
	}
	var sawErr bool
	for _, r := range result.AllResults {
		if r.Version == "4.19" && r.Error != nil {
			sawErr = true
		}
	}
	if !sawErr {
		t.Error("expected 4.19 result to carry the 403 error")
	}
}

func TestCheck_NotFoundEverywhereIsUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	result, err := c.Check(testDocURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsOutdated || result.LatestVersion != "4.18" {
		t.Errorf("got outdated=%v latest=%s, want outdated=false latest=4.18", result.IsOutdated, result.LatestVersion)
	}
}

// peakTracker records how many requests were in flight at the same time.
type peakTracker struct {
	mu      sync.Mutex
	current int
	peak    int
}

func (p *peakTracker) enter() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
	if p.current > p.peak {
		p.peak = p.current
	}
}

func (p *peakTracker) leave() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current--
}

func (p *peakTracker) max() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.peak
}

func TestCheck_ConcurrencyBound(t *testing.T) {
	const limit = 2

	tracker := &peakTracker{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.enter()
		// Hold the request open long enough for overlapping requests to show up.
		time.Sleep(20 * time.Millisecond)
		tracker.leave()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	c.SetVersions([]string{"4.19", "4.20", "4.21", "4.22", "5.0"})
	c.maxConcurrent = limit

	if _, err := c.Check(testDocURL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := tracker.max(); got != limit {
		t.Errorf("peak in-flight requests = %d, want %d", got, limit)
	}
}

func TestCheck_NonPositiveConcurrencyIsSerial(t *testing.T) {
	tracker := &peakTracker{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.enter()
		time.Sleep(5 * time.Millisecond)
		tracker.leave()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	c.SetVersions([]string{"4.19", "4.20", "4.21"})
	c.maxConcurrent = 0

	result, err := c.Check(testDocURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := tracker.max(); got != 1 {
		t.Errorf("peak in-flight requests = %d, want 1 with non-positive maxConcurrent", got)
	}
	if want := "4.19,4.20,4.21"; versionsOf(result.AllResults) != want {
		t.Errorf("AllResults versions = %s, want %s", versionsOf(result.AllResults), want)
	}
}

func TestCheck_ResultsOrderedRegardlessOfCompletionOrder(t *testing.T) {
	// Make earlier versions respond last so completion order is reversed.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/4.19/"):
			time.Sleep(60 * time.Millisecond)
		case strings.Contains(r.URL.Path, "/4.20/"):
			time.Sleep(30 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	c.SetVersions([]string{"4.18", "4.19", "4.20", "4.21"})
	c.maxConcurrent = 4

	result, err := c.Check(testDocURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := "4.19,4.20,4.21"; versionsOf(result.AllResults) != want {
		t.Errorf("AllResults versions = %s, want %s", versionsOf(result.AllResults), want)
	}
	if want := "4.19,4.20,4.21"; versionsOf(result.NewerVersions) != want {
		t.Errorf("NewerVersions = %s, want %s", versionsOf(result.NewerVersions), want)
	}
	if !result.IsOutdated || result.LatestVersion != "4.21" {
		t.Errorf("got outdated=%v latest=%s, want outdated=true latest=4.21", result.IsOutdated, result.LatestVersion)
	}
}

func TestCheck_NoNewerVersionsSkipsRequests(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	c.SetVersions([]string{"4.22", "5.0"})

	result, err := c.Check("https://docs.redhat.com/en/documentation/openshift_container_platform/5.0/html-single/storage/index")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("requests = %d, want 0 when no version is newer", got)
	}
	if result.IsOutdated || result.LatestVersion != "5.0" {
		t.Errorf("got outdated=%v latest=%s, want outdated=false latest=5.0", result.IsOutdated, result.LatestVersion)
	}
	if len(result.AllResults) != 0 {
		t.Errorf("AllResults = %d entries, want 0", len(result.AllResults))
	}
}

// versionsOf joins the version fields of results in slice order.
func versionsOf(results []VersionCheckResult) string {
	versions := make([]string, len(results))
	for i, r := range results {
		versions[i] = r.Version
	}
	return strings.Join(versions, ",")
}
