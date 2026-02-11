package config

import (
	"errors"
	"fmt"

	"github.com/flashbots/mev-boost/server/types"
)

var (
	ErrMuxIDEmpty               = errors.New("mux entry id cannot be empty")
	ErrDuplicateMuxID           = errors.New("duplicate mux entry id")
	ErrMuxNoRelays              = errors.New("mux entry must have at least one relay")
	ErrMuxNoPubkeys             = errors.New("mux entry must have at least one validator pubkey")
	ErrDuplicateValidatorPubkey = errors.New("validator pubkey appears in multiple mux entries")
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
			if existing, ok := muxMap[pubkey]; ok {
				return nil, fmt.Errorf("%w: %s in mux %q and %q", ErrDuplicateValidatorPubkey, pubkey, existing.ID, entry.ID)
			}
			muxMap[pubkey] = runtimeConfig
		}
	}

	return muxMap, nil
}
