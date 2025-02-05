package types

import "errors"

// ErrMissingRelayPubkey is returned if a new RelayEntry URL has no public key.
var ErrMissingRelayPubkey = errors.New("missing relay public key")

// ErrPointAtInfinityPubkey is returned if a new RelayEntry URL has point-at-infinity public key.
var ErrPointAtInfinityPubkey = errors.New("relay public key cannot be the point-at-infinity")

// ErrInvalidContentType is returned when the response content type is invalid.
var ErrInvalidContentType = errors.New("invalid content type")

// ErrMissingEthConsensusVersion is returned when the response is octet-stream but there is no "Eth-Consensus-Version" header.
var ErrMissingEthConsensusVersion = errors.New("missing eth-consensus-version")
