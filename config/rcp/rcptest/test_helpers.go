package rcptest

import (
	"io"
	"testing"

	"github.com/flashbots/mev-boost/config/rcm"
	"github.com/flashbots/mev-boost/config/relay"
	"github.com/flashbots/mev-boost/config/relay/reltest"
)

type MockConfig struct {
	relayCfg *relay.Config
	err      error

	calls []rcm.ConfigProvider
}

type MockOption func(cfg *MockConfig)

func MockRelayConfigProvider(opt ...MockOption) func() (*relay.Config, error) {
	cfg := &MockConfig{relayCfg: &relay.Config{}}
	if cfg.relayCfg.ProposerConfig == nil {
		cfg.relayCfg.ProposerConfig = make(relay.ProposerConfig)
	}

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

		return cfg.relayCfg, cfg.err
	}
}

func WithProposerRelays(pubKey relay.ValidatorPublicKey, relays relay.Set) MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.ProposerConfig[pubKey] = relay.Relay{
			Builder: &relay.Builder{
				Enabled: true,
				Relays:  relays.ToStringSlice(),
			},
		}
	}
}

func WithDisabledEmptyProposer(pubKey relay.ValidatorPublicKey) MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.ProposerConfig[pubKey] = relay.Relay{
			Builder: &relay.Builder{
				Enabled: false,
			},
		}
	}
}

func WithProposerWithNoBuilder(pubKey relay.ValidatorPublicKey) MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.ProposerConfig[pubKey] = relay.Relay{}
	}
}

func WithInvalidProposerRelays(t *testing.T) MockOption {
	t.Helper()

	return func(cfg *MockConfig) {
		pubKey := reltest.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: &relay.Builder{
					Enabled: true,
					Relays:  []string{"invalid-relay-url"},
				},
			},
		}
	}
}

func WithDefaultRelays(relays relay.Set) MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = &relay.Builder{
			Enabled: true,
			Relays:  relays.ToStringSlice(),
		}
	}
}

func WithDisabledEmptyDefaultRelays() MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = &relay.Builder{
			Enabled: false,
		}
	}
}

func WithDisabledDefaultRelays(relays relay.Set) MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = &relay.Builder{
			Enabled: false,
			Relays:  relays.ToStringSlice(),
		}
	}
}

func WithInvalidDefaultRelays() MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = &relay.Builder{
			Enabled: true,
			Relays:  []string{"htt://fef/"},
		}
	}
}

func WithProposerEnabledBuilderAndNoRelays(t *testing.T) MockOption {
	t.Helper()

	return func(cfg *MockConfig) {
		pubKey := reltest.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: &relay.Builder{
					Enabled: true,
				},
			},
		}
	}
}

func WithSomeDisabledProposerBuilders(t *testing.T) MockOption {
	t.Helper()

	return func(cfg *MockConfig) {
		pubKey := reltest.RandomBLSPublicKey(t).String()
		cfg.relayCfg.ProposerConfig = map[relay.ValidatorPublicKey]relay.Relay{
			pubKey: {
				Builder: &relay.Builder{
					Enabled: false,
					Relays:  reltest.RandomRelaySet(t, 3).ToStringSlice(),
				},
			},
		}
	}
}

func WithDefaultEnabledBuilderAndNoRelays() MockOption {
	return func(cfg *MockConfig) {
		cfg.relayCfg.DefaultConfig.Builder = &relay.Builder{
			Enabled: true,
		}
	}
}

func WithErr() MockOption {
	return func(cfg *MockConfig) {
		cfg.err = io.ErrUnexpectedEOF
	}
}

func WithCalls(calls []rcm.ConfigProvider) MockOption {
	return func(cfg *MockConfig) {
		cfg.calls = calls
	}
}
