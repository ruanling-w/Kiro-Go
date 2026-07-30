// Router — the guarded app shell wrapping the 6 routed pages. Login lives
// outside the shell (rendered by AuthGuard when there's no session). Pages are
// lazy so the initial bundle stays lean (Logs pulls in xterm; charts pull in
// recharts) — Suspense falls back to the shared hamster loader.
import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AuthGuard } from '@/features/login/AuthGuard'
import { AppShell } from '@/components/layout/AppShell'
import { FullPageLoader } from '@/components/shared/HamsterLoader'

const DashboardPage = lazy(() => import('@/features/dashboard/DashboardPage'))
const ProvidersPage = lazy(() => import('@/features/providers/ProvidersPage'))
const AccountsPage = lazy(() => import('@/features/accounts/AccountsPage'))
const ApiKeysPage = lazy(() => import('@/features/apikeys/ApiKeysPage'))
const CombosPage = lazy(() => import('@/features/combos/CombosPage'))
const SettingsPage = lazy(() => import('@/features/settings/SettingsPage'))
const LogsPage = lazy(() => import('@/features/logs/LogsPage'))

function Lazy({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<FullPageLoader />}>{children}</Suspense>
}

export const router = createBrowserRouter(
  [
    {
      path: '/',
      element: (
        <AuthGuard>
          <AppShell />
        </AuthGuard>
      ),
      children: [
        { index: true, element: <Lazy><DashboardPage /></Lazy> },
        { path: 'providers', element: <Lazy><ProvidersPage /></Lazy> },
        { path: 'providers/:provider', element: <Lazy><AccountsPage /></Lazy> },
        { path: 'accounts', element: <Lazy><AccountsPage /></Lazy> },
        { path: 'api-keys', element: <Lazy><ApiKeysPage /></Lazy> },
        { path: 'combos', element: <Lazy><CombosPage /></Lazy> },
        { path: 'settings', element: <Lazy><SettingsPage /></Lazy> },
        { path: 'logs', element: <Lazy><LogsPage /></Lazy> },
        { path: '*', element: <Navigate to="/" replace /> },
      ],
    },
  ],
  { basename: '/admin' },
)
