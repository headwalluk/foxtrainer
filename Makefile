# Build, test and lint foxtrainer.

GIT_TAG_VERSION := $(shell git describe --tags --dirty 2>/dev/null)
VERSION ?= $(if $(GIT_TAG_VERSION),$(patsubst v%,%,$(GIT_TAG_VERSION)),0.1.0-dev)
LINKER_FLAGS := -s -w -X github.com/headwalluk/foxtrainer/internal/buildinfo.stampedVersion=$(VERSION)
BINARY_DIR := bin
DIST_DIR := dist
RELEASE_PLATFORMS := linux/amd64 linux/arm64

.PHONY: all build test e2e lint fmt release clean

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
	shellcheck install.sh

fmt:
	golangci-lint fmt ./...

# Cross-compile release tarballs (foxtrainer_<os>_<arch>.tar.gz) and SHA256SUMS into dist/.
release:
	rm -rf $(DIST_DIR)
	set -eu; for PLATFORM in $(RELEASE_PLATFORMS); do \
		TARGET_OS="$${PLATFORM%/*}"; TARGET_ARCH="$${PLATFORM#*/}"; \
		STAGING_DIR="$(DIST_DIR)/foxtrainer_$${TARGET_OS}_$${TARGET_ARCH}"; \
		mkdir -p "$${STAGING_DIR}"; \
		CGO_ENABLED=0 GOOS="$${TARGET_OS}" GOARCH="$${TARGET_ARCH}" go build -trimpath -ldflags "$(LINKER_FLAGS)" -o "$${STAGING_DIR}/foxtrainer" ./cmd/foxtrainer; \
		cp LICENSE THIRD-PARTY-NOTICES.md README.md "$${STAGING_DIR}/"; \
		tar -C "$(DIST_DIR)" -czf "$${STAGING_DIR}.tar.gz" "$$(basename "$${STAGING_DIR}")"; \
		rm -rf "$${STAGING_DIR}"; \
	done
	cd $(DIST_DIR) && sha256sum foxtrainer_*.tar.gz > SHA256SUMS

clean:
	rm -rf $(BINARY_DIR) $(DIST_DIR)
