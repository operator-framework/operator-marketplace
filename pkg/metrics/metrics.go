package metrics

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/operator-framework/operator-marketplace/pkg/filemonitor"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

const (
	// metricsPath is the path that marketplace exposes its metrics at.
	metricsPath = "/metrics"

	// metricsPort is the port that marketplace exposes its metrics over http.
	metricsPort = 8383

	// metricsTLSPort is the port that marketplace exposes its metrics over https.
	metricsTLSPort = 8081
)

// ServePrometheus enables marketplace to serve prometheus metrics.
func ServePrometheus(cert, key string) error {
	// Register metrics for the operator with the prometheus.
	logrus.Info("[metrics] Registering marketplace metrics")

	err := registerMetrics()
	if err != nil {
		logrus.Infof("[metrics] Unable to register marketplace metrics: %v", err)
		return err
	}

	// Start the server and expose the registered metrics.
	logrus.Info("[metrics] Serving marketplace metrics")
	http.Handle(metricsPath, promhttp.Handler())

	if useTLS(cert, key) {
		tlsGetCertFn, err := filemonitor.OLMGetCertRotationFn(logrus.New(), cert, key)
		if err != nil {
			logrus.Errorf("Certificate monitoring for metrics (https) failed: %v", err)
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
					logrus.Errorf("Metrics (https) server closed")
					return
				}
				logrus.Errorf("Metrics (https) serving failed: %v", err)
			}
		}()
		return nil
	}

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), nil)
		if err != nil {
			if err == http.ErrServerClosed {
				logrus.Errorf("Metrics (http) server closed")
				return
			}
			logrus.Errorf("Metrics (http) serving failed: %v", err)
		}
	}()

	return nil
}

// registerMetrics registers marketplace prometheus metrics.
func registerMetrics() error {
	// Register all of the metrics in the standard registry.
	return nil
}

func useTLS(certPath, keyPath string) bool {
	if certPath != "" && keyPath == "" || certPath == "" && keyPath != "" {
		logrus.Warn("both --tls-key and --tls-crt must be provided for TLS to be enabled, falling back to non-https")
		return false
	}
	if certPath == "" && keyPath == "" {
		logrus.Info("TLS keys not set, using non-https for metrics")
		return false
	}

	logrus.Info("TLS keys set, using https for metrics")
	return true
}
