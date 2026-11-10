package types

import "errors"

// ErrMissingRelayPubkey is returned if a new RelayEntry URL has no public key.
var ErrMissingRelayPubkey = errors.New("missing builder public key")

// ErrDuplicateAuthData is returned when two builders in the same set expect the same auth data.
var ErrDuplicateAuthData = errors.New("builders share the same auth data, set a distinct auth_data for each in the -config YAML")

// ErrPointAtInfinityPubkey is returned if a new RelayEntry URL has point-at-infinity public key.
var ErrPointAtInfinityPubkey = errors.New("builder public key cannot be the point-at-infinity")

// ErrInvalidContentType is returned when the response content type is invalid.
var ErrInvalidContentType = errors.New("invalid content type")

// ErrMissingEthConsensusVersion is returned when the response is octet-stream but there is no "Eth-Consensus-Version" header.
var ErrMissingEthConsensusVersion = errors.New("missing eth-consensus-version")
