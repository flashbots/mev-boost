module github.com/flashbots/mev-middleware

go 1.16

require (
	github.com/ethereum/go-ethereum v1.10.11
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/stretchr/testify v1.7.0
)

replace (
	github.com/ethereum/go-ethereum => github.com/MariusVanDerWijden/go-ethereum v1.8.22-0.20211009100437-ac736f93f769
	github.com/gorilla/rpc => ./forked/gorilla/rpc
)
