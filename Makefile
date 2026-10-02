# Build, test and lint foxtrainer.

GIT_TAG_VERSION := $(shell git describe --tags --dirty 2>/dev/null)
VERSION ?= $(if $(GIT_TAG_VERSION),$(patsubst v%,%,$(GIT_TAG_VERSION)),0.1.0-dev)
LINKER_FLAGS := -s -w -X github.com/headwalluk/foxtrainer/internal/buildinfo.stampedVersion=$(VERSION)
BINARY_DIR := bin

.PHONY: all build test e2e lint fmt clean

all: lint test build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LINKER_FLAGS)" -o $(BINARY_DIR)/foxtrainer ./cmd/foxtrainer

test:
	go test -race ./...

# End-to-end tests against a real Firefox on a throwaway profile; needs firefox-devedition (or -args -firefox=PATH).
e2e:
	go test -tags e2e -count=1 -v ./internal/e2e/

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

clean:
	rm -rf $(BINARY_DIR)
