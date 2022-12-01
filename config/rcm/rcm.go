package rcm

import (
	"errors"

	"github.com/flashbots/mev-boost/config/relay"
)

// Errors returned by the package.
var (
	ErrConfigProviderFailure        = errors.New("config provider failure")
	ErrInvalidProposerConfig        = errors.New("invalid proposer config")
	ErrEmptyBuilderRelays           = errors.New("builder is enabled but has no relays")
	ErrCannotPopulateProposerRelays = errors.New("cannot populate proposer relays")
	ErrCannotPopulateDefaultRelays  = errors.New("cannot populate default relays")
)

// ConfigProvider provider relay configuration.
type ConfigProvider func() (*relay.Config, error)

// RelayRegistry provides read-only registry methods.
type RelayRegistry interface {
	RelaysForValidator(key relay.ValidatorPublicKey) relay.List
	AllRelays() relay.List
}
