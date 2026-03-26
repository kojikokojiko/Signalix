.PHONY: test test-integration build

test:
	go test ./...

test-integration:
	go test -tags=integration ./internal/integration/... -v -timeout 60s

build:
	go build ./cmd/api
	go build ./cmd/worker
