.PHONY: build frontend

frontend:
	cd web && npm ci && npm run build

build:
	$(MAKE) frontend
	goreleaser release --skip=publish --snapshot --clean
