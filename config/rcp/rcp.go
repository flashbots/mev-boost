package rcp

import (
	"net/url"

	"github.com/flashbots/go-boost-utils/types"
)

type ValidatorPublicKey = string

type RelayEntry interface {
	String() string
	PubKey() types.PublicKey
	RelayURL() *url.URL
	GetURI(path string) string
}

type RelayConfigProvider interface {
	RelaysByValidatorPublicKey(publicKey ValidatorPublicKey) ([]RelayEntry, error)
	RelaysByValidatorIndex(validatorIndex uint64) ([]RelayEntry, error)
}
