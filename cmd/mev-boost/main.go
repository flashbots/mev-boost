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

	executionURL := flag.String("executionURL", "http://127.0.0.1:8550", "url to execution client")
	consensusURL := flag.String("consensusURL", "http://127.0.0.1:28550", "url to consensus client")

	flag.Parse()

	router, err := lib.NewRouter(*executionURL, *consensusURL)
	if err != nil {
		panic(err)
	}

	log.Println("listening on: ", port)
	err = http.ListenAndServe(":"+strconv.Itoa(port), router)

	log.Println("error in server: ", err)
	panic(err)
}
