import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  // Lê apenas API_PREFIX do .env da raiz; o restante, inclusive segredos, não é exposto ao frontend.
  const rootEnv = loadEnv(mode, fileURLToPath(new URL('../../', import.meta.url)), 'API_PREFIX')
  const apiPrefix = (process.env.API_PREFIX || rootEnv.API_PREFIX || '/api/v1').replace(/\/$/, '')

  return {
    plugins: [react(), tailwindcss()],
    resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
    define: { 'import.meta.env.VITE_API_PREFIX': JSON.stringify(apiPrefix) },
    server: {
      host: '0.0.0.0',
      port: 5173,
      strictPort: true,
      // O navegador usa a mesma origem; só o Vite conhece o endereço interno do Go.
      proxy: { [apiPrefix]: { target: process.env.API_PROXY_TARGET ?? 'http://localhost:8080' } },
    },
  }
})
