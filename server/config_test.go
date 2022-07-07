package server

import (
	"encoding/json"
	"fmt"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func _NewRelayEntry(t *testing.T, relayURL string) RelayEntry {
	entry, err := NewRelayEntry(relayURL)
	require.NoError(t, err)

	return entry
}

func TestCreateConfigFromJSON(t *testing.T) {
	testCases := []struct {
		name       string
		rawMessage json.RawMessage

		expectedError         bool
		expectedConfiguration *configuration
	}{
		{
			name: "It creates a valid configuration",
			rawMessage: []byte(`
				{
					"builder_relays_groups": {
						"groupA": [
							"A1",
							"A2"
						],
						"groupB": [
							"B1",
							"B2"
						]
					},
					"proposer_config": {
						"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
							"fee_recipient": "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
							"builder_registration": {
								"enabled": true,
								"builder_relays": ["https://0x821961b64d99b997c934c22b4fd6109790acf00f7969322c4e9dbf1ca278c333148284c01c5ef551a1536ddd14b178b9@builder-relay-kiln.flashbots.net"],
								"gas_limit": "12345654321"
							}
						}
					},
					"default_config": {
						"fee_recipient": "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
						"builder_registration": {
							"enabled": false,
							"builder_relays": ["groupB"]
						}
					}	
				}
			`),
			expectedError: false,
			expectedConfiguration: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupA": {
						"A1",
						"A2",
					},
					"groupB": {
						"B1",
						"B2",
					},
				},
				ProposerConfig: map[string]rawProposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						BuilderRegistration: builderRegistrationConfig{
							Enabled:       true,
							BuilderRelays: []string{"https://0x821961b64d99b997c934c22b4fd6109790acf00f7969322c4e9dbf1ca278c333148284c01c5ef551a1536ddd14b178b9@builder-relay-kiln.flashbots.net"},
							GasLimit:      "12345654321",
						},
					},
				},
				DefaultConfig: rawProposerConfig{
					FeeRecipient: "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
					BuilderRegistration: builderRegistrationConfig{
						Enabled:       false,
						BuilderRelays: []string{"groupB"},
					},
				},
			},
		},
		{
			name:                  "It fails to read empty JSON",
			rawMessage:            []byte(""),
			expectedError:         true,
			expectedConfiguration: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := configFromJSON(tt.rawMessage)

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedConfiguration, config)
			}
		})
	}
}

func TestCreateConfigurationFromFile(t *testing.T) {
	testCases := []struct {
		name     string
		filename string

		expectedError         bool
		expectedConfiguration *configuration
	}{
		{
			name:                  "It fails to reads configuration from JSON file",
			filename:              "",
			expectedError:         true,
			expectedConfiguration: nil,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := configFromFile(tt.filename)

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedConfiguration, config)
			}
		})
	}
}

func TestBuildProposerConfigurationStorage(t *testing.T) {
	testCases := []struct {
		name string
		conf *configuration

		expectedError   bool
		expectedStorage proposerConfigurationStorage
	}{
		{
			name: "It detects non-existing group",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
				},
				ProposerConfig: map[string]rawProposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						BuilderRegistration: builderRegistrationConfig{
							Enabled:       true,
							BuilderRelays: []string{"groupB", "groupC"},
							GasLimit:      "12345654321",
						},
					},
				},
			},
			expectedError:   true,
			expectedStorage: nil,
		},
		{
			name: "It detects empty group",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
					"groupC": {},
				},
				ProposerConfig: map[string]rawProposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						BuilderRegistration: builderRegistrationConfig{
							Enabled:       true,
							BuilderRelays: []string{"groupB", "groupC"},
							GasLimit:      "12345654321",
						},
					},
				},
			},
			expectedError:   true,
			expectedStorage: nil,
		},
		{
			name: "It detects invalid relay URLs in group",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupA": {
						fmt.Sprintf("https://%s", "builder0-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x01}.String(),
							"builder1-relay-kiln.flashbots.net"),
					},
				},
				ProposerConfig: map[string]rawProposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						BuilderRegistration: builderRegistrationConfig{
							Enabled:       true,
							BuilderRelays: []string{"groupA"},
							GasLimit:      "12345654321",
						},
					},
				},
			},
			expectedError:   true,
			expectedStorage: nil,
		},
		{
			name: "It creates storage from group only",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupA": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x00}.String(),
							"builder0-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x01}.String(),
							"builder1-relay-kiln.flashbots.net"),
					},
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
				},
				ProposerConfig: map[string]rawProposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						BuilderRegistration: builderRegistrationConfig{
							Enabled:       true,
							BuilderRelays: []string{"groupB"},
							GasLimit:      "12345654321",
						},
					},
				},
			},
			expectedError: false,
			expectedStorage: map[types.PublicKey]*proposerConfiguration{
				_HexToPubkey("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a"): {
					FeeRecipient: _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
					Enabled:      true,
					Relays: []RelayEntry{
						_NewRelayEntry(t, fmt.Sprintf("https://%s@%s",
							types.PublicKey{0x02}.String(), "builder2-relay-kiln.flashbots.net")),
						_NewRelayEntry(t, fmt.Sprintf("https://%s@%s",
							types.PublicKey{0x03}.String(), "builder3-relay-kiln.flashbots.net")),
					},
					GasLimit: "12345654321",
				},
			},
		},
		{
			name: "It creates storage from raw URLs only",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupA": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x00}.String(),
							"builder0-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x01}.String(),
							"builder1-relay-kiln.flashbots.net"),
					},
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
				},
				ProposerConfig: map[string]rawProposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						BuilderRegistration: builderRegistrationConfig{
							Enabled: true,
							BuilderRelays: []string{
								fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
									"builder4-relay-kiln.flashbots.net"),
							},
							GasLimit: "12345654321",
						},
					},
				},
			},
			expectedError: false,
			expectedStorage: map[types.PublicKey]*proposerConfiguration{
				_HexToPubkey("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a"): {
					FeeRecipient: _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
					Enabled:      true,
					Relays: []RelayEntry{
						_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
							"builder4-relay-kiln.flashbots.net")),
					},
					GasLimit: "12345654321",
				},
			},
		},
		{
			name: "It creates storage from group and raw URLs",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
				},
				ProposerConfig: map[string]rawProposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						BuilderRegistration: builderRegistrationConfig{
							Enabled: true,
							BuilderRelays: []string{
								"groupB",
								fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
									"builder4-relay-kiln.flashbots.net"),
							},
							GasLimit: "12345654321",
						},
					},
				},
			},
			expectedError: false,
			expectedStorage: map[types.PublicKey]*proposerConfiguration{
				_HexToPubkey("0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a"): {
					FeeRecipient: _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
					Enabled:      true,
					Relays: []RelayEntry{
						_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net")),
						_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net")),
						_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
							"builder4-relay-kiln.flashbots.net")),
					},
					GasLimit: "12345654321",
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := tt.conf.buildProposerConfigurationStorage()

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, storage)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedStorage, storage)
			}
		})
	}
}

func TestBuildDefaultConfigurationStorage(t *testing.T) {
	testCases := []struct {
		name string
		conf *configuration

		expectedError   bool
		expectedStorage *proposerConfiguration
	}{
		{
			name: "It detects non-existing group",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
				},
				DefaultConfig: rawProposerConfig{
					FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					BuilderRegistration: builderRegistrationConfig{
						Enabled:       true,
						BuilderRelays: []string{"groupB", "groupC"},
						GasLimit:      "12345654321",
					},
				},
			},
			expectedError:   true,
			expectedStorage: nil,
		},
		{
			name: "It detects empty group",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
					"groupC": {},
				},
				DefaultConfig: rawProposerConfig{
					FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					BuilderRegistration: builderRegistrationConfig{
						Enabled:       true,
						BuilderRelays: []string{"groupB", "groupC"},
						GasLimit:      "12345654321",
					},
				},
			},
			expectedError:   true,
			expectedStorage: nil,
		},
		{
			name: "It detects invalid relay URLs in group",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupA": {
						fmt.Sprintf("https://%s", "builder0-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x01}.String(),
							"builder1-relay-kiln.flashbots.net"),
					},
				},
				DefaultConfig: rawProposerConfig{
					FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					BuilderRegistration: builderRegistrationConfig{
						Enabled:       true,
						BuilderRelays: []string{"groupA"},
						GasLimit:      "12345654321",
					},
				},
			},
			expectedError:   true,
			expectedStorage: nil,
		},
		{
			name: "It creates default configuration from group only",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
				},
				DefaultConfig: rawProposerConfig{
					FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					BuilderRegistration: builderRegistrationConfig{
						Enabled:       true,
						BuilderRelays: []string{"groupB"},
						GasLimit:      "12345654321",
					},
				},
			},
			expectedError: false,
			expectedStorage: &proposerConfiguration{
				FeeRecipient: _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
				Enabled:      true,
				Relays: []RelayEntry{
					_NewRelayEntry(t, fmt.Sprintf("https://%s@%s",
						types.PublicKey{0x02}.String(), "builder2-relay-kiln.flashbots.net")),
					_NewRelayEntry(t, fmt.Sprintf("https://%s@%s",
						types.PublicKey{0x03}.String(), "builder3-relay-kiln.flashbots.net")),
				},
				GasLimit: "12345654321",
			},
		},
		{
			name: "It creates storage from raw URLs only",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupA": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x00}.String(),
							"builder0-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x01}.String(),
							"builder1-relay-kiln.flashbots.net"),
					},
				},
				DefaultConfig: rawProposerConfig{
					FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					BuilderRegistration: builderRegistrationConfig{
						Enabled: true,
						BuilderRelays: []string{
							fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
								"builder4-relay-kiln.flashbots.net"),
						},
						GasLimit: "12345654321",
					},
				},
			},
			expectedError: false,
			expectedStorage: &proposerConfiguration{
				FeeRecipient: _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
				Enabled:      true,
				Relays: []RelayEntry{
					_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
						"builder4-relay-kiln.flashbots.net")),
				},
				GasLimit: "12345654321",
			},
		},
		{
			name: "It creates storage from group and raw URLs",
			conf: &configuration{
				BuilderRelaysGroups: map[string][]string{
					"groupB": {
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
							"builder2-relay-kiln.flashbots.net"),
						fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
							"builder3-relay-kiln.flashbots.net"),
					},
				},
				DefaultConfig: rawProposerConfig{
					FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
					BuilderRegistration: builderRegistrationConfig{
						Enabled: true,
						BuilderRelays: []string{
							"groupB",
							fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
								"builder4-relay-kiln.flashbots.net"),
						},
						GasLimit: "12345654321",
					},
				},
			},
			expectedError: false,
			expectedStorage: &proposerConfiguration{
				FeeRecipient: _HexToAddress("0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3"),
				Enabled:      true,
				Relays: []RelayEntry{
					_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x02}.String(),
						"builder2-relay-kiln.flashbots.net")),
					_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x03}.String(),
						"builder3-relay-kiln.flashbots.net")),
					_NewRelayEntry(t, fmt.Sprintf("https://%s@%s", types.PublicKey{0x04}.String(),
						"builder4-relay-kiln.flashbots.net")),
				},
				GasLimit: "12345654321",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			defaultStorage, err := tt.conf.buildDefaultConfiguration()

			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, defaultStorage)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedStorage, defaultStorage)
			}
		})
	}
}
