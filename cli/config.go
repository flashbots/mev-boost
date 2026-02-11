package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	errRelayConfiguredTwice = errors.New("relay is specified in both cli flags and config file")
	errNoRelaysSpecified    = errors.New("no relays specified, provide via --relays and/or --config")
)

type RelayConfigYAML struct {
	URL                  string `yaml:"url"`
	EnableTimingGames    bool   `yaml:"enable_timing_games"`
	TargetFirstRequestMs uint64 `yaml:"target_first_request_ms"`
	FrequencyGetHeaderMs uint64 `yaml:"frequency_get_header_ms"`
}

type MuxEntryYAML struct {
	ID                 string            `yaml:"id"`
	ValidatorPubkeys   []string          `yaml:"validator_pubkeys"`
	Relays             []RelayConfigYAML `yaml:"relays"`
	TimeoutGetHeaderMs uint64            `yaml:"timeout_get_header_ms,omitempty"`
	LateInSlotTimeMs   uint64            `yaml:"late_in_slot_time_ms,omitempty"`
}

// Config holds all configuration settings from the config file
type Config struct {
	TimeoutGetHeaderMs uint64            `yaml:"timeout_get_header_ms"`
	LateInSlotTimeMs   uint64            `yaml:"late_in_slot_time_ms"`
	Relays             []RelayConfigYAML `yaml:"relays"`
	Mux                []MuxEntryYAML    `yaml:"mux"`
}

type ConfigResult struct {
	RelayConfigs       map[string]types.RelayConfig
	TimeoutGetHeaderMs uint64
	LateInSlotTimeMs   uint64
	MuxMap             config.MuxMap
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

// MergeRelayConfigs merges relays passed via --relay with relays from config file.
// Returns an error if the same relay appears in both places.
// Users should specify each relay in exactly one place either cli or config file.
func MergeRelayConfigs(relays []types.RelayEntry, configMap map[string]types.RelayConfig) ([]types.RelayConfig, error) {
	configs := make([]types.RelayConfig, 0)

	for _, entry := range relays {
		urlStr := entry.String()
		if _, exists := configMap[urlStr]; exists {
			return nil, fmt.Errorf("%w: %s", errRelayConfiguredTwice, urlStr)
		}
		configs = append(configs, types.NewRelayConfig(entry))
	}

	for _, config := range configMap {
		configs = append(configs, config)
	}

	if len(configs) == 0 {
		return nil, errNoRelaysSpecified
	}

	return configs, nil
}

func parseConfig(cfg Config) (*ConfigResult, error) {
	timeoutGetHeaderMs := cfg.TimeoutGetHeaderMs
	if timeoutGetHeaderMs == 0 {
		timeoutGetHeaderMs = 950
	}

	lateInSlotTimeMs := cfg.LateInSlotTimeMs
	if lateInSlotTimeMs == 0 {
		lateInSlotTimeMs = 2000
	}

	configMap := make(map[string]types.RelayConfig)
	for _, relay := range cfg.Relays {
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

	// parse mux entries
	var muxMap config.MuxMap
	if len(cfg.Mux) > 0 {
		entries := make([]config.MuxEntryInput, 0, len(cfg.Mux))
		for _, muxYAML := range cfg.Mux {
			relayConfigs := make([]types.RelayConfig, 0, len(muxYAML.Relays))
			for _, relay := range muxYAML.Relays {
				relayEntry, err := types.NewRelayEntry(strings.TrimSpace(relay.URL))
				if err != nil {
					return nil, err
				}
				relayConfigs = append(relayConfigs, types.RelayConfig{
					RelayEntry:           relayEntry,
					EnableTimingGames:    relay.EnableTimingGames,
					TargetFirstRequestMs: relay.TargetFirstRequestMs,
					FrequencyGetHeaderMs: relay.FrequencyGetHeaderMs,
				})
			}

			// per mux timeouts if provieded otherwsie fall back to global defaults
			muxTimeout := muxYAML.TimeoutGetHeaderMs
			if muxTimeout == 0 {
				muxTimeout = timeoutGetHeaderMs
			}
			muxLateInSlot := muxYAML.LateInSlotTimeMs
			if muxLateInSlot == 0 {
				muxLateInSlot = lateInSlotTimeMs
			}

			entries = append(entries, config.MuxEntryInput{
				ID:                 muxYAML.ID,
				ValidatorPubkeys:   muxYAML.ValidatorPubkeys,
				RelayConfigs:       relayConfigs,
				TimeoutGetHeaderMs: muxTimeout,
				LateInSlotTimeMs:   muxLateInSlot,
			})
		}

		var err error
		muxMap, err = config.ValidateMuxEntries(entries)
		if err != nil {
			return nil, err
		}
	}

	return &ConfigResult{
		RelayConfigs:       configMap,
		TimeoutGetHeaderMs: timeoutGetHeaderMs,
		LateInSlotTimeMs:   lateInSlotTimeMs,
		MuxMap:             muxMap,
	}, nil
}
