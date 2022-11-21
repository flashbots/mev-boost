package relay

import (
	"errors"
	"net/url"
	"strings"
)

var ErrDuplicateEntry = errors.New("duplicate entry")

type MonitorList []*url.URL

func (rm *MonitorList) String() string {
	relayMonitors := make([]string, len(*rm))
	for i, relayMonitor := range *rm {
		relayMonitors[i] = relayMonitor.String()
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
