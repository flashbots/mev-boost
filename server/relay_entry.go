package server

import (
	"net/url"
	"strings"

	"github.com/flashbots/go-boost-utils/types"
)

// RelayEntry represents a relay that mev-boost connects to.
type RelayEntry struct {
	PublicKey types.PublicKey
	URL       *url.URL
}

func (r *RelayEntry) String() string {
	return r.URL.String()
}

// GetURI returns the full request URI with scheme, host, path and args.
func (r *RelayEntry) GetURI(path string) string {
	u2 := *r.URL
	u2.User = nil
	u2.Path = path
	return u2.String()
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

	q := entry.URL.Query()
	q.Set("boost", Version)
	entry.URL.RawQuery = q.Encode()

	err = entry.PublicKey.UnmarshalText([]byte(entry.URL.User.Username()))
	return entry, err
}
