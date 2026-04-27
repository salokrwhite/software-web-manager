import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  // Vite 在构建时会自动处理 process，这里不需要类型定义
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase = env.VITE_API_BASE || 'http://localhost:8080'

  return {
    plugins: [react()],
    server: {
      port: 5173,
      proxy: {
        '^/api/ws(?:/|$)': {
          target: apiBase.replace(/^http/, 'ws'),
          ws: true
        },
        '^/api(?:/|$)': {
          target: apiBase,
          changeOrigin: true
        }
      }
    }
  }
})
