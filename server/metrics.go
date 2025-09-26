package server

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const Namespace = "mev_boost"

// labels

const (
	Endpoint       = "endpoint"
	Relay          = "relay"
	HTTPStatusCode = "http_status_code"
)

var (
	metricsInitOnce  sync.Once
	BeaconNodeStatus *prometheus.CounterVec
	BidValues        *prometheus.HistogramVec
	BidsBelowMinBid  *prometheus.CounterVec
	WinningBidValue  *prometheus.HistogramVec
	RelayLatency     *prometheus.HistogramVec
	RelayStatusCode  *prometheus.CounterVec
	RelayLastSlot    *prometheus.GaugeVec
    MsIntoSlot       *prometheus.HistogramVec
)

func RegisterMetrics(registry *prometheus.Registry) {
	metricsInitOnce.Do(func() {
		BeaconNodeStatus = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "beacon_node_status_code_total",
				Help:      "http status code returned to beacon node",
			},
			[]string{HTTPStatusCode, Endpoint},
		)

		BidValues = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Name:      "bid_values",
				Help:      "Value of the bids seen per relay",
			},
			[]string{Relay},
		)

		BidsBelowMinBid = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "bids_below_min_bid",
				Help:      "Number of bids per relay which are below the min bid",
			},
			[]string{Relay},
		)

		WinningBidValue = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Name:      "winning_bid_value",
				Help:      "Value of the winning bid",
			},
			[]string{},
		)

		RelayLatency = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Name:      "relay_latency",
				Help:      "http latency by relay",
			},
			[]string{Endpoint, Relay},
		)

		RelayStatusCode = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: Namespace,
				Name:      "relay_status_code_total",
				Help:      "http status code received by relay",
			},
			[]string{HTTPStatusCode, Endpoint, Relay},
		)

		RelayLastSlot = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: Namespace,
				Name:      "relay_last_slot",
				Help:      "Last slot for which relay delivered a header",
			},
			[]string{Relay},
		)

		MsIntoSlot = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: Namespace,
				Name:      "millisec_into_slot",
				Help:      "Milliseconds into the slot when endpoint was called",
			},
			[]string{Endpoint},
		)
		registry.MustRegister(
			BeaconNodeStatus,
			BidValues,
			BidsBelowMinBid,
			WinningBidValue,
			RelayLatency,
			RelayStatusCode,
			RelayLastSlot,
            MsIntoSlot,
		)
	})
}

func IncrementBeaconNodeStatus(status, endpoint string) {
	if BeaconNodeStatus != nil {
		BeaconNodeStatus.WithLabelValues(status, endpoint).Inc()
	}
}

func RecordBidValue(relay string, value float64) {
	if BidValues != nil {
		BidValues.WithLabelValues(relay).Observe(value)
	}
}

func IncrementBidBelowMinBid(relay string) {
	if BidsBelowMinBid != nil {
		BidsBelowMinBid.WithLabelValues(relay).Inc()
	}
}

func RecordWinningBidValue(value float64) {
	if WinningBidValue != nil {
		WinningBidValue.WithLabelValues().Observe(value)
	}
}

func RecordRelayLatency(endpoint, relay string, latency float64) {
	if RelayLatency != nil {
		RelayLatency.WithLabelValues(endpoint, relay).Observe(latency)
	}
}

func RecordRelayStatusCode(httpStatus, endpoint, relay string) {
	if RelayStatusCode != nil {
		RelayStatusCode.WithLabelValues(httpStatus, endpoint, relay).Inc()
	}
}

func RecordRelayLastSlot(relay string, slot uint64) {
	if RelayLastSlot != nil {
		RelayLastSlot.WithLabelValues(relay).Set(float64(slot))
	}
}
