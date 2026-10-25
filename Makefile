.PHONY: help deps build build-web build-all build-unfaird install-unfaird dev-web clean ci

BIN         ?= bin/ipa-decrypter
GO          ?= go
BUN         ?= bun
UNFAIRD_DIR ?= third-party/unfaird
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     ?= -s -w -X github.com/ac0d3r/ipa-decrypter/internal/version.Version=$(VERSION)

help:
	@echo "ipa-decrypter"
	@echo ""
	@echo "  make deps           Install Go + bun dependencies"
	@echo "  make build          Build backend binary to $(BIN)"
	@echo "  make build-web      Build frontend to web/dist"
	@echo "  make build-all      Build backend + frontend"
	@echo "  make build-unfaird  Build unfaird iOS debs into third-party/unfaird/debs"
	@echo "  make install-unfaird SSH-install debs to a jailbroken device (root@ip:port)"
	@echo "  make dev-web        Start Vite dev server (proxies /api -> :8080)"
	@echo "  make ci             gofmt + go vet + go test ./..."
	@echo "  make clean          Remove bin/, web/dist, and unfaird build artifacts"

deps:
	$(GO) mod download
	cd web && $(BUN) install

build:
	@mkdir -p $(dir $(BIN))
	CGO_ENABLED=0 $(GO) build -trimpath -buildvcs=true -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/server

build-web:
	cd web && $(BUN) run build

build-all: build build-web

# Requires Theos, iPhoneOS SDK, and ldid. Override schemes with
# `make build-unfaird SCHEMES=rootless` or `SCHEMES=rootless roothide`.
build-unfaird:
	@if [ ! -f "$(UNFAIRD_DIR)/Makefile" ]; then \
		git submodule update --init "$(UNFAIRD_DIR)"; \
	fi
	$(MAKE) -C "$(UNFAIRD_DIR)" package-all $(if $(SCHEMES),SCHEMES="$(SCHEMES)")
	@ls -lh $(UNFAIRD_DIR)/debs/*.deb

# Install built debs onto a jailbroken device over SSH.
# Usage: `make install-unfaird DEVICE=root@192.168.1.2:2222` (port optional,
# defaults to 22). Without DEVICE you are prompted for root@ip:port.
install-unfaird:
	@debs="$(UNFAIRD_DIR)/debs/*.deb"; \
	if ! ls $$debs >/dev/null 2>&1; then \
		echo "no debs in $(UNFAIRD_DIR)/debs - run make build-unfaird first" >&2; exit 1; \
	fi; \
	if [ -n "$(DEVICE)" ]; then device="$(DEVICE)"; \
	else printf "Install to (root@ip:port): "; read device; fi; \
	if [ -z "$$device" ]; then echo "no device given" >&2; exit 1; fi; \
	host=$${device%:*}; port=$${device##*:}; \
	if [ "$$host" = "$$device" ]; then port=22; fi; \
	echo "Installing $$(ls $$debs | wc -l | tr -d ' ') deb(s) to $$host (port $$port)"; \
	scp -P $$port $$debs $$host:/tmp/ \
	&& ssh -p $$port $$host 'dpkg -i /tmp/*.deb'

dev-web:
	cd web && $(BUN) run dev

ci:
	@gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
	$(GO) vet ./...
	$(GO) test ./...

clean:
	@if [ -f "$(UNFAIRD_DIR)/Makefile" ]; then \
		$(MAKE) -C "$(UNFAIRD_DIR)" clean-package; \
	fi
	rm -rf $(UNFAIRD_DIR)/.build $(UNFAIRD_DIR)/.build-ios
	rm -rf $(dir $(BIN)) web/dist .cache
	rm -rf ipa-decrypter_latest.tar
