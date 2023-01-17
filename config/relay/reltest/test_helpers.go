package reltest

import (
	"net/url"
	"testing"

	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/stretchr/testify/require"
)

func RandomRelaySet(t *testing.T, num int) relay.Set {
	t.Helper()

	s := relay.NewRelaySet()
	for i := 0; i < num; i++ {
		s.Add(RandomRelayEntry(t))
	}

	return s
}

func RelaySetWithRelaysHavingTheSameURL(t *testing.T, num int) relay.Set {
	t.Helper()

	relayURL := RandomRelayURL(t)

	s := relay.NewRelaySet()
	for i := 0; i < num; i++ {
		s.Add(RelayEntryFromURL(t, relayURL))
	}

	return s
}

func RandomRelayList(t *testing.T, num int) relay.List {
	t.Helper()

	list := make(relay.List, num)
	for i := 0; i < num; i++ {
		list[i] = RandomRelayEntry(t)
	}

	return list
}

func RandomRelayEntry(t *testing.T) relay.Entry {
	t.Helper()

	return RelayEntryFromURL(t, RandomRelayURL(t))
}

func RelayEntryFromURL(t *testing.T, relayURL *url.URL) relay.Entry {
	t.Helper()

	relayEntry, err := relay.NewRelayEntry(relayURL.String())
	require.NoError(t, err)

	return relayEntry
}

func RandomRelayURL(t *testing.T) *url.URL {
	t.Helper()

	blsPublicKey := RandomBLSPublicKey(t)

	relayURL := &url.URL{
		Scheme: "https",
		User:   url.User(blsPublicKey.String()),
		Host:   "relay.test.net",
	}

	return relayURL
}

func RandomBLSPublicKey(t *testing.T) types.PublicKey {
	t.Helper()

	_, blsPublicKey, err := bls.GenerateNewKeypair()
	require.NoError(t, err)

	publicKey, err := types.BlsPublicKeyToPublicKey(blsPublicKey)
	require.NoError(t, err)

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
