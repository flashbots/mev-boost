package server

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// The point-at-infinity is 48 zero bytes.
var pointAtInfinityPubkey = [48]byte{}

// RelayEntry represents a relay that mev-boost connects to.
type RelayEntry struct {
	PublicKey phase0.BLSPubKey
	URL       *url.URL
}

func (r *RelayEntry) String() string {
	return r.URL.String()
}

// GetURI returns the full request URI with scheme, host, path and args for the relay.
func (r *RelayEntry) GetURI(path string) string {
	return GetURI(r.URL, path)
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (entry RelayEntry, err error) {
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

	// Convert the username string to a public key.
	pubkey, err := hexutil.Decode(entry.URL.User.Username())
	if err != nil {
		return entry, err
	}

	// Ensure that the provide public key is the correct length
	if len(pubkey) != len(pointAtInfinityPubkey) {
		return entry, ErrInvalidLengthPubkey
	}

	// Check if the public key is the point-at-infinity.
	if bytes.Equal(pubkey, pointAtInfinityPubkey[:]) {
		return entry, ErrPointAtInfinityPubkey
	}

	copy(entry.PublicKey[:], pubkey)
	return entry, nil
}

// RelayEntriesToStrings returns the string representation of a list of relay entries
func RelayEntriesToStrings(relays []RelayEntry) []string {
	ret := make([]string, len(relays))
	for i, entry := range relays {
		ret[i] = entry.String()
	}
	return ret
}
