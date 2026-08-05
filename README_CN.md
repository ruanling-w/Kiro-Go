# Kiro Proxy (Kiro-Go)

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![npm](https://img.shields.io/npm/v/proxy-kiro?logo=npm)](https://www.npmjs.com/package/proxy-kiro)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

将多个 AI 上游（Kiro / Claude、Grok、Codex、Antigravity、Remote Kiro）统一成 **Anthropic + OpenAI** 兼容网关，自带 Web 管理后台、API Key、多模型 Combo、实时日志。

| | |
|---|---|
| **作者** | **Vũ Trường（武长）** |
| **GitHub** | [@vtruong2k3](https://github.com/vtruong2k3) · [Kiro-Go](https://github.com/vtruong2k3/Kiro-Go) |
| **npm** | [`proxy-kiro`](https://www.npmjs.com/package/proxy-kiro) → CLI 命令 `kiroproxy` |
| **版本** | `1.1.5` |

[English README](README.md) | 中文

---

## 功能特性

- Anthropic `/v1/messages`、OpenAI `/v1/chat/completions` 与 `/v1/responses`
- 多账号池、同模型重试、配额冷却
- **Combo**：`fallback` / `round-robin` / `fusion` 多模型路由
- Web 管理台 `/admin`、API Key、RPM 限制、公开查 Key `/check/key`
- SSE 流式、Thinking 模式、出站 SOCKS5/HTTP 代理
- `npm i -g proxy-kiro` 一键安装（二进制内嵌前端）

---

## 快速开始（npm）

```bash
npm install -g proxy-kiro
kiroproxy
```

- 管理台：http://localhost:8080/admin
- 默认密码：`changeme`（请立即修改）
- 数据目录：`~/.kiroproxy/`

### Docker

```bash
docker run -d -p 8080:8080 \
  -e ADMIN_PASSWORD=your_password \
  -v /path/to/data:/app/data \
  ghcr.io/vtruong2k3/kiro-go:latest
```

### 源码构建

```bash
git clone https://github.com/vtruong2k3/Kiro-Go.git
cd Kiro-Go
pnpm --dir web/frontend install && pnpm --dir web/frontend build
go build -o kiro-go .
./kiro-go
```

---

## 客户端配置

```bash
# Claude / Anthropic 兼容
export ANTHROPIC_BASE_URL=http://localhost:8080
export ANTHROPIC_AUTH_TOKEN=sk-你的密钥

# OpenAI 兼容
export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-你的密钥
```

密钥在管理台 **API Keys** 中创建。

---

## CLI

```text
kiroproxy [--port N] [--host ADDR] [--config PATH] [--no-open] [--version]
```

端口默认 `8080`；被占用时自动尝试后续端口。

---

## 主要接口

| 路径 | 说明 |
|------|------|
| `POST /v1/messages` | Anthropic |
| `POST /v1/chat/completions` | OpenAI Chat |
| `POST /v1/responses` | OpenAI Responses |
| `GET /v1/models` | 模型列表 |
| `GET /admin` | 管理后台 |
| `GET /check/key` | 公钥用量查询 |
| `GET /health` | 健康检查 |

---

## Combo 多模型

在管理台创建 Combo，客户端使用 **Combo 名称** 作为 `model`：

- **fallback**：按顺序切换模型
- **round-robin**：轮询
- **fusion**：多路并行 + judge

直接请求某个模型时，仅在同模型账号池内重试（最多 3 个账号），不会自动跨模型跳转。

---

## 环境变量

| 变量 | 说明 |
|------|------|
| `CONFIG_PATH` | 配置文件路径 |
| `ADMIN_PASSWORD` | 覆盖管理员密码 |
| `LOG_LEVEL` | 日志级别 |
| `RUNTIME_DB_PATH` | SQLite 路径 |

配置解析顺序：`--config` / `CONFIG_PATH` → 已有 `./data/` → `~/.kiroproxy/config.json`。

---

## 数据目录

```text
~/.kiroproxy/
  bin/kiro-go
  config.json
  kiro-runtime.db
```

---

## 作者

**Vũ Trường** · 维护与发布  
GitHub: [vtruong2k3/Kiro-Go](https://github.com/vtruong2k3/Kiro-Go)  
npm: [proxy-kiro](https://www.npmjs.com/package/proxy-kiro)

---

## 免责声明

仅供学习与研究。与 Amazon / AWS / Kiro / Anthropic / OpenAI / xAI / Google 无关联。请遵守当地法律与各平台服务条款。

## License

[MIT](LICENSE)
