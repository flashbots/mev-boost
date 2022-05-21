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

var (
	version = "dev" // is set during build process

	// defaults
	defaultListenAddr     = getEnv("BOOST_LISTEN_ADDR", "localhost:18550")
	defaultRelayURLs      = getEnv("RELAY_URLS", "127.0.0.1:28545") // can be IP@PORT, PUBKEY@IP:PORT, https://IP, etc.
	defaultRelayTimeoutMs = getEnvInt("RELAY_TIMEOUT_MS", 2000)     // timeout for all the requests to the relay
	defaultRelayCheck     = os.Getenv("RELAY_STARTUP_CHECK") != ""

	// cli flags
	listenAddr     = flag.String("addr", defaultListenAddr, "listen-address for mev-boost server")
	relayURLs      = flag.String("relays", defaultRelayURLs, "relay urls - single entry or comma-separated list (pubkey@ip:port)")
	relayTimeoutMs = flag.Int("request-timeout", defaultRelayTimeoutMs, "timeout for requests to a relay [ms]")
	relayCheck     = flag.Bool("relay-check", defaultRelayCheck, "whether to check relay status on startup")
)

var log = logrus.WithField("module", "cmd/mev-boost")

func main() {
	flag.Parse()
	log.Printf("mev-boost %s\n", version)

	relays := parseRelayURLs(*relayURLs)
	log.WithField("relays", relays).Infof("using %d relays", len(relays))

	if *relayCheck {
		relayStartupCheck(relays)
	}

	relayTimeout := time.Duration(*relayTimeoutMs) * time.Millisecond
	server, err := server.NewBoostService(*listenAddr, relays, log, relayTimeout)
	if err != nil {
		log.WithError(err).Fatal("failed creating the server")
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

func parseRelayURLs(relayURLs string) []*server.RelayEntry {
	ret := []*server.RelayEntry{}
	for _, entry := range strings.Split(relayURLs, ",") {
		relay, err := server.NewRelayEntry(entry)
		if err != nil {
			log.WithError(err).WithField("relayURL", entry).Fatal("Invalid relay URL")
		}
		ret = append(ret, relay)
	}
	return ret
}

func relayStartupCheck(relays []*server.RelayEntry) error {
	log.Info("Checking relays...")
	for _, relay := range relays {
		log.WithField("relay", relay).Info("Checking relay")
		// err := relay.Check()
		// if err != nil {
		//     log.WithError(err).WithField("relay", relay).Fatal("Relay check failed")
		// }
	}
	return nil
}
