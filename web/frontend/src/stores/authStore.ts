// authStore — admin auth state for the cookie-session model.
//
// Unlike the old app (which held the raw admin password in localStorage and
// attached it as X-Admin-Password on every request), this store holds NO
// password after login. The backend issues an HttpOnly session cookie on
// POST /admin/api/login; the browser sends it automatically. We only keep:
//   - authed:  whether we believe a valid session exists (UI gate only; the
//              server is the real authority and returns 401 when it disagrees).
//   - csrfToken: the double-submit CSRF value read from the non-HttpOnly
//                kiro_csrf cookie, echoed back in X-CSRF-Token on mutations.
//
// The httpClient reads csrfToken from here; a 401 anywhere calls clearAuth().
import { create } from 'zustand'

// Read the CSRF token the backend set as a readable (non-HttpOnly) cookie.
export function readCsrfCookie(): string {
  const m = document.cookie.match(/(?:^|;\s*)kiro_csrf=([^;]+)/)
  return m ? decodeURIComponent(m[1]) : ''
}

interface AuthState {
  authed: boolean
  csrfToken: string
  setAuthed: (authed: boolean) => void
  syncCsrf: () => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  authed: false,
  csrfToken: readCsrfCookie(),
  setAuthed: (authed) => set({ authed, csrfToken: readCsrfCookie() }),
  syncCsrf: () => set({ csrfToken: readCsrfCookie() }),
  clearAuth: () => set({ authed: false, csrfToken: '' }),
}))

// Non-hook accessor for the CSRF token, used by the framework-free httpClient.
export function getCsrfToken(): string {
  return useAuthStore.getState().csrfToken || readCsrfCookie()
}

// Non-hook trigger so httpClient can force logout on a 401 without importing React.
export function forceLogout(): void {
  useAuthStore.getState().clearAuth()
}
