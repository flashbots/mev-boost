package relay_test

import (
	"flag"
	"fmt"
	"strings"
	"testing"

	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ flag.Value = (*relay.Set)(nil)

func TestRelaySet(t *testing.T) {
	t.Parallel()

	t.Run("it adds a relay entry", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := reltest.RandomRelayEntry(t)
		sut := relay.NewRelaySet()

		// act
		sut.Add(want)

		// assert
		assert.Contains(t, sut.ToList(), want)
	})

	t.Run("it adds a relay entry using relayURL", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := reltest.RandomRelayEntry(t)
		sut := relay.NewRelaySet()

		// act
		err := sut.AddURL(want.String())

		// assert
		require.NoError(t, err)
		assert.Contains(t, sut.ToList(), want)
	})

	t.Run("it fails to adds a relay with invalid relayURL", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := relay.NewRelaySet()

		// act
		err := sut.AddURL("invalid-relay-url")

		// assert
		assert.Error(t, err)
	})

	t.Run("it renders as a slice of relay urls", func(t *testing.T) {
		t.Parallel()

		// arrange
		relays := reltest.RandomRelayList(t, 2)
		want := []string{relays[0].String(), relays[1].String()}

		sut := relay.NewRelaySet()
		reltest.PopulateSetFromList(sut, relays)

		// act
		got := sut.ToStringSlice()

		// assert
		assert.ElementsMatch(t, want, got)
	})

	t.Run("it renders as a string", func(t *testing.T) {
		t.Parallel()

		// arrange
		relays := reltest.RandomRelayList(t, 2)
		want := fmt.Sprintf("%s,%s", relays[0].String(), relays[1].String())

		sut := relay.NewRelaySet()
		reltest.PopulateSetFromList(sut, relays)

		// act
		got := sut.String()

		// assert
		assertContainsTheSameRelays(t, want, got)
	})
}

func assertContainsTheSameRelays(t *testing.T, want, got string) {
	t.Helper()

	wantRelays := strings.Split(want, ",")
	gotRelays := strings.Split(got, ",")

	assert.ElementsMatch(t, wantRelays, gotRelays)
}
