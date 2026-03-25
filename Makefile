BINARY_NAME := terraxi
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build run test test-integration lint check clean install deps fmt

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/terraxi

run: build
	./$(BINARY_NAME) $(ARGS)

test:
	go test ./... -v -count=1

test-integration:
	go test -tags integration ./... -v -count=1

lint:
	golangci-lint run ./...

check: test lint

clean:
	rm -f $(BINARY_NAME)
	rm -rf imported/

install: build
	mv $(BINARY_NAME) $(GOPATH)/bin/ 2>/dev/null || mv $(BINARY_NAME) ~/go/bin/

deps:
	go mod tidy

fmt:
	go fmt ./...
