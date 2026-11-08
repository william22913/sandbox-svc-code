package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	srv "github.com/nexsoft-git/nexcommon/server"
	"github.com/nexsoft-git/nexcommon/server/metrics"
)

// StatusAwareMetricsMiddleware records inbound HTTP request metrics with the
// response's *actual* status code.
//
// nexcommon's own http/server/middleware does that too now — since v0.3.68 it
// wraps the writer and observes the real status instead of a hardcoded 200 — so
// this is no longer a workaround for a library bug. It is kept for the two things
// the library's middleware still gets wrong, both of which
// nexcommon-go-prometheus-standards asks for explicitly:
//
//   - The library reads its route label with
//     `mux.CurrentRoute(r).GetPathTemplate()`, which dereferences a nil route on
//     an unmatched request, and falls back to `r.URL.Path` — an unbounded label
//     any caller can mint new series with. routeTemplate below is nil-safe and
//     bounded.
//   - Its loggingResponseWriter embeds http.ResponseWriter without `Unwrap` or
//     `Flush`/`Hijack`, so http.ResponseController cannot reach the real writer.
//     statusRecorder exposes `Unwrap` for exactly that.
//
// It satisfies the same interface (SupportMetrics + Serve), so the router wires
// it identically.
type StatusAwareMetricsMiddleware struct {
	metrics metrics.Metrics
}

func NewStatusAwareMetricsMiddleware() *StatusAwareMetricsMiddleware {
	return &StatusAwareMetricsMiddleware{}
}

func (m *StatusAwareMetricsMiddleware) SupportMetrics(metrics metrics.Metrics) {
	m.metrics = metrics
}

func (m *StatusAwareMetricsMiddleware) Serve(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()

			srv.CORSOriginHandler(&w)

			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			if r.Method != http.MethodOptions {
				next.ServeHTTP(recorder, r)
			}

			// /metrics must not count itself, and CORS preflights are not
			// real traffic.
			if r.URL.Path == "/metrics" || r.Method == http.MethodOptions || m.metrics == nil {
				return
			}

			m.metrics.GetDefaultMetric().APIHist.WithLabelValues(
				routeTemplate(r),
				r.Method,
				strconv.Itoa(recorder.status),
			).Observe(float64(time.Since(now).Seconds()))
		},
	)
}

// routeTemplate returns the matched route's template, so series stay bounded by
// route rather than by the ids passing through them.
//
// An unmatched request falls back to a fixed label rather than r.URL.Path:
// nexcommon's own middleware uses the raw path here, which lets any caller mint
// unbounded metric series just by requesting new URLs.
func routeTemplate(r *http.Request) string {
	if route := mux.CurrentRoute(r); route != nil {
		if template, err := route.GetPathTemplate(); err == nil && template != "" {
			return template
		}
	}

	return "unmatched"
}

// statusRecorder captures the status code written by the handler chain.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.wroteHeader {
		return
	}

	s.status = code
	s.wroteHeader = true
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(body []byte) (int, error) {
	// A handler that writes without calling WriteHeader implicitly means 200.
	s.wroteHeader = true

	return s.ResponseWriter.Write(body)
}

// Unwrap lets http.ResponseController reach Flush/Hijack/SetDeadline on the
// underlying writer, so this wrapper doesn't have to reimplement every optional
// interface http.ResponseWriter may carry.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
