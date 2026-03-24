BINARY_NAME := terraxi
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
GO := /usr/local/go/bin/go

.PHONY: build run test lint clean install

build:
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/terraxi

run: build
	./$(BINARY_NAME) $(ARGS)

test:
	$(GO) test ./... -v

lint:
	$(GO) vet ./...

clean:
	rm -f $(BINARY_NAME)
	rm -rf imported/

install: build
	mv $(BINARY_NAME) $(GOPATH)/bin/ 2>/dev/null || mv $(BINARY_NAME) ~/go/bin/

deps:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...
