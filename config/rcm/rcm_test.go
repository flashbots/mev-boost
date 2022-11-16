package rcm_test

import (
	"net/url"
	"testing"

	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayConfigurationManager(t *testing.T) {
	t.Parallel()

	t.Run("it successfully returns a list of relays for a given validator by its public key", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := randomBLSPublicKey(t)
		want := randomRelayEntries(t, 3)

		rcp := &RelayConfigProviderMock{RelaysByValidatorPublicKeyFn: stubRelays(want)}
		sut := rcm.New(rcp)

		// act
		got, err := sut.RelaysByValidatorPublicKey(validatorPublicKey.String())

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

type RelayConfigProviderMock struct {
	RelaysByValidatorPublicKeyFn func(publicKey string) ([]server.RelayEntry, error)
}

func (m *RelayConfigProviderMock) RelaysByValidatorPublicKey(publicKey string) ([]server.RelayEntry, error) {
	return m.RelaysByValidatorPublicKeyFn(publicKey)
}

func stubRelays(relays []server.RelayEntry) func(publicKey string) ([]server.RelayEntry, error) {
	return func(publicKey string) ([]server.RelayEntry, error) {
		return relays, nil
	}
}

func randomRelayEntries(t *testing.T, num int) []server.RelayEntry {
	t.Helper()

	relays := make([]server.RelayEntry, num)

	for i := 0; i < num; i++ {
		relays[i] = randomRelayEntry(t)
	}

	return relays
}

func randomRelayEntry(t *testing.T) server.RelayEntry {
	t.Helper()

	blsPublicKey := randomBLSPublicKey(t)

	relayURL := url.URL{
		Scheme: "https",
		User:   url.User(blsPublicKey.String()),
		Host:   "relay.test.net",
	}

	relayEntry, err := server.NewRelayEntry(relayURL.String())
	require.NoError(t, err)

	return relayEntry
}

func randomBLSPublicKey(t *testing.T) types.PublicKey {
	t.Helper()

	_, blsPublicKey, err := bls.GenerateNewKeypair()
	require.NoError(t, err)

	publicKey, err := types.BlsPublicKeyToPublicKey(blsPublicKey)
	require.NoError(t, err)

	return publicKey
}
