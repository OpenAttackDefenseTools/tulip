import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    target: ['es2020']
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://flower-python:5000',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '')

      }
    }
  }
})
