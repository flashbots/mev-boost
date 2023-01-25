package relay

import "strings"

// Registry keeps an in-memory registry of all known relays.
//
// It contains a mapping for all added validators and their corresponding relays.
// It also has default fallback relays which are used when a provided validator has no relays.
type Registry struct {
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

// AddRelayForProposer adds a relay entry for the given proposer.
func (r *Registry) AddRelayForProposer(key ValidatorPublicKey, entry Entry) {
	r.relaysByPubKey.Add(key, entry)
}

// AddEmptyProposer adds a proposer with an empty relay set.
func (r *Registry) AddEmptyProposer(key ValidatorPublicKey) {
	r.relaysByPubKey.AddEmptyProposer(key)
}

// AddDefaultRelay adds a default relay.
func (r *Registry) AddDefaultRelay(entry Entry) {
	r.defaultRelays.Add(entry)
}

// RelaysForProposer returns all the relays for this given proposer.
//
// If there are no relays for the provided key, then the default relays are returned.
func (r *Registry) RelaysForProposer(key ValidatorPublicKey) List {
	return r.relaysByPubKey.GetOrDefault(key, r.defaultRelays)
}

// AllRelays returns all known relays.
//
// It returns a unique set of proposer relays and default relays.
func (r *Registry) AllRelays() List {
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
	r.initIfNotExists(key)
	r[key].Add(entry)
}

func (r relaysByValidatorPubKey) AddEmptyProposer(key ValidatorPublicKey) {
	r.initIfNotExists(key)
}

func (r relaysByValidatorPubKey) initIfNotExists(key ValidatorPublicKey) {
	key = strings.ToLower(key)
	if _, ok := r[key]; !ok {
		r[key] = NewRelaySet()
	}
}

func (r relaysByValidatorPubKey) GetOrDefault(key ValidatorPublicKey, def Set) List {
	key = strings.ToLower(key)
	if s, ok := r[key]; ok {
		return s.ToList()
	}

	return def.ToList()
}
