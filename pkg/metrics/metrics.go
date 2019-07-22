package metrics

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// Error to be returned when marketplace is unable to reach app registry
// to download manifests.
type OpsrcError string

const (
	// Default values which are to be used as labels
	// for prometheus metrics.
	Error     = "opsrc_error"
	OpsrcName = "opsrc_name"

	// Path, Port and Host where custom metrics would be exposed.
	MetricsPath = "/metrics"
	MetricsPort = 8383
	MetricsHost = "0.0.0.0"

	// UnreachableError is the generic error message returned when an opsrc fails to contact app registry.
	UnreachableError OpsrcError = "error_contacting_app_registry"

	// RegexPattern to extract HTTP Error code from error string.
	RegexPattern = "status\\s+[1-5][0-9][0-9]"
)

var (
	// opsrcErr is a prometheus historgram vector that reports
	// the error and the name of the operator source which fails
	// to reconcile.
	opsrcErr = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "opsrc_failure_count",
			Help:    "Number of times the operator source has failed to reconcile",
			Buckets: prometheus.LinearBuckets(0, 5, 6),
		}, []string{OpsrcName, Error},
	)

	// List of custom metrics which are to be collected.
	metricsList = []prometheus.Collector{
		opsrcErr,
	}

	// count is used to keep track of the number of opsrc errors.
	count = 0.0

	// mutex is used to synchronize count.
	mutex sync.Mutex
)

// registerMetrics registers the metrics with prometheus.
func registerMetrics() error {
	for _, metric := range metricsList {
		err := prometheus.Register(metric)
		if err != nil {
			return err
		}
	}
	return nil
}

// RecordMetrics starts the server, records the custom metrics and exposes
// the metrics at the specified endpoint.
func RecordMetrics() {
	// Register metrics for the operator with the prometheus.
	registerMetrics()

	// Start the server and expose the registered metrics.
	http.Handle(MetricsPath, promhttp.Handler())
	port := fmt.Sprintf(":%d", MetricsPort)
	go http.ListenAndServe(port, nil)
}

// UpdateOpsrcErrorHist updates the histogram vector and adds observations
// based on the granularity of the bin size.
func UpdateOpsrcErrorHist(opsrc string, opsrcError OpsrcError) {
	// Increment the value of count, and record the number of errors
	// with the help of histogram vector.
	mutex.Lock()
	count++
	opsrcErr.WithLabelValues(opsrc, opsrcError.opsrcErrorToString()).Observe(count)
	mutex.Unlock()
}

// opsrcErrorToString converts the error message of OpsrcError type to string.
func (operatorError OpsrcError) opsrcErrorToString() string {
	return string(operatorError)
}

// GetHTTPErrorCode extracts and returns the error code from HTTPRespose
// when app registry is not reachable.
func GetHTTPErrorCode(err string) string {
	re, error := regexp.Compile(RegexPattern)
	if error != nil {
		log.Info("Error in compiling regex pattern", err)
	}
	code := re.FindString(err)
	return code
}

// ErrorMessage returns the error which is to be published in telemetry.
func ErrorMessage(code string) OpsrcError {
	if code != "" {
		return OpsrcError(fmt.Sprintf("Error in downloading manifests. Returned response - (http %s)", code))
	}
	return UnreachableError
}
