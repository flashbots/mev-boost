package rcm_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/rcp"
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
		want := testutil.RandomRCPRelayEntries(t, 3)

		rcpMock := &RelayConfigProviderMock{RelaysByValidatorPublicKeyFn: stubRelays(want)}
		sut := rcm.New(rcpMock)

		// act
		got, err := sut.RelaysByValidatorPublicKey(validatorPublicKey.String())

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

type RelayConfigProviderMock struct {
	RelaysByValidatorPublicKeyFn func(publicKey string) ([]rcp.RelayEntry, error)
}

func (m *RelayConfigProviderMock) RelaysByValidatorPublicKey(publicKey string) ([]rcp.RelayEntry, error) {
	return m.RelaysByValidatorPublicKeyFn(publicKey)
}

func stubRelays(relays []rcp.RelayEntry) func(publicKey string) ([]rcp.RelayEntry, error) {
	return func(publicKey string) ([]rcp.RelayEntry, error) {
		return relays, nil
	}
}
