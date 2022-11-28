package testutil

import (
	"math/rand"
	"net/url"
	"sync"
	"sync/atomic"
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

func RelaySetWithRelayHavingTheSameURL(t *testing.T, num int) relay.Set {
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

func RunConcurrentlyAndCountFnCalls(numOfWorkers, num int64, fn func(*rand.Rand, int64)) uint64 {
	var (
		count uint64
		wg    sync.WaitGroup
	)

	for g := numOfWorkers; g > 0; g-- {
		r := rand.New(rand.NewSource(g))
		wg.Add(1)

		go func(r *rand.Rand) {
			defer wg.Done()

			for n := int64(1); n <= num; n++ {
				fn(r, n)
				atomic.AddUint64(&count, 1)
			}
		}(r)
	}

	wg.Wait()

	return count
}
