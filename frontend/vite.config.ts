import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  // Pre-bundle the reports chart library so a cold /reports visit does not race
  // Vite's on-demand optimizer (which surfaces as "Failed to fetch dynamically
  // imported module: …/Reports.tsx").
  optimizeDeps: {
    include: ['recharts'],
  },
  server: {
    // Match the Docker nginx proxy so the SPA always uses relative /api/v1.
    // Override target when the API is not on :8080 (e.g. scripts/dev-up.sh).
    proxy: {
      '/api': {
        target: process.env.API_PROXY_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
