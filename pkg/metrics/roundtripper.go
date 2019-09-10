package metrics

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/operator-framework/operator-marketplace/pkg/defaults"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	quayNamespaceQueryKey = "namespace"

	genericOpSrcName = "non-default-opsrc"

	codeLabel = "code"

	methodLabel = "method"

	opSrcLabel = "opsrc"
)

// appRegistryRoundTripperCounter is a middleware that wraps the provided
// http.RoundTripper to observe the request result with the provided CounterVec.
func appRegistryRoundTripperCounter(counter *prometheus.CounterVec, next http.RoundTripper) promhttp.RoundTripperFunc {
	return promhttp.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		resp, err := next.RoundTrip(r)
		if err == nil {
			opsrc, opsrcFound := extractOpSrcName(r.URL)
			if opsrcFound {
				counter.With(labels(r.Method, resp.StatusCode, opsrc)).Inc()
			}
		}
		return resp, err
	})
}

// appRegistryRoundTripperDuration is a middleware that wraps the provided
// http.RoundTripper to observe the request duration with the provided
// ObserverVec.
func appRegistryRoundTripperDuration(obs prometheus.ObserverVec, next http.RoundTripper) promhttp.RoundTripperFunc {
	return promhttp.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		start := time.Now()
		resp, err := next.RoundTrip(r)
		if err == nil {
			opsrc, opsrcFound := extractOpSrcName(r.URL)
			if opsrcFound {
				labels := prometheus.Labels{}
				labels[opSrcLabel] = sanitizeOpsrc(opsrc)
				obs.With(labels).Observe(time.Since(start).Seconds())
			}
		}
		return resp, err
	})
}

// getOpSrcName returns the opsrc name the request was made against
// a valid Operator Registry.
func extractOpSrcName(url *url.URL) (string, bool) {
	namespaceValues, ok := url.Query()[quayNamespaceQueryKey]
	if ok && len(namespaceValues) > 0 {
		return namespaceValues[0], true
	}

	return "", false
}

// labels adds the expected labels for the AppRegistry Counter.
func labels(reqMethod string, status int, quayNamespace string) prometheus.Labels {
	labels := prometheus.Labels{}
	labels[methodLabel] = strings.ToLower(reqMethod)
	labels[codeLabel] = sanitizeCode(status)
	labels[opSrcLabel] = sanitizeOpsrc(quayNamespace)

	return labels
}

// sanitizeOpsrc returns the opsrc name if it
// matches a default OperatorSource. Otherwise, a generic
// name is returned to prevent leaking customer data to telemetry.
func sanitizeOpsrc(quayNamespace string) string {
	if defaults.IsDefaultSource(quayNamespace) {
		return quayNamespace
	}
	return genericOpSrcName
}

// If the wrapped http.Handler has not set a status code, i.e. the value is
// currently 0, santizeCode will return 200, for consistency with behavior in
// the stdlib.
func sanitizeCode(s int) string {
	if s == 0 {
		s = 200
	}
	return strconv.Itoa(s)
}
