package server

import (
	"sort"
	"strconv"
	"strings"
)

// AcceptEntry represents a parsed Accept header entry with q-weight.
type AcceptEntry struct {
	MediaType string
	QValue    float64
}

// ParseAcceptHeader parses and sorts the Accept header by q-values.
func ParseAcceptHeader(header string) []AcceptEntry {
	rawAcceptValues := strings.Split(header, ",")
	entries := make([]AcceptEntry, 0, len(rawAcceptValues))

	for _, part := range rawAcceptValues {
		mediaQ := strings.Split(strings.TrimSpace(part), ";")
		mediaType := mediaQ[0]
		qValue := 1.0 // Default q-value if not specified

		if len(mediaQ) > 1 && strings.HasPrefix(mediaQ[1], "q=") {
			if q, err := strconv.ParseFloat(strings.TrimPrefix(mediaQ[1], "q="), 64); err == nil {
				qValue = q
			}
		}

		entries = append(entries, AcceptEntry{MediaType: mediaType, QValue: qValue})
	}

	// Sort by q-value (highest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].QValue > entries[j].QValue
	})

	return entries
}
