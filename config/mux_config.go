package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/flashbots/mev-boost/server/types"
)

var (
	ErrMuxIDEmpty               = errors.New("mux entry id cannot be empty")
	ErrDuplicateMuxID           = errors.New("duplicate mux entry id")
	ErrMuxNoRelays              = errors.New("mux entry must have at least one relay")
	ErrMuxNoPubkeys             = errors.New("mux entry must have at least one validator pubkey")
	ErrDuplicateValidatorPubkey = errors.New("validator pubkey appears in multiple mux entries")
	ErrInvalidValidatorPubkey   = errors.New("invalid validator pubkey")

	errMissingHexPrefix  = errors.New("missing 0x prefix")
	errWrongPubkeyLength = errors.New("wrong length")
)

type RuntimeMuxConfig struct {
	ID                 string
	RelayConfigs       []types.RelayConfig
	TimeoutGetHeaderMs uint64
	LateInSlotTimeMs   uint64
}

// MuxMap maps validator pubkeys to their RuntimeMuxConfig
type MuxMap map[string]*RuntimeMuxConfig

type MuxEntryInput struct {
	ID                 string
	ValidatorPubkeys   []string
	RelayConfigs       []types.RelayConfig
	TimeoutGetHeaderMs uint64
	LateInSlotTimeMs   uint64
}

// ValidateMuxEntries validates a list of mux entry inputs and returns a MuxMap
func ValidateMuxEntries(entries []MuxEntryInput) (MuxMap, error) {
	if len(entries) == 0 {
		return MuxMap{}, nil
	}

	muxMap := make(MuxMap)
	seenIDs := make(map[string]bool)

	for _, entry := range entries {
		if entry.ID == "" {
			return nil, ErrMuxIDEmpty
		}
		if seenIDs[entry.ID] {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateMuxID, entry.ID)
		}
		seenIDs[entry.ID] = true

		if len(entry.RelayConfigs) == 0 {
			return nil, fmt.Errorf("mux %s: %w", entry.ID, ErrMuxNoRelays)
		}
		if len(entry.ValidatorPubkeys) == 0 {
			return nil, fmt.Errorf("mux %s: %w", entry.ID, ErrMuxNoPubkeys)
		}

		runtimeConfig := &RuntimeMuxConfig{
			ID:                 entry.ID,
			RelayConfigs:       entry.RelayConfigs,
			TimeoutGetHeaderMs: entry.TimeoutGetHeaderMs,
			LateInSlotTimeMs:   entry.LateInSlotTimeMs,
		}

		for _, pubkey := range entry.ValidatorPubkeys {
			// Validate and canonicalize. getHeader looks the validator up by
			// the pubkey the beacon node puts in the request URL, so a mux
			// keyed on a malformed or differently-cased string can never
			// match, and does so silently.
			canonical, err := canonicalValidatorPubkey(pubkey)
			if err != nil {
				return nil, fmt.Errorf("mux %s: %w %q: %w", entry.ID, ErrInvalidValidatorPubkey, pubkey, err)
			}

			if existing, ok := muxMap[canonical]; ok {
				return nil, fmt.Errorf("%w: %s in mux %q and %q", ErrDuplicateValidatorPubkey, canonical, existing.ID, entry.ID)
			}
			muxMap[canonical] = runtimeConfig
		}
	}

	return muxMap, nil
}

// blsPubkeyLen is the length of a BLS public key in bytes.
const blsPubkeyLen = 48

// canonicalValidatorPubkey checks that pubkey is a 0x-prefixed, 48-byte hex
// string and returns it lowercased.
//
// This is deliberately a syntactic check rather than a full BLS point
// decompression: these pubkeys are only ever used as lookup keys against the
// pubkey in a getHeader URL, which mev-boost does not curve-check either.
// Validating the encoding catches the mistakes that actually silence a mux --
// a truncated or mistyped key, a missing 0x prefix, or the wrong case --
// without rejecting the well-formed placeholder keys used in test and staging
// configs.
func canonicalValidatorPubkey(pubkey string) (string, error) {
	// The prefix is matched case-insensitively for the same reason the body
	// is: hex does not carry case.
	if len(pubkey) < 2 || !strings.EqualFold(pubkey[:2], "0x") {
		return "", errMissingHexPrefix
	}
	decoded, err := hex.DecodeString(pubkey[2:])
	if err != nil {
		return "", err
	}
	if len(decoded) != blsPubkeyLen {
		return "", fmt.Errorf("%w: got %d bytes, want %d", errWrongPubkeyLength, len(decoded), blsPubkeyLen)
	}
	return strings.ToLower(pubkey), nil
}
