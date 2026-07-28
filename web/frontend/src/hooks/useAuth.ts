// useAuth — session lifecycle bound to the cookie-session backend.
//
// - useSessionProbe(): on mount, verifies whether a session cookie is still
//   valid (GET /status). Drives the initial "authed?" gate on app boot/reload.
// - useLogin(): POST /login, then flips authStore.authed + syncs the CSRF token.
// - useLogout(): POST /logout, then clears auth state.
//
// The store's `authed` is only a UI hint; the server is the source of truth and
// returns 401 when it disagrees (httpClient forces logout on any 401).
import { useMutation, useQuery } from '@tanstack/react-query'
import { login as loginReq, logout as logoutReq, verifySession } from '@/services/auth.service'
import { useAuthStore } from '@/stores/authStore'

export function useSessionProbe() {
  const setAuthed = useAuthStore((s) => s.setAuthed)
  return useQuery({
    queryKey: ['session'],
    queryFn: async () => {
      try {
        await verifySession()
        setAuthed(true)
        return true
      } catch {
        setAuthed(false)
        return false
      }
    },
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: Infinity,
  })
}

export function useLogin() {
  const setAuthed = useAuthStore((s) => s.setAuthed)
  return useMutation({
    mutationFn: (password: string) => loginReq(password),
    onSuccess: () => setAuthed(true),
  })
}

export function useLogout() {
  const clearAuth = useAuthStore((s) => s.clearAuth)
  return useMutation({
    mutationFn: () => logoutReq(),
    onSettled: () => clearAuth(),
  })
}
