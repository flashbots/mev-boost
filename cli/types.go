package cli

import (
	"errors"
	"strings"

	"github.com/flashbots/mev-boost/server/types"
)

var errDuplicateEntry = errors.New("duplicate entry")

type relayList []types.RelayEntry

func (r *relayList) String() string {
	return strings.Join(types.RelayEntriesToStrings(*r), ",")
}

func (r *relayList) Contains(relay types.RelayEntry) bool {
	for _, entry := range *r {
		if relay.String() == entry.String() {
			return true
		}
	}
	return false
}

func (r *relayList) Set(value string) error {
	relay, err := types.NewRelayEntry(value)
	if err != nil {
		return err
	}
	if r.Contains(relay) {
		return errDuplicateEntry
	}
	*r = append(*r, relay)
	return nil
}
