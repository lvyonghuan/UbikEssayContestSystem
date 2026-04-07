import path from 'node:path'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  server: {
    proxy: {
      '/api/admin': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        rewrite: (sourcePath) => sourcePath.replace(/^\/api\/admin/, '/api/v1'),
      },
      '/api/system': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (sourcePath) => sourcePath.replace(/^\/api\/system/, '/api/v1'),
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/tests/setup.ts'],
    exclude: ['tests/e2e/**', 'node_modules/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['src/services/**/*.ts', 'src/stores/**/*.ts'],
    },
  },
})
