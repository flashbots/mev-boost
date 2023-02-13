package server

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetHeaderResponseJSON(t *testing.T) {
	testHeaderFiles := []string{
		"../testdata/signed-builder-bid-bellatrix.json",
		"../testdata/signed-builder-bid-capella.json",
	}

	for _, fn := range testHeaderFiles {
		t.Run(fn, func(t *testing.T) {
			jsonBytes, err := os.ReadFile(fn)
			require.NoError(t, err)

			getHeaderResponse := new(GetHeaderResponse)
			err = DecodeJSON(bytes.NewReader(jsonBytes), getHeaderResponse)

			require.NoError(t, err)
			o := new(bytes.Buffer)
			err = json.NewEncoder(o).Encode(&getHeaderResponse)
			require.NoError(t, err)

			i := new(bytes.Buffer)
			err = json.Compact(i, jsonBytes)
			require.NoError(t, err)

			require.Equal(t, i.String(), strings.TrimSpace(o.String()))
		})
	}
}
