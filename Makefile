VERSION ?= $(shell git describe --tags --always --dirty="-dev")
DOCKER_REPO := flashbots/mev-boost

# Set linker flags to:
#   -w: disables DWARF debugging information.
GO_BUILD_LDFLAGS += -w
#   -s: disables symbol table information.
GO_BUILD_LDFLAGS += -s
#   -X: sets the value of the symbol.
GO_BUILD_LDFLAGS += -X 'github.com/flashbots/mev-boost/config.Version=$(VERSION)'

# Remove all file system paths from the executable.
GO_BUILD_FLAGS += -trimpath
# Add linker flags to build flags.
GO_BUILD_FLAGS += -ldflags "$(GO_BUILD_LDFLAGS)"

.PHONY: all
all: build

.PHONY: v
v:
	@echo "${VERSION}"

.PHONY: build
build:
	@go version
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o mev-boost ./cmd/mev-boost

.PHONY: build-testcli
build-testcli:
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o test-cli ./cmd/test-cli

.PHONY: test
test:
	CGO_ENABLED=0 go test ./...

.PHONY: test-race
test-race:
	CGO_ENABLED=1 go test -race ./...

.PHONY: lint
lint:
	gofmt -d -s .
	gofumpt -d -extra .
	staticcheck ./...
	golangci-lint run

.PHONY: lt
lt: lint test

.PHONY: fmt
fmt:
	gofmt -s -w .
	gofumpt -extra -w .
	gci write .
	go mod tidy

.PHONY: test-coverage
test-coverage:
	CGO_ENABLED=0 go test -v -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func coverage.out

.PHONY: cover
cover:
	CGO_ENABLED=0 go test -coverprofile=coverage.out ./...
	go tool cover -func coverage.out
	unlink coverage.out

.PHONY: cover-html
cover-html:
	CGO_ENABLED=0 go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	unlink coverage.out

.PHONY: run-mergemock-integration
run-mergemock-integration: build
	./scripts/run_mergemock_integration.sh

.PHONY: docker-image
docker-image:
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 --build-arg VERSION=${VERSION} . -t mev-boost
	docker tag mev-boost:latest ${DOCKER_REPO}:${VERSION}
	docker tag mev-boost:latest ${DOCKER_REPO}:latest

.PHONY: clean
clean:
	git clean -fdx
