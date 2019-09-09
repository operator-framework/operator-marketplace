package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	// metricsPath is the path that marketplace exposes its metrics at.
	metricsPath = "/metrics"

	// metricsPort is the port that marketplace exposes its metrics at.
	metricsPort = 8383
)

var (
	// Wraps the default RoundTripperFunc with functions that record prometheus metrics.
	roundTripper = http.DefaultTransport

	// appRegistryRequestCounter is a prometheus counter vector that stores
	// information about requests to an AppRegistry.
	appRegistryRequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_registry_request_total",
			Help: "A counter that stores the results of reaching out to an AppRegistry.",
		}, []string{codeLabel, methodLabel, opSrcLabel},
	)

	// appRegistryHistVec is a prometheus HistogramVec that stores latency
	// information about requests to an AppRegistry.
	appRegistryHistVec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "app_registry_request_duration_seconds",
			Help:    "A histogram of AppRegistry request latencies.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{opSrcLabel},
	)
)

// init enables marketplace to serve prometheus metrics.
func init() {
	// Register metrics for the operator with the prometheus.
	log.Info("[metrics] Registering marketplace metrics")
	err := registerMetrics()
	if err != nil {
		log.Infof("[metrics] Unable to register marketplace metrics: %v", err)
		return
	}

	// Wrap the default RoundTripper with middleware.
	log.Info("[metrics] Creating marketplace metrics RoundTripperFunc")
	roundTripper = appRegistryRoundTripperCounter(appRegistryRequestCounter,
		appRegistryRoundTripperDuration(appRegistryHistVec, http.DefaultTransport),
	)

	// Start the server and expose the registered metrics.
	log.Info("[metrics] Serving marketplace metrics")
	http.Handle(metricsPath, promhttp.Handler())
	port := fmt.Sprintf(":%d", metricsPort)
	go http.ListenAndServe(port, nil)
}

// registerMetrics registers marketplace prometheus metrics.
func registerMetrics() error {
	// Register all of the metrics in the standard registry.
	err := prometheus.Register(appRegistryRequestCounter)
	if err != nil {
		return err
	}

	err = prometheus.Register(appRegistryHistVec)
	if err != nil {
		return err
	}

	return nil
}

// GetRoundTripper returns the RoundTripper used to collect metrics
// from marketplace.
func GetRoundTripper() http.RoundTripper {
	return roundTripper
}
