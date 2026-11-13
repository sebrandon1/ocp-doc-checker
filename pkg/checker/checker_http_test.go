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

// countRequests records how many requests hit each URL path.
type countRequests struct {
	mu     sync.Mutex
	counts map[string]int
	total  int
}

func (c *countRequests) add(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts == nil {
		c.counts = map[string]int{}
	}
	c.counts[path]++
	c.total++
}

func (c *countRequests) totalCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

func (c *countRequests) forPath(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[path]
}

func TestCheck_ReusesFetchedPageAcrossAnchors(t *testing.T) {
	counter := &countRequests{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.add(r.URL.Path)
		_, _ = w.Write([]byte(`<div id="one"></div><div id="two"></div>`))
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	c.SetVersions([]string{"4.17", "4.18", "4.19"})

	for _, anchor := range []string{"one", "two"} {
		result, err := c.Check("https://docs.redhat.com/en/documentation/openshift_container_platform/4.17/html-single/storage/index#" + anchor)
		if err != nil {
			t.Fatalf("unexpected error for anchor %q: %v", anchor, err)
		}
		if len(result.NewerVersions) != 2 {
			t.Errorf("anchor %q: NewerVersions = %d, want 2", anchor, len(result.NewerVersions))
		}
	}

	// Two newer versions, fetched once each even though two anchors were checked.
	wantRequests := 2
	if got := counter.totalCount(); got != wantRequests {
		t.Errorf("requests = %d, want %d", got, wantRequests)
	}
	if got := counter.forPath("/en/documentation/openshift_container_platform/4.18/html-single/storage/index"); got != 1 {
		t.Errorf("requests for 4.18 page = %d, want 1", got)
	}
}

func TestCheck_HeadOnlyCacheUpgradesToGetForAnchor(t *testing.T) {
	var gets, heads atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			heads.Add(1)
		} else {
			gets.Add(1)
		}
		_, _ = w.Write([]byte(`<div id="sec"></div>`))
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	pageURL := srv.URL + "/page"

	exists, _, hasAnchor, err := c.checkURL(pageURL)
	if err != nil || !exists || hasAnchor {
		t.Fatalf("first check: exists=%v hasAnchor=%v err=%v, want exists=true hasAnchor=false", exists, hasAnchor, err)
	}

	exists, anchorExists, hasAnchor, err := c.checkURL(pageURL + "#sec")
	if err != nil || !exists || !hasAnchor || !anchorExists {
		t.Fatalf("anchor check: exists=%v anchorExists=%v hasAnchor=%v err=%v, want all true", exists, anchorExists, hasAnchor, err)
	}

	// A second anchored check for a different anchor must reuse the IDs.
	exists, anchorExists, _, err = c.checkURL(pageURL + "#missing")
	if err != nil || !exists || anchorExists {
		t.Fatalf("second anchor check: exists=%v anchorExists=%v err=%v, want exists=true anchorExists=false", exists, anchorExists, err)
	}

	if got := heads.Load(); got != 1 {
		t.Errorf("HEAD requests = %d, want 1", got)
	}
	if got := gets.Load(); got != 1 {
		t.Errorf("GET requests = %d, want 1 after upgrading the HEAD-only entry", got)
	}

	// A repeated no-anchor check is served entirely from the cache.
	exists, _, _, err = c.checkURL(pageURL)
	if err != nil || !exists {
		t.Fatalf("repeat no-anchor check: exists=%v err=%v, want exists=true", exists, err)
	}
	if got := heads.Load() + gets.Load(); got != 2 {
		t.Errorf("total requests = %d, want 2 (repeat check hits the cache)", got)
	}
}

func TestCheck_CacheWorksOnZeroValueMaps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<div id="sec"></div>`))
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	c.cache, c.inflight = nil, nil // simulate a Checker that skipped NewChecker

	exists, anchorExists, _, err := c.checkURL(srv.URL + "/page#sec")
	if err != nil || !exists || !anchorExists {
		t.Fatalf("exists=%v anchorExists=%v err=%v, want all true", exists, anchorExists, err)
	}
}

func TestCheck_CachesNotFound(t *testing.T) {
	counter := &countRequests{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.add(r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	pageURL := srv.URL + "/missing"
	for i := 0; i < 3; i++ {
		exists, _, _, err := c.checkURL(pageURL)
		if err != nil || exists {
			t.Fatalf("check %d: exists=%v err=%v, want exists=false err=nil", i, exists, err)
		}
	}

	if got := counter.totalCount(); got != 1 {
		t.Errorf("requests = %d, want 1 (404 is cached)", got)
	}
}

func TestCheck_DoesNotCacheErrors(t *testing.T) {
	counter := &countRequests{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.add(r.URL.Path)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	pageURL := srv.URL + "/blocked"
	for i := 0; i < 2; i++ {
		exists, _, _, err := c.checkURL(pageURL)
		if err == nil || exists {
			t.Fatalf("check %d: exists=%v err=%v, want exists=false and an error", i, exists, err)
		}
	}

	if got := counter.totalCount(); got != 2 {
		t.Errorf("requests = %d, want 2 (errors are not cached)", got)
	}
}

func TestCheck_ConcurrentFetchesShareOneRequest(t *testing.T) {
	counter := &countRequests{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.add(r.URL.Path)
		time.Sleep(20 * time.Millisecond)
		_, _ = w.Write([]byte(`<div id="sec"></div>`))
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	pageURL := srv.URL + "/shared"

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			exists, anchorExists, _, err := c.checkURL(pageURL + "#sec")
			if err != nil || !exists || !anchorExists {
				t.Errorf("exists=%v anchorExists=%v err=%v, want all true", exists, anchorExists, err)
			}
		}()
	}
	wg.Wait()

	if got := counter.totalCount(); got != 1 {
		t.Errorf("requests = %d, want 1 (concurrent fetches are deduplicated)", got)
	}
}

func TestCheck_ConcurrentFetchesShareOneError(t *testing.T) {
	counter := &countRequests{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.add(r.URL.Path)
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := newTestChecker(t, srv)
	pageURL := srv.URL + "/blocked"

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, _, err := c.checkURL(pageURL); err == nil {
				t.Error("expected an error for a blocked page")
			}
		}()
	}
	wg.Wait()

	if got := counter.totalCount(); got != 1 {
		t.Errorf("requests = %d, want 1 (waiters join the in-flight fetch)", got)
	}
}
