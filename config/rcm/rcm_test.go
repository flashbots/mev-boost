package rcm_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/server"
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayConfigurationManager(t *testing.T) {
	t.Parallel()

	t.Run("it successfully returns a list of relays for a given validator by its public key", func(t *testing.T) {
		t.Parallel()

		// arrange
		validatorPublicKey := testutil.RandomBLSPublicKey(t)
		want := testutil.RandomRelayEntries(t, 3)

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
