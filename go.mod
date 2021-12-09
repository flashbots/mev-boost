module github.com/flashbots/mev-middleware

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ethereum/go-ethereum v1.10.11
	github.com/fjl/gencodec v0.0.0-20191126094850-e283372f291f // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/stretchr/testify v1.7.0
)

replace (
	github.com/ethereum/go-ethereum => github.com/MariusVanDerWijden/go-ethereum v1.8.22-0.20211203090255-c65940378e9f
	github.com/gorilla/rpc => ./forked/gorilla/rpc
)
