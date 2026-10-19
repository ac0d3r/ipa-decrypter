# syntax=docker/dockerfile:1

FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS ipatool
# CGO required: byteness/keyring → onepassword-sdk-go fails to compile with CGO_ENABLED=0 on linux.
RUN apk add --no-cache git patch gcc musl-dev
WORKDIR /src/ipatool
ARG IPATOOL_VERSION=v2.7.0
RUN git clone --depth 1 --branch "${IPATOOL_VERSION}" https://github.com/majd/ipatool.git .
COPY integrations/ipatool/keyring-backend-flag.patch /tmp/keyring-backend-flag.patch
RUN git apply --check /tmp/keyring-backend-flag.patch \
    && git apply /tmp/keyring-backend-flag.patch
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /ipatool .

FROM golang:1.25-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ipa-decrypter ./cmd/server

FROM alpine:3.21
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
    CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1
ENTRYPOINT ["/app/ipa-decrypter"]
