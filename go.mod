module github.com/flashbots/mev-boost

go 1.18

require (
	github.com/ethereum/go-ethereum v1.10.17
	github.com/fjl/gencodec v0.0.0-20191126094850-e283372f291f
	github.com/gorilla/mux v1.8.0
	github.com/sirupsen/logrus v1.2.0
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/garslo/gogen v0.0.0-20170306192744-1d203ffc1f61 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.0.0-20220321153916-2c7772ba3064 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/sys v0.0.0-20220330033206-e17cdc41300f // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/tools v0.1.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace (
	github.com/ethereum/go-ethereum => github.com/MariusVanDerWijden/go-ethereum v1.8.22-0.20211208130742-dd90624af970
	github.com/gorilla/rpc => ./forked/gorilla/rpc
)
