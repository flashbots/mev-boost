build:
	go build ./cmd/mev-boost

test:
	go test ./lib/... ./cmd/...

lint:
	revive ./lib ./cmd

run:
	go run cmd/mev-boost/main.go
