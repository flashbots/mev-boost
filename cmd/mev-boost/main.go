package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/flashbots/mev-middleware/lib"
)

const port = 18550

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	router, err := lib.NewRouter("http://127.0.0.1:8550", "http://127.0.0.1:28550")
	if err != nil {
		panic(err)
	}

	fmt.Println("listening on", port)
	err = http.ListenAndServe(":"+strconv.Itoa(port), router)

	fmt.Println("error in server", err)
	panic(err)
}
