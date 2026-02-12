package cli

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewConfigWatcher(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configYAML := `
timeout_get_header_ms: 1000
late_in_slot_time_ms: 1500
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay.example.com
    enable_timing_games: true
    target_first_request_ms: 200
    frequency_get_header_ms: 100
`
		err := os.WriteFile(configPath, []byte(configYAML), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{}, log)
		require.NoError(t, err)
		require.NotNil(t, watcher)
		require.Equal(t, configPath, watcher.configPath)
	})

	t.Run("invalid config file path", func(t *testing.T) {
		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher("/wrong/path/config.yaml", []types.RelayEntry{}, log)
		require.Error(t, err)
		require.Nil(t, watcher)
	})

	t.Run("invalid yaml content", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		invalidYAML := `invalid: yaml: content: [`
		err := os.WriteFile(configPath, []byte(invalidYAML), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{}, log)
		require.Error(t, err)
		require.Nil(t, watcher)
	})
}

func TestConfigWatcherWatchLive(t *testing.T) {
	t.Run("config change triggers callback", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `
timeout_get_header_ms: 900
late_in_slot_time_ms: 1000
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com
    enable_timing_games: false
`
		err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{}, log)
		require.NoError(t, err)

		var (
			receivedConfig *ConfigResult
			callbackMutex  sync.Mutex
			once           sync.Once
			done           = make(chan struct{})
		)

		watcher.Watch(func(newConfig *ConfigResult) {
			callbackMutex.Lock()
			receivedConfig = newConfig
			callbackMutex.Unlock()
			once.Do(func() {
				close(done)
			})
		})

		time.Sleep(100 * time.Millisecond)

		updatedConfig := `
timeout_get_header_ms: 1200
late_in_slot_time_ms: 1500
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay2.example.com
    enable_timing_games: true
    target_first_request_ms: 300
    frequency_get_header_ms: 150
`
		err = os.WriteFile(configPath, []byte(updatedConfig), 0o644)
		require.NoError(t, err)

		select {
		case <-done:
			callbackMutex.Lock()
			require.NotNil(t, receivedConfig)
			require.Equal(t, uint64(1200), receivedConfig.TimeoutGetHeaderMs)
			require.Equal(t, uint64(1500), receivedConfig.LateInSlotTimeMs)
			require.Len(t, receivedConfig.RelayConfigs, 1)
			callbackMutex.Unlock()
		case <-time.After(2 * time.Second):
			t.Fatal("callback was not called within timeout")
		}
	})

	t.Run("invalid config change keeps old config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `
timeout_get_header_ms: 900
late_in_slot_time_ms: 1000
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com
    enable_timing_games: false
`
		err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{}, log)
		require.NoError(t, err)

		callbackCallCount := 0
		var callbackMutex sync.Mutex

		watcher.Watch(func(_ *ConfigResult) {
			callbackMutex.Lock()
			callbackCallCount++
			callbackMutex.Unlock()
		})

		time.Sleep(200 * time.Millisecond)

		// writin invalid yaml which will cause the config watcher to fail
		invalidConfig := `invalid: yaml: [`
		err = os.WriteFile(configPath, []byte(invalidConfig), 0o644)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		callbackMutex.Lock()
		require.Equal(t, 0, callbackCallCount)
		callbackMutex.Unlock()
	})

	t.Run("config with invalid relay keeps old config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `
timeout_get_header_ms: 900
late_in_slot_time_ms: 1000
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com
    enable_timing_games: false
`
		err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{}, log)
		require.NoError(t, err)

		callbackCallCount := 0
		var callbackMutex sync.Mutex

		watcher.Watch(func(_ *ConfigResult) {
			callbackMutex.Lock()
			callbackCallCount++
			callbackMutex.Unlock()
		})

		time.Sleep(100 * time.Millisecond)

		invalidConfig := `
timeout_get_header_ms: 1200
late_in_slot_time_ms: 1500
relays:
  - url: https://invalid-relay-url.com
    enable_timing_games: false
`
		err = os.WriteFile(configPath, []byte(invalidConfig), 0o644)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		callbackMutex.Lock()
		require.Equal(t, 0, callbackCallCount)
		callbackMutex.Unlock()
	})

	t.Run("multiple config updates", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `
timeout_get_header_ms: 900
late_in_slot_time_ms: 1000
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com
    enable_timing_games: false
`
		err := os.WriteFile(configPath, []byte(initialConfig), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{}, log)
		require.NoError(t, err)

		callbackCallCount := 0
		var callbackMutex sync.Mutex
		var receivedConfigs []*ConfigResult

		watcher.Watch(func(newConfig *ConfigResult) {
			callbackMutex.Lock()
			callbackCallCount++
			receivedConfigs = append(receivedConfigs, newConfig)
			callbackMutex.Unlock()
		})

		time.Sleep(100 * time.Millisecond)

		// first update
		config1 := `
timeout_get_header_ms: 1000
late_in_slot_time_ms: 1100
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com
    enable_timing_games: true
    target_first_request_ms: 200
`
		err = os.WriteFile(configPath, []byte(config1), 0o644)
		require.NoError(t, err)
		time.Sleep(400 * time.Millisecond)

		// second update
		config2 := `
timeout_get_header_ms: 1200
late_in_slot_time_ms: 1500
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay2.example.com
    enable_timing_games: false
`
		err = os.WriteFile(configPath, []byte(config2), 0o644)
		require.NoError(t, err)
		time.Sleep(400 * time.Millisecond)

		callbackMutex.Lock()
		callbackCount := callbackCallCount
		configsCount := len(receivedConfigs)

		// at least 2 callbacks and 2 configs should be received
		// due to updates in the config file
		require.GreaterOrEqual(t, callbackCount, 2)
		require.GreaterOrEqual(t, configsCount, 2)

		liveConfig := receivedConfigs[configsCount-1]
		require.Equal(t, uint64(1200), liveConfig.TimeoutGetHeaderMs)
		require.Equal(t, uint64(1500), liveConfig.LateInSlotTimeMs)

		callbackMutex.Unlock()
	})

	t.Run("empty relays in config file allows keep using cli relays", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		cliRelay, err := types.NewRelayEntry("https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@cli-relay.example.com")
		require.NoError(t, err)

		initialConfig := `
timeout_get_header_ms: 900
late_in_slot_time_ms: 1000
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com
    enable_timing_games: false
`
		err = os.WriteFile(configPath, []byte(initialConfig), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{cliRelay}, log)
		require.NoError(t, err)

		var (
			receivedConfig *ConfigResult
			mergedConfigs  []types.RelayConfig
			callbackMutex  sync.Mutex
			once           sync.Once
			done           = make(chan struct{})
		)

		watcher.Watch(func(newConfig *ConfigResult) {
			callbackMutex.Lock()
			receivedConfig = newConfig
			mergedConfigs, _ = MergeRelayConfigs([]types.RelayEntry{cliRelay}, newConfig.RelayConfigs)
			callbackMutex.Unlock()
			once.Do(func() {
				close(done)
			})
		})

		time.Sleep(100 * time.Millisecond)

		// sp now we are updating the config and removing all the relays
		emptyConfig := `
timeout_get_header_ms: 1200
late_in_slot_time_ms: 1500
relays: []
`
		err = os.WriteFile(configPath, []byte(emptyConfig), 0o644)
		require.NoError(t, err)

		select {
		case <-done:
			callbackMutex.Lock()
			require.NotNil(t, receivedConfig)
			require.Equal(t, uint64(1200), receivedConfig.TimeoutGetHeaderMs)
			require.Equal(t, uint64(1500), receivedConfig.LateInSlotTimeMs)
			require.Empty(t, receivedConfig.RelayConfigs)
			require.Len(t, mergedConfigs, 1)
			callbackMutex.Unlock()
		case <-time.After(2 * time.Second):
			t.Fatal("callback was not called within timeout")
		}
	})

	t.Run("merges CLI relays with config file relays", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		cliRelay, err := types.NewRelayEntry("https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@cli-relay.example.com")
		require.NoError(t, err)

		configYAML := `
timeout_get_header_ms: 1000
late_in_slot_time_ms: 1500
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@config-relay.example.com
    enable_timing_games: true
    target_first_request_ms: 200
    frequency_get_header_ms: 100
`
		err = os.WriteFile(configPath, []byte(configYAML), 0o644)
		require.NoError(t, err)

		log := logrus.NewEntry(logrus.New())
		watcher, err := NewConfigWatcher(configPath, []types.RelayEntry{cliRelay}, log)
		require.NoError(t, err)

		var (
			receivedConfig *ConfigResult
			mergedConfigs  []types.RelayConfig
			callbackMutex  sync.Mutex
			once           sync.Once
			done           = make(chan struct{})
		)

		watcher.Watch(func(newConfig *ConfigResult) {
			callbackMutex.Lock()
			receivedConfig = newConfig
			mergedConfigs, _ = MergeRelayConfigs([]types.RelayEntry{cliRelay}, newConfig.RelayConfigs)
			callbackMutex.Unlock()
			once.Do(func() {
				close(done)
			})
		})

		time.Sleep(100 * time.Millisecond)

		// update config to add another relay
		updatedConfig := `
timeout_get_header_ms: 1200
late_in_slot_time_ms: 1600
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@config-relay.example.com
    enable_timing_games: true
    target_first_request_ms: 300
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@new-relay.example.com
    enable_timing_games: false
`
		err = os.WriteFile(configPath, []byte(updatedConfig), 0o644)
		require.NoError(t, err)

		select {
		case <-done:
			callbackMutex.Lock()
			require.NotNil(t, receivedConfig)
			require.Equal(t, uint64(1200), receivedConfig.TimeoutGetHeaderMs)
			require.Equal(t, uint64(1600), receivedConfig.LateInSlotTimeMs)
			require.Len(t, receivedConfig.RelayConfigs, 2)
			require.Len(t, mergedConfigs, 3)
			callbackMutex.Unlock()
		case <-time.After(2 * time.Second):
			t.Fatal("callback was not called within timeout")
		}
	})
}

func TestMergeRelayConfigs_DuplicateDetection(t *testing.T) {
	relay, err := types.NewRelayEntry("https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay.example.com")
	require.NoError(t, err)

	t.Run("Error when no relays in cli or config", func(t *testing.T) {
		_, err := MergeRelayConfigs([]types.RelayEntry{}, map[string]types.RelayConfig{})
		require.Error(t, err)
		require.ErrorIs(t, err, errNoRelaysSpecified)
	})

	t.Run("Success with config only and empty cli relays", func(t *testing.T) {
		configMap := map[string]types.RelayConfig{
			relay.String(): {
				RelayEntry:        relay,
				EnableTimingGames: true,
			},
		}
		configs, err := MergeRelayConfigs([]types.RelayEntry{}, configMap)
		require.NoError(t, err)
		require.Len(t, configs, 1)
	})

	t.Run("Success with cli relays only and empty config map", func(t *testing.T) {
		configs, err := MergeRelayConfigs([]types.RelayEntry{relay}, map[string]types.RelayConfig{})
		require.NoError(t, err)
		require.Len(t, configs, 1)
	})

	t.Run("Error when same relay in both CLI and config", func(t *testing.T) {
		configMap := map[string]types.RelayConfig{
			relay.String(): {
				RelayEntry:        relay,
				EnableTimingGames: true,
			},
		}
		_, err := MergeRelayConfigs([]types.RelayEntry{relay}, configMap)
		require.Error(t, err)
		require.Contains(t, err.Error(), "relay is specified in both cli flags and config file")
	})

	t.Run("Success when relays are different", func(t *testing.T) {
		configRelay, err := types.NewRelayEntry("https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@other-relay.example.com")
		require.NoError(t, err)

		configMap := map[string]types.RelayConfig{
			configRelay.String(): {
				RelayEntry:        configRelay,
				EnableTimingGames: true,
			},
		}

		configs, err := MergeRelayConfigs([]types.RelayEntry{relay}, configMap)
		require.NoError(t, err)
		require.Len(t, configs, 2)
	})

	t.Run("Success with CLI only", func(t *testing.T) {
		configs, err := MergeRelayConfigs([]types.RelayEntry{relay}, map[string]types.RelayConfig{})
		require.NoError(t, err)
		require.Len(t, configs, 1)
	})

	t.Run("Success with config only", func(t *testing.T) {
		configMap := map[string]types.RelayConfig{
			relay.String(): {
				RelayEntry:        relay,
				EnableTimingGames: true,
			},
		}
		configs, err := MergeRelayConfigs([]types.RelayEntry{}, configMap)
		require.NoError(t, err)
		require.Len(t, configs, 1)
	})
}

func TestParseConfigWithMux(t *testing.T) {
	t.Run("Valid mux config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configYAML := `
timeout_get_header_ms: 950
late_in_slot_time_ms: 2000
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay.example.com
mux:
  - id: "lido"
    validator_pubkeys:
      - "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
    timeout_get_header_ms: 900
    late_in_slot_time_ms: 1500
    relays:
      - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@lido-relay.example.com
        enable_timing_games: true
        target_first_request_ms: 200
        frequency_get_header_ms: 100
  - id: "rocket-pool"
    validator_pubkeys:
      - "0x8d1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca252"
    relays:
      - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@rocketpool-relay.example.com
`
		err := os.WriteFile(configPath, []byte(configYAML), 0o644)
		require.NoError(t, err)

		result, err := LoadConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, result.MuxMap)
		require.Len(t, result.MuxMap, 2) // 2 val pubkeys

		// check lido mux config
		lidoMux := result.MuxMap["0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"]
		require.NotNil(t, lidoMux)
		require.Equal(t, "lido", lidoMux.ID)
		require.Len(t, lidoMux.RelayConfigs, 1)
		require.Equal(t, uint64(900), lidoMux.TimeoutGetHeaderMs)
		require.Equal(t, uint64(1500), lidoMux.LateInSlotTimeMs)
		require.True(t, lidoMux.RelayConfigs[0].EnableTimingGames)

		// check rocket pool mux config
		rocketMux := result.MuxMap["0x8d1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca252"]
		require.NotNil(t, rocketMux)
		require.Equal(t, "rocket-pool", rocketMux.ID)
		require.Len(t, rocketMux.RelayConfigs, 1)
		require.Equal(t, uint64(950), rocketMux.TimeoutGetHeaderMs) // use global vals since not set per mux
		require.Equal(t, uint64(2000), rocketMux.LateInSlotTimeMs)  // same
		require.False(t, rocketMux.RelayConfigs[0].EnableTimingGames)
	})

	t.Run("No mux config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configYAML := `
timeout_get_header_ms: 950
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay.example.com
`
		err := os.WriteFile(configPath, []byte(configYAML), 0o644)
		require.NoError(t, err)

		result, err := LoadConfigFile(configPath)
		require.NoError(t, err)
		require.Nil(t, result.MuxMap)
	})

	t.Run("Duplicate validator pubkey across mux entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configYAML := `
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay.example.com
mux:
  - id: "lido"
    validator_pubkeys:
      - "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
    relays:
      - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com
  - id: "rocket-pool"
    validator_pubkeys:
      - "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
    relays:
      - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay2.example.com
`
		err := os.WriteFile(configPath, []byte(configYAML), 0o644)
		require.NoError(t, err)

		_, err = LoadConfigFile(configPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "validator pubkey appears in multiple mux entries")
	})

	t.Run("Mux entry with no relays", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configYAML := `
relays:
  - url: https://0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay.example.com
mux:
  - id: "empty"
    validator_pubkeys:
      - "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
    relays: []
`
		err := os.WriteFile(configPath, []byte(configYAML), 0o644)
		require.NoError(t, err)

		_, err = LoadConfigFile(configPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must have at least one relay")
	})
}
