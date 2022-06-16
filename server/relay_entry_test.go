package server

import (
	"strconv"
	"testing"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
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

// Tests that reputations is calculated accordingly. Hardcoded
// for a window of 5 slots
func TestGetRelayReputation(t *testing.T) {
	testCases := []struct {
		responseStatus []int

		//  0 -> Relay not selected in that slot
		// -1 -> Relay withdrawn the payload
		// +1 -> Relay returned ok the payload
		expectedReputation float64
	}{
		{
			// Never selected, neutral reputation = 0
			responseStatus:     []int{0, 0, 0, 0, 0},
			expectedReputation: 0,
		},
		{
			// Always selected, best reputation = 1
			responseStatus:     []int{1, 1, 1, 1, 1},
			expectedReputation: 1,
		},
		{
			// Always failed, worst reputation = -1
			responseStatus:     []int{-1, -1, -1, -1, -1},
			expectedReputation: -1,
		},
		{
			// Failed in the past, but with recent OK responses
			responseStatus:     []int{-1, 0, 0, 1, 1},
			expectedReputation: 0.733343,
		},
		{
			// Never failed in the past and with recent OK responses
			responseStatus:     []int{0, 0, 0, 1, 1},
			expectedReputation: 0.767394,
		},
		{
			// Failed very recently
			responseStatus:     []int{0, 1, 1, -1, -1},
			expectedReputation: -0.568843,
		},
		{
			// Failed in the past, but recently OK
			responseStatus:     []int{0, -1, -1, 1, 1},
			expectedReputation: 0.5688430,
		},
	}

	for i, tt := range testCases {
		i := "Relay_" + strconv.Itoa(i)
		t.Run(i, func(t *testing.T) {
			relayEntry, err := NewRelayEntry(i)
			require.NoError(t, err)
			relayEntry.ResponseStatus = tt.responseStatus

			// Now perform content assertions.
			require.InDelta(t, tt.expectedReputation, relayEntry.GetRelayReputation(), 0.00001)
		})
	}
}

// Tests that the window storing past performance (committed/withdrawn) is updated correctly
func TestSetResponseStatus(t *testing.T) {
	relayEntry, err := NewRelayEntry("relay")
	require.NoError(t, err)

	// ResponseStatus works as a sliding windows where each new sample
	// destroys the first one. Only x samples are kept (hardcoded to 5)
	// Its content shall be modified with SetResponseStatus()
	relayEntry.ResponseStatus = []int{0, 0, 0, 0, 0}

	// Note that SetResponseStatus is expected to be called by the server on every slot
	// This is just an assumption to simplify it

	// Observe how the windows is sliding left with each new sample added on the right
	// Observe also how the reputation evolves with each new sample
	relayEntry.SetResponseStatus(PayloadWithdrawn)
	require.Equal(t, []int{0, 0, 0, 0, -1}, relayEntry.ResponseStatus)
	require.Equal(t, -0.50866, relayEntry.GetRelayReputation())

	relayEntry.SetResponseStatus(PayloadWithdrawn)
	require.Equal(t, []int{0, 0, 0, -1, -1}, relayEntry.ResponseStatus)
	require.Equal(t, -0.7673949956, relayEntry.GetRelayReputation())

	relayEntry.SetResponseStatus(PayloadReturned)
	require.Equal(t, []int{0, 0, -1, -1, 1}, relayEntry.ResponseStatus)
	require.Equal(t, 0.11831686153810395, relayEntry.GetRelayReputation())

	relayEntry.SetResponseStatus(NotSelected)
	require.Equal(t, []int{0, -1, -1, 1, 0}, relayEntry.ResponseStatus)
	require.Equal(t, 0.06018305478997199, relayEntry.GetRelayReputation())

	relayEntry.SetResponseStatus(PayloadReturned)
	require.Equal(t, []int{-1, -1, 1, 0, 1}, relayEntry.ResponseStatus)
	require.Equal(t, 0.5392727126494672, relayEntry.GetRelayReputation())

	relayEntry.SetResponseStatus(PayloadReturned)
	require.Equal(t, []int{-1, 1, 0, 1, 1}, relayEntry.ResponseStatus)
	require.Equal(t, 0.8002871612838351, relayEntry.GetRelayReputation())
}
