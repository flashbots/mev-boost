package relay_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
)

func TestProposerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("it returns relays for a given validator", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := testutil.RandomRelaySet(t, 3)
		pubKey := testutil.RandomBLSPublicKey(t).String()

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, want.ToList()...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.Equal(t, got, want)
	})

	t.Run("it returns default relays if validator is unknown", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := testutil.RandomRelaySet(t, 3)
		pubKey := testutil.RandomBLSPublicKey(t).String()

		sut := relay.NewProposerRegistry()
		addDefaultRelays(sut, want.ToList()...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.Equal(t, got, want)
	})

	t.Run("it only adds the last proposer relay if a few relays with the same url are added", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := testutil.RandomBLSPublicKey(t).String()
		want := testutil.RelaySetWithRelayHavingTheSameURL(t, 3)

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, want.ToList()...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.Equal(t, got, want)
		assert.Len(t, want, 1)
	})

	t.Run("it only adds the last default relay if a few relays with the same url are added", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := testutil.RandomBLSPublicKey(t).String()
		want := testutil.RelaySetWithRelayHavingTheSameURL(t, 3)

		sut := relay.NewProposerRegistry()
		addDefaultRelays(sut, want.ToList()...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.Equal(t, got, want)
		assert.Len(t, want, 1)
	})

	t.Run("it returns proposer and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := testutil.RandomBLSPublicKey(t).String()
		want := testutil.RandomRelaySet(t, 3).ToList()

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, want[0])
		addDefaultRelays(sut, want[1:]...)

		// act
		got := sut.AllRelays()

		// assert
		assert.ElementsMatch(t, want, got.ToList())
		assert.Len(t, want, 3)
	})

	t.Run("it returns a unique set of proposer relays and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := testutil.RandomBLSPublicKey(t).String()
		proposerRelays := testutil.RelaySetWithRelayHavingTheSameURL(t, 3)
		defaultRelays := testutil.RelaySetWithRelayHavingTheSameURL(t, 2)
		want := testutil.JoinSets(proposerRelays, defaultRelays)

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, proposerRelays.ToList()...)
		addDefaultRelays(sut, defaultRelays.ToList()...)

		// act
		got := sut.AllRelays()

		// assert
		assert.Equal(t, want, got)
		assert.Len(t, want, 2)
	})
}

func addRelaysForValidator(r *relay.Registry, pubKey relay.ValidatorPublicKey, entry ...relay.Entry) {
	for _, e := range entry {
		r.AddRelayForValidator(pubKey, e)
	}
}

func addDefaultRelays(r *relay.Registry, entry ...relay.Entry) {
	for _, e := range entry {
		r.AddDefaultRelay(e)
	}
}
