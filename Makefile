MERGEMOCK_DIR=../mergemock
MERGEMOCK_BIN=./mergemock

build:
	go build ./cmd/mev-boost

test:
	go test ./lib/... ./cmd/...

lint:
	revive -set_exit_status ./lib ./cmd

generate:
	go generate ./...

run:
	./mev-boost

run-boost-with-mergemock:
	./mev-boost -executionUrl http://127.0.0.1:8550

run-dev:
	go run cmd/mev-boost/main.go

run-mergemock-engine:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) engine

run-mergemock-consensus:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) consensus --slot-time=4s --engine http://127.0.0.1:18550

run-mergemock-integration: build
	make -j3 run-boost-with-mergemock run-mergemock-engine run-mergemock-consensus
