package rcp_test

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/flashbots/mev-boost/config/rcp"
	"github.com/flashbots/mev-boost/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	_, b, _, _                      = runtime.Caller(0)
	root                            = filepath.Join(filepath.Dir(b), "../..")
	validProposerConfigFilePath     = filepath.Join(root, "testdata", "valid-proposer-config.json")
	corruptedProposerConfigFilePath = filepath.Join(root, "testdata", "corrupted-proposer-config.json")
)

func TestFileRelayConfigProvider(t *testing.T) {
	t.Parallel()

	t.Run("it returns relay config", func(t *testing.T) {
		t.Parallel()

		// arrange
		want := testdata.ValidProposerConfig(t)
		sut := rcp.NewFile(validProposerConfigFilePath)

		// act
		got, err := sut.FetchConfig()

		// assert
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("it returns an error if it cannot open the config file", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := rcp.NewFile("/non/existent/file/path")

		// act
		_, err := sut.FetchConfig()

		// assert
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("it returns an error if the config file has malformed contents", func(t *testing.T) {
		t.Parallel()

		// arrange
		sut := rcp.NewFile(corruptedProposerConfigFilePath)

		// act
		_, err := sut.FetchConfig()

		// assert
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}
