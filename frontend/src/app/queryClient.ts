import { QueryClient } from '@tanstack/react-query'

/**
 * Shared query client. Defaults are tuned for an internal admin app: data is
 * fresh enough for 30s (avoids refetch storms while clicking around), and a
 * failed request is not retried, because the API's 4xx responses are
 * deliberate answers (403/409/422) rather than transient faults — retrying
 * them only delays showing the user the real message.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: false,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
    },
  },
})
