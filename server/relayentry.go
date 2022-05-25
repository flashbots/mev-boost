package server

import (
	"net/url"
	"strings"

	"github.com/flashbots/go-boost-utils/types"
)

// RelayEntry represents a relay that mev-boost connects to
// Address will be schema://hostname:port
type RelayEntry struct {
	Address string
	Pubkey  types.PublicKey
	URL     *url.URL
}

func (r *RelayEntry) String() string {
	return r.URL.String()
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (entry RelayEntry, err error) {
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "http://" + relayURL
	}

	entry.URL, err = url.Parse(relayURL)
	if err != nil {
		return entry, err
	}
	entry.Address = entry.URL.Scheme + "://" + entry.URL.Host
	if entry.URL.User.Username() != "" {
		err = entry.Pubkey.UnmarshalText([]byte(entry.URL.User.Username()))
	}
	return entry, err
}
