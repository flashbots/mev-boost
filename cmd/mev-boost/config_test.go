package main

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateConfigFromJSON(t *testing.T) {
	testCases := []struct {
		name       string
		rawMessage json.RawMessage

		expectedError         error
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
							"validator_registration": {
								"enabled": true,
								"builder_relays": ["https://0x821961b64d99b997c934c22b4fd6109790acf00f7969322c4e9dbf1ca278c333148284c01c5ef551a1536ddd14b178b9@builder-relay-kiln.flashbots.net"],
								"gas_limit": "12345654321"
							}
						}
					}
				}
			`),
			expectedError: nil,
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
				ProposerConfig: map[string]proposerConfig{
					"0xa057816155ad77931185101128655c0191bd0214c201ca48ed887f6c4c6adf334070efcd75140eada5ac83a92506dd7a": {
						FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
						ValidatorRegistration: validatorRegistration{
							Enabled:       true,
							BuilderRelays: []string{"https://0x821961b64d99b997c934c22b4fd6109790acf00f7969322c4e9dbf1ca278c333148284c01c5ef551a1536ddd14b178b9@builder-relay-kiln.flashbots.net"},
							GasLimit:      "12345654321",
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			config, err := configFromJSON(tt.rawMessage)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedConfiguration, config)
			}
		})
	}
}
