// AuthGuard — gates the authed app behind a valid session.
//
// On boot it probes GET /status (useSessionProbe): while pending it shows the
// full-page loader; if the session is valid it renders the app, otherwise the
// LoginPage. authStore.authed is the live signal — a 401 anywhere flips it off
// (httpClient → forceLogout) and this re-renders back to login without a reload.
import type { ReactNode } from 'react'
import { useSessionProbe } from '@/hooks/useAuth'
import { useAuthStore } from '@/stores/authStore'
import { FullPageLoader } from '@/components/shared/HamsterLoader'
import { LoginPage } from './LoginPage'

export function AuthGuard({ children }: { children: ReactNode }) {
  const probe = useSessionProbe()
  const authed = useAuthStore((s) => s.authed)

  if (probe.isPending) return <FullPageLoader />
  if (!authed) return <LoginPage />
  return <>{children}</>
}
