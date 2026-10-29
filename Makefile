run:
	go run ./cmd/server

test:
	go test -race ./...

generate:
	go generate ./...

build:
	go build -o bin/server ./cmd/server

.PHONY: run test generate build
