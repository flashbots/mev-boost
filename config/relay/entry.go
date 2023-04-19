package relay

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/flashbots/go-boost-utils/types"
)

var (
	// ErrMissingRelayPubKey is returned if a new Entry relayURL has no public key.
	ErrMissingRelayPubKey = fmt.Errorf("missing relay public key")
	// ErrInvalidRelayURL is returned if a new Entry has malformed relayURL.
	ErrInvalidRelayURL = fmt.Errorf("invalid relay url")
	// ErrPointAtInfinityPubkey is returned if a new RelayEntry URL has an all-zero public key.
	ErrPointAtInfinityPubkey = fmt.Errorf("relay public key cannot be the point-at-infinity")
)

type (
	ValidatorPublicKey = string
	ValidatorIndex     = uint64
)

// The point-at-infinity is 48 zero bytes.
var pointAtInfinityPubKey = [48]byte{}

// Entry represents a relay that mev-boost connects to.
type Entry struct {
	publicKey types.PublicKey
	relayURL  *url.URL
}

func (r Entry) String() string {
	return r.relayURL.String()
}

func (r Entry) PublicKey() types.PublicKey {
	return r.publicKey
}

func (r Entry) RelayURL() *url.URL {
	return r.relayURL
}

// GetURI returns the full request URI with scheme, host, path and args for the relay.
func (r Entry) GetURI(path string) string {
	return GetURI(r.relayURL, path)
}

// NewRelayEntry creates a new instance based on an input string
// relayURL can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
func NewRelayEntry(relayURL string) (Entry, error) {
	relayURL = strings.TrimSpace(relayURL)

	// Add protocol scheme prefix if it does not exist.
	if !strings.HasPrefix(relayURL, "http") {
		relayURL = "http://" + relayURL
	}

	// Parse the provided relay's relayURL and save the parsed relayURL in the Entry.
	parsedURL, err := url.ParseRequestURI(relayURL)
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %w: %s", ErrInvalidRelayURL, err, relayURL)
	}

	pubKeyHex := parsedURL.User.Username()

	// Extract the relay's public key from the parsed relayURL.
	if pubKeyHex == "" {
		return Entry{}, ErrMissingRelayPubKey
	}

	var pubKey types.PublicKey
	if err := pubKey.UnmarshalText([]byte(pubKeyHex)); err != nil {
		return Entry{}, fmt.Errorf("%w: %s", err, relayURL)
	}

	// Check if the public key is the point-at-infinity.
	if bytes.Equal(pubKey[:], pointAtInfinityPubKey[:]) {
		return Entry{}, ErrPointAtInfinityPubkey
	}

	return Entry{
		relayURL:  parsedURL,
		publicKey: pubKey,
	}, nil
}

// GetURI returns the full request URI with scheme, host, path and args.
func GetURI(url *url.URL, path string) string {
	u2 := *url
	u2.User = nil
	u2.Path = path

	return u2.String()
}
