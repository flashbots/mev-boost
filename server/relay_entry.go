package server

import (
	"net/url"
	"strings"

	"github.com/flashbots/go-boost-utils/types"
)

// RelayEntry represents a relay that mev-boost connects to.
// Address will be schema://hostname:port
// PublicKey holds the relay's BLS public key used to verify message signatures.
type RelayEntry struct {
	Address   string
	PublicKey types.PublicKey
	URL       *url.URL

	// Naive reputation mechanism OKResponses/NOKResponses is the hit rate of the relayer
	// TODO: uint64 will eventually overflow
	OKResponses  uint64
	NOKResponses uint64
}

func (r *RelayEntry) String() string {
	return r.URL.String()
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (entry RelayEntry, err error) {
	// Add protocol scheme prefix if it does not exist.
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "http://" + relayURL
	}

	// Parse the provided relay's URL and save the parsed URL in the RelayEntry.
	entry.URL, err = url.Parse(relayURL)
	if err != nil {
		return entry, err
	}

	// Build the relay's address.
	entry.Address = entry.URL.Scheme + "://" + entry.URL.Host

	// Extract the relay's public key from the parsed URL.
	// TODO: Remove the if condition, as it is mandatory to verify relay's message signature.
	if entry.URL.User.Username() != "" {
		err = entry.PublicKey.UnmarshalText([]byte(entry.URL.User.Username()))
	}

	return entry, err
}

// Rates the reputation of a relay based on the OK vs NOK requests
func (r *RelayEntry) GetRelayReputation() float64 {
	if r.NOKResponses == 0 && r.OKResponses == 0 {
		// init reputation
		return 10
	} else if r.NOKResponses == 0 {
		// never failed
		return float64(r.OKResponses)
	}
	return float64(r.OKResponses) / float64(r.NOKResponses)
}
