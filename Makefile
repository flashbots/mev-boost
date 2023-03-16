VERSION ?= $(shell git describe --tags --always --dirty="-dev")
DOCKER_REPO := flashbots/mev-boost

# Force os/user & net to be pure Go.
GO_BUILD_FLAGS += -tags osusergo,netgo
# Remove all file system paths from the executable.
GO_BUILD_FLAGS += -trimpath
# Make the build more verbose.
GO_BUILD_FLAGS += -v

# Set linker flags to:
#   -w: disables DWARF debugging information.
GO_BUILD_LDFLAGS += -w
#   -s: disables symbol table information.
GO_BUILD_LDFLAGS += -s
#   -X: sets the value of the symbol.
GO_BUILD_LDFLAGS += -X 'github.com/flashbots/mev-boost/config.Version=$(VERSION)'

.PHONY: all
all: build

.PHONY: v
v:
	@echo "${VERSION}"

.PHONY: build
build:
	$(EXTRA_ENV) go build $(GO_BUILD_FLAGS) -ldflags "$(GO_BUILD_LDFLAGS)" -o mev-boost .

.PHONY: build-portable
build-portable: EXTRA_ENV = CGO_CFLAGS="-O -D__BLST_PORTABLE__"
build-portable: build

.PHONY: build-static
build-static: GO_BUILD_LDFLAGS += "-extldflags=-static"
build-static: build

.PHONY: build-portable-static
build-portable-static: EXTRA_ENV = CGO_CFLAGS="-O -D__BLST_PORTABLE__"
build-portable-static: GO_BUILD_LDFLAGS += -extldflags=-static
build-portable-static: build

.PHONY: build-testcli
build-testcli:
	go build $(GO_BUILD_FLAGS) -ldflags "$(GO_BUILD_LDFLAGS)" -o test-cli ./cmd/test-cli

.PHONY: test
test:
	go test ./...

.PHONY: test-race
test-race:
	go test -race ./...

.PHONY: lint
lint:
	gofmt -d -s .
	gofumpt -d -extra .
	staticcheck ./...
	golangci-lint run

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
