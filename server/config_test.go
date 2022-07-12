package server

import (
	"fmt"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func _newRelayEntry(t *testing.T, relayURL string) RelayEntry {
	entry, err := NewRelayEntry(relayURL)
	require.NoError(t, err)

	return entry
}

func _newRelayEntries(t *testing.T, l, h int) []RelayEntry {
	var res []RelayEntry

	for i := l; i < h; i++ {
		pubKey := types.PublicKey{byte(i)}.String()
		newEntry := fmt.Sprintf("https://%s@%s%d%s", pubKey, "builder", i, "-relay-kiln.flashbots.net/")

		res = append(res, _newRelayEntry(t, newEntry))
	}

	return res
}

func _newGasLimit(t *testing.T, input string) types.U256Str {
	gasLimit := types.U256Str{}
	err := gasLimit.UnmarshalText([]byte(input))

	require.NoError(t, err)

	return gasLimit
}

func TestCreateNewRawConfiguration(t *testing.T) {
	testCases := []struct {
		name     string
		filename string

		expectedError                bool
		expectedRawConfigurationFile *rawConfigurationFile
	}{
		{
			name:                         "It detects non-existing file",
			filename:                     "deadbeef",
			expectedError:                true,
			expectedRawConfigurationFile: nil,
		},
		{
			name:                         "It detects invalid JSON",
			filename:                     "testdata/invalid_json.input",
			expectedError:                true,
			expectedRawConfigurationFile: nil,
		},
		{
			name:          "It creates a valid raw configuration from file",
			filename:      "testdata/valid_json.input",
			expectedError: false,
			expectedRawConfigurationFile: &rawConfigurationFile{
				BuilderRelaysGroups: make(map[string][]string),
				ProposerConfig:      make(map[string]rawConfiguration),
				DefaultConfig: rawConfiguration{
					ValidatorRegistration: struct {
						BuilderRelays []string `json:"builder_relays"`
						Enabled       bool     `json:"enabled"`
						GasLimit      string   `json:"gas_limit"`
					}(struct {
						BuilderRelays []string
						Enabled       bool
						GasLimit      string
					}{BuilderRelays: []string{}, Enabled: false, GasLimit: ""}),
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			rcf, err := newRawConfigurationFile(tt.filename)

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, rcf)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRawConfigurationFile, rcf)
			}
		})
	}
}

func TestCreateNewConfigurationStorage(t *testing.T) {
	relay0 := fmt.Sprintf("https://%s@%s", types.PublicKey{0x00}.String(), "builder0-relay-kiln.flashbots.net/")
	relay1 := fmt.Sprintf("https://%s@%s", types.PublicKey{0x01}.String(), "builder1-relay-kiln.flashbots.net/")
	gasLimit := "123456"
	feeRecipient := _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3")

	testCases := []struct {
		name    string
		rawConf *rawConfiguration
		groups  map[string][]string

		expectedError                bool
		expectedConfigurationStorage *ConfigurationStorage
	}{
		{
			name: "It detects invalid fee recipient",
			rawConf: &rawConfiguration{
				FeeRecipient: "0xdeadbeef",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{
						relay0,
					},
					GasLimit: gasLimit,
				},
			},
			groups: map[string][]string{
				"groupA": {relay0},
			},
			expectedError:                true,
			expectedConfigurationStorage: nil,
		},
		{
			name: "It detects invalid relay0 entry in raw configuration",
			rawConf: &rawConfiguration{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{
						"deadbeef",
					},
					GasLimit: gasLimit,
				},
			},
			groups: map[string][]string{
				"groupA": {relay0},
			},
			expectedError:                true,
			expectedConfigurationStorage: nil,
		},
		{
			name: "It detects empty group",
			rawConf: &rawConfiguration{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{
						"groupA",
					},
					GasLimit: gasLimit,
				},
			},
			groups: map[string][]string{
				"groupA": {},
			},
			expectedError:                true,
			expectedConfigurationStorage: nil,
		},
		{
			name: "It detects invalid relay0 entry in group",
			rawConf: &rawConfiguration{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{
						"groupA",
					},
					GasLimit: gasLimit,
				},
			},
			groups: map[string][]string{
				"groupA": {
					"deadbeef",
				},
			},
			expectedError:                true,
			expectedConfigurationStorage: nil,
		},
		{
			name: "It detects empty relay array in proposer configuration",
			rawConf: &rawConfiguration{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{},
					GasLimit:      gasLimit,
				},
			},
			groups: map[string][]string{
				"groupA": {
					"deadbeef",
				},
			},
			expectedError:                true,
			expectedConfigurationStorage: nil,
		},
		{
			name: "It creates valid configuration storage from group only",
			rawConf: &rawConfiguration{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{
						"groupA",
					},
					GasLimit: gasLimit,
				},
			},
			groups: map[string][]string{
				"groupA": {
					relay0,
				},
			},
			expectedError: false,
			expectedConfigurationStorage: &ConfigurationStorage{
				FeeRecipient: feeRecipient,
				Enabled:      false,
				Relays:       _newRelayEntries(t, 0, 1),
				GasLimit:     _newGasLimit(t, gasLimit),
			},
		},
		{
			name: "It creates valid configuration storage from raw relay0 entries only",
			rawConf: &rawConfiguration{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{
						relay0,
					},
					GasLimit: gasLimit,
				},
			},
			groups:        map[string][]string{},
			expectedError: false,
			expectedConfigurationStorage: &ConfigurationStorage{
				FeeRecipient: feeRecipient,
				Enabled:      false,
				Relays:       _newRelayEntries(t, 0, 1),
				GasLimit:     _newGasLimit(t, gasLimit),
			},
		},
		{
			name: "It creates valid configuration storage from both raw relay0 entries and groups",
			rawConf: &rawConfiguration{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				ValidatorRegistration: struct {
					BuilderRelays []string `json:"builder_relays"`
					Enabled       bool     `json:"enabled"`
					GasLimit      string   `json:"gas_limit"`
				}{
					BuilderRelays: []string{
						"groupA",
						relay1,
					},
					GasLimit: gasLimit,
				},
			},
			groups: map[string][]string{
				"groupA": {
					relay0,
				},
			},
			expectedError: false,
			expectedConfigurationStorage: &ConfigurationStorage{
				FeeRecipient: feeRecipient,
				Enabled:      false,
				Relays:       _newRelayEntries(t, 0, 2),
				GasLimit:     _newGasLimit(t, gasLimit),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			configurationStorage, err := newConfigurationStorage(tt.rawConf, tt.groups)

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, configurationStorage)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedConfigurationStorage, configurationStorage)
			}
		})
	}
}

func TestCreateNewProposerConfigurationStorage(t *testing.T) {
	testCases := []struct {
		name     string
		filename string

		expectedError                        bool
		expectedProposerConfigurationStorage *ProposerConfigurationStorage
	}{
		{
			name:          "It creates a valid raw configuration from file",
			filename:      "testdata/valid_config.json",
			expectedError: false,
			expectedProposerConfigurationStorage: &ProposerConfigurationStorage{
				proposerConfigurations: map[types.PublicKey]*ConfigurationStorage{
					_HexToPubkey("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a"): {
						FeeRecipient: _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
						Enabled:      true,
						Relays:       _newRelayEntries(t, 2, 6),
						GasLimit:     _newGasLimit(t, "123456"),
					},
				},
				defaultConfiguration: &ConfigurationStorage{
					FeeRecipient: _HexToAddress("0x0000000000000000000000000000000000000000"),
					Enabled:      false,
					Relays:       _newRelayEntries(t, 6, 7),
					GasLimit:     _newGasLimit(t, "333333"),
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewProposerConfigurationStorage(tt.filename)

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, storage)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedProposerConfigurationStorage, storage)
			}
		})
	}
}

func TestGetProposerConfiguration(t *testing.T) {
	proposerPubKey := _HexToPubkey("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a")
	feeRecipient := _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3")
	gasLimit := "12345654321"

	testCases := []struct {
		name    string
		storage ProposerConfigurationStorage

		expectedConfiguration *ConfigurationStorage
	}{
		{
			name: "It gets specific configuration",
			storage: ProposerConfigurationStorage{
				proposerConfigurations: map[types.PublicKey]*ConfigurationStorage{
					proposerPubKey: {
						FeeRecipient: feeRecipient,
						Enabled:      true,
						Relays:       _newRelayEntries(t, 0, 1),
						GasLimit:     _newGasLimit(t, gasLimit),
					},
				},
			},
			expectedConfiguration: &ConfigurationStorage{
				FeeRecipient: feeRecipient,
				Enabled:      true,
				Relays:       _newRelayEntries(t, 0, 1),
				GasLimit:     _newGasLimit(t, gasLimit),
			},
		},
		{
			name: "It gets default configuration",
			storage: ProposerConfigurationStorage{
				defaultConfiguration: &ConfigurationStorage{
					FeeRecipient: feeRecipient,
					Enabled:      true,
					Relays:       _newRelayEntries(t, 0, 2),
					GasLimit:     _newGasLimit(t, gasLimit),
				},
			},
			expectedConfiguration: &ConfigurationStorage{
				FeeRecipient: feeRecipient,
				Enabled:      true,
				Relays:       _newRelayEntries(t, 0, 2),
				GasLimit:     _newGasLimit(t, gasLimit),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			configurationStorage := tt.storage.GetProposerConfiguration(proposerPubKey)

			require.NotNil(t, configurationStorage)
			require.Equal(t, tt.expectedConfiguration, configurationStorage)
		})
	}
}
