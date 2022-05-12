module github.com/flashbots/mev-boost

go 1.18

require (
	github.com/ethereum/go-ethereum v1.10.17
	github.com/ferranbt/fastssz v0.0.0-20220303160658-88bb965b6747
	github.com/gorilla/mux v1.8.0
	github.com/prysmaticlabs/prysm v1.4.4
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0-20190314233015-f79a8a8ca69d // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto v0.0.4-0.20210318174700-74754f61e018 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/herumi/bls-eth-go-binary v0.0.0-20210130185500-57372fb27371 // indirect
	github.com/klauspost/cpuid/v2 v2.0.4 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prysmaticlabs/eth2-types v0.0.0-20210303084904-c9735a06829d // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/supranational/blst v0.3.4 // indirect
	github.com/urfave/cli/v2 v2.3.0 // indirect
	golang.org/x/crypto v0.0.0-20220321153916-2c7772ba3064 // indirect
	golang.org/x/sys v0.0.0-20220330033206-e17cdc41300f // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace (
	github.com/ethereum/go-ethereum => github.com/MariusVanDerWijden/go-ethereum v1.8.22-0.20211208130742-dd90624af970
	github.com/gorilla/rpc => ./forked/gorilla/rpc
)
