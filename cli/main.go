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
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

const (
	genesisForkVersionMainnet = "0x00000000"
	genesisForkVersionSepolia = "0x90000069"
	genesisForkVersionGoerli  = "0x00001020"
	genesisForkVersionHolesky = "0x01017000"

	genesisTimeMainnet = 1606824023
	genesisTimeSepolia = 1655733600
	genesisTimeGoerli  = 1614588812
	genesisTimeHolesky = 1695902400
)

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
		log.Infof("mev-boost %s\n", config.Version)
		return nil
	}

	if err := setupLogging(cmd); err != nil {
		flag.Usage()
		log.WithError(err).Fatal("failed setting up logging")
	}

	var (
		genesisForkVersion, genesisTime      = setupGenesis(cmd)
		relays, monitors, minBid, relayCheck = setupRelays(cmd)
		listenAddr                           = cmd.String(addrFlag.Name)
	)

	opts := server.BoostServiceOpts{
		Log:                      log,
		ListenAddr:               listenAddr,
		Relays:                   relays,
		RelayMonitors:            monitors,
		GenesisForkVersionHex:    genesisForkVersion,
		GenesisTime:              genesisTime,
		RelayCheck:               relayCheck,
		RelayMinBid:              minBid,
		RequestTimeoutGetHeader:  time.Duration(cmd.Int(timeoutGetHeaderFlag.Name)) * time.Millisecond,
		RequestTimeoutGetPayload: time.Duration(cmd.Int(timeoutGetPayloadFlag.Name)) * time.Millisecond,
		RequestTimeoutRegVal:     time.Duration(cmd.Int(timeoutRegValFlag.Name)) * time.Millisecond,
		RequestMaxRetries:        int(cmd.Int(maxRetriesFlag.Name)),
	}
	service, err := server.NewBoostService(opts)
	if err != nil {
		log.WithError(err).Fatal("failed creating the server")
	}

	if relayCheck && service.CheckRelays() == 0 {
		log.Error("no relay passed the health-check!")
	}

	log.Infof("Listening on %v", listenAddr)
	return service.StartHTTPServer()
}

func setupRelays(cmd *cli.Command) (relayList, relayMonitorList, types.U256Str, bool) {
	// For backwards compatibility with the -relays flag.
	var (
		relays   relayList
		monitors relayMonitorList
	)
	if cmd.IsSet(relaysFlag.Name) {
		relayURLs := cmd.StringSlice(relaysFlag.Name)
		for _, urls := range relayURLs {
			for _, url := range strings.Split(urls, ",") {
				if err := relays.Set(strings.TrimSpace(url)); err != nil {
					log.WithError(err).WithField("relay", url).Fatal("Invalid relay URL")
				}
			}
		}
	}

	if len(relays) == 0 {
		log.Fatal("no relays specified")
	}
	log.Infof("using %d relays", len(relays))
	for index, relay := range relays {
		log.Infof("relay #%d: %s", index+1, relay.String())
	}

	// For backwards compatibility with the -relay-monitors flag.
	if cmd.IsSet(relayMonitorFlag.Name) {
		monitorURLs := cmd.StringSlice(relayMonitorFlag.Name)
		for _, urls := range monitorURLs {
			for _, url := range strings.Split(urls, ",") {
				if err := monitors.Set(strings.TrimSpace(url)); err != nil {
					log.WithError(err).WithField("relayMonitor", url).Fatal("Invalid relay monitor URL")
				}
			}
		}
	}

	if len(monitors) > 0 {
		log.Infof("using %d relay monitors", len(monitors))
		for index, relayMonitor := range monitors {
			log.Infof("relay-monitor #%d: %s", index+1, relayMonitor.String())
		}
	}

	relayMinBidWei, err := sanitizeMinBid(cmd.Float(minBidFlag.Name))
	if err != nil {
		log.WithError(err).Fatal("Failed sanitizing min bid")
	}
	if relayMinBidWei.BigInt().Sign() > 0 {
		log.Infof("Min bid set to %v eth (%v wei)", cmd.Float(minBidFlag.Name), relayMinBidWei)
	}
	return relays, monitors, *relayMinBidWei, cmd.Bool(relayCheckFlag.Name)
}

func setupGenesis(cmd *cli.Command) (string, uint64) {
	var (
		genesisForkVersion string
		genesisTime        uint64
	)

	switch {
	case cmd.Bool(customGenesisForkFlag.Name):
		genesisForkVersion = cmd.String(customGenesisForkFlag.Name)
	case cmd.Bool(sepoliaFlag.Name):
		genesisForkVersion = genesisForkVersionSepolia
		genesisTime = genesisTimeSepolia
	case cmd.Bool(holeskyFlag.Name):
		genesisForkVersion = genesisForkVersionHolesky
		genesisTime = genesisTimeHolesky
	case cmd.Bool(mainnetFlag.Name):
		genesisForkVersion = genesisForkVersionMainnet
		genesisTime = genesisTimeMainnet
	default:
		flag.Usage()
		log.Fatal("please specify a genesis fork version (eg. -mainnet / -sepolia / -goerli / -holesky / -genesis-fork-version flags)")
	}

	if cmd.IsSet(customGenesisTimeFlag.Name) {
		genesisTime = cmd.Uint(customGenesisTimeFlag.Name)
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
