package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/flashbots/mev-middleware/lib"
)

const port = 18550

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("mev-boost: ")

	executionURL := flag.String("executionUrl", "http://127.0.0.1:18545", "url to execution client")
	relayURL := flag.String("relayUrl", "http://127.0.0.1:28545", "url to relay")

	flag.Parse()

	router, err := lib.NewRouter(*executionURL, *relayURL)
	if err != nil {
		panic(err)
	}

	log.Println("listening on: ", port)
	err = http.ListenAndServe(":"+strconv.Itoa(port), router)

	log.Println("error in server: ", err)
	panic(err)
}
