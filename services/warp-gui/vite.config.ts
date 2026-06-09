import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  base: './',
  root: __dirname,
  build: {
    outDir: 'build',
    emptyOutDir: true,
  },
})