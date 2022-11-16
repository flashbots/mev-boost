package rcm

import (
	"net/url"

	"github.com/flashbots/go-boost-utils/types"
)

type (
	ValidatorPublicKey = string
	ValidatorIndex     = uint64
)

type RelayEntry interface {
	PubKey() types.PublicKey
	RelayURL() *url.URL
	GetURI(path string) string
}
