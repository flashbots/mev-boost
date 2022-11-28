package relay

import (
	"strings"
	"sync"
)

// Registry keeps a thread-save in-memory registry of all known relays.
//
// It contains a mapping for all added validators and their corresponding relays.
// It also has default fallback relays which are used when a provided validator has no relays.
type Registry struct {
	relaysMu       sync.RWMutex
	defaultRelays  Set
	relaysByPubKey relaysByValidatorPubKey
}

// NewProposerRegistry creates a new instance of Registry.
func NewProposerRegistry() *Registry {
	return &Registry{
		defaultRelays:  NewRelaySet(),
		relaysByPubKey: newRelaysByValidatorPubKey(),
	}
}

// AddRelayForValidator adds a relay entry for the given validator.
func (r *Registry) AddRelayForValidator(key ValidatorPublicKey, entry Entry) {
	r.relaysMu.Lock()
	defer r.relaysMu.Unlock()

	r.relaysByPubKey.Add(key, entry)
}

// AddDefaultRelay adds a default relay.
func (r *Registry) AddDefaultRelay(entry Entry) {
	r.relaysMu.Lock()
	defer r.relaysMu.Unlock()

	r.defaultRelays.Add(entry)
}

// RelaysForValidator returns all the relays for this given validator.
//
// If there are no relays for the provided key, then the default relays are returned.
func (r *Registry) RelaysForValidator(key ValidatorPublicKey) List {
	r.relaysMu.RLock()
	defer r.relaysMu.RUnlock()

	return r.relaysByPubKey.GetOrDefault(key, r.defaultRelays)
}

// AllRelays returns all known relays.
//
// It returns a unique set of validator relays and default relays.
func (r *Registry) AllRelays() List {
	r.relaysMu.RLock()
	defer r.relaysMu.RUnlock()

	relayList := NewRelaySet()

	for _, relays := range r.relaysByPubKey {
		for _, entry := range relays {
			relayList[entry.RelayURL().String()] = entry
		}
	}

	for _, entry := range r.defaultRelays {
		relayList[entry.RelayURL().String()] = entry
	}

	return relayList.ToList()
}

type relaysByValidatorPubKey map[ValidatorPublicKey]Set

func newRelaysByValidatorPubKey() relaysByValidatorPubKey {
	return make(relaysByValidatorPubKey)
}

func (r relaysByValidatorPubKey) Add(key ValidatorPublicKey, entry Entry) {
	key = strings.ToLower(key)
	if _, ok := r[key]; !ok {
		r[key] = make(Set)
	}

	r[key].Add(entry)
}

func (r relaysByValidatorPubKey) GetOrDefault(key ValidatorPublicKey, def Set) List {
	key = strings.ToLower(key)
	if s, ok := r[key]; ok {
		return s.ToList()
	}

	return def.ToList()
}
