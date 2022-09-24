package cli

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestTrim(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		input := []string{}
		expected := []string{}
		require.Equal(t, expected, trim(input))
	})

	t.Run("single item no padding", func(t *testing.T) {
		input := []string{"a"}
		expected := []string{"a"}
		require.Equal(t, expected, trim(input))
	})

	t.Run("single item with padding", func(t *testing.T) {
		input := []string{" a "}
		expected := []string{"a"}
		require.Equal(t, expected, trim(input))
	})

	t.Run("multiple items with padding", func(t *testing.T) {
		input := []string{" a ", " b "}
		expected := []string{"a", "b"}
		require.Equal(t, expected, trim(input))
	})

	t.Run("multiple items with padding one without", func(t *testing.T) {
		input := []string{" a ", "b", " c "}
		expected := []string{"a", "b", "c"}
		require.Equal(t, expected, trim(input))
	})

	t.Run("single item tab padding", func(t *testing.T) {
		input := []string{"\ta\t"}
		expected := []string{"a"}
		require.Equal(t, expected, trim(input))
	})

	t.Run("single item spaces and tabs", func(t *testing.T) {
		input := []string{" \t a \t "}
		expected := []string{"a"}
		require.Equal(t, expected, trim(input))
	})

	t.Run("filters out empty string", func(t *testing.T) {
		input := []string{""}
		expected := []string{}
		require.Equal(t, expected, trim(input))
	})

	t.Run("filters out string with only whitespace", func(t *testing.T) {
		input := []string{" "}
		expected := []string{}
		require.Equal(t, expected, trim(input))
	})

	t.Run("filters out multiple strings with only whitespace", func(t *testing.T) {
		input := []string{" ", "", " "}
		expected := []string{}
		require.Equal(t, expected, trim(input))
	})
}

func TestUnique(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		input := []string{}
		expected := []string{}
		require.Equal(t, expected, unique(input))
	})

	t.Run("single item", func(t *testing.T) {
		input := []string{"a"}
		expected := []string{"a"}
		require.Equal(t, expected, unique(input))
	})

	t.Run("already unique", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		expected := []string{"a", "b", "c"}
		require.Equal(t, expected, unique(input))
	})

	t.Run("single duplicate", func(t *testing.T) {
		input := []string{"a", "a"}
		expected := []string{"a"}
		require.Equal(t, expected, unique(input))
	})

	t.Run("multiple duplicates", func(t *testing.T) {
		input := []string{"a", "a", "a"}
		expected := []string{"a"}
		require.Equal(t, expected, unique(input))
	})

	t.Run("multiple duplicates different items", func(t *testing.T) {
		input := []string{"a", "b", "a", "b"}
		expected := []string{"a", "b"}
		require.Equal(t, expected, unique(input))
	})

	t.Run("single duplicate not adjacent", func(t *testing.T) {
		input := []string{"a", "b", "c", "a"}
		expected := []string{"a", "b", "c"}
		require.Equal(t, expected, unique(input))
	})
}

var (
	aRelay = "https://0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@example.com"
	bRelay = "https://0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb@example.com"
	cRelay = "https://0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc@example.com"
)

func TestParseRelayURLs(t *testing.T) {
	t.Run("no relay", func(t *testing.T) {
		input := ""
		require.Len(t, parseRelayURLs(input), 0)
	})

	t.Run("no relay but comma", func(t *testing.T) {
		input := ","
		require.Len(t, parseRelayURLs(input), 0)
	})

	t.Run("no relay but commas and whitespace", func(t *testing.T) {
		input := " ,\t, "
		require.Len(t, parseRelayURLs(input), 0)
	})

	t.Run("single relay", func(t *testing.T) {
		input := aRelay
		require.Len(t, parseRelayURLs(input), 1)
	})

	t.Run("multiple relays", func(t *testing.T) {
		input := aRelay + "," + bRelay
		require.Len(t, parseRelayURLs(input), 2)
	})

	t.Run("one relay and a duplicate", func(t *testing.T) {
		input := aRelay + "," + aRelay
		require.Len(t, parseRelayURLs(input), 1)
	})

	t.Run("two relays and a duplicate", func(t *testing.T) {
		input := aRelay + "," + bRelay + "," + aRelay
		require.Len(t, parseRelayURLs(input), 2)
	})

	t.Run("three relays", func(t *testing.T) {
		input := aRelay + "," + bRelay + "," + cRelay
		require.Len(t, parseRelayURLs(input), 3)
	})

	t.Run("single relay comma at end", func(t *testing.T) {
		input := aRelay + ","
		require.Len(t, parseRelayURLs(input), 1)
	})

	t.Run("single relay comma at beginning", func(t *testing.T) {
		input := "," + aRelay
		require.Len(t, parseRelayURLs(input), 1)
	})

	t.Run("two relays space after comma", func(t *testing.T) {
		input := aRelay + ", " + bRelay
		require.Len(t, parseRelayURLs(input), 2)
	})

	t.Run("two relays tab after comma", func(t *testing.T) {
		input := aRelay + ",\t" + bRelay
		require.Len(t, parseRelayURLs(input), 2)
	})
}
