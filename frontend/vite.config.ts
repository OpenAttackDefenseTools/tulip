import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig(({ command, mode }) => {


  const env = loadEnv(mode, process.cwd(), '')

  return ({
    plugins: [react()],
    build: {
      target: ['es2020']
    },
    server: {
      proxy: {
        '/api': {
          target: env["API_SERVER_ENDPOINT"] ?? "http://localhost:5000/",
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        }
      }
    }
  })
})
