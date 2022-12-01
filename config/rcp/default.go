package rcp

import (
	"github.com/flashbots/mev-boost/config/relay"
)

// Default returns a relay config based on the given relay.Set.
type Default struct {
	relays relay.Set
}

// NewDefault creates a new instance of Default.
//
// It takes a relay set as an argument.
// If the given relay set is nil, a new one will be created.
func NewDefault(relays relay.Set) *Default {
	if relays == nil {
		relays = relay.NewRelaySet()
	}

	return &Default{
		relays: relays,
	}
}

// FetchConfig returns a relay config based on the given relay.Set.
//
// It always returns the *relay.Config with only `default_config` section populated.
// It doesn't have `proposer_config` section.
// It doesn't return any errors.
func (d *Default) FetchConfig() (*relay.Config, error) {
	cfg := &relay.Config{
		DefaultConfig: relay.Relay{
			Builder: relay.Builder{
				Enabled: true,
				Relays:  d.relays.ToStringSlice(),
			},
		},
	}

	return cfg, nil
}
