import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    // LiveKit is isolated in the lazy meeting route; keep the home route below
    // Vite's default 500 kB budget while accepting the on-demand media chunk.
    chunkSizeWarningLimit: 650,
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    exclude: ['e2e/**', 'node_modules/**'],
    restoreMocks: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['src/hooks/useAuth.ts', 'src/hooks/useMeetingSocket.ts', 'src/services/api.ts'],
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
})
