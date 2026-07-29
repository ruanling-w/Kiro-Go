import { defineConfig, type PluginOption } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'

// Dev-only: the app is served under base /admin/, so hitting the dev server root
// shows Vite's "did you mean /admin/?" notice. Redirect / (and other non-prefixed
// paths) to /admin/ so a plain localhost:5173 or a reload lands on the app.
function redirectRootToAdmin(): PluginOption {
  return {
    name: 'redirect-root-to-admin',
    apply: 'serve',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        const url = req.url ?? '/'
        if (url === '/' || url === '/index.html') {
          res.writeHead(302, { Location: '/admin' })
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
    proxy: {
      // Explicit config (not the string shorthand) so SSE (/admin/api/logs/stream)
      // isn't buffered: changeOrigin fixes the Host header and http-proxy streams
      // text/event-stream through without waiting for the response to close.
      '/admin/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/check/api': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
