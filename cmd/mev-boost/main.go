package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/flashbots/mev-middleware/lib"
)

const port = 28545

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	router, err := lib.NewRouter("http://localhost:8545", "http://localhost:9545")
	if err != nil {
		panic(err)
	}

	fmt.Println("listening on", port)
	err = http.ListenAndServe(":"+strconv.Itoa(port), router)

	fmt.Println("error in server", err)
	panic(err)
}
