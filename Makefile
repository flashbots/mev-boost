build:
	go build ./cmd/mev-boost

test:
	go test ./lib/... ./cmd/...

lint:
	revive ./lib ./cmd

generate:
	go generate ./...

run:
	go run cmd/mev-boost/main.go
