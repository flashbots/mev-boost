package server

import (
	"fmt"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
)

func TestParseRelaysURLs(t *testing.T) {
	// Used to fake a relay's public key.
	publicKey := types.PublicKey{0x01}

	testCases := []struct {
		name     string
		relayURL string
		path     string

		expectedErr       error
		expectedURI       string // full URI with scheme, host, path and args
		expectedPublicKey string
		expectedURL       string
		expectedWeight    float64
	}{
		{
			name:     "Relay URL with protocol scheme",
			relayURL: fmt.Sprintf("http://%s@foo.com", publicKey.String()),

			expectedURI:       "http://foo.com",
			expectedPublicKey: publicKey.String(),
			expectedURL:       fmt.Sprintf("http://%s@foo.com", publicKey.String()),
			expectedWeight:    1.0,
		},
		{
			name:     "Relay URL without protocol scheme, without public key",
			relayURL: "foo.com",

			expectedErr: ErrMissingRelayPubkey,
		},
		{
			name:     "Relay URL without protocol scheme and with public key",
			relayURL: publicKey.String() + "@foo.com",

			expectedURI:       "http://foo.com",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@foo.com",
			expectedWeight:    1.0,
		},
		{
			name:     "Relay URL with public key host and port",
			relayURL: publicKey.String() + "@foo.com:9999",

			expectedURI:       "http://foo.com:9999",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@foo.com:9999",
			expectedWeight:    1.0,
		},
		{
			name:     "Relay URL with IP and port",
			relayURL: publicKey.String() + "@12.345.678:9999",

			expectedURI:       "http://12.345.678:9999",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@12.345.678:9999",
			expectedWeight:    1.0,
		},
		{
			name:     "Relay URL with https IP and port",
			relayURL: "https://" + publicKey.String() + "@12.345.678:9999",

			expectedURI:       "https://12.345.678:9999",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "https://" + publicKey.String() + "@12.345.678:9999",
			expectedWeight:    1.0,
		},
		{
			name:     "Invalid relay public key",
			relayURL: "http://0x123456@foo.com",

			expectedErr: types.ErrLength,
		},
		{
			name:     "Relay URL with query arg",
			relayURL: fmt.Sprintf("http://%s@foo.com?id=foo&bar=1", publicKey.String()),

			expectedURI:       "http://foo.com?id=foo&bar=1",
			expectedPublicKey: publicKey.String(),
			expectedURL:       fmt.Sprintf("http://%s@foo.com?id=foo&bar=1", publicKey.String()),
			expectedWeight:    1.0,
		},
		{
			name:     "Weighted relay with https, IP and port",
			relayURL: fmt.Sprintf("123#https://%s@foo.com", publicKey.String()),

			expectedURI:       "https://foo.com",
			expectedPublicKey: publicKey.String(),
			expectedURL:       fmt.Sprintf("https://%s@foo.com", publicKey.String()),
			expectedWeight:    123.0,
		},
		{
			name:     "Weighted relay URL without protocol scheme and with public key",
			relayURL: fmt.Sprintf("0.11#%s@foo.com", publicKey.String()),

			expectedURI:       "http://foo.com",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@foo.com",
			expectedWeight:    0.11,
		},
		{
			name:     "Double weighted relay URL",
			relayURL: fmt.Sprintf("10#0.9#%s@foo.com", publicKey.String()),

			expectedErr: fmt.Errorf("invalid weighted relay entry format: %s", fmt.Sprintf("10#0.9#%s@foo.com", publicKey.String())),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			relayEntry, err := NewRelayEntry(tt.relayURL)

			// Check errors.
			require.Equal(t, tt.expectedErr, err)

			// Now perform content assertions.
			if tt.expectedErr == nil {
				require.Equal(t, tt.expectedURI, relayEntry.GetURI(tt.path))
				require.Equal(t, tt.expectedPublicKey, relayEntry.PublicKey.String())
				require.Equal(t, tt.expectedURL, relayEntry.String())
				require.Equal(t, tt.expectedWeight, relayEntry.Weight)
			}
		})
	}
}
