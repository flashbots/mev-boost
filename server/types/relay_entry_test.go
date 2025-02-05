package types

import (
	"fmt"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/go-boost-utils/utils"
	"github.com/stretchr/testify/require"
)

func TestParseRelaysURLs(t *testing.T) {
	// Used to fake a relay's public key.
	publicKey, err := utils.HexToPubkey("0x82f6e7cc57a2ce68ec41321bebc55bcb31945fe66a8e67eb8251425fab4c6a38c10c53210aea9796dd0ba0441b46762a")
	require.NoError(t, err)

	testCases := []struct {
		name     string
		relayURL string
		path     string

		expectedErr       error
		expectedURI       string // full URI with scheme, host, path and args
		expectedPublicKey string
		expectedURL       string
	}{
		{
			name:              "Relay URL with protocol scheme",
			relayURL:          fmt.Sprintf("http://%s@foo.com", publicKey.String()),
			expectedURI:       "http://foo.com",
			expectedPublicKey: publicKey.String(),
			expectedURL:       fmt.Sprintf("http://%s@foo.com", publicKey.String()),
		},
		{
			name:        "Relay URL without protocol scheme, without public key",
			relayURL:    "foo.com",
			expectedErr: ErrMissingRelayPubkey,
		},
		{
			name:              "Relay URL without protocol scheme and with public key",
			relayURL:          publicKey.String() + "@foo.com",
			expectedURI:       "http://foo.com",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@foo.com",
		},
		{
			name:              "Relay URL with public key host and port",
			relayURL:          publicKey.String() + "@foo.com:9999",
			expectedURI:       "http://foo.com:9999",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@foo.com:9999",
		},
		{
			name:              "Relay URL with IP and port",
			relayURL:          publicKey.String() + "@12.345.678:9999",
			expectedURI:       "http://12.345.678:9999",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "http://" + publicKey.String() + "@12.345.678:9999",
		},
		{
			name:              "Relay URL with https IP and port",
			relayURL:          "https://" + publicKey.String() + "@12.345.678:9999",
			expectedURI:       "https://12.345.678:9999",
			expectedPublicKey: publicKey.String(),
			expectedURL:       "https://" + publicKey.String() + "@12.345.678:9999",
		},
		{
			name:        "Relay URL with an invalid public key (wrong length)",
			relayURL:    "http://0x123456@foo.com",
			expectedErr: types.ErrLength,
		},
		{
			name:        "Relay URL with an invalid public key (not on the curve)",
			relayURL:    "http://0xac6e78dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@foo.com",
			expectedErr: utils.ErrInvalidPubkey,
		},
		{
			name:        "Relay URL with an invalid public key (point-at-infinity)",
			relayURL:    "http://0xc00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000@foo.com",
			expectedErr: ErrPointAtInfinityPubkey,
		},
		{
			name:        "Relay URL with an invalid public key (all zero)",
			relayURL:    "http://0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000@foo.com",
			expectedErr: utils.ErrInvalidPubkey,
		},
		{
			name:              "Relay URL with query arg",
			relayURL:          fmt.Sprintf("http://%s@foo.com?id=foo&bar=1", publicKey.String()),
			expectedURI:       "http://foo.com?id=foo&bar=1",
			expectedPublicKey: publicKey.String(),
			expectedURL:       fmt.Sprintf("http://%s@foo.com?id=foo&bar=1", publicKey.String()),
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
			}
		})
	}
}
