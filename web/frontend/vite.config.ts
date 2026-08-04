import { defineConfig, type PluginOption } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'
import fs from 'node:fs'
import type { IncomingMessage, ServerResponse } from 'node:http'

// Dev-only: the app is served under base /admin/, so hitting the dev server root
// shows Vite's "did you mean /admin/?" notice. Redirect / (and other bare paths)
// to /admin/ so a plain localhost:3008 lands on the admin SPA.
//
// Public check-key UI must stay OUTSIDE /admin — production Go serves it at
// /check and /check/key. Mirror that in dev so bookmarks and docs match.
function publicRoutes(): PluginOption {
  const CHECK_PATHS = new Set(['/check', '/check/', '/check/key', '/check/key/'])
  // Legacy / mistaken admin-prefixed URLs → canonical public path.
  const LEGACY_ADMIN_CHECK = new Set([
    '/admin/check',
    '/admin/check/',
    '/admin/check.html',
    '/admin/check/key',
    '/admin/check/key/',
    '/check.html',
  ])

  function pathOnly(url: string): string {
    return url.split('?')[0] ?? url
  }

  function sendCheckHtml(
    server: { transformIndexHtml: (url: string, html: string) => Promise<string> },
    res: ServerResponse,
    next: (err?: unknown) => void,
  ) {
    const file = resolve(__dirname, 'check.html')
    fs.readFile(file, 'utf-8', (err, html) => {
      if (err) {
        next(err)
        return
      }
      // Transform as /check.html so Vite rewrites module URLs against base=/admin/
      // (shared assets still live under /admin/assets in both dev and prod).
      void server
        .transformIndexHtml('/check.html', html)
        .then((transformed) => {
          res.statusCode = 200
          res.setHeader('Content-Type', 'text/html')
          res.end(transformed)
        })
        .catch(next)
    })
  }

  return {
    name: 'public-check-and-admin-root',
    apply: 'serve',
    configureServer(server) {
      // Run first so /admin/check never falls through to the admin SPA.
      server.middlewares.use((req: IncomingMessage, res: ServerResponse, next) => {
        const path = pathOnly(req.url ?? '/')

        if (path === '/' || path === '/index.html') {
          // Trailing slash is required: base is '/admin/', so Vite's transform
          // middleware does not serve a bare '/admin' (it 404s).
          res.writeHead(302, { Location: '/admin/' })
          res.end()
          return
        }

        if (LEGACY_ADMIN_CHECK.has(path)) {
          res.writeHead(302, { Location: '/check/key' })
          res.end()
          return
        }

        if (CHECK_PATHS.has(path)) {
          sendCheckHtml(server, res, next)
          return
        }

        next()
      })
    },
  }
}

// Admin SPA is served by the Go backend under /admin/, so asset paths must be
// prefixed accordingly. The public check page lives at /check/key but its JS/CSS
// are emitted into the same /admin/ asset dir (base) — the Go static handler maps
// /admin/* to web/dist and serves check.html at /check and /check/key. This
// project lives at web/frontend, so build output goes one level up into web/dist.
export default defineConfig({
  base: '/admin/',
  plugins: [react(), tailwindcss(), publicRoutes()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  build: {
    outDir: '../dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        check: resolve(__dirname, 'check.html'),
      },
    },
  },
  server: {
    port: 3008,
    // Fail loudly instead of silently sliding to 3009 — scripts and bookmarks
    // that assume 3008 would otherwise hit nothing.
    strictPort: true,
    proxy: {
      // Explicit config (not the string shorthand) so SSE (/admin/api/logs/stream)
      // isn't buffered: changeOrigin fixes the Host header and http-proxy streams
      // text/event-stream through without waiting for the response to close.
      '/admin/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/check/api': { target: 'http://localhost:8080', changeOrigin: true },
      // Public OpenAI/Anthropic-compatible surface used by the API docs page
      // (model catalog, curl samples). Same-origin in dev so no CORS dance.
      '/v1': { target: 'http://localhost:8080', changeOrigin: true },
      '/health': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
