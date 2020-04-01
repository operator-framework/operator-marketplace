package metrics

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/operator-framework/operator-marketplace/pkg/filemonitor"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	// metricsPath is the path that marketplace exposes its metrics at.
	metricsPath = "/metrics"

	// metricsPort is the port that marketplace exposes its metrics over http.
	metricsPort = 8383

	// metricsTLSPort is the port that marketplace exposes its metrics over https.
	metricsTLSPort = 8081

	// ResourceTypeOpsrc indicates that the resource in an OperatorSource
	ResourceTypeOpsrc = "OperatorSource"

	// ResourceTypeCSC indicates that the resource is a CatalogSourceConfig
	ResourceTypeCSC = "CatalogSourceConfig"

	// ResourceTypeLabel is a label for indicating the type of the resource.
	ResourceTypeLabel = "customResourceType"
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
	customResourceGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "custom_resource_total",
			Help: "A gauge that stores the count of custom OperatorSource and CatalogSourceConfig resources in the cluster.",
		},
		[]string{ResourceTypeLabel},
	)
)

// ServePrometheus enables marketplace to serve prometheus metrics.
func ServePrometheus(useTLS bool, cert, key string) error {
	// Register metrics for the operator with the prometheus.
	log.Info("[metrics] Registering marketplace metrics")
	err := registerMetrics()
	if err != nil {
		log.Infof("[metrics] Unable to register marketplace metrics: %v", err)
		return err
	}

	// Wrap the default RoundTripper with middleware.
	log.Info("[metrics] Creating marketplace metrics RoundTripperFunc")
	roundTripper = appRegistryRoundTripperCounter(appRegistryRequestCounter,
		appRegistryRoundTripperDuration(appRegistryHistVec, http.DefaultTransport),
	)

	// Start the server and expose the registered metrics.
	log.Info("[metrics] Serving marketplace metrics")
	http.Handle(metricsPath, promhttp.Handler())

	if useTLS {
		tlsGetCertFn, err := filemonitor.OLMGetCertRotationFn(log.New(), cert, key)
		if err != nil {
			log.Errorf("Certificate monitoring for metrics (https) failed: %v", err)
			return err
		}

		go func() {
			httpsServer := &http.Server{
				Addr:    fmt.Sprintf(":%d", metricsTLSPort),
				Handler: nil,
				TLSConfig: &tls.Config{
					GetCertificate: tlsGetCertFn,
				},
			}
			err := httpsServer.ListenAndServeTLS("", "")
			if err != nil {
				if err == http.ErrServerClosed {
					log.Errorf("Metrics (https) server closed")
					return
				}
				log.Errorf("Metrics (https) serving failed: %v", err)
			}
		}()
	} else {
		go func() {
			err := http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil)
			if err != nil {
				if err == http.ErrServerClosed {
					log.Errorf("Metrics (http) server closed")
					return
				}
				log.Errorf("Metrics (http) serving failed: %v", err)
			}
		}()
	}
	return nil
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

	err = prometheus.Register(customResourceGaugeVec)
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

// RegisterCustomResource increases the count of the custom_resource_total metric
func RegisterCustomResource(resourceType string) {
	customResourceGaugeVec.With(prometheus.Labels{ResourceTypeLabel: resourceType}).Inc()
}

// DeregisterCustomResource decreases the count of the custom_resource_total metric
func DeregisterCustomResource(resourceType string) {
	customResourceGaugeVec.With(prometheus.Labels{ResourceTypeLabel: resourceType}).Dec()
}
