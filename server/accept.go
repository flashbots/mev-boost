package server

import (
	"strconv"
	"strings"
)

type AcceptEntry struct {
	MediaType string
	Quality   float64
	pos       int // position in the header (lower=earlier)
}

// ParseAcceptHeader takes an Accept header string and returns a slice of valid entries.
// It splits the header by commas and for each part, it splits by semicolon to find
// a possible "q" parameter. If the part is malformed (for example, a trailing semicolon
// with no parameter or a q value that cannot be parsed or is not in [0,1]), the entry is ignored.
func ParseAcceptHeader(header string) []AcceptEntry {
	parts := strings.Split(header, ",")
	entries := make([]AcceptEntry, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Split on semicolon; first token is the media-range.
		tokens := strings.Split(part, ";")
		mediaType := strings.TrimSpace(tokens[0])
		if mediaType == "" {
			continue
		}
		// Default q is 1.
		quality := 1.0
		valid := true
		// Process parameters.
		for _, token := range tokens[1:] {
			token = strings.TrimSpace(token)
			// If a parameter is empty (for example, a trailing semicolon) then consider the entry malformed.
			if token == "" {
				valid = false
				break
			}
			// Look for a "q" parameter.
			if strings.HasPrefix(token, "q=") {
				kv := strings.SplitN(token, "=", 2)
				if len(kv) != 2 {
					valid = false
					break
				}
				value := strings.TrimSpace(kv[1])
				f, err := strconv.ParseFloat(value, 64)
				if err != nil || f < 0 || f > 1 {
					valid = false
					break
				}
				quality = f
			}
			// Other parameters are ignored.
		}
		if !valid {
			continue
		}
		entries = append(entries, AcceptEntry{
			MediaType: mediaType,
			Quality:   quality,
			pos:       i,
		})
	}
	return entries
}

// SelectHighestQualityValueMediaType takes the parsed Accept header entries (from ParseAcceptHeader)
// and a slice of supported media types, then returns the one with the highest quality factor.
// A supported type is considered matching if it is an exact match or if the accept entry is "*/*".
// When quality factors are equal, the later (higher pos) accept entry “wins” – and if even that is tied,
// the order of supportedMediaTypes is used as a final tie‐breaker.
func SelectHighestQualityValueMediaType(entries []AcceptEntry, supportedMediaTypes []string) string {
	type candidate struct {
		mediaType string
		q         float64
		pos       int // position of the matching accept entry
		index     int // the index in the supportedMediaTypes slice
	}
	var bestCandidate *candidate
	// For each supported media type, find its best matching accept entry.
	for idx, supp := range supportedMediaTypes {
		bestQ := -1.0
		bestPos := -1
		found := false
		for _, e := range entries {
			if e.MediaType == supp || e.MediaType == "*/*" {
				// If this entry has a higher q or, in case of equal q, a later position,
				// then it is preferred.
				if e.Quality > bestQ || (e.Quality == bestQ && e.pos > bestPos) {
					bestQ = e.Quality
					bestPos = e.pos
					found = true
				}
			}
		}
		if found {
			cand := candidate{
				mediaType: supp,
				q:         bestQ,
				pos:       bestPos,
				index:     idx,
			}
			if bestCandidate == nil {
				bestCandidate = &cand
			} else {
				// Compare candidates: higher q wins; if equal, then later accept header position wins;
				// if still equal, then the supportedMediaTypes order (lower index wins).
				if cand.q > bestCandidate.q ||
					(cand.q == bestCandidate.q && cand.pos > bestCandidate.pos) ||
					(cand.q == bestCandidate.q && cand.pos == bestCandidate.pos && cand.index < bestCandidate.index) {
					bestCandidate = &cand
				}
			}
		}
	}
	if bestCandidate != nil {
		return bestCandidate.mediaType
	}
	return ""
}
