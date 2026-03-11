.PHONY: build test

build:
	go build ./cmd/spirited-env

test:
	go test ./...
