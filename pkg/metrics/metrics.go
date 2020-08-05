package metrics

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/operator-framework/operator-marketplace/pkg/filemonitor"

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
)

var (
	// Wraps the default RoundTripperFunc with functions that record prometheus metrics.
	roundTripper = http.DefaultTransport
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

	return nil
}
