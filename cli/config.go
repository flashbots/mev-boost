package cli

import (
	"os"
	"strings"

	"github.com/flashbots/mev-boost/server/types"
	"gopkg.in/yaml.v3"
)

type RelayConfigYAML struct {
	URL                  string `yaml:"url"`
	ID                   string `yaml:"id"`
	EnableTimingGames    bool   `yaml:"enable_timing_games"`
	TargetFirstRequestMs uint64 `yaml:"target_first_request_ms"`
	FrequencyGetHeaderMs uint64 `yaml:"frequency_getheader_ms"`
}

// Config holds all configuration settings from the config file
type Config struct {
	TimeoutGetHeaderMs uint64            `yaml:"timeout_get_header_ms"`
	LateInSlotTimeMs   uint64            `yaml:"late_in_slot_time_ms"`
	Relays             []RelayConfigYAML `yaml:"relays"`
}

type ConfigResult struct {
	RelayConfigs       map[string]types.RelayConfig
	TimeoutGetHeaderMs uint64
	LateInSlotTimeMs   uint64
}

// LoadConfigFile loads configurations from a YAML file
func LoadConfigFile(configPath string) (*ConfigResult, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	timeoutGetHeaderMs := config.TimeoutGetHeaderMs
	if timeoutGetHeaderMs == 0 {
		timeoutGetHeaderMs = 900
	}

	lateInSlotTimeMs := config.LateInSlotTimeMs
	if lateInSlotTimeMs == 0 {
		lateInSlotTimeMs = 1000
	}

	configMap := make(map[string]types.RelayConfig)
	for _, relay := range config.Relays {
		relayEntry, err := types.NewRelayEntry(strings.TrimSpace(relay.URL))
		if err != nil {
			return nil, err
		}
		if relay.ID != "" {
			relayEntry.ID = relay.ID
		} else {
			relayEntry.ID = relayEntry.URL.String()
		}
		relayConfig := types.RelayConfig{
			RelayEntry:           relayEntry,
			EnableTimingGames:    relay.EnableTimingGames,
			TargetFirstRequestMs: relay.TargetFirstRequestMs,
			FrequencyGetHeaderMs: relay.FrequencyGetHeaderMs,
		}
		configMap[relayEntry.String()] = relayConfig
	}

	return &ConfigResult{
		RelayConfigs:       configMap,
		TimeoutGetHeaderMs: timeoutGetHeaderMs,
		LateInSlotTimeMs:   lateInSlotTimeMs,
	}, nil
}

// MergeRelayConfigs merges relays passed via --relays with config file settings.
// this allows the users to still use --relays if they dont want to provide a config file
func MergeRelayConfigs(relays []types.RelayEntry, configMap map[string]types.RelayConfig) []types.RelayConfig {
	configs := make([]types.RelayConfig, 0)
	processedURLs := make(map[string]bool)

	for _, entry := range relays {
		urlStr := entry.String()
		if config, exists := configMap[urlStr]; exists {
			config.RelayEntry = entry
			configs = append(configs, config)
		} else {
			configs = append(configs, types.NewRelayConfig(entry))
		}
		processedURLs[urlStr] = true
	}

	for urlStr, config := range configMap {
		if !processedURLs[urlStr] {
			configs = append(configs, config)
		}
	}
	return configs
}
