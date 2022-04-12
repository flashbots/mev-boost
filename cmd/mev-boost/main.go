package main

import (
	"flag"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flashbots/mev-boost/lib"
	"github.com/sirupsen/logrus"
)

var (
	version = "dev" // is set during build process

	// defaults
	defaultPort      = 18550
	defaultRelayURLs = getEnv("RELAY_URLS", "http://127.0.0.1:28545")

	// cli flags
	port      = flag.Int("port", defaultPort, "port for mev-boost to listen on")
	relayURLs = flag.String("relayUrl", defaultRelayURLs, "relay urls - single entry or comma-separated list")
)

func main() {
	rand.Seed(time.Now().UnixNano()) // warning: not a cryptographically secure seed

	flag.Parse()
	log := logrus.WithField("prefix", "cmd/mev-boost")
	log.Printf("mev-boost %s\n", version)

	_relayURLs := []string{}
	for _, entry := range strings.Split(*relayURLs, ",") {
		_relayURLs = append(_relayURLs, strings.Trim(entry, " "))
	}

	routerOpts := lib.RouterOptions{
		RelayURLs: _relayURLs,
		Store:     lib.NewStoreWithCleanup(),
		Log:       log,
	}
	router, err := lib.NewRouter(routerOpts)
	if err != nil {
		panic(err)
	}

	log.Println("listening on: ", *port)
	err = http.ListenAndServe(":"+strconv.Itoa(*port), router)

	log.Fatalf("error in server: %v", err)
}

func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
