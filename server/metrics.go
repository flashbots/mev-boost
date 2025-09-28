package server

import (
	"fmt"

	"github.com/VictoriaMetrics/metrics"
)

var winningBidValue = metrics.NewHistogram("mev_boost_winning_bid_value")

const (
	beaconNodeStatusLabel = `mev_boost_beacon_node_status_code_total{http_status_code="%s",endpoint="%s"}`
	bidValuesLabel        = `mev_boost_bid_values{relay="%s"}`
	bidsBelowMinBidLabel  = `mev_boost_bids_below_min_bid_total{relay="%s"}`
	relayLatencyLabel     = `mev_boost_relay_latency{endpoint="%s",relay="%s"}`
	relayStatusCodeLabel  = `mev_boost_relay_status_code_total{http_status_code="%s",endpoint="%s",relay="%s"}`
	relayLastSlotLabel    = `mev_boost_relay_last_slot{relay="%s"}`
	msIntoSlotLabel       = `mev_boost_millisec_into_slot{endpoint="%s"}`
)

func IncrementBeaconNodeStatus(status, endpoint string) {
	l := fmt.Sprintf(beaconNodeStatusLabel, status, endpoint)
	metrics.GetOrCreateCounter(l).Inc()
}

func RecordBidValue(relay string, value float64) {
	l := fmt.Sprintf(bidValuesLabel, relay)
	metrics.GetOrCreateHistogram(l).Update(value)
}

func IncrementBidBelowMinBid(relay string) {
	l := fmt.Sprintf(bidsBelowMinBidLabel, relay)
	metrics.GetOrCreateCounter(l).Inc()
}

func RecordWinningBidValue(value float64) {
	winningBidValue.Update(value)
}

func RecordRelayLatency(endpoint, relay string, latency float64) {
	l := fmt.Sprintf(relayLatencyLabel, endpoint, relay)
	metrics.GetOrCreateHistogram(l).Update(latency)
}

func RecordRelayStatusCode(httpStatus, endpoint, relay string) {
	l := fmt.Sprintf(relayStatusCodeLabel, httpStatus, endpoint, relay)
	metrics.GetOrCreateCounter(l).Inc()
}

func RecordRelayLastSlot(relay string, slot uint64) {
	l := fmt.Sprintf(relayLastSlotLabel, relay)
	metrics.GetOrCreateGauge(l, nil).Set(float64(slot))
}

func RecordMsIntoSlot(endpoint string, ms float64) {
	l := fmt.Sprintf(msIntoSlotLabel, endpoint)
	metrics.GetOrCreateHistogram(l).Update(ms)
}
