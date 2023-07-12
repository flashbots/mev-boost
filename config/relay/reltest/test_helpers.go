package reltest

import (
	"net/url"
	"testing"

	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/stretchr/testify/require"
)

func RandomRelaySet(tb testing.TB, num int) relay.Set {
	tb.Helper()

	s := relay.NewRelaySet()
	for i := 0; i < num; i++ {
		s.Add(RandomRelayEntry(tb))
	}

	return s
}

func RelaySetWithRelaysHavingTheSameURL(tb testing.TB, num int) relay.Set {
	tb.Helper()

	relayURL := RandomRelayURL(tb)

	s := relay.NewRelaySet()
	for i := 0; i < num; i++ {
		s.Add(RelayEntryFromURL(tb, relayURL))
	}

	return s
}

func RandomRelayList(tb testing.TB, num int) relay.List {
	tb.Helper()

	list := make(relay.List, num)
	for i := 0; i < num; i++ {
		list[i] = RandomRelayEntry(tb)
	}

	return list
}

func RandomRelayEntry(tb testing.TB) relay.Entry {
	tb.Helper()

	return RelayEntryFromURL(tb, RandomRelayURL(tb))
}

func RelayEntryFromURL(tb testing.TB, relayURL *url.URL) relay.Entry {
	tb.Helper()

	relayEntry, err := relay.NewRelayEntry(relayURL.String())
	require.NoError(tb, err)

	return relayEntry
}

func RandomRelayURL(tb testing.TB) *url.URL {
	tb.Helper()

	blsPublicKey := RandomBLSPublicKey(tb)

	relayURL := &url.URL{
		Scheme: "https",
		User:   url.User(blsPublicKey.String()),
		Host:   "relay.test.net",
	}

	return relayURL
}

func RandomBLSPublicKey(tb testing.TB) types.PublicKey {
	tb.Helper()

	_, blsPublicKey, err := bls.GenerateNewKeypair()
	require.NoError(tb, err)

	publicKey, err := types.BlsPublicKeyToPublicKey(blsPublicKey)
	require.NoError(tb, err)

	return publicKey
}

func JoinSets(sets ...relay.Set) relay.Set {
	want := relay.NewRelaySet()

	for _, set := range sets {
		for _, entry := range set {
			want.Add(entry)
		}
	}

	return want
}

func PopulateSetFromList(s relay.Set, relays relay.List) {
	for _, entry := range relays {
		s.Add(entry)
	}
}

func RelaySetFromList(relays relay.List) relay.Set {
	s := relay.NewRelaySet()
	PopulateSetFromList(s, relays)

	return s
}
