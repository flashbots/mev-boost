package main

import (
	"encoding/json"
)

type validatorRegistration struct {
	Enabled       bool     `json:"enabled"`
	BuilderRelays []string `json:"builder_relays"`
	GasLimit      string   `json:"gas_limit"`
}

type proposerConfig struct {
	FeeRecipient          string                `json:"fee_recipient"`
	ValidatorRegistration validatorRegistration `json:"validator_registration"`
}

// configuration is used by mev-boost to allow validators to only trust a specific list of relays.
// It also contains the validators preferences.
type configuration struct {
	BuilderRelaysGroups map[string][]string       `json:"builder_relays_groups"`
	ProposerConfig      map[string]proposerConfig `json:"proposer_config"`
}

// configFromJSON builds a new configuration from a given JSON raw object.
func configFromJSON(raw json.RawMessage) (*configuration, error) {
	config := &configuration{
		BuilderRelaysGroups: make(map[string][]string),
		ProposerConfig:      make(map[string]proposerConfig),
	}

	// Tries to unmarshal content in JSON.
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}

	return config, nil
}
