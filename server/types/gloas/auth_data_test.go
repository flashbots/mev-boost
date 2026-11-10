package gloas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAuthData(t *testing.T) {
	data, err := ParseAuthData("relay-a")
	require.NoError(t, err)
	require.Equal(t, []byte("relay-a"), data)

	data, err = ParseAuthData("0x0102")
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2}, data)

	_, err = ParseAuthData("")
	require.ErrorIs(t, err, ErrEmptyAuthData)
	_, err = ParseAuthData("0x")
	require.ErrorIs(t, err, ErrEmptyAuthData)
	_, err = ParseAuthData("0xzz")
	require.Error(t, err)
	_, err = ParseAuthData(string(make([]byte, MaxBuilderAuthDataSize+1)))
	require.ErrorIs(t, err, ErrAuthDataTooLarge)
}

// TestDefaultAuthData uses the table from builder-specs specs/gloas/validator.md.
func TestDefaultAuthData(t *testing.T) {
	cases := map[string]string{
		"https://builder.example.com/":             "builder.example.com",
		"HTTPS://Builder.Example.com:443/bids?x=1": "builder.example.com",
		"https://builder.example.com:8080":         "builder.example.com",
		"https://user:pw@builder.example.com/":     "builder.example.com",
		"https://10.0.0.5:18550/eth/v1/builder":    "10.0.0.5",
		"https://[0:0:0:0:0:0:0:1]:8443/":          "[::1]",
		"https://[::ffff:192.0.2.1]/":              "[::ffff:c000:201]",
		// Relay URLs as mev-boost configures them, with the relay pubkey as userinfo.
		"https://0xac6e77dfe25ecd6110b8e780608cce0dab71fdd5ebea22a16c0205200f2f8e2e3ad3b71d3499c54ad14d6c21b41a37ae@boost-relay.flashbots.net": "boost-relay.flashbots.net",
		"http://localhost:18550": "localhost",
		// Additional RFC 5952 cases.
		"http://[2001:db8:0:0:1:0:0:1]/":    "[2001:db8::1:0:0:1]",
		"http://[2001:0db8::0001]/":         "[2001:db8::1]",
		"http://[0:0:0:0:0:0:0:0]/":         "[::]",
		"http://[2001:db8:0:1:1:1:1:1]/":    "[2001:db8:0:1:1:1:1:1]",
		"http://[1:0:0:2:0:0:0:3]/":         "[1:0:0:2::3]",
		"http://[fe80::1]:8080/":            "[fe80::1]",
		"http://[2001:DB8:AAAA:BBBB::CC]/":  "[2001:db8:aaaa:bbbb::cc]",
		"http://[::ffff:0.0.0.0]/":          "[::ffff:0:0]",
		"http://[::1234:5678:1.2.3.4]:443/": "[::1234:5678:102:304]",
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := DefaultAuthData(input)
			require.NoError(t, err)
			require.Equal(t, expected, string(got))
		})
	}

	_, err := DefaultAuthData("https:///path")
	require.ErrorIs(t, err, ErrMissingHostname)
	_, err = DefaultAuthData("://bad")
	require.Error(t, err)
}
