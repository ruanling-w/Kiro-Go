# kiroproxy

Multi-provider AI gateway. One install, one command.

```bash
npm install -g @vtruong2k3/kiroproxy
kiroproxy
```

The admin UI opens at `http://localhost:8080/admin`. Add your accounts there.

## Endpoints

| Path | Protocol |
|---|---|
| `/v1/messages` | Anthropic Claude |
| `/v1/chat/completions` | OpenAI Chat Completions |
| `/v1/responses` | OpenAI Responses |
| `/admin` | Web admin UI |
| `/check/key` | Public key/quota lookup |

Point any Claude- or OpenAI-compatible client at the base URL and use an API key
created in the admin UI.

```bash
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_AUTH_TOKEN=sk-...   # created in /admin
```

## Options

```
-p, --port <n>       port to listen on (default 8080, next free port if busy)
-h, --host <addr>    address to bind (default 0.0.0.0)
-c, --config <path>  config file (default ~/.kiroproxy/config.json)
    --no-open        do not open the browser
-v, --version        print version
    --help           show help
```

## Where state lives

Everything is under `~/.kiroproxy/`:

```
~/.kiroproxy/
  bin/kiro-go         server binary (downloaded on install)
  config.json         accounts, API keys, settings
  kiro-runtime.db     request logs, combos, rate-limit state
```

Back up `config.json` and you have your whole setup.

## How the install works

This package is a small launcher with no runtime dependencies. On install it
downloads the Go server binary for your platform from the matching
[GitHub release](https://github.com/vtruong2k3/Kiro-Go/releases) and verifies it
against the release's `SHA256SUMS`. The binary embeds the admin UI, so nothing
else needs to be fetched at runtime.

If the download fails (offline, proxy, firewall), the install still succeeds and
the launcher retries the next time you run `kiroproxy`.

Supported: Linux, macOS and Windows on x64 and arm64.

### Behind a proxy or air-gapped

```bash
# skip the download during install, supply the binary yourself
KIROPROXY_SKIP_DOWNLOAD=1 npm i -g @vtruong2k3/kiroproxy
mkdir -p ~/.kiroproxy/bin
cp /path/to/kiro-go ~/.kiroproxy/bin/kiro-go && chmod +x ~/.kiroproxy/bin/kiro-go
echo "$(kiroproxy --version)" > ~/.kiroproxy/bin/.version
```

`KIROPROXY_RELEASE_BASE` points the downloader at an internal mirror serving the
same asset names plus `SHA256SUMS`.

## Docker

```bash
docker run -p 8080:8080 -v $PWD/data:/app/data ghcr.io/vtruong2k3/kiro-go
```

## License

MIT
