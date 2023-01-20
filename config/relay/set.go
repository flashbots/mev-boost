package relay

// Set is a set of relay entries.
type Set map[string]Entry

// NewRelaySet creates a new instance of Set.
func NewRelaySet() Set {
	return make(Set)
}

// Add adds a new relay entry to the list.
func (s Set) Add(entry Entry) {
	s[entry.RelayURL().String()] = entry
}

// AddURL creates a new relay entry from relayURL and adds it to the list.
func (s Set) AddURL(relayURL string) error {
	entry, err := NewRelayEntry(relayURL)
	if err != nil {
		return err
	}

	s[entry.RelayURL().String()] = entry

	return nil
}

// Set creates a new relay entry from relayURL and adds it to the list.
//
// Implements flag.Value interface.
func (s Set) Set(relayURL string) error {
	return s.AddURL(relayURL)
}

// String returns a comma separated string of relay urls.
// Implements fmt.Stringer interface.
func (s Set) String() string {
	return s.ToList().String()
}

// ToStringSlice returns a string slice of relay urls.
func (s Set) ToStringSlice() []string {
	return s.ToList().ToStringSlice()
}

// ToList returns a slice of all relay entries.
func (s Set) ToList() List {
	relayList := make(List, 0, len(s))
	for _, entry := range s {
		relayList = append(relayList, entry)
	}

	return relayList
}
