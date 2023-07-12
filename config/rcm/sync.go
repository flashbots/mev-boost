package rcm

import (
	"context"
	"errors"
	"time"

	"github.com/flashbots/mev-boost/config/relay"
)

// DefaultSyncInterval is the default config synchronisation interval.
// It equals to half an epoch.
// Every epoch has 32 slots each of which lasts 12 seconds.
const DefaultSyncInterval = 32 * 12 / 2 * time.Second

// OnSyncHandler an even handler which is invoked on every synchronisation call.
type OnSyncHandler = func(t time.Time, err error, relays relay.List)

// NopSyncHandler the default sync handler which does nothing.
func NopSyncHandler(_ time.Time, _ error, _ relay.List) {}

// SyncConfig holds synchronisation options.
type SyncConfig struct {
	interval      time.Duration
	onSyncHandler OnSyncHandler
}

// SyncOption is a synchronisation option.
type SyncOption = func(cfg *SyncConfig)

// SyncerWithOnSyncHandler specifies an OnSyncHandler.
func SyncerWithOnSyncHandler(h OnSyncHandler) SyncOption {
	return func(cfg *SyncConfig) {
		cfg.onSyncHandler = h
	}
}

// SyncerWithInterval specifies synchronisation interval.
func SyncerWithInterval(d time.Duration) SyncOption {
	return func(cfg *SyncConfig) {
		cfg.interval = d
	}
}

// Syncer synchronises relay configuration with the given RCP.
type Syncer struct {
	configurator  *Configurator
	interval      time.Duration
	onSyncHandler OnSyncHandler
}

// NewSyncer creates a new instance of Syncer.
//
// It takes a Configurator instance as a required param.
// It may take numerous optional params.
//
// It panics if Configurator is not passed.
// If no interval option is passed, then the DefaultSyncInterval will be used.
func NewSyncer(configurator *Configurator, opt ...SyncOption) *Syncer {
	if configurator == nil {
		panic("configurator is required and cannot be nil")
	}

	cfg := &SyncConfig{}
	for _, o := range opt {
		o(cfg)
	}

	if cfg.interval < 1 {
		cfg.interval = DefaultSyncInterval
	}

	if cfg.onSyncHandler == nil {
		cfg.onSyncHandler = NopSyncHandler
	}

	return &Syncer{
		configurator:  configurator,
		interval:      cfg.interval,
		onSyncHandler: cfg.onSyncHandler,
	}
}

// SyncConfig runs a background job which synchronises the configuration.
//
// It runs a synchronisation job every given interval of time.
// The job will finish once the context is done.
// A custom interval maybe specified via a constructor option.
// If OnSyncHandler is passed, it will run every time the config is synced.
//
// This function will block, until the context is done.
// A good usage example may look as follows:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	go rcm.SyncConfig(ctx)
func (s *Syncer) SyncConfig(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	go func() {
		for {
			if s.isDone(ctx) {
				return
			}

			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				s.onSyncHandler(t, s.configurator.SyncConfig(), s.configurator.AllRelays())
			}
		}
	}()

	// block until context is done
	<-ctx.Done()
}

func (s *Syncer) isDone(ctx context.Context) bool {
	err := ctx.Err()

	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
