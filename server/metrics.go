package server

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const Namespace = "mev_boost"

var (
	metricsInitOnce  sync.Once
	BeaconNodeStatus *prometheus.CounterVec
	RelayHeaderValue *prometheus.GaugeVec
	RelayLatency     *prometheus.HistogramVec
	RelayStatusCode  *prometheus.CounterVec
)

func RegisterMetrics(registry *prometheus.Registry) {
	metricsInitOnce.Do(func() {
		BeaconNodeStatus = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "beacon_node_status_code",
				Help:      "http status code returned to beacon node",
			},
			[]string{"http_status_code", "endpoint"},
		)

		RelayHeaderValue = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Name:      "relay_header_value",
				Help:      "header value in gwei delivered by relay",
			},
			[]string{"relay"},
		)

		RelayLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Name:      "relay_latency",
				Help:      "http latency by relay",
			},
			[]string{"endpoint", "relay"},
		)

		RelayStatusCode = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "relay_status_code",
				Help:      "http status code received by relay",
			},
			[]string{"http_status_code", "endpoint", "relay"},
		)

		registry.MustRegister(
			BeaconNodeStatus,
			RelayHeaderValue,
			RelayLatency,
			RelayStatusCode,
		)
	})
}
