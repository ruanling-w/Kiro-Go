// httpClient — the single entry point for every admin API call, built on axios.
//
// Auth model (see plan "Bảo mật dashboard"): the backend issues an HttpOnly
// session cookie on login, so requests carry it automatically via
// `withCredentials`. Mutations additionally send the double-submit CSRF token in
// the `X-CSRF-Token` header, which axios reads from the readable `kiro_csrf`
// cookie (xsrfCookieName/xsrfHeaderName). No password is ever stored or sent
// after login.
//
// A 401 means the session is gone/expired. A response interceptor flips auth
// state off (forceLogout) so the guard bounces to /login, then every error is
// normalized to ApiError{status, message, code} so callers/React Query see a
// consistent shape.
//
// A 403 with code "csrf_mismatch" means the `kiro_csrf` cookie was missing or
// stale when the mutation went out, so axios could not attach the header. The
// cookie is scoped Path=/admin, SameSite=Strict and Secure behind TLS, so it can
// silently go missing after a deployment change. We re-probe /status once to let
// the server re-issue the cookie pair, then retry the request exactly once; if
// it fails again the session really is unusable, so force a logout with an
// actionable message instead of surfacing a bare 403.
import axios, { AxiosError, type AxiosRequestConfig } from 'axios'
import { forceLogout } from '@/stores/authStore'

const ADMIN_API_BASE = '/admin/api'

export class ApiError extends Error {
  status: number
  code?: string
  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export const axiosClient = axios.create({
  baseURL: ADMIN_API_BASE,
  withCredentials: true,
  // axios attaches the CSRF cookie's value as X-CSRF-Token automatically.
  xsrfCookieName: 'kiro_csrf',
  xsrfHeaderName: 'X-CSRF-Token',
  headers: { 'Content-Type': 'application/json' },
})

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null
}

/** Marks a config we've already retried, so a second 403 cannot loop. */
interface RetriableConfig extends AxiosRequestConfig {
  _csrfRetried?: boolean
}

// Normalize any axios failure into an ApiError, and force logout on 401.
axiosClient.interceptors.response.use(
  (res) => res,
  async (err: AxiosError) => {
    const status = err.response?.status ?? 0
    const data = err.response?.data
    const msg =
      (isRecord(data) && typeof data.error === 'string' && data.error) ||
      err.message ||
      'Request failed'
    const code = isRecord(data) && typeof data.code === 'string' ? data.code : undefined

    // 401 = session gone/expired. Skip the boot probe (/status) — its caller
    // interprets the failure itself instead of bouncing mid-boot.
    if (status === 401 && err.config?.url !== '/status') forceLogout()

    // 403 CSRF mismatch: the csrf cookie was absent/stale so no header went out.
    // Re-probe /status to have the server re-issue it, then retry once.
    const cfg = err.config as RetriableConfig | undefined
    if (status === 403 && code === 'csrf_mismatch' && cfg && !cfg._csrfRetried) {
      cfg._csrfRetried = true
      try {
        await axiosClient.get('/status')
        return await axiosClient.request(cfg)
      } catch {
        forceLogout()
        return Promise.reject(
          new ApiError(
            403,
            'Session security token expired. Please sign in again.',
            'csrf_mismatch',
          ),
        )
      }
    }

    return Promise.reject(new ApiError(status, msg, code))
  },
)

export interface ApiRequestOptions extends AxiosRequestConfig {
  /** Return the raw AxiosResponse (e.g. to read headers / blobs). */
  raw?: boolean
}

/** Verb helpers. Bodies are JSON-serialized by axios; T is the parsed payload. */
export const http = {
  get: <T>(path: string, opts?: ApiRequestOptions) =>
    axiosClient.get<T>(path, opts).then((r) => r.data),
  post: <T>(path: string, body?: unknown, opts?: ApiRequestOptions) =>
    axiosClient.post<T>(path, body, opts).then((r) => r.data),
  put: <T>(path: string, body?: unknown, opts?: ApiRequestOptions) =>
    axiosClient.put<T>(path, body, opts).then((r) => r.data),
  delete: <T>(path: string, opts?: ApiRequestOptions) =>
    axiosClient.delete<T>(path, opts).then((r) => r.data),
}
