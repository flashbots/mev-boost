package server

import (
	"testing"

	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestRelayConfig(url string) types.RelayConfig {
	entry, _ := types.NewRelayEntry(url)
	return types.RelayConfig{RelayEntry: entry}
}

func TestGetConfigForValidator(t *testing.T) {
	relayConfig1 := newTestRelayConfig("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com")
	relayConfig2 := newTestRelayConfig("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay2.example.com")
	relayConfig3 := newTestRelayConfig("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay3.example.com")

	defaultRelayConfigs := []types.RelayConfig{relayConfig1, relayConfig2, relayConfig3}

	lidoMuxConfig := &config.RuntimeMuxConfig{
		ID:                 "lido",
		RelayConfigs:       []types.RelayConfig{relayConfig1},
		TimeoutGetHeaderMs: 900,
		LateInSlotTimeMs:   1500,
	}
	rocketMuxConfig := &config.RuntimeMuxConfig{
		ID:                 "rocket-pool",
		RelayConfigs:       []types.RelayConfig{relayConfig2},
		TimeoutGetHeaderMs: 950,
		LateInSlotTimeMs:   2000,
	}

	muxMap := config.MuxMap{
		"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249": lidoMuxConfig,
		"0x8b1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249": rocketMuxConfig,
	}

	service := &BoostService{
		relayConfigs:       defaultRelayConfigs,
		muxMap:             muxMap,
		timeoutGetHeaderMs: 950,
		lateInSlotTimeMs:   2000,
		log:                logrus.NewEntry(logrus.New()),
	}

	t.Run("Validator with mux config returns mux specific relays", func(t *testing.T) {
		lidoValidator := "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relayConfigs, timeoutMs, lateMs := service.GetConfigForValidator(lidoValidator)
		require.Len(t, relayConfigs, 1)
		require.Equal(t, relayConfig1.RelayEntry.URL.String(), relayConfigs[0].RelayEntry.URL.String())
		require.Equal(t, uint64(900), timeoutMs)
		require.Equal(t, uint64(1500), lateMs)

		rocketValidator := "0x8b1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relayConfigs, timeoutMs, lateMs = service.GetConfigForValidator(rocketValidator)
		require.Len(t, relayConfigs, 1)
		require.Equal(t, relayConfig2.RelayEntry.URL.String(), relayConfigs[0].RelayEntry.URL.String())
		require.Equal(t, uint64(950), timeoutMs)
		require.Equal(t, uint64(2000), lateMs)
	})

	t.Run("Validator without mux config returns defaults", func(t *testing.T) {
		unknownValidator := "0x8c1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relayConfigs, timeoutMs, lateMs := service.GetConfigForValidator(unknownValidator)
		require.Len(t, relayConfigs, 3)
		require.Equal(t, uint64(950), timeoutMs)
		require.Equal(t, uint64(2000), lateMs)
	})

	t.Run("No mux config returns defaults", func(t *testing.T) {
		serviceNoMux := &BoostService{
			relayConfigs:       defaultRelayConfigs,
			muxMap:             nil,
			timeoutGetHeaderMs: 950,
			lateInSlotTimeMs:   2000,
			log:                logrus.NewEntry(logrus.New()),
		}

		lidoValidator := "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relayConfigs, timeoutMs, lateMs := serviceNoMux.GetConfigForValidator(lidoValidator)
		require.Len(t, relayConfigs, 3)
		require.Equal(t, uint64(950), timeoutMs)
		require.Equal(t, uint64(2000), lateMs)
	})
}

func TestAllRelayConfigs(t *testing.T) {
	relayConfig1 := newTestRelayConfig("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com")
	relayConfig2 := newTestRelayConfig("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay2.example.com")
	relayConfig3 := newTestRelayConfig("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay3.example.com")

	t.Run("No mux returns default relays", func(t *testing.T) {
		service := &BoostService{
			relayConfigs: []types.RelayConfig{relayConfig1, relayConfig2},
			muxMap:       nil,
			log:          logrus.NewEntry(logrus.New()),
		}
		all := service.AllRelayConfigs()
		require.Len(t, all, 2)
	})

	t.Run("Deduplicates shared relays", func(t *testing.T) {
		muxMap := config.MuxMap{
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249": {
				ID:                 "lido",
				RelayConfigs:       []types.RelayConfig{relayConfig1, relayConfig3},
				TimeoutGetHeaderMs: 900,
				LateInSlotTimeMs:   1500,
			},
		}

		service := &BoostService{
			relayConfigs: []types.RelayConfig{relayConfig1, relayConfig2},
			muxMap:       muxMap,
			log:          logrus.NewEntry(logrus.New()),
		}
		all := service.AllRelayConfigs()
		require.Len(t, all, 3)
	})

	t.Run("Adds mux only relays", func(t *testing.T) {
		muxMap := config.MuxMap{
			"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249": {
				ID:                 "lido",
				RelayConfigs:       []types.RelayConfig{relayConfig3},
				TimeoutGetHeaderMs: 900,
				LateInSlotTimeMs:   1500,
			},
		}

		service := &BoostService{
			relayConfigs: []types.RelayConfig{relayConfig1, relayConfig2},
			muxMap:       muxMap,
			log:          logrus.NewEntry(logrus.New()),
		}
		all := service.AllRelayConfigs()
		require.Len(t, all, 3)
	})
}
