package testutil

import (
	"net/url"
	"testing"

	"github.com/flashbots/go-boost-utils/bls"
	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/server"
	"github.com/stretchr/testify/require"
)

func RandomRCPRelayEntries(t *testing.T, num int) []rcm.RelayEntry {
	t.Helper()

	return server.RelayEntriesToRCPRelayEntries(RandomRelayEntries(t, num))
}

func RandomRelayEntries(t *testing.T, num int) []server.RelayEntry {
	t.Helper()

	relays := make([]server.RelayEntry, num)

	for i := 0; i < num; i++ {
		relays[i] = RandomRelayEntry(t)
	}

	return relays
}

func RandomRelayEntry(t *testing.T) server.RelayEntry {
	t.Helper()

	blsPublicKey := RandomBLSPublicKey(t)

	relayURL := url.URL{
		Scheme: "https",
		User:   url.User(blsPublicKey.String()),
		Host:   "relay.test.net",
	}

	relayEntry, err := server.NewRelayEntry(relayURL.String())
	require.NoError(t, err)

	return relayEntry
}

func RandomBLSPublicKey(t *testing.T) types.PublicKey {
	t.Helper()

	_, blsPublicKey, err := bls.GenerateNewKeypair()
	require.NoError(t, err)

	publicKey, err := types.BlsPublicKeyToPublicKey(blsPublicKey)
	require.NoError(t, err)

	return publicKey
}
