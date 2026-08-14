# proxy-kiro

> Kiro Proxy — multi-provider AI gateway by **Vũ Trường** ([@vtruong2k3](https://github.com/vtruong2k3))

Cài một lệnh, chạy một lệnh:

```bash
npm install -g proxy-kiro
kiroproxy
```

- Admin: http://localhost:8080/admin
- Mật khẩu mặc định: `changeme`
- State: `~/.kiroproxy/`

Tài liệu đầy đủ: [github.com/vtruong2k3/Kiro-Go](https://github.com/vtruong2k3/Kiro-Go#readme)

## Endpoints

| Path | Protocol |
|------|----------|
| `/v1/messages` | Anthropic Claude |
| `/v1/chat/completions` | OpenAI Chat Completions |
| `/v1/responses` | OpenAI Responses |
| `/v1/models` | Model list |
| `/admin` | Web admin |
| `/check/key` | Public key / quota lookup |
| `/health` | Health |

## Client

```bash
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_AUTH_TOKEN=sk-...   # tạo trong /admin

export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-...
```

## CLI

```
kiroproxy [options]

  -p, --port <n>       port (default 8080, next free if busy)
  -h, --host <addr>    bind address
  -c, --config <path>  config file (default ~/.kiroproxy/config.json)
      --no-open        do not open browser
  -v, --version
      --help
```

## Data directory

```text
~/.kiroproxy/
  bin/kiro-go      # server binary (downloaded from GitHub Releases)
  config.json      # accounts, API keys, settings
  kiro-runtime.db  # logs, combos, rate limits
```

## How install works

Package này chỉ là launcher (~9 KB). `postinstall` tải binary Go đúng OS/arch từ
[GitHub Releases](https://github.com/vtruong2k3/Kiro-Go/releases), verify
`SHA256SUMS`, rồi `kiroproxy` spawn server (admin UI đã embed trong binary).

Supported: Linux, macOS, Windows · x64, arm64 · Node ≥ 18.

### Offline

```bash
KIROPROXY_SKIP_DOWNLOAD=1 npm i -g proxy-kiro
mkdir -p ~/.kiroproxy/bin
cp ./kiro-go-linux-amd64 ~/.kiroproxy/bin/kiro-go && chmod +x ~/.kiroproxy/bin/kiro-go
echo "1.1.6" > ~/.kiroproxy/bin/.version
```

## Docker

```bash
docker run -p 8080:8080 \
  -e ADMIN_PASSWORD=changeme \
  -v "$PWD/data:/app/data" \
  ghcr.io/vtruong2k3/kiro-go:latest
```

## Author

**Vũ Trường** · [github.com/vtruong2k3](https://github.com/vtruong2k3)

## License

MIT
