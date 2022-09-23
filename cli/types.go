package cli

import (
	"github.com/flashbots/mev-boost/server"
	"strings"
	"net/url"
	"errors"
)

var ErrDuplicateEntry = errors.New("duplicate entry")

type relayList []server.RelayEntry

func (r *relayList) String() string {
	return strings.Join(server.RelayEntriesToStrings(*r), ",")
}

func (r *relayList) Contains(relay server.RelayEntry) bool {
	for _, entry := range *r {
		if relay.String() == entry.String() {
			return true
		}
	}
	return false
}

func (r *relayList) Set(value string) error {
	relay, err := server.NewRelayEntry(value)
	if err != nil {
		return err
	}
	if r.Contains(relay) {
		return ErrDuplicateEntry
	}
	*r = append(*r, relay)
	return nil
}

type relayMonitorList []*url.URL

func (rm *relayMonitorList) String() string {
	relayMonitors := make([]string, len(*rm))
	for _, relayMonitor := range *rm {
		relayMonitors = append(relayMonitors, relayMonitor.String())
	}
	return strings.Join(relayMonitors, ",")
}

func (rm *relayMonitorList) Contains(relayMonitor *url.URL) bool {
	for _, entry := range *rm {
		if relayMonitor.String() == entry.String() {
			return true
		}
	}
	return false
}

func (rm *relayMonitorList) Set(value string) error {
	relayMonitor, err := url.Parse(value)
	if err != nil {
		return err
	}
	if rm.Contains(relayMonitor) {
		return ErrDuplicateEntry
	}
	*rm = append(*rm, relayMonitor)
	return nil
}
