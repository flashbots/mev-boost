package server

import "fmt"

// ErrMissingRelayPubkey is returned if a new RelayEntry URL has no public key.
var ErrMissingRelayPubkey = fmt.Errorf("missing relay public key")

// ErrPointAtInfinityPubkey is returned if a new RelayEntry URL has an all-zero public key.
var ErrPointAtInfinityPubkey = fmt.Errorf("relay public key cannot be the point-at-infinity")
