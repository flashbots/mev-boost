package relay

import (
	"errors"
	"net/url"
	"strings"
)

var ErrDuplicateEntry = errors.New("duplicate entry")

type List []Entry

func (r *List) String() string {
	return strings.Join(EntriesToStrings(*r), ",")
}

func (r *List) Contains(relay Entry) bool {
	for _, entry := range *r {
		if relay.String() == entry.String() {
			return true
		}
	}
	return false
}

func (r *List) Set(value string) error {
	relay, err := NewRelayEntry(value)
	if err != nil {
		return err
	}
	if r.Contains(relay) {
		return ErrDuplicateEntry
	}
	*r = append(*r, relay)
	return nil
}

type MonitorList []*url.URL

func (rm *MonitorList) String() string {
	relayMonitors := []string{}
	for _, relayMonitor := range *rm {
		relayMonitors = append(relayMonitors, relayMonitor.String())
	}
	return strings.Join(relayMonitors, ",")
}

func (rm *MonitorList) Contains(relayMonitor *url.URL) bool {
	for _, entry := range *rm {
		if relayMonitor.String() == entry.String() {
			return true
		}
	}
	return false
}

func (rm *MonitorList) Set(value string) error {
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
