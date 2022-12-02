package rcp_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRelayConfigProvider(t *testing.T) {
	t.Parallel()

	t.Run("it returns relay config", func(t *testing.T) {
		t.Parallel()

		// arrange
		relays := reltest.RandomRelaySet(t, 3)
		want := expectDefaultRelaysConfig(relays.ToList())

		sut := rcp.NewDefault(relays)

		// act
		got, err := sut.FetchConfig()

		// assert
		require.NoError(t, err)
		assert.Equal(t, want.DefaultConfig.Builder.Enabled, got.DefaultConfig.Builder.Enabled)
		assert.ElementsMatch(t, want.DefaultConfig.Builder.Relays, got.DefaultConfig.Builder.Relays)
	})

	t.Run("it uses an empty relay set, if relays are nil", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := expectConfigWithNoDefaultRelays()

		sut := rcp.NewDefault(nil)

		// act
		got, err := sut.FetchConfig()

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}

func expectDefaultRelaysConfig(entries relay.List) *relay.Config {
	return &relay.Config{
		DefaultConfig: relay.Relay{
			Builder: relay.Builder{
				Enabled: true,
				Relays:  entries.ToStringSlice(),
			},
		},
	}
}

func expectConfigWithNoDefaultRelays() *relay.Config {
	return &relay.Config{
		DefaultConfig: relay.Relay{
			Builder: relay.Builder{
				Enabled: true,
				Relays:  []string{},
			},
		},
	}
}
