package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/flashbots/mev-boost/common"
	"github.com/flashbots/mev-boost/config"
	"github.com/flashbots/mev-boost/server"
	serverTypes "github.com/flashbots/mev-boost/server/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

const (
	genesisForkVersionMainnet = "0x00000000"
	genesisForkVersionSepolia = "0x90000069"
	genesisForkVersionHolesky = "0x01017000"
	genesisForkVersionHoodi   = "0x10000910"

	genesisTimeMainnet = 1606824023
	genesisTimeSepolia = 1655733600
	genesisTimeHolesky = 1695902400
	genesisTimeHoodi   = 1742213400
)

type RelaySetupResult struct {
	RelayConfigs       []serverTypes.RelayConfig
	MinBid             types.U256Str
	RelayCheck         bool
	TimeoutGetHeaderMs uint64
	LateInSlotTimeMs   uint64
	CLIRelays          []serverTypes.RelayEntry // CLI-provided relays for hot-reload merging
}

var (
	// errors
	errInvalidLoglevel = errors.New("invalid loglevel")
	errNegativeBid     = errors.New("please specify a non-negative minimum bid")
	errLargeMinBid     = errors.New("minimum bid is too large, please ensure min-bid is denominated in Ethers")

	log = logrus.NewEntry(logrus.New())
)

func Main() {
	cmd := &cli.Command{
		Name:   "mev-boost",
		Usage:  "mev-boost implementation, see help for more info",
		Action: start,
		Flags:  flags,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// start starts the mev-boost cli
func start(_ context.Context, cmd *cli.Command) error {
	// Only print the version if the flag is set
	if cmd.IsSet(versionFlag.Name) {
		fmt.Fprintf(cmd.Writer, "mev-boost %s\n", config.Version)
		return nil
	}

	if err := setupLogging(cmd); err != nil {
		flag.Usage()
		log.WithError(err).Fatal("failed setting up logging")
	}

	var (
		genesisForkVersion, genesisTime = setupGenesis(cmd)
		muxConfig                       = setupMuxConfig(cmd)
		listenAddr                      = cmd.String(addrFlag.Name)
		metricsEnabled                  = cmd.Bool(metricsFlag.Name)
		metricsAddr                     = cmd.String(metricsAddrFlag.Name)
	)

	relaySetup, err := setupRelays(cmd)
	if err != nil {
		return err
	}

	opts := server.BoostServiceOpts{
		Log:                      log,
		ListenAddr:               listenAddr,
		RelayConfigs:             relaySetup.RelayConfigs,
		MuxConfig:                muxConfig,
		GenesisForkVersionHex:    genesisForkVersion,
		GenesisTime:              genesisTime,
		RelayCheck:               relaySetup.RelayCheck,
		RelayMinBid:              relaySetup.MinBid,
		RequestTimeoutGetHeader:  time.Duration(cmd.Int(timeoutGetHeaderFlag.Name)) * time.Millisecond,
		RequestTimeoutGetPayload: time.Duration(cmd.Int(timeoutGetPayloadFlag.Name)) * time.Millisecond,
		RequestTimeoutRegVal:     time.Duration(cmd.Int(timeoutRegValFlag.Name)) * time.Millisecond,
		RequestMaxRetries:        cmd.Int(maxRetriesFlag.Name),
		MetricsAddr:              metricsAddr,
		TimeoutGetHeaderMs:       relaySetup.TimeoutGetHeaderMs,
		LateInSlotTimeMs:         relaySetup.LateInSlotTimeMs,
	}
	service, err := server.NewBoostService(opts)
	if err != nil {
		log.WithError(err).Fatal("failed creating the server")
	}

	if relaySetup.RelayCheck && service.CheckRelays() == 0 {
		log.Error("no relay passed the health-check!")
	}

	// enable hot reloading only if both --config and --watch-config flags are set
	if cmd.IsSet(relayConfigFlag.Name) && cmd.Bool(watchConfigFlag.Name) {
		configPath := cmd.String(relayConfigFlag.Name)
		watcher, err := NewConfigWatcher(configPath, relaySetup.CLIRelays, log)
		if err != nil {
			log.WithError(err).Warn("failed to set up config watcher")
			return err
		}
		// register a callback which gets invoked when config file changes
		watcher.Watch(func(newConfig *ConfigResult) {
			mergedConfigs, err := MergeRelayConfigs(relaySetup.CLIRelays, newConfig.RelayConfigs)
			if err != nil {
				log.WithError(err).Error("failed to merge relay configs, keeping old config")
				return
			}
			if len(mergedConfigs) == 0 {
				log.Error("merged config has no relays (neither from CLI nor config file), keeping old config")
				return
			}
			service.UpdateConfig(mergedConfigs, newConfig.TimeoutGetHeaderMs, newConfig.LateInSlotTimeMs)
		})
	}

	if metricsEnabled {
		go func() {
			log.Infof("metrics server listening on %v", opts.MetricsAddr)
			if err := service.StartMetricsServer(); err != nil {
				log.WithError(err).Error("metrics server exited with error")
			}
		}()
	}

	log.Infof("listening on %v", listenAddr)
	return service.StartHTTPServer()
}

func setupRelays(cmd *cli.Command) (*RelaySetupResult, error) {
	// For backwards compatibility with the -relays flag.
	var relays relayList
	if cmd.IsSet(relaysFlag.Name) {
		relayURLs := cmd.StringSlice(relaysFlag.Name)
		for _, urls := range relayURLs {
			for _, url := range strings.Split(urls, ",") {
				if err := relays.Set(strings.TrimSpace(url)); err != nil {
					log.WithError(err).WithField("relay", url).Fatal("invalid relay URL")
				}
			}
		}
	}

	// load configuration via config file
	var configMap map[string]serverTypes.RelayConfig
	var timeoutGetHeaderMs uint64 = 950
	var lateInSlotTimeMs uint64 = 2000
	if cmd.IsSet(relayConfigFlag.Name) {
		configPath := cmd.String(relayConfigFlag.Name)
		log.Infof("loading config from: %s", configPath)
		configResult, err := LoadConfigFile(configPath)
		if err != nil {
			log.WithError(err).Fatal("failed to load config file")
			return nil, err
		}
		configMap = configResult.RelayConfigs
		timeoutGetHeaderMs = configResult.TimeoutGetHeaderMs
		lateInSlotTimeMs = configResult.LateInSlotTimeMs
	}
	relayConfigs, err := MergeRelayConfigs(relays, configMap)
	if err != nil {
		log.WithError(err).Fatal("failed to merge relay configs")
		return nil, err
	}

	log.Infof("using %d relays", len(relayConfigs))
	for index, config := range relayConfigs {
		if config.EnableTimingGames {
			log.Infof("relay #%d: %s timing games: enabled", index+1, config.RelayEntry.String())
		} else {
			log.Infof("relay #%d: %s", index+1, config.RelayEntry.String())
		}
	}

	relayMinBidWei, err := sanitizeMinBid(cmd.Float(minBidFlag.Name))
	if err != nil {
		log.WithError(err).Fatal("failed sanitizing min bid")
	}
	if relayMinBidWei.BigInt().Sign() > 0 {
		log.Infof("min bid set to %v eth (%v wei)", cmd.Float(minBidFlag.Name), relayMinBidWei)
	}
	return &RelaySetupResult{
		RelayConfigs:       relayConfigs,
		MinBid:             *relayMinBidWei,
		RelayCheck:         cmd.Bool(relayCheckFlag.Name),
		TimeoutGetHeaderMs: timeoutGetHeaderMs,
		LateInSlotTimeMs:   lateInSlotTimeMs,
		CLIRelays:          []serverTypes.RelayEntry(relays),
	}, nil
}

func setupGenesis(cmd *cli.Command) (string, uint64) {
	var (
		genesisForkVersion string
		genesisTime        uint64
	)

	switch {
	case cmd.IsSet(customGenesisForkFlag.Name):
		genesisForkVersion = cmd.String(customGenesisForkFlag.Name)
	case cmd.Bool(sepoliaFlag.Name):
		genesisForkVersion = genesisForkVersionSepolia
		genesisTime = genesisTimeSepolia
	case cmd.Bool(holeskyFlag.Name):
		genesisForkVersion = genesisForkVersionHolesky
		genesisTime = genesisTimeHolesky
	case cmd.Bool(hoodiFlag.Name):
		genesisForkVersion = genesisForkVersionHoodi
		genesisTime = genesisTimeHoodi
	case cmd.Bool(mainnetFlag.Name):
		genesisForkVersion = genesisForkVersionMainnet
		genesisTime = genesisTimeMainnet
	default:
		flag.Usage()
		log.Fatal("please specify a genesis fork version (eg. -mainnet / -sepolia / -holesky / -hoodi / -genesis-fork-version flags)")
	}

	if cmd.IsSet(customGenesisTimeFlag.Name) {
		genesisTime = uint64(cmd.Uint(customGenesisTimeFlag.Name))
	}
	log.Infof("using genesis fork version: %s time: %d", genesisForkVersion, genesisTime)
	return genesisForkVersion, genesisTime
}

func setupLogging(cmd *cli.Command) error {
	// setup logging
	log.Logger.SetOutput(os.Stdout)
	if cmd.IsSet(jsonFlag.Name) {
		log.Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.RFC3339Milli,
		})
	} else {
		log.Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: config.RFC3339Milli,
			ForceColors:     cmd.Bool(colorFlag.Name),
		})
	}

	logLevel := cmd.String(logLevelFlag.Name)
	if cmd.IsSet(debugFlag.Name) {
		logLevel = "debug"
	}
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidLoglevel, logLevel)
	}
	log.Logger.SetLevel(lvl)

	if cmd.IsSet(logServiceFlag.Name) {
		log = log.WithField("service", cmd.String(logServiceFlag.Name))
	}

	// Add version to logs and say hello
	if cmd.Bool(logNoVersionFlag.Name) {
		log.Infof("starting mev-boost %s", config.Version)
	} else {
		log = log.WithField("version", config.Version)
		log.Infof("starting mev-boost")
	}
	log.Debug("debug logging enabled")
	return nil
}

func sanitizeMinBid(minBid float64) (*types.U256Str, error) {
	if minBid < 0.0 {
		return nil, errNegativeBid
	}
	if minBid > 1000000.0 {
		return nil, errLargeMinBid
	}
	return common.FloatEthTo256Wei(minBid)
}

func setupMuxConfig(cmd *cli.Command) *config.MuxConfig {
	configPath := cmd.String(muxConfigFlag.Name)
	if configPath == "" {
		log.Info("no mux config file specified, using default relay selection for all validators")
		return nil
	}
	muxConfig, err := config.LoadMuxConfig(configPath)
	if err != nil {
		log.WithError(err).Fatal("failed to load mux configuration")
		return nil
	}

	return muxConfig
}
