// Package middleware verifies the service's own HTTP metrics middleware by
// scraping it, per the prometheus standards' verification requirement: a real
// scrape, not just the absence of an error.
//
// The claim under test is the one REQ-007 rests on. nexcommon's own
// http/server/middleware hardcodes `statusCode := 200`, so a 422 cap rejection
// would be recorded as a success and the alert built on it would never fire.
// What follows is evidence that this service's middleware records the status
// the client actually received, keeps its label cardinality bounded, and does
// not count scrapes as traffic.
//
// No config import — config's init() calls log.Fatal on a bad environment.
package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/nexsoft-git/nexcommon/server/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	svcmiddleware "github.com/william22913/sandbox-svc-code/middleware"
)

// NewMetrics registers its collectors on the default registry and ignores the
// registration error, so a second call in the same process yields collectors
// nothing scrapes. One construction per process, shared by every test here.
var (
	once    sync.Once
	mw      *svcmiddleware.StatusAwareMetricsMiddleware
	handler http.Handler
)

func build() {
	metric := metrics.NewMetrics()
	mw = svcmiddleware.NewStatusAwareMetricsMiddleware()
	mw.SupportMetrics(metric)

	r := mux.NewRouter()
	r.HandleFunc("/v1/items/{item_id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}).Methods(http.MethodDelete)

	r.HandleFunc("/v1/items", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	// Deliberately registered the way the real router does it: promhttp on the
	// default registry, outside anything else.
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	r.Use(mw.Serve)
	handler = r
}

func scrape(t *testing.T) string {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("scraping /metrics answered %d", rec.Code)
	}
	return rec.Body.String()
}

func call(method, path string) int {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec.Code
}

// TestScrapeSeparatesSuccessFromCapRejection is the point of the whole
// middleware: a 422 must not be filed under 200.
func TestScrapeSeparatesSuccessFromCapRejection(t *testing.T) {
	once.Do(build)

	if got := call(http.MethodGet, "/v1/items"); got != http.StatusOK {
		t.Fatalf("GET /v1/items answered %d", got)
	}
	if got := call(http.MethodDelete, "/v1/items/10"); got != http.StatusUnprocessableEntity {
		t.Fatalf("DELETE /v1/items/10 answered %d", got)
	}

	body := scrape(t)

	// Prometheus renders labels in alphabetical order: method, path, status.
	if !strings.Contains(body, `my_app_http_hist_count{method="GET",path="/v1/items",status="200"} 1`) {
		t.Error("the successful request is not scraped as a 200")
	}
	if !strings.Contains(body, `my_app_http_hist_count{method="DELETE",path="/v1/items/{item_id}",status="422"} 1`) {
		t.Error("the rejected request is not scraped as a 422 — it was folded into another status")
	}
}

// TestScrapeLabelsByRouteTemplateNotByID — the panel for this metric is only
// readable if its series are bounded by route. Labelling by the raw path lets
// any caller mint a new series per id, which is both an unusable graph and a
// memory-growth vector.
func TestScrapeLabelsByRouteTemplateNotByID(t *testing.T) {
	once.Do(build)

	call(http.MethodDelete, "/v1/items/11")
	call(http.MethodDelete, "/v1/items/12")

	body := scrape(t)

	if strings.Contains(body, `path="/v1/items/11"`) || strings.Contains(body, `path="/v1/items/12"`) {
		t.Error("metric series are labelled by raw path, so ids create unbounded series")
	}
	if !strings.Contains(body, `path="/v1/items/{item_id}"`) {
		t.Error("no series carries the route template")
	}
}

// TestScrapeDoesNotCountItself — a metric that climbs every time it is scraped
// is a metric nobody can reason about.
func TestScrapeDoesNotCountItself(t *testing.T) {
	once.Do(build)

	scrape(t)
	body := scrape(t)

	if strings.Contains(body, `path="/metrics"`) {
		t.Error("/metrics counts its own scrapes")
	}
}

// TestUnmatchedRequestsAreNotRecorded pins a real gap so it stays visible
// rather than being mistaken for coverage.
//
// gorilla/mux applies middleware only after a route matches — an unmatched
// request goes straight to the 404 handler and never reaches this middleware.
// A 404 therefore produces no metric at all, not a series under a fixed label
// and not one under its raw URL. The routeTemplate fallback still exists to
// guard the nil case the standard warns about (a matched route whose template
// lookup fails), it just is not reachable by a 404.
//
// Consequence, recorded in dev-test-results.md: this histogram measures the
// traffic that reached a handler and cannot be used to alert on 404 volume.
// If this test starts failing because a series appeared, the router has gained
// coverage of unmatched requests — assert its label then instead.
func TestUnmatchedRequestsAreNotRecorded(t *testing.T) {
	once.Do(build)

	before := scrape(t)

	if got := call(http.MethodGet, "/no/such/route/at/all"); got != http.StatusNotFound {
		t.Fatalf("unmatched request answered %d, expected 404", got)
	}

	after := scrape(t)
	if series(after) != series(before) {
		t.Errorf("an unmatched request added metric series (%d -> %d)", series(before), series(after))
	}
	if strings.Contains(after, "/no/such/route/at/all") {
		t.Error("an unmatched request minted a series from its raw URL")
	}
}

// TestRequestsWithNoMatchedRouteUseTheFixedLabel exercises the label fallback
// where a mux 404 cannot: the middleware sitting over a plain handler, so no
// route was ever set on the request.
//
// Outside a mux match this is the only thing standing between the histogram
// and an unbounded series per URL — the failure mode the prometheus standard
// calls out in nexcommon's own middleware, which falls back to r.URL.Path.
func TestRequestsWithNoMatchedRouteUseTheFixedLabel(t *testing.T) {
	once.Do(build)

	mw.Serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/some/unrouted/path", nil),
	)

	body := scrape(t)
	if !strings.Contains(body, `path="unmatched"`) {
		t.Error("a request with no matched route did not get the fixed label")
	}
	if strings.Contains(body, `path="/some/unrouted/path"`) {
		t.Error("a request with no matched route minted a series from its raw URL")
	}
}

// series counts the sample lines belonging to the request histogram.
func series(body string) int {
	n := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "my_app_http_hist_") || strings.HasPrefix(line, "my_app_http_hist{") {
			n++
		}
	}
	return n
}
