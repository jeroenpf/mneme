import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// Default target is the TLS endpoint. Cert validation stays ON: Node does
// not read the macOS Keychain, so `make dev` passes the mkcert root CA via
// NODE_EXTRA_CA_CERTS. Escape hatch: VITE_API_TARGET=http://localhost:8080.
const API_TARGET = process.env.VITE_API_TARGET ?? 'https://mneme.dev:8443'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // 5173–5175 are claimed by the parallel TradeGod project.
    // Mneme uses 5273 so both can run side-by-side.
    port: 5273,
    strictPort: true,
    proxy: {
      '/api': { target: API_TARGET, changeOrigin: true },
      '/health': { target: API_TARGET, changeOrigin: true },
    },
  },
})
