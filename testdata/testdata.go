package testdata

import (
	_ "embed"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/flashbots/mev-boost/config/relay"
	"github.com/stretchr/testify/require"
)

//go:embed valid-proposer-config.json
var ValidProposerConfigBytes []byte

var (
	_, curPath, _, _ = runtime.Caller(0)
	curDir           = filepath.Join(filepath.Dir(curPath))

	ValidProposerConfigFilePath     = filepath.Join(curDir, "valid-proposer-config.json")
	CorruptedProposerConfigFilePath = filepath.Join(curDir, "corrupted-proposer-config.json")
)

func ValidProposerConfig(t *testing.T) *relay.Config {
	t.Helper()

	want := validProposerConfig()

	// we want to make sure that json version of config
	// is the same as the manually crafted config struct
	var got *relay.Config
	require.NoError(t, json.Unmarshal(ValidProposerConfigBytes, &got))
	require.Equal(t, want, got)

	return want
}

func validProposerConfig() *relay.Config {
	return &relay.Config{
		ProposerConfig: relay.ProposerConfig{
			"0xa1d1ad0714035353258038e964ae9675dc0252ee22cea896825c01458e1807bfad2f9969338798548d9858a571f7425c": relay.Relay{
				FeeRecipient: "0x50155530FCE8a85ec7055A5F8b2bE214B3DaeFd3",
				Builder: relay.Builder{
					Enabled: true,
					Relays: []string{
						"https://0x84f4b47474dfd6f2e0f4f05d9a056fbf8414e8c5a02e3363a29272262ee136c2bca0872b2f3cfa10e2017d52ce03a9d6@bloxroute.ethical.blxrbdn.com",
					},
					GasLimit: "12345654321",
				},
			},
			"0xb2ff4716ed345b05dd1dfc6a5a9fa70856d8c75dcc9e881dd2f766d5f891326f0d10e96f3a444ce6c912b69c22c6754d": relay.Relay{
				FeeRecipient: "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5",
				Builder: relay.Builder{
					Enabled: true,
					Relays: []string{
						"https://0x8b66d47c8bd32211c0ed807048e7f6467a8a046db020db2d7097f5afd6996545ecc07b827efaa8f1524cb14ca268afa2@boost-relay.flashbots.net",
						"https://0xb075a426a903150254d8c4b5672b9d6d6b1c925b8d0d2d74b7798229a9b8141c20716d3bff452adacf8c4a61f859bce6@bloxroute.max-profit.blxrbdn.com",
					},
				},
			},
			"0x8e323fd501233cd4d1b9d63d74076a38de50f2f584b001a5ac2412e4e46adb26d2fb2a6041e7e8c57cd4df0916729219": relay.Relay{
				FeeRecipient: "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5",
				Builder: relay.Builder{
					Enabled: false,
				},
			},
			"0x8e323fd501233cd4d1b9d63d74076a38de50f2f584b001a5ac2412e4e46adb26d2fb2a6041e7e8c57cd4df0916727225": relay.Relay{
				FeeRecipient: "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5",
				Builder: relay.Builder{
					Enabled: true,
				},
			},
		},
		DefaultConfig: relay.Relay{
			FeeRecipient: "0x6e35733c5af9B61374A128e6F85f553aF09ff89A",
			Builder: relay.Builder{
				Enabled: true,
				Relays: []string{
					"https://0x8b66d47c8bd32211c0ed807048e7f6467a8a046db020db2d7097f5afd6996545ecc07b827efaa8f1524cb14ca268afa2@boost-relay.flashbots.net",
					"https://0x84f4b47474dfd6f2e0f4f05d9a056fbf8414e8c5a02e3363a29272262ee136c2bca0872b2f3cfa10e2017d52ce03a9d6@bloxroute.ethical.blxrbdn.com",
					"https://0xad587acbbfe7e92fe17fc48c39ef8b86b9c7ceacbcdaad5b401141a81f841849c595105c53f03cc76a2774a5749eaa34@builder-relay-mainnet.blocknative.com",
					"https://0xadbc68cbb87a05171e4025a7536037e5e3f08e5a4dad85386fe29158be307bec565be59a6ab361daef45e499578d7c54@relay.edennetwork.io",
					"https://0x863d2d9fe2d4114a5963f086db4cdf8cf730cafc37ab52e4e6069a3ad62217c707cd10a65a386a1215753a63f54b36de@relayooor.wtf",
				},
			},
		},
	}
}
