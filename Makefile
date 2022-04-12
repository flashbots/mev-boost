MERGEMOCK_DIR=../mergemock
MERGEMOCK_BIN=./mergemock

GIT_VER := $(shell git describe --tags --always --dirty="-dev")
DOCKER_REPO := flashbots/mev-boost

build:
	go build ./cmd/mev-boost

test:
	go test ./lib/... ./cmd/...

test-coverage:
	go test ./lib/... ./cmd/... -v -covermode=count -coverprofile=coverage.out

cover-html:
	go test -coverprofile=/tmp/go-sim-lb.cover.tmp ./...
	go tool cover -html=/tmp/go-sim-lb.cover.tmp
	unlink /tmp/go-sim-lb.cover.tmp

lint:
	revive -set_exit_status ./lib ./cmd
	go vet ./...
	staticcheck ./...

generate:
	go generate ./...

run:
	./mev-boost

run-boost-with-relay:
	./mev-boost -relayUrl http://127.0.0.1:28545

run-dev:
	go run cmd/mev-boost/main.go

run-mergemock-engine:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) engine --listen-addr 127.0.0.1:8550

run-mergemock-consensus:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) consensus --slot-time=4s --engine http://127.0.0.1:8550 --builder http://127.0.0.1:18550 --slot-bound 10

run-mergemock-relay:
	cd $(MERGEMOCK_DIR) && $(MERGEMOCK_BIN) relay --listen-addr 127.0.0.1:28545

run-mergemock-integration: build
	make -j4 run-boost-with-relay run-mergemock-engine run-mergemock-consensus run-mergemock-relay

build-for-docker:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${GIT_VER}" -v -o mev-boost ./cmd/mev-boost

docker-image:
	DOCKER_BUILDKIT=1 docker build . -t mev-boost
	docker tag mev-boost:latest ${DOCKER_REPO}:${GIT_VER}
	docker tag mev-boost:latest ${DOCKER_REPO}:kintsugi

docker-push:
	docker push ${DOCKER_REPO}:${GIT_VER}
	docker push ${DOCKER_REPO}:kintsugi
