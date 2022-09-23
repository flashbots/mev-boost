package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricNamespace = "mev_boost"
)

// MevMetrics stores the pointers to server metricOpts
type MevMetrics struct {
	relays   prometheus.Gauge
	bids     prometheus.Gauge
	withheld prometheus.Counter
}

// NewMevMetrics takes in a prometheus registry and initializes
// and registers relay metrics. It returns those registered MevMetrics.
func NewMevMetrics(r prometheus.Registerer) *MevMetrics {
	return &MevMetrics{
		relays: promauto.With(r).NewGauge(
			prometheus.GaugeOpts{
				Name: "mev_boost_relays_total",
				Help: "the total relays configured",
			}),
		bids: promauto.With(r).NewGauge(
			prometheus.GaugeOpts{
				Name: "mev_boost_bids_total",
				Help: "the total bids currently active",
			}),
		withheld: promauto.With(r).NewCounter(
			prometheus.CounterOpts{
				Name: "mev_boost_withheld_bids_total",
				Help: "the total number of failed or withheld bids",
			}),
	}
}
