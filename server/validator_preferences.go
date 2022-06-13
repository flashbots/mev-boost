package server

import (
	"github.com/flashbots/go-boost-utils/types"
	"sync"
	"time"
)

// ValidatorPreferences is used to store each validator preferences,
// which will then be used to update the relay's state.
//
// Using a map ensures that the most up-to-date preferences is the one stored.
type ValidatorPreferences struct {
	mu          sync.RWMutex
	preferences map[string]types.SignedValidatorRegistration

	interval time.Duration
}

// Save adds the given validator preferences to the safe storage.
func (v *ValidatorPreferences) Save(svr types.SignedValidatorRegistration) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.preferences[svr.Message.Pubkey.String()] = svr
}

// PreparePayload uses the safe map for validator preferences to create a list of these.
func (v *ValidatorPreferences) PreparePayload() []types.SignedValidatorRegistration {
	// We don't allocate memory here because the map's size might have increased before the mutex is unlocked.
	var payload []types.SignedValidatorRegistration

	v.mu.RLock()
	defer v.mu.RUnlock()

	for _, preferences := range v.preferences {
		payload = append(payload, preferences)
	}

	return payload
}
