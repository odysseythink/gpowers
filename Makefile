# gpowers Makefile
# Manage builds, tests, and releases for all modules.

# ── Version ──────────────────────────────────────────────────────────
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# ── Go settings ──────────────────────────────────────────────────────
GO      := go
GOFLAGS := -trimpath
CGO     := CGO_ENABLED=0

# ── Directories ──────────────────────────────────────────────────────
BUILD_DIR  := build
DIST_DIR   := dist

# ── Binaries ─────────────────────────────────────────────────────────
BIN_INSTALL    := $(BUILD_DIR)/install
BIN_REWRITE    := $(BUILD_DIR)/_gpowers-rewrite-browser
BIN_BROWSE     := $(BUILD_DIR)/browse
BIN_TERMINAL   := $(BUILD_DIR)/terminal-agent

ALL_BINS := $(BIN_INSTALL) $(BIN_REWRITE) $(BIN_BROWSE) $(BIN_TERMINAL)

# ── Phony targets ────────────────────────────────────────────────────
.PHONY: all build build-install build-rewrite build-browse build-terminal \
        test test-root test-browse tidy tidy-root tidy-browse \
        lint shellcheck \
        release snapshot \
        clean help

# ── Default ──────────────────────────────────────────────────────────
all: build

# ── Build ────────────────────────────────────────────────────────────
build: build-install build-rewrite build-browse build-terminal

build-install: $(BIN_INSTALL)

build-rewrite: $(BIN_REWRITE)

build-browse: $(BIN_BROWSE)

build-terminal: $(BIN_TERMINAL)

$(BIN_INSTALL): cmd/install/*.go go.mod
	$(CGO) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ ./cmd/install

$(BIN_REWRITE): bin/gpowers-rewrite-browser.go go.mod
	$(CGO) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ ./bin/gpowers-rewrite-browser.go

$(BIN_BROWSE): tools/skills/browse-go/cmd/browse/*.go tools/skills/browse-go/go.mod
	cd tools/skills/browse-go && $(CGO) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o ../../$@ ./cmd/browse

$(BIN_TERMINAL): tools/skills/browse-go/cmd/terminal-agent/*.go tools/skills/browse-go/go.mod
	cd tools/skills/browse-go && $(CGO) $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o ../../$@ ./cmd/terminal-agent

# ── Test ─────────────────────────────────────────────────────────────
test: test-root test-browse

test-root:
	$(GO) test ./...

test-browse:
	cd tools/skills/browse-go && $(GO) test ./...

# ── Tidy ─────────────────────────────────────────────────────────────
tidy: tidy-root tidy-browse

tidy-root:
	$(GO) mod tidy

tidy-browse:
	cd tools/skills/browse-go && $(GO) mod tidy

# ── Lint ─────────────────────────────────────────────────────────────
lint:
	$(GO) vet ./...
	cd tools/skills/browse-go && $(GO) vet ./...

shellcheck:
	shellcheck -S warning bin/*.sh lib/*.sh tools/bin/* core/hooks/session-start || true
	shellcheck -S error bin/gpowers bin/gpowers-path bin/gpowers-init \
	                    bin/gpowers-detect-platforms bin/gpowers-upgrade \
	                    bin/gpowers-migrate bin/gpowers-platforms \
	                    uninstall

# ── GoReleaser ───────────────────────────────────────────────────────
release:
	goreleaser release --clean

snapshot:
	goreleaser release --clean --snapshot

# ── Clean ────────────────────────────────────────────────────────────
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)

# ── Help ─────────────────────────────────────────────────────────────
help:
	@echo "gpowers Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Build targets:"
	@echo "  all              Build all binaries (default)"
	@echo "  build            Same as all"
	@echo "  build-install    Build the install binary"
	@echo "  build-rewrite    Build the _gpowers-rewrite-browser binary"
	@echo "  build-browse     Build the browse daemon"
	@echo "  build-terminal   Build the terminal-agent"
	@echo ""
	@echo "Test targets:"
	@echo "  test             Run all tests"
	@echo "  test-root        Run root module tests"
	@echo "  test-browse      Run browse-go module tests"
	@echo ""
	@echo "Utility targets:"
	@echo "  tidy             Run go mod tidy on all modules"
	@echo "  lint             Run go vet on all modules"
	@echo "  shellcheck       Run shellcheck on shell scripts"
	@echo ""
	@echo "Release targets:"
	@echo "  release          Run GoReleaser release (requires git tag)"
	@echo "  snapshot         Run GoReleaser snapshot (no tag required)"
	@echo ""
	@echo "  clean            Remove build and dist directories"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  COMMIT=$(COMMIT)"
	@echo "  DATE=$(DATE)"
