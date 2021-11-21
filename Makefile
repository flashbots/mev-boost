test:
	go test ./lib/... ./cmd/...

lint:
	revive ./lib ./cmd
