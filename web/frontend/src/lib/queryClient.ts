import { QueryClient } from '@tanstack/react-query'

// Single app-wide client. Defaults tuned for an admin panel that polls a few
// endpoints: retries are cheap-but-bounded, and refetch-on-focus is off so
// switching tabs doesn't hammer the Go backend.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5_000,
    },
  },
})
