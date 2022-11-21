package relay

import "strings"

// List is a read only list of relay entities.
//
// It is utilised by the other components, e.g. relay.Set.
// It must not be modified once created.
// It must not be instantiated directly, except for testing purposes.
type List []Entry

// String returns a comma separated string of relay urls.
//
// Implements fmt.Stringer interface
func (l List) String() string {
	return strings.Join(l.ToStringSlice(), ",")
}

// ToStringSlice returns a string slice of relay urls.
func (l List) ToStringSlice() []string {
	relays := make([]string, len(l))
	for i, entry := range l {
		relays[i] = entry.String()
	}

	return relays
}
