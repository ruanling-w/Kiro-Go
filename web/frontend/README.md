# Kiro-Go Admin FE

Giao diện quản trị cho Kiro-Go, viết bằng **React 19 + Vite + TypeScript**. Build ra `web/dist/` và được backend Go phục vụ tĩnh dưới đường dẫn `/admin/`.

- Quy tắc codebase: xem [AGENT.md](./AGENT.md) (bắt buộc đọc trước khi code).
- Tech stack: React 19, Vite 8, TypeScript strict, TanStack Query, Tailwind v4 + shadcn/ui, Zustand, react-i18next, React Router, Recharts, xterm.js, sonner.

## Yêu cầu

- Node.js 20+
- pnpm (bật qua `corepack enable`)
- Backend Go chạy sẵn ở `http://localhost:8080` (để dev proxy gọi API)

## Cài đặt

```bash
cd web/frontend
corepack enable          # nếu chưa bật pnpm
pnpm install --frozen-lockfile
```

## Chạy dev

Vite dev server có HMR, tự proxy API sang backend Go (xem `server.proxy` trong `vite.config.ts`).

```bash
# Terminal 1 — backend Go (từ gốc repo)
go run .

# Terminal 2 — frontend
cd web/frontend
pnpm dev
```

Mở `http://localhost:3008` (port cố định, xem `server.port` trong vite.config.ts) → redirect sang `/admin/`. Trang public check key: `http://localhost:3008/check/key`. Các request `/admin/api` và `/check/api` được proxy sang `http://localhost:8080`, nên auth cookie-session + CSRF hoạt động như production.

## Các lệnh

| Lệnh | Việc |
|---|---|
| `pnpm dev` | Dev server + HMR |
| `pnpm build` | `tsc -b` rồi `vite build` → xuất ra `web/dist/` |
| `pnpm typecheck` | Chỉ chạy `tsc -b` (không emit) |
| `pnpm lint` | Chạy oxlint |
| `pnpm preview` | Xem thử bản build production tại chỗ |

## Build production

```bash
cd web/frontend
pnpm build
```

Output đi vào `../dist` (tức `web/dist/`) — gồm `index.html` (admin SPA) và `check.html` (trang public), cùng thư mục `assets/`. Backend Go map `/admin/*` → `web/dist` và phục vụ `check.html` ở `/check` + `/check/key` (canonical), có SPA fallback cho route admin không có phần mở rộng.

Không cần chỉnh gì phía Go sau khi build; chỉ cần `web/dist/` được cập nhật.

## Docker

`Dockerfile` (ở gốc repo) có stage `fe` tự chạy `pnpm build` trong `web/frontend/` rồi copy `web/dist` vào image cuối. Không cần build FE thủ công trước khi `docker build`.

## Cấu trúc

```
web/frontend/
├── index.html          # entry admin SPA (/admin/)
├── check.html          # entry trang public check key (/check/key)
├── vite.config.ts      # base /admin/, outDir ../dist, dev proxy
├── locales/            # vi / en / zh (i18n)
├── public/             # asset tĩnh copy nguyên
└── src/
    ├── router.tsx      # route + AuthGuard + AppShell
    ├── services/       # gọi API (axios) — nơi DUY NHẤT chứa URL
    ├── hooks/          # queries + mutations (TanStack Query)
    ├── stores/         # Zustand — chỉ UI state
    ├── components/     # shared / ui / animate
    ├── features/       # dashboard, accounts, providers, apikeys, settings, logs, auth-modals
    ├── config/         # queryKeys, providers, regions
    ├── lib/            # format, mask, chartColors, cn, t
    └── check/          # trang public check key
```

Kiến trúc phân tầng bắt buộc: `component → hook → service → httpClient`. Chi tiết trong [AGENT.md](./AGENT.md).
