.PHONY: build frontend test release

BINARY ?= bin/pure-live

frontend:
	npm --prefix web ci
	npm --prefix web run build

# The frontend is embedded in this native executable; no runtime assets are needed.
build: frontend
	mkdir -p $(dir $(BINARY))
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) .

test:
	npm --prefix web run typecheck
	go vet ./...
	go test ./...

# Optional: generate all release archives locally with GoReleaser installed.
release: frontend
	goreleaser release --skip=publish --snapshot --clean
