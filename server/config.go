package server

import (
	"encoding/json"
	"errors"
	"github.com/flashbots/go-boost-utils/types"
	"io/ioutil"
)

type builderRegistrationConfig struct {
	Enabled       bool     `json:"enabled"`
	BuilderRelays []string `json:"builder_relays"`
	GasLimit      string   `json:"gas_limit"`
}

type proposerConfig struct {
	FeeRecipient        string                    `json:"fee_recipient"`
	BuilderRegistration builderRegistrationConfig `json:"builder_registration"`
}

// configuration is used by mev-boost to allow validators to only trust a specific list of relays.
// It also contains the validators preferences.
type configuration struct {
	BuilderRelaysGroups map[string][]string       `json:"builder_relays_groups"`
	ProposerConfig      map[string]proposerConfig `json:"proposer_config"`
	DefaultConfig       proposerConfig            `json:"default_config"`
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

// configFromFile reads a JSON file and creates a configuration out of it.
func configFromFile(filename string) (*configuration, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return configFromJSON(bytes)
}

type proposerConfigurationStorage map[types.PublicKey][]RelayEntry

// buildProposerConfigurationStorage creates a storage containing a mapping of each proposer
// address extracted from the configuration file and its preferred relays.
func (c *configuration) buildProposerConfigurationStorage() (proposerConfigurationStorage, error) {
	// Initialize the storage.
	storage := make(proposerConfigurationStorage)

	for proposer, config := range c.ProposerConfig {
		// First, we'll verify if the proposer address is valid.
		// It will be used as the key in the storage to reference this proposer's preferences.
		address, err := types.HexToPubkey(proposer)
		if err != nil {
			return nil, err
		}

		for _, builderRelay := range config.BuilderRegistration.BuilderRelays {
			if c.BuilderRelaysGroups[builderRelay] == nil {
				// At this point, builderRelay can either be an empty or non-existing group,
				// or a relay entry.
				entry, err := NewRelayEntry(builderRelay)
				if err != nil {
					return nil, err
				}

				// Save this relay as the preference for this validator.
				storage[address] = append(storage[address], entry)
				continue
			}

			// At this point, builderRelay is a group of relay URLs.
			// TODO : Maybe verify if the group's name matches a regex / is not empty ?
			if len(c.BuilderRelaysGroups[builderRelay]) == 0 {
				// Empty group.
				return nil, errors.New("group contains nothing")
			}

			for _, relayURL := range c.BuilderRelaysGroups[builderRelay] {
				entry, err := NewRelayEntry(relayURL)
				if err != nil {
					return nil, err
				}

				// Save this each relay of this group as the preference for this validator.
				storage[address] = append(storage[address], entry)
			}
		}
	}

	// TODO : Maybe remove duplicates ? For example, when a proposer contains a fusion of two groups with common relay URLs.
	return storage, nil
}
