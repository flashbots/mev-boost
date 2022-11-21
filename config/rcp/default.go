package rcp

import (
	"github.com/flashbots/mev-boost/config/relay"
)

type Default struct {
	relays relay.Set
}

func NewDefault(relays relay.Set) *Default {
	return &Default{
		relays: relays,
	}
}

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
