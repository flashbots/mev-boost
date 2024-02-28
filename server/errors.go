package server

import "errors"

// ErrMissingRelayPubkey is returned if a new RelayEntry URL has no public key.
var ErrMissingRelayPubkey = errors.New("missing relay public key")

// ErrPointAtInfinityPubkey is returned if a new RelayEntry URL has an all-zero public key.
var ErrPointAtInfinityPubkey = errors.New("relay public key cannot be the point-at-infinity")
