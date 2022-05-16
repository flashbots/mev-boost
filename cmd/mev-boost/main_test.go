package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRelayURLs(t *testing.T) {
	tests := map[string]string{
		"foo.com":            "http://foo.com",
		"0x12341234@foo.com": "http://foo.com",
		"http://foo.com":     "http://foo.com",

		"https://0x12341234@foo.com":      "https://foo.com",
		"https://0x12341234@foo.com:3272": "https://foo.com:3272",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			e, err := parseRelayURL(test)
			require.NoError(t, err)
			require.Equal(t, test, e.Address)
		})
	}
}
