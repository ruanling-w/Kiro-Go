// AppShell — the authed layout frame: fixed Sidebar + Topbar, scrollable routed
// content via <Outlet />. Rendered inside the guarded route tree. Each route's
// content fades+rises in via MotionPage, keyed on pathname so switching pages
// re-triggers the transition.
import { Outlet, useLocation } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { Topbar } from './Topbar'
import { MotionPage } from '@/components/ui/animate/MotionPage'

export function AppShell() {
  const location = useLocation()
  return (
    <div className="flex h-dvh overflow-hidden bg-background text-foreground">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar />
        <main className="flex-1 overflow-auto p-6">
          <MotionPage key={location.pathname}>
            <Outlet />
          </MotionPage>
        </main>
      </div>
    </div>
  )
}
