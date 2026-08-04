// gatewayBase — resolve the public gateway origin used in CLI/SDK snippets.
// Admin UI origin ≠ gateway origin in Vite dev (3008 vs 8080).

/** Page origin of the admin SPA (Vite or Go-served). */
export function pageOrigin(): string {
  return typeof window !== 'undefined' ? window.location.origin : ''
}

/**
 * Best-guess public gateway base for client configs (Claude Code, Codex, cURL…).
 *
 * - Production: admin is served by the Go binary → same origin as the page.
 * - Vite dev: admin is on the Vite port; gateway listens on `settings.port`
 *   (default 8080). `settings.host` is a bind address (often 0.0.0.0) and must
 *   never be pasted into URLs — we keep the page hostname instead.
 */
export function resolveGatewayBaseURL(opts?: { port?: number }): string {
  if (typeof window === 'undefined') return 'http://localhost:8080'
  const { protocol, hostname } = window.location
  if (!import.meta.env.DEV) return window.location.origin

  const port = opts?.port && opts.port > 0 ? opts.port : 8080
  const host = hostname || 'localhost'
  const isDefault =
    (protocol === 'https:' && port === 443) || (protocol === 'http:' && port === 80)
  return isDefault ? `${protocol}//${host}` : `${protocol}//${host}:${port}`
}

/** True when `url` is the Vite admin origin (wrong default for snippets in dev). */
export function isDevAdminOrigin(url: string): boolean {
  if (!import.meta.env.DEV || typeof window === 'undefined' || !url) return false
  try {
    return new URL(url).origin === window.location.origin
  } catch {
    return false
  }
}

/**
 * True when the docs base points at a local gateway we can reach via the Vite
 * `/v1` proxy (same-origin fetch avoids CORS during `pnpm dev`).
 */
export function isLocalGatewayBase(url: string): boolean {
  if (!url || typeof window === 'undefined') return true
  try {
    const u = new URL(url)
    const host = u.hostname
    return (
      host === 'localhost' ||
      host === '127.0.0.1' ||
      host === '[::1]' ||
      host === window.location.hostname
    )
  } catch {
    return false
  }
}
