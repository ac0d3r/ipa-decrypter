.PHONY: help deps build build-web build-all dev-web clean test fmt vet

BIN        ?= bin/ipa-decrypter
GO         ?= go
NPM        ?= npm
LDFLAGS    ?= -s -w

help:
	@echo "ipa-decrypter"
	@echo ""
	@echo "  make deps        Install Go + npm dependencies"
	@echo "  make build       Build backend binary to $(BIN)"
	@echo "  make build-web   Build frontend to web/dist"
	@echo "  make build-all   Build backend + frontend"
	@echo "  make dev-web     Start Vite dev server (proxies /api -> :8080)"
	@echo "  make test        go test ./..."
	@echo "  make fmt         gofmt -w Go sources"
	@echo "  make vet         go vet ./..."
	@echo "  make clean       Remove $(BIN) and web/dist"

deps:
	$(GO) mod download
	cd web && $(NPM) install

build:
	@mkdir -p $(dir $(BIN))
	CGO_ENABLED=0 $(GO) build -trimpath -buildvcs=true -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/server

build-web:
	cd web && $(NPM) run build

build-all: build build-web

dev-web:
	cd web && $(NPM) run dev

test:
	$(GO) test ./...

fmt:
	@gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BIN) web/dist
