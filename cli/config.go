package cli

import (
	"os"
	"strings"

	"github.com/flashbots/mev-boost/server/types"
	"gopkg.in/yaml.v3"
)

type RelayConfigYAML struct {
	URL                  string `yaml:"url"`
	EnableTimingGames    bool   `yaml:"enable_timing_games"`
	TargetFirstRequestMs uint64 `yaml:"target_first_request_ms"`
	FrequencyGetHeaderMs uint64 `yaml:"frequency_getheader_ms"`
}

type TimingGamesConfig struct {
	Relays []RelayConfigYAML `yaml:"relays"`
}

// LoadRelayConfigFile loads relay configurations from a YAML file
func LoadRelayConfigFile(configPath string) (map[string]types.RelayConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config TimingGamesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	configMap := make(map[string]types.RelayConfig)
	for _, relay := range config.Relays {
		relayEntry, err := types.NewRelayEntry(strings.TrimSpace(relay.URL))
		if err != nil {
			return nil, err
		}
		relayConfig := types.RelayConfig{
			RelayEntry:           relayEntry,
			EnableTimingGames:    relay.EnableTimingGames,
			TargetFirstRequestMs: relay.TargetFirstRequestMs,
			FrequencyGetHeaderMs: relay.FrequencyGetHeaderMs,
		}
		configMap[relayEntry.String()] = relayConfig
	}

	return configMap, nil
}

// MergeRelayConfigs merges relays passed via --relays to config file settings.
// this allows the users to still use --relays if they dont want to provide a config file
func MergeRelayConfigs(relays []types.RelayEntry, configMap map[string]types.RelayConfig) []types.RelayConfig {
	configs := make([]types.RelayConfig, 0, len(relays))

	for _, entry := range relays {
		if config, exists := configMap[entry.String()]; exists {
			config.RelayEntry = entry
			configs = append(configs, config)
		} else {
			configs = append(configs, types.NewRelayConfig(entry))
		}
	}

	return configs
}
