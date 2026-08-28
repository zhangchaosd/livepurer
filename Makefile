.PHONY:build
build:
	goreleaser release --skip=publish --snapshot --clean
