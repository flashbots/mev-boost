module github.com/flashbots/mev-middleware

go 1.16

require (
	github.com/ethereum/go-ethereum v1.10.11
	github.com/fatih/color v1.13.0
	github.com/fjl/gencodec v0.0.0-20191126094850-e283372f291f
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/rpc v1.2.0
	github.com/minio/sha256-simd v0.1.1
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2
)

replace (
	github.com/ethereum/go-ethereum => github.com/MariusVanDerWijden/go-ethereum v1.8.22-0.20211208130742-dd90624af970
	github.com/gorilla/rpc => ./forked/gorilla/rpc
)
