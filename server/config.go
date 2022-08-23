package server

import (
	"encoding/json"
	"errors"
	"github.com/flashbots/go-boost-utils/types"
	"os"
)

type rawConfiguration struct {
	FeeRecipient          string `json:"fee_recipient"`
	ValidatorRegistration struct {
		BuilderRelays []string `json:"builder_relays"`
		Enabled       bool     `json:"enabled"`
		GasLimit      string   `json:"gas_limit"`
	} `json:"validator_registration"`
}

type rawConfigurationFile struct {
	BuilderRelaysGroups map[string][]string `json:"builder_relays_groups"`

	ProposerConfig map[string]rawConfiguration `json:"proposer_config"`
	DefaultConfig  rawConfiguration            `json:"default_config"`
}

// newRawConfigurationFile creates a temporary rawConfigurationFile used to build a
// ProposerConfigurationStorage, by reading content from a JSON file.
func newRawConfigurationFile(filename string) (*rawConfigurationFile, error) {
	// Read JSON file content.
	bytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Tries to unmarshal content in JSON.
	placeholder := &rawConfigurationFile{}
	if err := json.Unmarshal(bytes, placeholder); err != nil {
		return nil, err
	}

	return placeholder, nil
}

// ProposerConfig holds one proposer configuration.
type ProposerConfig struct {
	FeeRecipient types.Address
	Enabled      bool
	Relays       []RelayEntry
	GasLimit     types.U256Str
}

// ProposerConfigurationStorage holds both the default configuration and the proposers ones.
type ProposerConfigurationStorage struct {
	proposerConfigurations map[types.PublicKey]*ProposerConfig
	defaultConfiguration   *ProposerConfig
}

// NewProposerConfigurationStorage creates a new storage holding each proposer preferences using
// the content extracted from a JSON file.
func NewProposerConfigurationStorage(filename string) (*ProposerConfigurationStorage, error) {
	// Tries to create the raw configuration extracted from the JSON file.
	raw, err := newRawConfigurationFile(filename)
	if err != nil {
		return nil, err
	}

	// Initialize the storage and save default configuration.
	pcs := &ProposerConfigurationStorage{
		proposerConfigurations: map[types.PublicKey]*ProposerConfig{},
	}

	pcs.defaultConfiguration, err = newConfigurationStorage(&raw.DefaultConfig, raw.BuilderRelaysGroups)
	if err != nil {
		return nil, err
	}

	// For each proposer, save its own configuration.
	for proposer, configuration := range raw.ProposerConfig {
		address, err := types.HexToPubkey(proposer)
		if err != nil {
			return nil, err
		}

		configurationStorage, err := newConfigurationStorage(&configuration, raw.BuilderRelaysGroups)
		if err != nil {
			return nil, err
		}

		pcs.proposerConfigurations[address] = configurationStorage
	}

	return pcs, nil
}

// GetProposerConfiguration looks for a specific configuration for the given proposer, if not found it
// returns the default configuration.
func (s *ProposerConfigurationStorage) GetProposerConfiguration(proposer types.PublicKey) *ProposerConfig {
	res := s.proposerConfigurations[proposer]
	if res == nil {
		res = s.defaultConfiguration
	}

	return res
}

// newConfigurationStorage creates a new ProposerConfig from a rawConfiguration
// previously extracted from a JSON file and the relay groups available.
// Used to create the default configuration and each proposer's one.
func newConfigurationStorage(rawConf *rawConfiguration, groups map[string][]string) (*ProposerConfig, error) {
	feeRecipient, err := types.HexToAddress(rawConf.FeeRecipient)
	if err != nil {
		return nil, err
	}

	gasLimit := types.U256Str{}
	err = gasLimit.UnmarshalText([]byte(rawConf.ValidatorRegistration.GasLimit))
	if err != nil {
		return nil, err
	}

	configuration := &ProposerConfig{
		FeeRecipient: feeRecipient,
		Enabled:      rawConf.ValidatorRegistration.Enabled,
		GasLimit:     gasLimit,
	}

	if len(rawConf.ValidatorRegistration.BuilderRelays) == 0 {
		return nil, errors.New("no builder is associated to this proposer")
	}

	for _, builderRelay := range rawConf.ValidatorRegistration.BuilderRelays {
		if groups[builderRelay] == nil {
			// At this point, builderRelay can either be an empty or non-existing group,
			// or a relay entry.
			entry, err := NewRelayEntry(builderRelay)
			if err != nil {
				return nil, err
			}

			// Save this relay as the preference for this validator.
			configuration.Relays = append(configuration.Relays, entry)
			continue
		}

		// At this point, builderRelay is a group of relay URLs.
		// TODO : Maybe verify if the group's name matches a regex / is not empty ?
		if len(groups[builderRelay]) == 0 {
			// Empty group.
			return nil, errors.New("group contains nothing")
		}

		for _, relayURL := range groups[builderRelay] {
			entry, err := NewRelayEntry(relayURL)
			if err != nil {
				return nil, err
			}

			// Save this each relay of this group as the preference for this validator.
			configuration.Relays = append(configuration.Relays, entry)
		}
	}

	// TODO : Maybe remove duplicates ? For example, when a configuration contains a fusion of two groups with common relay URLs.
	return configuration, nil
}

// FromRelayList creates a default configuration with the provided list of relays.
func (s *ProposerConfigurationStorage) FromRelayList(relays []RelayEntry) *ProposerConfigurationStorage {
	return &ProposerConfigurationStorage{
		defaultConfiguration: &ProposerConfig{
			Relays: relays,
		},
	}
}

// GetAllRelays returns all registered relays.
func (s *ProposerConfigurationStorage) GetAllRelays() []RelayEntry {
	relays := s.defaultConfiguration.Relays

	for _, configurationStorage := range s.proposerConfigurations {
		relays = append(relays, configurationStorage.Relays...)
	}

	return relays
}
