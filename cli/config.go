package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/flashbots/mev-boost/server/types"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type RelayConfigYAML struct {
	URL                  string `yaml:"url"`
	EnableTimingGames    bool   `yaml:"enable_timing_games"`
	TargetFirstRequestMs uint64 `yaml:"target_first_request_ms"`
	FrequencyGetHeaderMs uint64 `yaml:"frequency_get_header_ms"`
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

// ConfigWatcher provides hot reloading of config files
type ConfigWatcher struct {
	v              *viper.Viper
	configPath     string
	cliRelays      []types.RelayEntry
	onConfigChange func(*ConfigResult)
	log            *logrus.Entry
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
	return parseConfig(config)
}

// NewConfigWatcher creates a new config file watcher
func NewConfigWatcher(configPath string, cliRelays []types.RelayEntry, log *logrus.Entry) (*ConfigWatcher, error) {
	v := viper.New()
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	v.SetConfigFile(absPath)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	return &ConfigWatcher{
		v:          v,
		configPath: absPath,
		cliRelays:  cliRelays,
		log:        log,
	}, nil
}

// Watch starts watching the config file for changes
func (cw *ConfigWatcher) Watch(onConfigChange func(*ConfigResult)) {
	cw.onConfigChange = onConfigChange

	cw.v.OnConfigChange(func(_ fsnotify.Event) {
		cw.log.Info("config file changed, reloading...")

		// explicitly read the file to get the latest content since viper
		// may cache the old value and not read the file immediately.
		data, err := os.ReadFile(cw.configPath)
		if err != nil {
			cw.log.WithError(err).Error("failed to read new config file, keeping old config")
			return
		}

		var config Config
		if err := yaml.Unmarshal(data, &config); err != nil {
			cw.log.WithError(err).Error("failed to unmarshal new config, keeping old config")
			return
		}
		newConfig, err := parseConfig(config)
		if err != nil {
			cw.log.WithError(err).Error("failed to parse new config, keeping old config")
			return
		}

		cw.log.Infof("successfully loaded new config with %d relays from config file", len(newConfig.RelayConfigs))

		if cw.onConfigChange != nil {
			cw.onConfigChange(newConfig)
		}
	})

	cw.v.WatchConfig()
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

func parseConfig(config Config) (*ConfigResult, error) {
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
