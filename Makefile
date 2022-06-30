MERGEMOCK_DIR=../mergemock
MERGEMOCK_BIN=./mergemock

VERSION ?= $(shell git describe --tags --always --dirty="-dev")
DOCKER_REPO := flashbots/mev-boost

v:
	@echo "${VERSION}"

build:
	go build -ldflags "-X main.version=${VERSION}" -v -o mev-boost ./cmd/mev-boost

build-testcli:
	go build -ldflags "-X main.version=${VERSION}" -v -o test-cli ./cmd/test-cli

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	revive -set_exit_status ./...
	go vet ./...
	staticcheck ./...

generate-ssz:
	rm -f types/builder_encoding.go
	sszgen --path types --include ../go-ethereum/common/hexutil --objs Eth1Data,BeaconBlockHeader,SignedBeaconBlockHeader,ProposerSlashing,Checkpoint,AttestationData,IndexedAttestation,AttesterSlashing,Attestation,Deposit,VoluntaryExit,SyncAggregate,ExecutionPayloadHeader,VersionedExecutionPayloadHeader,BlindedBeaconBlockBody,BlindedBeaconBlock,RegisterValidatorRequestMessage,BuilderBid,SignedBuilderBid

test-coverage:
	go test -race -v -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func coverage.out

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func coverage.out
	unlink coverage.out

cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	unlink coverage.out

run:
	./mev-boost

run-boost-with-relay:
	./mev-boost -mainnet -relays http://0x821961b64d99b997c934c22b4fd6109790acf00f7969322c4e9dbf1ca278c333148284c01c5ef551a1536ddd14b178b9@127.0.0.1:28545

run-dev:
	go run cmd/mev-boost/main.go

run-mergemock-engine:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) engine --listen-addr 127.0.0.1:8551

run-mergemock-consensus:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) consensus --slot-time=4s --engine http://127.0.0.1:8551 --builder http://127.0.0.1:18550 --slot-bound 10

run-mergemock-relay:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) relay --listen-addr 127.0.0.1:28545 --secret-key 1e64a14cb06073c2d7c8b0b891e5dc3dc719b86e5bf4c131ddbaa115f09f8f52

run-mergemock-integration: build
	make -j3 run-boost-with-relay run-mergemock-consensus run-mergemock-relay

build-for-docker:
	GOOS=linux go build -ldflags "-X main.version=${VERSION}" -v -o mev-boost ./cmd/mev-boost

docker-image:
	DOCKER_BUILDKIT=1 docker build . -t mev-boost
	docker tag mev-boost:latest ${DOCKER_REPO}:${VERSION}
	docker tag mev-boost:latest ${DOCKER_REPO}:latest

docker-push:
	docker push ${DOCKER_REPO}:${VERSION}
	docker push ${DOCKER_REPO}:latest
