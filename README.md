# Kiro Proxy (Kiro-Go)

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![npm](https://img.shields.io/npm/v/proxy-kiro?logo=npm)](https://www.npmjs.com/package/proxy-kiro)
[![Release](https://img.shields.io/github/v/release/vtruong2k3/Kiro-Go?logo=github)](https://github.com/vtruong2k3/Kiro-Go/releases)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Multi-provider AI gateway** — một server, nhiều upstream (Kiro / Claude, Grok, Codex, Antigravity/Gemini, Remote Kiro), API tương thích **Anthropic** + **OpenAI**, có admin web, API key, combo multi-model, log realtime.

| | |
|---|---|
| **Tác giả** | **Vũ Trường** |
| **GitHub** | [@vtruong2k3](https://github.com/vtruong2k3) · [Kiro-Go](https://github.com/vtruong2k3/Kiro-Go) |
| **npm** | [`proxy-kiro`](https://www.npmjs.com/package/proxy-kiro) → lệnh CLI `kiroproxy` |
| **Phiên bản** | `1.1.6` |

[English](#kiro-proxy-kiro-go) · tài liệu này viết song ngữ ngắn gọn EN/VI trong cùng file.

---

## Mục lục

1. [Tính năng](#tính-năng)
2. [Cài đặt nhanh (npm)](#cài-đặt-nhanh-npm)
3. [Docker](#docker)
4. [Build từ source](#build-từ-source)
5. [Lần đầu chạy](#lần-đầu-chạy)
6. [CLI `kiroproxy`](#cli-kiroproxy)
7. [API endpoints](#api-endpoints)
8. [Dùng với Claude Code / client](#dùng-với-claude-code--client)
9. [Providers & tài khoản](#providers--tài-khoản)
10. [API Keys](#api-keys)
11. [Combos (multi-model)](#combos-multi-model)
12. [Thinking mode](#thinking-mode)
13. [Outbound proxy](#outbound-proxy)
14. [Cấu hình & biến môi trường](#cấu-hình--biến-môi-trường)
15. [Thư mục dữ liệu](#thư-mục-dữ-liệu)
16. [Release & cập nhật](#release--cập-nhật)
17. [Deploy Zeabur](#deploy-zeabur)
18. [Cấu trúc repo](#cấu-trúc-repo)
19. [Đóng góp](#đóng-góp)
20. [Disclaimer & License](#disclaimer--license)

---

## Tính năng

- **API tương thích**
  - Anthropic: `POST /v1/messages`
  - OpenAI Chat: `POST /v1/chat/completions`
  - OpenAI Responses: `POST /v1/responses`
  - Models: `GET /v1/models`
  - Health: `GET /health`
- **Nhiều provider upstream**
  - Kiro (Builder ID / SSO / API key)
  - Grok (xAI)
  - Codex (OpenAI-compatible)
  - Antigravity (Gemini)
  - Remote Kiro (proxy peer `sk-…`)
- **Account pool** — round-robin, retry cùng model (tối đa 3 account), cooldown khi quota/lỗi
- **Combos** — chuỗi multi-model do bạn định nghĩa: `fallback` · `round-robin` · `fusion`
- **Admin web** (`/admin`) — accounts, API keys, combos, logs SSE, settings
- **Admin Chat** (`/admin/chat`) — persisted text streaming, secure image upload/vision, image generation, stop/retry, gallery và export; xem [hướng dẫn vận hành](docs/admin-chat.md)
- **API key** — tạo key `sk-…`, RPM limit, tra cứu public `/check/key`
- **Streaming SSE**, thinking mode, outbound SOCKS5/HTTP
- **Portable binary** — SPA admin embed trong binary; cài global không cần clone repo

---

## Cài đặt nhanh (npm)

Yêu cầu: **Node.js ≥ 18**, Linux / macOS / Windows (x64 hoặc arm64).

```bash
npm install -g proxy-kiro
kiroproxy
```

- Mở admin: **http://localhost:8080/admin**
- Mật khẩu mặc định: `changeme` → đổi ngay trong Settings
- State: `~/.kiroproxy/` (config + DB + binary)

Gỡ:

```bash
npm uninstall -g proxy-kiro
```

> Tên package npm là **`proxy-kiro`** (unscoped). Lệnh chạy vẫn là **`kiroproxy`**.

---

## Docker

### Docker Compose

```bash
git clone https://github.com/vtruong2k3/Kiro-Go.git
cd Kiro-Go
mkdir -p data
# tạo .env với ADMIN_PASSWORD=...
docker compose up -d --build
```

### Docker Run

```bash
docker run -d \
  --name kiro-go \
  -p 8080:8080 \
  -e ADMIN_PASSWORD='your_secure_password' \
  -e CONFIG_PATH=/app/data/config.json \
  -v /path/to/data:/app/data \
  --restart unless-stopped \
  ghcr.io/vtruong2k3/kiro-go:latest
```

Binary trong image đã **embed** admin UI; volume `/app/data` giữ accounts & config.

---

## Build từ source

```bash
git clone https://github.com/vtruong2k3/Kiro-Go.git
cd Kiro-Go

# 1) Frontend (bắt buộc trước go build — //go:embed web/dist)
pnpm --dir web/frontend install
pnpm --dir web/frontend build

# 2) Backend
go build -ldflags="-s -w" -o kiro-go .

# 3) Chạy
./kiro-go
# hoặc
./kiro-go --port 8080 --config ./data/config.json
```

Yêu cầu toolchain: **Go 1.23+**, **Node 22 / pnpm 9** (chỉ khi build FE).

---

## Lần đầu chạy

1. Mở **http://localhost:8080/admin**
2. Đăng nhập (`changeme` hoặc `ADMIN_PASSWORD`)
3. **Accounts** → thêm provider (Kiro / Grok / …)
4. **API Keys** → tạo key `sk-…` cho client
5. (Tuỳ chọn) **Combos** → chuỗi multi-model
6. Trỏ Claude Code / OpenAI client vào gateway (mục dưới)

---

## CLI `kiroproxy`

```
kiroproxy [options]

  -p, --port <n>       Port listen (mặc định 8080; bận thì tự +1 … +10)
  -h, --host <addr>    Bind address (mặc định theo config / 0.0.0.0)
  -c, --config <path>  File config (mặc định ~/.kiroproxy/config.json)
      --no-open        Không mở browser
  -v, --version        In version
      --help           Help
```

Ví dụ:

```bash
kiroproxy --port 9090 --no-open
kiroproxy --config /etc/kiro/config.json
```

Khi cài qua npm, postinstall tải binary đúng OS/arch từ  
[GitHub Releases](https://github.com/vtruong2k3/Kiro-Go/releases) và verify **SHA256SUMS**.

Offline / air-gap:

```bash
KIROPROXY_SKIP_DOWNLOAD=1 npm i -g proxy-kiro
mkdir -p ~/.kiroproxy/bin
cp /path/to/kiro-go-linux-amd64 ~/.kiroproxy/bin/kiro-go
chmod +x ~/.kiroproxy/bin/kiro-go
echo "1.1.6" > ~/.kiroproxy/bin/.version
kiroproxy --no-open
```

---

## API endpoints

| Method | Path | Mô tả |
|--------|------|--------|
| `POST` | `/v1/messages` | Anthropic Messages (Claude) |
| `POST` | `/v1/chat/completions` | OpenAI Chat Completions |
| `POST` | `/v1/responses` | OpenAI Responses |
| `GET` | `/v1/models` | Danh sách model (+ combo nếu advertise) |
| `GET` | `/v1/stats` | Stats (cần API key) |
| `GET` | `/health` | Health check |
| `GET` | `/admin` | Admin SPA |
| `GET` | `/check/key` | Tra cứu key/quota (public UI) |

---

## Dùng với Claude Code / client

### Anthropic-compatible

```bash
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_AUTH_TOKEN=sk-your-key-from-admin   # hoặc ANTHROPIC_API_KEY
```

```bash
curl http://localhost:8080/v1/messages \
  -H "content-type: application/json" \
  -H "x-api-key: sk-your-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4.5",
    "max_tokens": 1024,
    "messages": [{"role":"user","content":"Hello"}]
  }'
```

### OpenAI-compatible

```bash
export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-your-key-from-admin
```

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "content-type: application/json" \
  -H "authorization: Bearer sk-your-key" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role":"user","content":"Hello"}]
  }'
```

Model id là **tên model upstream** hoặc **tên Combo** bạn tạo trong admin.

---

## Providers & tài khoản

Trong **Admin → Accounts / Providers**:

| Provider | Ghi chú |
|----------|---------|
| **Kiro** | Builder ID, IAM SSO, SSO token, credentials JSON, Kiro API key |
| **Grok** | xAI OAuth / key |
| **Codex** | OpenAI-compatible Codex pool |
| **Antigravity** | Google / Gemini path |
| **Remote Kiro** | Peer gateway: base URL + `sk-` key |

Direct model request chỉ retry **cùng model** trên pool account (tối đa 3). Muốn Claude → Grok khi fail → dùng **Combo** `strategy=fallback`, không còn auto ModelFallback ẩn.

---

## API Keys

- Tạo trong **Admin → API Keys**
- Format `sk-…`
- Có thể gắn **RPM limit**
- User tra cứu usage: **http://localhost:8080/check/key**

Bật bắt buộc key (nếu cấu hình `requireApiKey`) trong Settings.

---

## Combos (multi-model)

**Admin → Combos** — chuỗi model có tên, client gọi bằng **tên combo**.

| Strategy | Hành vi |
|----------|---------|
| `fallback` | Thử model 1 → hết account/fail → model 2 → … |
| `round-robin` | Xoay vòng theo sticky limit |
| `fusion` | Chạy song song panel + judge model |

Ví dụ fallback: `claude-opus-5 → grok-4.5`.  
UI hiển thị chip màu + icon provider giống cột Model trong Logs.

---

## Thinking mode

- Suffix model (mặc định `-thinking`), vd. `claude-sonnet-4.5-thinking`
- Hoặc body Claude có `thinking`: `{"type":"enabled","budget_tokens":2048}` / `{"type":"adaptive"}`
- Format output chỉnh trong **Settings → Thinking Mode**

---

## Outbound proxy

**Settings → Outbound Proxy**: SOCKS5 / HTTP. Áp dụng ngay, không cần restart — hữu ích khi upstream bị chặn theo vùng.

---

## Cấu hình & biến môi trường

| Biến | Mô tả | Mặc định |
|------|--------|----------|
| `CONFIG_PATH` | Đường dẫn `config.json` | xem [thư mục dữ liệu](#thư-mục-dữ-liệu) |
| `ADMIN_PASSWORD` | Password admin (override file) | — |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `RUNTIME_DB_PATH` | SQLite runtime DB | cạnh config |
| `KIROPROXY_SKIP_DOWNLOAD` | Bỏ tải binary lúc `npm i` | — |
| `KIROPROXY_RELEASE_BASE` | Mirror URL release assets | GitHub Releases |
| `KIROPROXY_REPO` | `owner/repo` cho download | `vtruong2k3/Kiro-Go` |

Flags binary:

```text
--config PATH   --host ADDR   --port N   --version
```

---

## Thư mục dữ liệu

Thứ tự chọn config:

1. `--config` / `CONFIG_PATH`
2. `./data/config.json` nếu đã tồn tại (Docker / checkout cũ)
3. `~/.kiroproxy/config.json` (cài npm global)

```text
~/.kiroproxy/                 # npm install
  bin/kiro-go                 # server binary
  bin/.version
  config.json                 # accounts, keys, settings
  kiro-runtime.db             # logs, combos, RPM stats

./data/                       # Docker / dev checkout
  config.json
  kiro-runtime.db
```

Backup: copy `config.json` (+ DB nếu cần lịch sử log).

---

## Release & cập nhật

- **GitHub Release:** https://github.com/vtruong2k3/Kiro-Go/releases  
  Assets: `kiro-go-{linux,darwin,windows}-{amd64,arm64}` + `SHA256SUMS`
- **npm:** `npm i -g proxy-kiro@latest` rồi chạy lại `kiroproxy` (tự tải binary khớp version)

Publish tay (không dùng Actions):

```bash
# build FE + binary các platform, upload Release + SHA256SUMS
cd cli && npm version patch && npm publish --access public
```

---

## Deploy Zeabur

1. Fork repo → Zeabur **Deploy from GitHub**
2. Expose port **8080**, gắn domain
3. Biến: `ADMIN_PASSWORD=...`
4. Volume mount **`/app/data`**

Hoặc CLI:

```bash
npm i -g zeabur
zeabur auth login
zeabur deploy   # từ root repo; đừng commit .zeabur/
```

---

## Cấu trúc repo

```text
Kiro-Go/
  main.go                 # entry, flags, embed hook
  config/                 # config.json, paths, version
  proxy/                  # HTTP handlers, providers, combos, static FS
  pool/                   # account pool
  auth/                   # OAuth / SSO helpers
  store/                  # SQLite runtime
  web/
    embed.go              # //go:embed all:dist
    dist/                 # Vite build output (admin SPA)
    frontend/             # React + Vite source
  cli/                    # npm package proxy-kiro (launcher)
  Dockerfile
  .github/workflows/      # docker + release (optional CI)
```

---

## Đóng góp

Issue / PR welcome trên [github.com/vtruong2k3/Kiro-Go](https://github.com/vtruong2k3/Kiro-Go).

Project được phát triển và duy trì bởi **Vũ Trường** ([@vtruong2k3](https://github.com/vtruong2k3)).

```text
Maintainer : Vũ Trường
npm        : proxy-kiro
CLI        : kiroproxy
```

---

## Disclaimer & License

Chỉ dùng cho mục đích học tập / nghiên cứu / hạ tầng cá nhân hợp pháp.  
**Không** liên kết với Amazon, AWS, Kiro, Anthropic, OpenAI, xAI hay Google.  
Người dùng tự chịu trách nhiệm tuân thủ ToS và pháp luật địa phương.

**License:** [MIT](LICENSE)

---

<p align="center">
  Made with ❤️ by <strong>Vũ Trường</strong> ·
  <a href="https://github.com/vtruong2k3/Kiro-Go">GitHub</a> ·
  <a href="https://www.npmjs.com/package/proxy-kiro">npm</a>
</p>
