package common

import "fmt"

var (
	// ErrMissingRelayPubkey is returned if a new RelayEntry URL has no public key
	ErrMissingRelayPubkey = fmt.Errorf("missing relay public key")
)
