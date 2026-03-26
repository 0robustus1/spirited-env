.PHONY: build test smoke

build:
	go build ./cmd/spirited-env

test:
	go test ./...

smoke:
	SPIRITED_ENV_SMOKE=1 go test ./internal/app -run 'TestNoEnvExecSmoke_' -v
