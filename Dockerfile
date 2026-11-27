# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27

# -----------------------------------------------------------------------------
# Stage 1: Shared Base
# -----------------------------------------------------------------------------
FROM golang:${GO_VERSION}-bookworm AS base
WORKDIR /app
ENV CGO_ENABLED=0

# -----------------------------------------------------------------------------
# Stage 2: Dependencies (cached Go module layer)
# -----------------------------------------------------------------------------
FROM base AS deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# -----------------------------------------------------------------------------
# Target A: Local / Development (used when APP_ENV != "production")
# -----------------------------------------------------------------------------
FROM deps AS local
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/air-verse/air@latest && \
    useradd --uid 10001 --create-home --shell /bin/bash appuser && \
    mkdir -p /go/pkg/mod /tmp/go-build && \
    chown -R appuser:appuser /app /go /tmp/go-build

ENV APP_ENV=local \
    HTTP_ADDR=0.0.0.0:8080 \
    GOCACHE=/tmp/go-build

COPY --chown=appuser:appuser . .
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD ["curl", "-fsS", "http://127.0.0.1:8080/api/v1/health"]
CMD ["air", "-c", ".air.api.toml"]

# -----------------------------------------------------------------------------
# Stage 3: Compiled Static Binary Builder (used for both UAT and Production)
# -----------------------------------------------------------------------------
FROM deps AS build
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# -----------------------------------------------------------------------------
# Target B: Hardened Distroless Compiled Runner (used when APP_ENV is "uat" or "production")
# -----------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS production
ARG APP_ENV=production
WORKDIR /app
ENV APP_ENV=${APP_ENV} \
    HTTP_ADDR=0.0.0.0:8080

COPY --from=build --chown=nonroot:nonroot /out/api /app/api
COPY --from=build --chown=nonroot:nonroot /out/worker /app/worker
COPY --from=build --chown=nonroot:nonroot /out/migrate /app/migrate

USER nonroot:nonroot
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/api", "-healthcheck"]

ENTRYPOINT ["/app/api"]
