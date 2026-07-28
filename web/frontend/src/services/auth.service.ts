// auth.service — admin session lifecycle against the cookie-session backend.
//
// login:  POST /admin/api/login {password} → backend sets the HttpOnly session
//         cookie + readable CSRF cookie. No token is returned in the body.
// logout: POST /admin/api/logout → backend destroys the session and clears cookies.
// verify: GET /admin/api/status → cheap authenticated probe used to confirm an
//         existing session is still valid (e.g. on app boot / reload).
import { http } from './httpClient'
import type { StatusSnapshot } from '@/types/common'

export interface LoginResponse {
  success: boolean
}

export function login(password: string): Promise<LoginResponse> {
  return http.post<LoginResponse>('/login', { password })
}

export function logout(): Promise<{ success: boolean }> {
  return http.post<{ success: boolean }>('/logout')
}

/** Probes an authenticated endpoint; resolves the status when the session is valid. */
export function verifySession(): Promise<StatusSnapshot> {
  return http.get<StatusSnapshot>('/status')
}
