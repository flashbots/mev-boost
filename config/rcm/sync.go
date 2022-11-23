package rcm

import (
	"context"
	"time"
)

// DefaultSyncTime is the default config sync time.
// It equals to half an epoch.
// Every epoch has 32 slots each of which lasts 12 seconds.
const DefaultSyncTime = 32 * 12 / 2 * time.Second

// OnSyncHandler an even handler which is invoked on every synchronisation call.
type OnSyncHandler = func(t time.Time, err error)

// NopSyncHandler the default sync handler which does nothing.
func NopSyncHandler(_ time.Time, _ error) {}

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
	configManager *Configurator
	interval      time.Duration
	onSyncHandler OnSyncHandler
}

// NewSyncer creates a new instance of Syncer.
//
// It takes configManager instance as a required param.
// It may take numerous optional params.
//
// It panics if no configManager is passed.
// If no interval option is passed, then the DefaultSyncTime will be used.
func NewSyncer(configManager *Configurator, opt ...SyncOption) *Syncer {
	if configManager == nil {
		panic("configManager is require and cannot be nil")
	}

	cfg := &SyncConfig{}
	for _, o := range opt {
		o(cfg)
	}

	if cfg.interval < 1 {
		cfg.interval = DefaultSyncTime
	}

	if cfg.onSyncHandler == nil {
		cfg.onSyncHandler = NopSyncHandler
	}

	return &Syncer{
		configManager: configManager,
		interval:      cfg.interval,
		onSyncHandler: cfg.onSyncHandler,
	}
}

// SyncConfig runs a background job which synchronises the configuration.
//
// The job will finish once the context is done.
// It the job runs a synchronisation task every given interval of time.
// A custom interval maybe specified via constructor option.
func (s *Syncer) SyncConfig(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				s.onSyncHandler(t, s.configManager.SyncConfig())
			}
		}
	}()

	<-ctx.Done()
}
