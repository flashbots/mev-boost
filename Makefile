VERSION ?= $(shell git describe --tags --always --dirty="-dev")
DOCKER_REPO := flashbots/mev-boost

.PHONY: all
all: build

.PHONY: v
v:
	@echo "${VERSION}"

.PHONY: build
build:
	go build -trimpath -ldflags "-s -X 'github.com/flashbots/mev-boost/config.Version=${VERSION}'" -v -o mev-boost .

.PHONY: build-portable
build-portable:
	CGO_CFLAGS="-O -D__BLST_PORTABLE__" go build -trimpath -ldflags "-s -X 'github.com/flashbots/mev-boost/config.Version=${VERSION}'" -v -o mev-boost .

.PHONY: build-testcli
build-testcli:
	go build -trimpath -ldflags "-s -X 'github.com/flashbots/mev-boost/config.Version=${VERSION}'" -v -o test-cli ./cmd/test-cli

.PHONY: test
test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: lint
lint:
	gofumpt -d -extra .
	revive -set_exit_status ./...
	go vet ./...
	staticcheck ./...
	golangci-lint run --config=.golangci.yml

.PHONY: test-coverage
test-coverage:
	go test -race -v -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func coverage.out

.PHONY: cover
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func coverage.out
	unlink coverage.out

.PHONY: cover-html
cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	unlink coverage.out

.PHONY: run-mergemock-integration
run-mergemock-integration: build
	./scripts/run_mergemock_integration.sh

.PHONY: docker-image
docker-image:
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 --build-arg CGO_CFLAGS="" --build-arg VERSION=${VERSION} . -t mev-boost
	docker tag mev-boost:latest ${DOCKER_REPO}:${VERSION}
	docker tag mev-boost:latest ${DOCKER_REPO}:latest

.PHONY: docker-image-portable
docker-image-portable:
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 --build-arg CGO_CFLAGS="-O -D__BLST_PORTABLE__" --build-arg VERSION=${VERSION}  . -t mev-boost
	docker tag mev-boost:latest ${DOCKER_REPO}:${VERSION}

.PHONY: docker-push-version
docker-push-version:
	docker push ${DOCKER_REPO}:${VERSION}

.PHONY: clean
clean:
	git clean -fdx
