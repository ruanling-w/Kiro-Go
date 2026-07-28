import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'node:path'

// Admin SPA is served by the Go backend under /admin/, so asset paths must be
// prefixed accordingly. The public check page lives at /check but its assets are
// emitted into the same /admin/ asset dir (base) — the Go static handler maps
// /admin/* to web/dist and serves check.html at /check. This project lives at
// web/frontend, so build output goes one level up into web/dist.
export default defineConfig({
  base: '/admin/',
  plugins: [react(), tailwindcss()],
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
      '/admin/api': 'http://localhost:8080',
      '/check/api': 'http://localhost:8080',
    },
  },
})
