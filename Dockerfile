# fe 阶段：构建 React + Vite 前端，输出到 web/dist（Vite build.outDir）
FROM node:20-alpine AS fe
WORKDIR /app/web/frontend
RUN corepack enable
COPY web/frontend/package.json web/frontend/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
COPY web/frontend/ ./
RUN pnpm build

# builder 阶段始终运行在构建机原生平台（amd64），用 Go 交叉编译目标平台二进制
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o kiro-go .

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/kiro-go .
COPY --from=builder /app/web ./web
# 用前端构建产物覆盖 web/dist（builder 阶段的 COPY . . 不包含被 .dockerignore 忽略的 dist）
COPY --from=fe /app/web/dist ./web/dist
RUN mkdir -p /app/data

EXPOSE 8080
VOLUME /app/data

CMD ["./kiro-go"]
