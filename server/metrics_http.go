package server

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	labelPath   = "path"
	labelMethod = "method"
	labelCode   = "code"
)

// OutboundHTTPMetrics stores the pointers to outbound http metrics
type OutboundHTTPMetrics struct {
	requestsSent          *prometheus.CounterVec
	requestErrors         *prometheus.CounterVec
	requestClientFailures *prometheus.CounterVec
	requestDuration       *prometheus.HistogramVec
	requestBodyBytes      *prometheus.HistogramVec
	responseBodyBytes     *prometheus.HistogramVec
}

// NewOutboundHTTPMetrics takes in a prometheus registry and initializes
// and registers server configuration. It returns those registered metrics
// these are for requests that mev-boost makes
func NewOutboundHTTPMetrics(r prometheus.Registerer) *OutboundHTTPMetrics {
	return &OutboundHTTPMetrics{
		requestsSent: promauto.With(r).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricNamespace,
				Name:      "outbound_http_requests_total",
				Help:      "the total http requests sent",
			}, []string{labelPath, labelMethod, labelCode},
		),
		requestErrors: promauto.With(r).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricNamespace,
				Name:      "outbound_http_requests_errors_total",
				Help:      "the total http requests that failed due to a response error",
			}, []string{labelPath, labelMethod, labelCode}),
		requestClientFailures: promauto.With(r).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricNamespace,
				Name:      "outbound_http_bad_requests_total",
				Help:      "the total http requests that failed due to a request error",
			}, []string{labelPath, labelMethod}),
		requestDuration: promauto.With(r).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metricNamespace,
				Name:      "outbound_http_request_duration_milliseconds",
				Help:      "the total milliseconds taken for a response",
				Buckets:   prometheus.ExponentialBuckets(50, 3, 6),
			}, []string{labelPath, labelMethod}),
		requestBodyBytes: promauto.With(r).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metricNamespace,
				Name:      "outbound_http_request_bytes",
				Help:      "the total bytes sent as http request body",
				Buckets:   prometheus.ExponentialBuckets(500, 3, 8),
			}, []string{labelPath, labelMethod}),
		responseBodyBytes: promauto.With(r).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metricNamespace,
				Name:      "outbound_http_response_bytes",
				Help:      "the total bytes sent as http response body",
				Buckets:   prometheus.ExponentialBuckets(500, 3, 8),
			}, []string{labelPath, labelMethod}),
	}
}

// InboundHTTPMetrics stores the pointers to inbound http metrics
type InboundHTTPMetrics struct {
	requestsSent      *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	requestBodyBytes  *prometheus.HistogramVec
	responseBodyBytes *prometheus.HistogramVec
}

// NewInboundHTTPMetrics takes in a prometheus registry and initializes
// and registers server configuration. It returns those registered metrics
// these are for requests made to mev-boost
func NewInboundHTTPMetrics(r prometheus.Registerer) *InboundHTTPMetrics {
	return &InboundHTTPMetrics{
		requestsSent: promauto.With(r).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricNamespace,
				Name:      "inbound_http_requests_total",
				Help:      "the total http requests sent to mev-boost",
			}, []string{labelPath, labelMethod, labelCode},
		),
		requestDuration: promauto.With(r).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metricNamespace,
				Name:      "inbound_http_request_duration_milliseconds",
				Help:      "the total milliseconds taken for a response to mev-boost",
				Buckets:   prometheus.ExponentialBuckets(50, 3, 6),
			}, []string{labelPath, labelMethod}),
		requestBodyBytes: promauto.With(r).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metricNamespace,
				Name:      "inbound_http_request_bytes",
				Help:      "the total bytes sent as http request body to mev-boost",
				Buckets:   prometheus.ExponentialBuckets(500, 3, 8),
			}, []string{labelPath, labelMethod}),
		responseBodyBytes: promauto.With(r).NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: metricNamespace,
				Name:      "inbound_http_response_bytes",
				Help:      "the total bytes sent as http response body to mev-boost",
				Buckets:   prometheus.ExponentialBuckets(500, 3, 8),
			}, []string{labelPath, labelMethod}),
	}
}

// httpStateRecorder wraps a request
type httpStateRecorder struct {
	http.ResponseWriter
	status int
	Body   []byte
}

func (r *httpStateRecorder) Header() http.Header {
	return r.ResponseWriter.Header()
}

// WriteHeader implements the ResponseWriter.WriteHeader interface
func (r *httpStateRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write implements the ResponseWriter.Write interface and sets the value
func (r *httpStateRecorder) Write(bytes []byte) (int, error) {
	r.Body = bytes
	return r.ResponseWriter.Write(bytes)

}

// Middleware wraps a http handler
type Middleware func(http.Handler) http.Handler

// Chain a helper for chaining middleware functions
func Chain(handler http.Handler, middleware ...Middleware) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

// InboundHTTPMetricMiddleware exports prometheus metrics for the http tier
func InboundHTTPMetricMiddleware(metrics *InboundHTTPMetrics) Middleware {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			start := time.Now()
			recorder := &httpStateRecorder{
				ResponseWriter: rw,
			}

			// handle the request
			handler.ServeHTTP(recorder, req)

			// do nothing if it's not a /eth request
			if !strings.HasPrefix(req.URL.Path, "/eth") {
				metrics.requestsSent.With(
					map[string]string{labelPath: "not_recorded",
						labelMethod: req.Method,
						labelCode:   strconv.Itoa(recorder.status),
					},
				).Inc()
				return
			}

			labels := map[string]string{labelPath: filterPath(req.URL.Path), labelMethod: req.Method}

			reqBodyBuffer, _ := new(bytes.Buffer).ReadFrom(req.Body)
			if reqBodyBuffer != 0 {
				metrics.responseBodyBytes.With(labels).Observe(float64(reqBodyBuffer))
			}
			if len(recorder.Body) != 0 {
				metrics.responseBodyBytes.With(labels).Observe(float64(len(recorder.Body)))
			}

			// only record duration for successful requests
			if recorder.status >= http.StatusOK && recorder.status < http.StatusMultipleChoices {
				metrics.requestDuration.With(labels).Observe(time.Since(start).Seconds())
			}

			labels[labelCode] = strconv.Itoa(recorder.status)
			metrics.requestsSent.With(labels).Inc()

		})
	}
}
