# Frontend: React + Vite SPA → web/dist (Vite build.outDir).
FROM node:22-alpine AS fe
WORKDIR /app/web/frontend
RUN corepack enable
COPY web/frontend/package.json web/frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
COPY web/frontend/ ./
RUN pnpm build

# Go builder always runs on the build host and cross-compiles to the target.
# web/dist must be present before `go build` because //go:embed all:dist
# (web/embed.go) bakes the SPA into the binary — the runtime image no longer
# needs a separate web/ tree.
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
# Overlay the real SPA on top of the committed placeholder so the binary
# embeds the actual admin UI rather than the "not built" stub.
COPY --from=fe /app/web/dist ./web/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o kiro-go .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/kiro-go .
# State directory — bind-mounted in production. CONFIG_PATH is set so the
# binary keeps using /app/data even though ResolveConfigPath would otherwise
# prefer ~/.kiroproxy inside the container.
RUN mkdir -p /app/data
ENV CONFIG_PATH=/app/data/config.json

EXPOSE 8080
VOLUME /app/data

CMD ["./kiro-go"]
