package relay

import (
	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseRelaysURLs(t *testing.T) {
	zeroPublicKey := types.PublicKey{0x0}
	// Used to fake a relay's public key.
	publicKey := types.PublicKey{0x01}

	testCases := []struct {
		name     string
		relayURL string

		expectedErr       error
		expectedAddress   string
		expectedPublicKey string
		expectedURL       string
	}{
		{
			name:     "Relay URL with protocol scheme",
			relayURL: "http://foo.com",

			expectedErr:       nil,
			expectedAddress:   "http://foo.com",
			expectedPublicKey: zeroPublicKey.String(),
			expectedURL:       "http://foo.com",
		},
		{
			name:     "Relay URL without protocol scheme",
			relayURL: "foo.com",

			expectedErr:       nil,
			expectedAddress:   "http://foo.com",
			expectedPublicKey: zeroPublicKey.String(),
			expectedURL:       "http://foo.com",
		},
		{
			name:     "Relay URL without protocol scheme and with public key",
			relayURL: publicKey.String() + "@foo.com",

			expectedErr:       nil,
			expectedAddress:   "http://foo.com",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@foo.com",
		},
		{
			name:     "Relay URL with public key host and port",
			relayURL: publicKey.String() + "@foo.com:9999",

			expectedErr:       nil,
			expectedAddress:   "http://foo.com:9999",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@foo.com:9999",
		},
		{
			name:     "Relay URL with IP and port",
			relayURL: "12.345.678:9999",

			expectedErr:       nil,
			expectedAddress:   "http://12.345.678:9999",
			expectedPublicKey: zeroPublicKey.String(),
			expectedURL:       "http://12.345.678:9999",
		},
		{
			name:     "Relay URL with https IP and port",
			relayURL: "https://12.345.678:9999",

			expectedErr:       nil,
			expectedAddress:   "https://12.345.678:9999",
			expectedPublicKey: zeroPublicKey.String(),
			expectedURL:       "https://12.345.678:9999",
		},
		{
			name:     "Invalid relay public key",
			relayURL: "http://0x123456@foo.com",

			expectedErr:       types.ErrLength,
			expectedAddress:   "",
			expectedPublicKey: "",
			expectedURL:       "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			relayEntry, err := NewRelayEntry(tt.relayURL)

			// Check errors.
			require.Equal(t, tt.expectedErr, err)

			// Now perform content assertions.
			if tt.expectedErr == nil {
				require.Equal(t, tt.expectedAddress, relayEntry.Address)
				require.Equal(t, tt.expectedPublicKey, relayEntry.PublicKey.String())
				require.Equal(t, tt.expectedURL, relayEntry.String())
			}
		})
	}
}
