import { defineConfig, type PluginOption } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'

// Dev-only: the app is served under base /admin/, so hitting the dev server root
// shows Vite's "did you mean /admin/?" notice. Redirect / (and other non-prefixed
// paths) to /admin/ so a plain localhost:3008 or a reload lands on the app.
function redirectRootToAdmin(): PluginOption {
  return {
    name: 'redirect-root-to-admin',
    apply: 'serve',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const url = req.url ?? '/'
        if (url === '/' || url === '/index.html') {
          // Trailing slash is required: base is '/admin/', so Vite's transform
          // middleware does not serve a bare '/admin' (it 404s).
          res.writeHead(302, { Location: '/admin/' })
          res.end()
          return
        }
        next()
      })
    },
  }
}

// Admin SPA is served by the Go backend under /admin/, so asset paths must be
// prefixed accordingly. The public check page lives at /check but its assets are
// emitted into the same /admin/ asset dir (base) — the Go static handler maps
// /admin/* to web/dist and serves check.html at /check. This project lives at
// web/frontend, so build output goes one level up into web/dist.
export default defineConfig({
  base: '/admin/',
  plugins: [react(), tailwindcss(), redirectRootToAdmin()],
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
