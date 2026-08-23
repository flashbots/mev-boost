package config

import (
	"strings"
	"testing"

	"github.com/flashbots/mev-boost/server/types"
	"github.com/stretchr/testify/require"
)

const (
	testPubkeyLower = "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
	testRelayURL    = "0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com"
)

func testRelayConfigs(t *testing.T) []types.RelayConfig {
	t.Helper()
	entry, err := types.NewRelayEntry(testRelayURL)
	require.NoError(t, err)
	return []types.RelayConfig{{RelayEntry: entry}}
}

// A validator pubkey that is not a well-formed BLS pubkey can never match a
// getHeader request, so accepting it silently leaves the mux permanently
// inactive. Reject it at startup instead.
func TestValidateMuxEntriesRejectsMalformedPubkey(t *testing.T) {
	for _, pubkey := range []string{
		"0xdeadbeef",           // too short
		"not-a-pubkey",         // missing 0x prefix
		"",                     // empty
		testPubkeyLower + "ff", // too long
	} {
		t.Run(pubkey, func(t *testing.T) {
			_, err := ValidateMuxEntries([]MuxEntryInput{{
				ID:               "mux",
				ValidatorPubkeys: []string{pubkey},
				RelayConfigs:     testRelayConfigs(t),
			}})
			require.Error(t, err, "malformed validator pubkey %q was accepted", pubkey)
		})
	}
}

// Hex pubkeys are case-insensitive, and the getHeader path looks the validator
// up by the pubkey the beacon node put in the URL. Keying the map on the raw
// config string makes an uppercase entry silently miss.
func TestValidateMuxEntriesCanonicalizesPubkeys(t *testing.T) {
	muxMap, err := ValidateMuxEntries([]MuxEntryInput{{
		ID:               "mux",
		ValidatorPubkeys: []string{"0x" + strings.ToUpper(testPubkeyLower[2:])},
		RelayConfigs:     testRelayConfigs(t),
	}})
	require.NoError(t, err)
	require.Contains(t, muxMap, testPubkeyLower,
		"mux map is not keyed by the canonical lowercase pubkey")

	// An uppercased 0X prefix is hex too.
	muxMap, err = ValidateMuxEntries([]MuxEntryInput{{
		ID:               "mux",
		ValidatorPubkeys: []string{strings.ToUpper(testPubkeyLower)},
		RelayConfigs:     testRelayConfigs(t),
	}})
	require.NoError(t, err)
	require.Contains(t, muxMap, testPubkeyLower)
}

// The duplicate check exists to stop one validator being claimed by two muxes.
// Comparing raw strings lets the same pubkey slip through in a different case.
func TestValidateMuxEntriesDetectsDuplicateAcrossCase(t *testing.T) {
	_, err := ValidateMuxEntries([]MuxEntryInput{
		{
			ID:               "mux-a",
			ValidatorPubkeys: []string{testPubkeyLower},
			RelayConfigs:     testRelayConfigs(t),
		},
		{
			ID:               "mux-b",
			ValidatorPubkeys: []string{strings.ToUpper(testPubkeyLower)},
			RelayConfigs:     testRelayConfigs(t),
		},
	})
	require.ErrorIs(t, err, ErrDuplicateValidatorPubkey)
}
