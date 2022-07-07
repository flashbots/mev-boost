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

run-dev:
	go run cmd/mev-boost/main.go

run-mergemock-integration: build
	./scripts/run_mergemock_integration.sh

build-for-docker:
	GOOS=linux go build -ldflags "-X main.version=${VERSION}" -v -o mev-boost ./cmd/mev-boost

docker-image:
	DOCKER_BUILDKIT=1 docker build . -t mev-boost
	docker tag mev-boost:latest ${DOCKER_REPO}:${VERSION}
	docker tag mev-boost:latest ${DOCKER_REPO}:latest

docker-push:
	docker push ${DOCKER_REPO}:${VERSION}
	docker push ${DOCKER_REPO}:latest
