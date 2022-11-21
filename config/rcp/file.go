package rcp

import (
	"encoding/json"
	"os"

	"github.com/flashbots/mev-boost/config/relay"
)

type File struct {
	filePath string
}

func NewFile(filePath string) *File {
	return &File{
		filePath: filePath,
	}
}

func (f *File) FetchConfig() (*relay.Config, error) {
	configFile, err := os.Open(f.filePath)
	if err != nil {
		return nil, err
	}

	defer configFile.Close()

	var cfg *relay.Config
	if err := json.NewDecoder(configFile).Decode(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
