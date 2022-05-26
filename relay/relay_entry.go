package relay

import (
	"github.com/flashbots/go-boost-utils/types"
	"net/url"
	"strings"
)

// Address will be schema://hostname:port
// PublicKey holds the relay's BLS public key used to verify message signatures.
type Entry struct {
	Address   string
	PublicKey types.PublicKey
	URL       *url.URL
}

func (r *Entry) String() string {
	return r.URL.String()
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (entry Entry, err error) {
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
