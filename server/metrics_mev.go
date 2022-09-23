package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricNamespace = "mev_boost"

	labelRelayHost = "relay_host"
	labelVersion   = "version"
	labelPubkey    = "pubkey"
)

// MevMetrics stores the pointers to server metricOpts
type MevMetrics struct {
	validatorIdentities *prometheus.GaugeVec
	version             *prometheus.GaugeVec
	genesisForkVersion  prometheus.Gauge
	relays              *prometheus.GaugeVec
	relayCount          prometheus.Gauge
	bids                prometheus.Gauge
}

// NewMevMetrics takes in a prometheus registry and initializes
// and registers relay metrics. It returns those registered MevMetrics.
func NewMevMetrics(r prometheus.Registerer) *MevMetrics {
	return &MevMetrics{
		validatorIdentities: promauto.With(r).NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: metricNamespace,
				Name:      "validator_info",
				Help:      "validator identity information",
			}, []string{labelPubkey}),
		version: promauto.With(r).NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: metricNamespace,
				Name:      "version",
				Help:      "the service version",
			}, []string{labelVersion}),
		genesisForkVersion: promauto.With(r).NewGauge(
			prometheus.GaugeOpts{
				Namespace: metricNamespace,
				Name:      "genesis_fork_version",
				Help:      "the genesis fork version",
			}),
		relays: promauto.With(r).NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: metricNamespace,
				Name:      "relays",
				Help:      "the relay hosts",
			}, []string{labelRelayHost}),
		relayCount: promauto.With(r).NewGauge(
			prometheus.GaugeOpts{
				Namespace: metricNamespace,
				Name:      "relays_total",
				Help:      "the total relay_count configured",
			}),
		bids: promauto.With(r).NewGauge(
			prometheus.GaugeOpts{
				Namespace: metricNamespace,
				Name:      "bids_total",
				Help:      "the total bids currently active",
			}),
	}
}
