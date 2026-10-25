# syntax=docker/dockerfile:1

FROM oven/bun:1-alpine AS web
WORKDIR /src/web
COPY web/package.json web/bun.lock ./
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM golang:1.27-alpine AS ipatool
# CGO required: byteness/keyring → onepassword-sdk-go fails to compile with CGO_ENABLED=0 on linux.
RUN apk add --no-cache git gcc musl-dev
WORKDIR /src/ipatool
ARG IPATOOL_VERSION=latest
# "latest" resolves to the newest git tag at build time; pass `--build-arg IPATOOL_VERSION=<tag>` to pin
# (CI passes the resolved tag so layer caching busts only on new ipatool releases).
RUN if [ "$IPATOOL_VERSION" = "latest" ]; then \
        IPATOOL_VERSION="$(git ls-remote --tags --sort=-v:refname https://github.com/majd/ipatool.git \
            | grep -v '\^{}' | head -n1 | awk -F/ '{print $NF}')"; \
    fi; \
    echo "Building ipatool ${IPATOOL_VERSION}" >&2; \
    [ -n "$IPATOOL_VERSION" ] || { echo "failed to resolve ipatool version" >&2; exit 1; }; \
    git clone --depth 1 --branch "${IPATOOL_VERSION}" https://github.com/majd/ipatool.git .
COPY third-party/ipatool/keyring-backend-flag.patch /tmp/keyring-backend-flag.patch
RUN git apply --check /tmp/keyring-backend-flag.patch \
    && git apply /tmp/keyring-backend-flag.patch
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /ipatool .

FROM golang:1.27-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY third-party/fairplay/ ./third-party/fairplay/
COPY --from=web /src/web/dist ./web/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ipa-decrypter ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 1000 app
WORKDIR /app
COPY --from=backend /ipa-decrypter ./ipa-decrypter
COPY --from=web /src/web/dist ./web/dist
COPY --from=ipatool /ipatool /usr/local/bin/ipatool
ENV DATA_DIR=/data \
    WEB_DIR=/app/web/dist \
    LISTEN_ADDR=:8080 \
    IPATOOL_BIN=/usr/local/bin/ipatool \
    KEYCHAIN_PASSPHRASE=ipatool-ipa-decrypter
RUN mkdir -p /data && chown -R app:app /data /app
USER app
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/app/ipa-decrypter"]
