import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClientProvider } from '@tanstack/react-query'
import '../index.css'
import '../lib/i18n'
import { queryClient } from '@/lib/queryClient'
import { initTheme } from '@/stores/uiStore'
import { Toaster } from '@/components/ui/sonner'
import { CheckPage } from './CheckPage'

initTheme()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <CheckPage />
      <Toaster position="top-right" richColors closeButton />
    </QueryClientProvider>
  </StrictMode>,
)
