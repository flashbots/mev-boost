package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flashbots/mev-boost/server"
	"github.com/sirupsen/logrus"
)

const (
	genesisForkVersionMainnet = "0x00000000"
	genesisForkVersionKiln    = "0x70000069"
	genesisForkVersionRopsten = "0x80000069"
)

var (
	version = "dev" // is set during build process

	// defaults
	defaultListenAddr         = getEnv("BOOST_LISTEN_ADDR", "localhost:18550")
	defaultRelayTimeoutMs     = getEnvInt("RELAY_TIMEOUT_MS", 2000) // timeout for all the requests to the relay
	defaultRelayCheck         = os.Getenv("RELAY_STARTUP_CHECK") != ""
	defaultGenesisForkVersion = getEnv("GENESIS_FORK_VERSION", "")

	// cli flags
	listenAddr     = flag.String("addr", defaultListenAddr, "listen-address for mev-boost server")
	relayURLs      = flag.String("relays", "", "relay urls - single entry or comma-separated list (schema://pubkey@host)")
	relayTimeoutMs = flag.Int("request-timeout", defaultRelayTimeoutMs, "timeout for requests to a relay [ms]")
	relayCheck     = flag.Bool("relay-check", defaultRelayCheck, "whether to check relay status on startup")

	// helpers
	useGenesisForkVersionMainnet = flag.Bool("mainnet", false, "use Mainnet genesis fork version 0x00000000 (for signature validation)")
	useGenesisForkVersionKiln    = flag.Bool("kiln", false, "use Kiln genesis fork version 0x70000069 (for signature validation)")
	useGenesisForkVersionRopsten = flag.Bool("ropsten", false, "use Ropsten genesis fork version 0x80000069 (for signature validation)")
	useCustomGenesisForkVersion  = flag.String("genesis-fork-version", defaultGenesisForkVersion, "use a custom genesis fork version (for signature validation)")
)

var log = logrus.WithField("module", "cmd/mev-boost")

func main() {
	flag.Parse()
	log.Printf("mev-boost %s", version)

	genesisForkVersionHex := ""
	if *useCustomGenesisForkVersion != "" {
		genesisForkVersionHex = *useCustomGenesisForkVersion
	} else if *useGenesisForkVersionMainnet {
		genesisForkVersionHex = genesisForkVersionMainnet
	} else if *useGenesisForkVersionKiln {
		genesisForkVersionHex = genesisForkVersionKiln
	} else if *useGenesisForkVersionRopsten {
		genesisForkVersionHex = genesisForkVersionRopsten
	} else {
		log.Fatal("Please specify a genesis fork version (eg. -mainnet or -kiln or -ropsten or -genesis-fork-version flags)")
	}
	log.Infof("Using genesis fork version: %s", genesisForkVersionHex)

	relays := parseRelayURLs(*relayURLs)
	if len(relays) == 0 {
		log.Fatal("No relays specified")
	}
	log.WithField("relays", relays).Infof("using %d relays", len(relays))

	relayTimeout := time.Duration(*relayTimeoutMs) * time.Millisecond
	server, err := server.NewBoostService(*listenAddr, relays, log, genesisForkVersionHex, relayTimeout)
	if err != nil {
		log.WithError(err).Fatal("failed creating the server")
	}

	if *relayCheck && !server.CheckRelays() {
		log.Fatal("relays unavailable")
	}

	log.Println("listening on", *listenAddr)
	log.Fatal(server.StartHTTPServer())
}

func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		val, err := strconv.Atoi(value)
		if err == nil {
			return val
		}
	}
	return defaultValue
}

func parseRelayURLs(relayURLs string) []server.RelayEntry {
	ret := []server.RelayEntry{}
	for _, entry := range strings.Split(relayURLs, ",") {
		relay, err := server.NewRelayEntry(entry)
		if err != nil {
			log.WithError(err).WithField("relayURL", entry).Fatal("Invalid relay URL")
		}
		ret = append(ret, relay)
	}
	return ret
}
