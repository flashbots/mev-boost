package relay_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
	"github.com/stretchr/testify/assert"
)

func TestProposerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("it returns relays for a given validator", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := reltest.RandomRelaySet(t, 3).ToList()
		pubKey := reltest.RandomBLSPublicKey(t).String()

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, want...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.ElementsMatch(t, got, want)
	})

	t.Run("it returns default relays if validator is unknown", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := reltest.RandomRelaySet(t, 3).ToList()
		pubKey := reltest.RandomBLSPublicKey(t).String()

		sut := relay.NewProposerRegistry()
		addDefaultRelays(sut, want...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.ElementsMatch(t, got, want)
	})

	t.Run("it only adds the last proposer relay if a few relays with the same url are added", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := reltest.RandomBLSPublicKey(t).String()
		want := reltest.RelaySetWithRelayHavingTheSameURL(t, 3).ToList()

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, want...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.ElementsMatch(t, got, want)
		assert.Len(t, want, 1)
	})

	t.Run("it only adds the last default relay if a few relays with the same url are added", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := reltest.RandomBLSPublicKey(t).String()
		want := reltest.RelaySetWithRelayHavingTheSameURL(t, 3).ToList()

		sut := relay.NewProposerRegistry()
		addDefaultRelays(sut, want...)

		// act
		got := sut.RelaysForValidator(pubKey)

		// assert
		assert.ElementsMatch(t, got, want)
		assert.Len(t, want, 1)
	})

	t.Run("it returns proposer and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := reltest.RandomBLSPublicKey(t).String()
		want := reltest.RandomRelaySet(t, 3).ToList()

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, want[0])
		addDefaultRelays(sut, want[1:]...)

		// act
		got := sut.AllRelays()

		// assert
		assert.ElementsMatch(t, want, got)
		assert.Len(t, want, 3)
	})

	t.Run("it returns a unique set of proposer relays and default relays", func(t *testing.T) {
		t.Parallel()

		// arrange
		pubKey := reltest.RandomBLSPublicKey(t).String()
		proposerRelays := reltest.RelaySetWithRelayHavingTheSameURL(t, 3)
		defaultRelays := reltest.RelaySetWithRelayHavingTheSameURL(t, 2)
		want := reltest.JoinSets(proposerRelays, defaultRelays).ToList()

		sut := relay.NewProposerRegistry()
		addRelaysForValidator(sut, pubKey, proposerRelays.ToList()...)
		addDefaultRelays(sut, defaultRelays.ToList()...)

		// act
		got := sut.AllRelays()

		// assert
		assert.ElementsMatch(t, want, got)
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
