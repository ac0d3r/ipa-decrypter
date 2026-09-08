.PHONY: help deps build build-web build-all build-unfaird dev-web clean clean-unfaird test fmt vet

BIN         ?= bin/ipa-decrypter
GO          ?= go
NPM         ?= npm
LDFLAGS     ?= -s -w
UNFAIRD_DIR ?= unfaird

help:
	@echo "ipa-decrypter"
	@echo ""
	@echo "  make deps           Install Go + npm dependencies"
	@echo "  make build          Build backend binary to $(BIN)"
	@echo "  make build-web      Build frontend to web/dist"
	@echo "  make build-all      Build backend + frontend"
	@echo "  make build-unfaird  Build unfaird iOS debs and copy them to integrations/unfaird"
	@echo "  make dev-web        Start Vite dev server (proxies /api -> :8080)"
	@echo "  make test           go test ./..."
	@echo "  make fmt            gofmt -w Go sources"
	@echo "  make vet            go vet ./..."
	@echo "  make clean          Remove $(BIN), web/dist, and unfaird build artifacts"
	@echo "  make clean-unfaird  Remove unfaird debs and Theos/Swift build dirs"

deps:
	$(GO) mod download
	cd web && $(NPM) install

build:
	@mkdir -p $(dir $(BIN))
	CGO_ENABLED=0 $(GO) build -trimpath -buildvcs=true -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/server

build-web:
	cd web && $(NPM) run build

build-all: build build-web

# Requires Theos, iPhoneOS SDK, and ldid. Override schemes with
# `make build-unfaird SCHEMES=rootless` or `SCHEMES=rootless roothide`.
build-unfaird:
	@if [ ! -f "$(UNFAIRD_DIR)/Makefile" ]; then \
		git submodule update --init "$(UNFAIRD_DIR)"; \
	fi
	$(MAKE) -C "$(UNFAIRD_DIR)" package-all $(if $(SCHEMES),SCHEMES="$(SCHEMES)")
	@mkdir -p integrations/unfaird
	@found=0; \
	for f in $(UNFAIRD_DIR)/debs/*.deb; do \
		[ -f "$$f" ] || continue; \
		cp -f "$$f" integrations/unfaird/; \
		found=1; \
	done; \
	if [ "$$found" -eq 0 ]; then echo "no debs in $(UNFAIRD_DIR)/debs" >&2; exit 1; fi; \
	ls -lh integrations/unfaird/*.deb

dev-web:
	cd web && $(NPM) run dev

test:
	$(GO) test ./...

fmt:
	@gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	$(GO) vet ./...

clean: clean-unfaird
	rm -rf $(BIN) web/dist

clean-unfaird:
	@if [ -f "$(UNFAIRD_DIR)/Makefile" ]; then \
		$(MAKE) -C "$(UNFAIRD_DIR)" clean-package; \
	fi
	rm -rf $(UNFAIRD_DIR)/.build $(UNFAIRD_DIR)/.build-ios
	rm -f integrations/unfaird/*.deb
