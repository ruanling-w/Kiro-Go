// AppShell — the authed layout frame: fixed Sidebar (desktop) / Sheet nav
// (mobile) + Topbar, scrollable routed content via <Outlet />.
import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Sidebar, SidebarBrand, SidebarNav } from './Sidebar'
import { Topbar } from './Topbar'
import { MotionPage } from '@/components/ui/animate/MotionPage'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

export function AppShell() {
  const location = useLocation()
  const { t } = useTranslation()
  const [navOpen, setNavOpen] = useState(false)

  // Close the drawer on route change (back/forward or programmatic navigate).
  useEffect(() => {
    setNavOpen(false)
  }, [location.pathname])

  const isChat = location.pathname === '/chat'

  return (
    <div className="flex h-dvh overflow-hidden bg-background text-foreground">
      <Sidebar />

      <Sheet open={navOpen} onOpenChange={setNavOpen}>
        <SheetContent side="left" className="w-[min(20rem,calc(100%-2.5rem))] gap-0 bg-sidebar p-0">
          <SheetHeader className="sr-only">
            <SheetTitle>{t('nav.menu')}</SheetTitle>
          </SheetHeader>
          <SidebarBrand />
          <SidebarNav animated={false} onNavigate={() => setNavOpen(false)} />
        </SheetContent>
      </Sheet>

      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar onOpenNav={() => setNavOpen(true)} />
        <main className={isChat ? 'flex-1 overflow-hidden p-3 md:p-4' : 'flex-1 overflow-auto p-4 md:p-6'}>
          {isChat ? <Outlet /> : <MotionPage key={location.pathname}><Outlet /></MotionPage>}
        </main>
      </div>
    </div>
  )
}
