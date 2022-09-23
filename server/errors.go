package server

import "fmt"

// ErrMissingRelayPubkey is returned if a new RelayEntry URL has no public key
var ErrMissingRelayPubkey = fmt.Errorf("missing relay public key")
