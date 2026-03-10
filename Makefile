MODULE  := github.com/frankcruz/tasklin
BINARY  := tasklin
BIN_DIR := bin

# Injected at link time
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(DATE)

GO      := go
GOFLAGS := -trimpath

# Cross-compile targets: OS/ARCH pairs
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: all build build-all run test install clean help

## all: build for the current architecture (default)
all: build

## build: compile for the current OS/ARCH into bin/
build:
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

## build-all: cross-compile for all target platforms into bin/
build-all:
	@mkdir -p $(BIN_DIR)
	@$(foreach PLATFORM,$(PLATFORMS), \
		$(eval OS   := $(word 1,$(subst /, ,$(PLATFORM)))) \
		$(eval ARCH := $(word 2,$(subst /, ,$(PLATFORM)))) \
		$(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
		$(eval OUT  := $(BIN_DIR)/$(BINARY)-$(OS)-$(ARCH)$(EXT)) \
		echo "  building $(OUT) ..."; \
		GOOS=$(OS) GOARCH=$(ARCH) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(OUT) . || exit 1; \
	)
	@echo "All builds complete. Artifacts in $(BIN_DIR)/"

## run: build and run the application
run: build
	./$(BIN_DIR)/$(BINARY)

## test: run all unit tests with race detector and coverage
test:
	$(GO) test -race -cover ./...

## test-ci: run tests in CI mode (no colour, junit-friendly output)
test-ci:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

## install: install the binary to GOPATH/bin (or GOBIN if set)
install:
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" .

## clean: remove build artefacts
clean:
	@rm -rf $(BIN_DIR) coverage.out

## help: list available targets
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
