package rcp

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/flashbots/mev-boost/config/relay"
)

// File loads relay configuration from a file.
type File struct {
	filePath string
}

// NewFile creates a new instance of File.
//
// It takes a file path as a parameter.
func NewFile(filePath string) *File {
	return &File{
		filePath: filePath,
	}
}

// FetchConfig loads relay configuration from a file.
//
// It returns *relay.Config on success.
// It returns an error if it cannot read the file.
// It returns an error if the file has unexpected contents.
func (f *File) FetchConfig() (*relay.Config, error) {
	configFile, err := os.Open(f.filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch config: %w", err)
	}

	defer configFile.Close()

	var cfg *relay.Config
	if err := json.NewDecoder(configFile).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("cannot fetch config: %w", err)
	}

	return cfg, nil
}
