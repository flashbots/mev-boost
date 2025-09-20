package server

import (
	"testing"

	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGetRelaysForValidator(t *testing.T) {
	relay1, err := types.NewRelayEntry("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com")
	require.NoError(t, err)
	relay2, err := types.NewRelayEntry("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay2.example.com")
	require.NoError(t, err)
	relay3, err := types.NewRelayEntry("0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay3.example.com")
	require.NoError(t, err)

	defaultRelays := []types.RelayEntry{relay1, relay2, relay3}

	muxConfig := &config.MuxConfig{
		Policies: []config.Policy{
			{
				Name: "lido-policy",
				Relayers: []config.Relayer{
					{
						Name: "relay1",
						URL:  "0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com",
					},
				},
			},
			{
				Name: "rocket-policy",
				Relayers: []config.Relayer{
					{
						Name: "relay2",
						URL:  "0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay2.example.com",
					},
				},
			},
		},
		Mappings: []config.Mapping{
			{
				Name:   "lido-keys",
				Policy: "lido-policy",
				Filters: config.Filters{
					PublicKeys: []string{
						"0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
					},
				},
			},
			{
				Name:   "rocket-keys",
				Policy: "rocket-policy",
				Filters: config.Filters{
					PublicKeys: []string{
						"0x8b1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
					},
				},
			},
		},
	}
	service := &BoostService{
		relays:    defaultRelays,
		muxConfig: muxConfig,
		log:       logrus.NewEntry(logrus.New()),
	}

	t.Run("Validator with policy", func(t *testing.T) {
		lidoValidator := "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relays := service.getRelaysForValidator(lidoValidator)
		require.Len(t, relays, 1)
		require.Equal(t, relay1.URL.String(), relays[0].URL.String())

		rocketValidator := "0x8b1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relays = service.getRelaysForValidator(rocketValidator)
		require.Len(t, relays, 1)
		require.Equal(t, relay2.URL.String(), relays[0].URL.String())
	})

	t.Run("Validator without policy", func(t *testing.T) {
		unknownValidator := "0x8c1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relays := service.getRelaysForValidator(unknownValidator)
		// should return default relays
		require.Len(t, relays, 3)
		require.Equal(t, defaultRelays, relays)
	})

	t.Run("No mux config", func(t *testing.T) {
		serviceNoMux := &BoostService{
			relays:    defaultRelays,
			muxConfig: nil,
			log:       logrus.NewEntry(logrus.New()),
		}

		lidoValidator := "0x8a1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relays := serviceNoMux.getRelaysForValidator(lidoValidator)
		// should return default relays
		require.Len(t, relays, 3)
		require.Equal(t, defaultRelays, relays)
	})

	t.Run("Invalid policy", func(t *testing.T) {
		invalidMuxConfig := &config.MuxConfig{
			Policies: []config.Policy{
				{
					Name: "lido-policy",
					Relayers: []config.Relayer{
						{
							Name: "relay1",
							URL:  "0x9000009807ed12c1f08bf4e81c6da3ba8e3fc3d953898ce0102433094e5f22f21102ec057841fcb81978ed1ea0fa8246@relay1.example.com",
						},
					},
				},
			},
			Mappings: []config.Mapping{
				{
					Name:   "rocket-keys",
					Policy: "rocket-policy",
					Filters: config.Filters{
						PublicKeys: []string{
							"0x8b1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249",
						},
					},
				},
			},
		}

		serviceInvalid := &BoostService{
			relays:    defaultRelays,
			muxConfig: invalidMuxConfig,
			log:       logrus.NewEntry(logrus.New()),
		}

		invalidValidator := "0x8d1d7b8dd64e0aafe7ea7b6c95065c9364cf99d38470c12ee807d55f7de1529ad29ce2c422e0b65e3d5a05c02caca249"
		relays := serviceInvalid.getRelaysForValidator(invalidValidator)
		// should return default relays
		require.Len(t, relays, 3)
		require.Equal(t, defaultRelays, relays)
	})
}
