package rcp

import (
	"github.com/flashbots/go-boost-utils/types"
)

type ValidatorPublicKey = string

type RelayEntry interface {
	String() string
	PubKey() types.PublicKey
	GetURI(path string) string
}

type RelayConfigProvider interface {
	RelaysByValidatorPublicKey(publicKey ValidatorPublicKey) ([]RelayEntry, error)
}
