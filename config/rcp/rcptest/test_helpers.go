package rcptest

import (
	"io"
	"testing"

	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/testutil"
)

type MockConfig struct {
	relayCfg *relay.Config
	err      error

	calls []rcm.ConfigProvider
}

type MockOption func(cfg *MockConfig)

func MockRelayConfigProvider(opt ...MockOption) func() (*relay.Config, error) {
	cfg := &MockConfig{relayCfg: &relay.Config{}}
	for _, o := range opt {
		o(cfg)
	}

	curCall := 0

	return func() (*relay.Config, error) {
		if len(cfg.calls) > curCall {
			nextCall := cfg.calls[curCall]
			curCall++

			return nextCall()
		}

		return stubRelayConfigProvider(cfg.relayCfg, cfg.err)
	}
}

func stubRelayConfigProvider(cfg *relay.Config, err error) (*relay.Config, error) {
	return cfg, err
}

func WithErr() MockOption {
	return func(cfg *MockConfig) {
		cfg.err = io.ErrUnexpectedEOF
	}
}

func WithProposerRelays(pubKey relay.ValidatorPublicKey, relays relay.Set) MockOption {
	return func(cfg *MockConfig) {
		proposerConfig := cfg.relayCfg.ProposerConfig
		if proposerConfig == nil {
			proposerConfig = make(relay.ProposerConfig, len(relays))
		}

		_, ok := proposerConfig[pubKey]
		if !ok {
			proposerConfig[pubKey] = relay.Relay{
				Builder: relay.Builder{
					Enabled: true,
					Relays:  relays.ToStringSlice(),
				},
			}

			cfg.relayCfg.ProposerConfig = proposerConfig

			return
		}

		panic("not implemented")

		//reg.Builder = relay.Builder{
		//	Enabled: true,
		//	Relays:  append(reg.Builder.Relays, relays...),
		//}
		//
		//cfg.relayCfg.ProposerConfig[pubKey] = reg
	}
}

func WithInvalidProposerRelays(t *testing.T) MockOption {
	t.Helper()

	return func(cfg *MockConfig) {
		pubKey := testutil.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: true,
					Relays:  []string{"invalid-relay-url"},
				},
			},
		}
	}
}

func WithDefaultRelays(relays relay.Set) MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
			Relays:  relays.ToStringSlice(),
		}
	}
}

func WithInvalidDefaultRelays() MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
			Relays:  []string{"htt://fef/"},
		}
	}
}

func WithProposerEnabledBuilderAndNoRelays(t *testing.T) MockOption {
	t.Helper()

	return func(cfg *MockConfig) {
		pubKey := testutil.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: true,
				},
			},
		}
	}
}

func WithSomeDisabledProposerBuilders(t *testing.T) MockOption {
	t.Helper()

	return func(cfg *MockConfig) {
		pubKey := testutil.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: relay.Builder{
					Enabled: false,
					Relays:  testutil.RandomRelaySet(t, 3).ToStringSlice(),
				},
			},
		}
	}
}

func WithDefaultEnabledBuilderAndNoRelays() MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = relay.Builder{
			Enabled: true,
		}
	}
}

func WithCalls(calls []rcm.ConfigProvider) MockOption {
	return func(cfg *MockConfig) {
		cfg.calls = calls
	}
}
