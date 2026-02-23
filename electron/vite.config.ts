import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  base: './',
  root: 'src/renderer',
  build: {
    outDir: '../../dist/renderer',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('monaco-editor') || id.includes('@monaco-editor')) {
              return 'monaco';
            }
            if (id.includes('@xyflow') || id.includes('dagre')) {
              return 'react-flow';
            }
            if (id.includes('js-yaml') || id.includes('marked')) {
              return 'utils';
            }
            if (id.includes('react-dom') || id.includes('/react/')) {
              return 'react-vendor';
            }
          }
        }
      }
    }
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src/renderer'),
    },
  },
  server: {
    port: 5173,
  },
})
