package cli

import (
	"github.com/flashbots/mev-boost/server"
	"strings"
	"net/url"
)

type relayList []server.RelayEntry

func (r *relayList) String() string {
	return strings.Join(server.RelayEntriesToStrings(*r), ",")
}

func (r *relayList) Set(value string) error {
	relay, err := server.NewRelayEntry(value)
	if err != nil {
		return err
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

func (rm *relayMonitorList) Set(value string) error {
	relayMonitor, err := url.Parse(value)
	if err != nil {
		return err
	}
	*rm = append(*rm, relayMonitor)
	return nil
}
