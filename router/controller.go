package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	"github.com/nexsoft-git/nexcommon/http/endpoint/health"
	mdl "github.com/nexsoft-git/nexcommon/http/server/middleware"
	"github.com/nexsoft-git/nexlogger/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/william22913/sandbox-svc-code/config"
)

// maxRequestBodyBytes caps inbound bodies. nexcommon reads the body with an
// unbounded io.ReadAll before any validation runs, so without this the caller
// decides how much memory the process allocates.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// FavoriteExportPath is FEAT-003's read-only CSV route.
//
// It is registered directly on the router rather than through nexcommon's
// HandleFunc: the response is a file with its own headers, not the JSON
// envelope the wrapper forces. The constant lives here because
// favorite_endpoint already imports router, and the reverse would be a cycle.
const FavoriteExportPath = V1PrefixPath + "/favorites/export"

var router *mux.Router

func InitHttpService(
	config config.Configuration,
	metricMiddleware mdl.MetricMiddleware,
	httpValidator *http_validator.HTTPController,
	healthChecker health.HealthEndpoint,
	exportHandler http.HandlerFunc,
) {
	router = mux.NewRouter()
	httpValidator.Router(router)

	// /metrics sits outside user-token validation deliberately; access is
	// restricted by the deployment, not by a route validator.
	router.HandleFunc("/metrics", promhttp.Handler().ServeHTTP).Methods(http.MethodGet)
	router.HandleFunc("/v1/health", healthChecker.CheckHealthConnection).Methods(http.MethodGet)

	// FEAT-003's CSV route. Registered on the same router as everything else,
	// so router.Use(metricMiddleware.Serve) below observes it with no separate
	// wiring, and so it cannot be shadowed by another entity's path.
	if exportHandler != nil {
		router.HandleFunc(FavoriteExportPath, exportHandler).Methods(http.MethodGet)
	}

	router.Use(metricMiddleware.Serve)
}

func StartService(
	config config.Configuration,
	httpValidator *http_validator.HTTPController,
) {
	// Reads the metadata accumulated by InitEndpoint's registrations, so it has
	// to run after them and before listening.
	httpValidator.GenerateSwaggerDocs("/inspect", "sandbox-svc Application Documentation", true)

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
			router.ServeHTTP(w, r)
		},
	)

	// Never http.ListenAndServe directly: it builds a server with every timeout
	// at its zero value, which is no timeout at all.
	server := &http.Server{
		Addr:              fmt.Sprintf("0.0.0.0:%d", config.Server.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	log.Info().
		Str("action", "server.start").
		Int("port", config.Server.Port).
		Msg("HTTP Server Start.")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal().
			Str("action", "server.stop").
			Err(err).
			Msg("HTTP Server Stopped.")
	}
}
