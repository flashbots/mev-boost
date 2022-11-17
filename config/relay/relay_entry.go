package relay

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/flashbots/go-boost-utils/types"
)

// ErrMissingRelayPubKey is returned if a new Entry URL has no public key
var ErrMissingRelayPubKey = fmt.Errorf("missing relay public key")

// Entry represents a relay that mev-boost connects to.
type Entry struct {
	PublicKey types.PublicKey
	URL       *url.URL
}

func (r Entry) String() string {
	return r.URL.String()
}

func (r Entry) PubKey() types.PublicKey {
	return r.PublicKey
}

func (r Entry) RelayURL() *url.URL {
	return r.URL
}

// GetURI returns the full request URI with scheme, host, path and args for the relay.
func (r Entry) GetURI(path string) string {
	return GetURI(r.URL, path)
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (entry Entry, err error) {
	// Add protocol scheme prefix if it does not exist.
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "http://" + relayURL
	}

	// Parse the provided relay's URL and save the parsed URL in the Entry.
	entry.URL, err = url.ParseRequestURI(relayURL)
	if err != nil {
		return entry, err
	}

	// Extract the relay's public key from the parsed URL.
	if entry.URL.User.Username() == "" {
		return entry, ErrMissingRelayPubKey
	}

	err = entry.PublicKey.UnmarshalText([]byte(entry.URL.User.Username()))
	return entry, err
}

// EntriesToStrings returns the string representation of a list of relay entries
func EntriesToStrings(relays []Entry) []string {
	ret := make([]string, len(relays))
	for i, entry := range relays {
		ret[i] = entry.String()
	}
	return ret
}

// GetURI returns the full request URI with scheme, host, path and args.
func GetURI(url *url.URL, path string) string {
	u2 := *url
	u2.User = nil
	u2.Path = path
	return u2.String()
}
