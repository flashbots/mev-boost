package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAcceptHeader(t *testing.T) {
	testCases := []struct {
		header   string
		expected string
	}{
		{"", MediaTypeJSON},
		{"*/*", MediaTypeJSON},
		{"application/json", MediaTypeJSON},
		{"application/octet-stream", MediaTypeOctetStream},
		{"application/invalid", MediaTypeJSON},
		{"application/invalid;q=1,application/octet-stream;q=0.1", MediaTypeOctetStream},
		{"application/octet-stream;q=0.5,application/json;q=1", MediaTypeJSON},
		{"application/octet-stream;q=1,application/json;q=0.1", MediaTypeOctetStream},
		{"application/octet-stream;q=1,application/json;q=0.9", MediaTypeOctetStream},
		{"application/octet-stream;q=1,*/*;q=0.9", MediaTypeOctetStream},
		{"application/octet-stream,application/json;q=0.1", MediaTypeOctetStream},
		{"application/octet-stream;,application/json;q=0.1", MediaTypeJSON},          // Malformed entry
		{"application/octet-stream;q=2,application/json;q=0.1", MediaTypeJSON},       // Invalid q-value >1, fallback
		{"application/octet-stream;q=invalid,application/json;q=0.1", MediaTypeJSON}, // Invalid q-value, ignored
		{"application/octet-stream  ; q=0.5 , application/json ; q=1", MediaTypeJSON},
		{"application/octet-stream  ; q=1 , application/json ; q=0.1", MediaTypeOctetStream},
		{"application/octet-stream;q=1,application/json;q=0.1", MediaTypeOctetStream},
		{
			// Default Chrome Accept header, JSON should be preferred
			"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
			MediaTypeJSON,
		},
		// Order-dependent tests (last one wins if q-values are the same)
		{"application/octet-stream;q=1,application/json;q=1", MediaTypeJSON},
		{"application/json;q=1,application/octet-stream;q=1", MediaTypeOctetStream},
	}

	supportedMediaTypes := []string{MediaTypeJSON, MediaTypeOctetStream}

	for _, tc := range testCases {
		t.Run(tc.header, func(t *testing.T) {
			parsed := ParseAcceptHeader(tc.header)
			selected := SelectHighestQualityValueMediaType(parsed, supportedMediaTypes, MediaTypeJSON)
			require.Equal(t, tc.expected, selected)
		})
	}
}
