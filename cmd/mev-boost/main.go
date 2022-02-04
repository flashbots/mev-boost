package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"

	"github.com/flashbots/mev-middleware/lib"
	"github.com/sirupsen/logrus"
)

var (
	version = "dev" // is set during build process

	// defaults
	defaultPort         = 18550
	defaultExecutionURL = getEnv("EXECUTION_URL", "http://127.0.0.1:18545")
	defaultRelayURL     = getEnv("RELAY_URL", "http://127.0.0.1:28545")

	// cli flags
	port         = flag.Int("port", defaultPort, "port for mev-boost to listen on")
	executionURL = flag.String("executionUrl", defaultExecutionURL, "url to execution client")
	relayURL     = flag.String("relayUrl", defaultRelayURL, "url to relay")
)

func main() {
	flag.Parse()
	log := logrus.WithField("prefix", "cmd/mev-boost")
	log.Printf("mev-boost %s\n", version)

	store := lib.NewStore()
	router, err := lib.NewRouter(*executionURL, *relayURL, store, log)
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
