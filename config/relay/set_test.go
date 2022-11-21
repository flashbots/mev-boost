package relay_test

import (
	"testing"

	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRelaySet(t *testing.T) {
	t.Parallel()

	t.Run("it adds a relay entry", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := testutil.RandomRelayEntry(t)
		sut := relay.NewRelaySet()

		// act
		sut.Add(want)

		// assert
		assert.Contains(t, sut.ToList(), want)
	})
}
