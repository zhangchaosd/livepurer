import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  server: { proxy: { '/api': 'http://127.0.0.1:8800' } },
  plugins: [vue()],
  resolve: { alias: { vue: 'vue/dist/vue.esm-bundler.js' } },
  build: { outDir: '../app/server/internal/frontend/assets', emptyOutDir: true }
})
