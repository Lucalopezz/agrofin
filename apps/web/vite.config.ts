import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    // O navegador usa a mesma origem; só o Vite conhece o endereço interno do Go.
    proxy: { '/api': { target: process.env.API_PROXY_TARGET ?? 'http://localhost:8080' } },
  },
})
