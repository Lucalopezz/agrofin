import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  define: { 'import.meta.env.VITE_API_PREFIX': JSON.stringify('/api/v1') },
  test: { environment: 'jsdom', setupFiles: ['./src/test-setup.ts'] },
})
