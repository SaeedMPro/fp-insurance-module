import { QueryClientProvider } from '@tanstack/react-query'

import { AuthProvider } from './context/AuthContext'
import { ToastProvider } from './context/ToastContext'
import { AppRouter } from './app/router'
import { queryClient } from './app/queryClient'

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <ToastProvider>
          <AppRouter />
        </ToastProvider>
      </AuthProvider>
    </QueryClientProvider>
  )
}

export default App
