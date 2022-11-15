package server

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/flashbots/go-boost-utils/types"
)

// RelayEntry represents a relay that mev-boost connects to.
type RelayEntry struct {
	PublicKey types.PublicKey
	URL       *url.URL
	Weight    float64
}

func (r *RelayEntry) String() string {
	return r.URL.String()
}

// GetURI returns the full request URI with scheme, host, path and args for the relay.
func (r *RelayEntry) GetURI(path string) string {
	return GetURI(r.URL, path)
}

// NewRelayEntry creates a new instance based on an input string, an optional weight prefix is supported.
// relayURL can be WEIGHT#IP@PORT, IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (entry RelayEntry, err error) {
	// Entry weight is 1 by default.
	weight := 1.0

	if strings.Contains(relayURL, "#") {
		parts := strings.Split(relayURL, "#")
		if len(parts) != 2 {
			return entry, fmt.Errorf("invalid weighted relay entry format: %s", relayURL)
		}

		// Parse the weight as a float
		weight, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return entry, err
		}

		// Parse the relay entry
		relayURL = parts[1]
	}

	entry.Weight = weight

	// Add protocol scheme prefix if it does not exist.
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "http://" + relayURL
	}

	// Parse the provided relay's URL and save the parsed URL in the RelayEntry.
	entry.URL, err = url.ParseRequestURI(relayURL)
	if err != nil {
		return entry, err
	}

	// Extract the relay's public key from the parsed URL.
	if entry.URL.User.Username() == "" {
		return entry, ErrMissingRelayPubkey
	}

	err = entry.PublicKey.UnmarshalText([]byte(entry.URL.User.Username()))
	return entry, err
}

// RelayEntriesToStrings returns the string representation of a list of relay entries
func RelayEntriesToStrings(relays []RelayEntry) []string {
	ret := make([]string, len(relays))
	for i, entry := range relays {
		ret[i] = entry.String()
	}
	return ret
}
