# Optimize Bundle Size with Code Splitting

**Type:** Performance optimization

## Problem

Build output shows:

```
../../dist/renderer/assets/index-COXP_iGu.js   649.26 kB │ gzip: 203.83 kB

(!) Some chunks are larger than 500 kB after minification. Consider:
- Using dynamic import() to code-split the application
- Use build.rollupOptions.output.manualChunks to improve chunking
- Adjust chunk size limit for this warning via build.chunkSizeWarningLimit.
```

The main bundle is 649 KB which exceeds the recommended 500 KB limit.

## Solution Options

### Option 1: Manual Chunks (Recommended)

Update `vite.config.ts` to split vendor dependencies:

```typescript
export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom'],
          'monaco': ['@monaco-editor/react'],
          'react-flow': ['@xyflow/react', 'dagre'],
        }
      }
    }
  }
})
```

### Option 2: Dynamic Imports

Lazy-load heavy components like Monaco editor and React Flow:

```typescript
const MonacoEditor = lazy(() => import('./components/MonacoFileEditor'))
const DagCanvas = lazy(() => import('./components/DagCanvas'))
```

### Option 3: Increase Warning Limit (Not recommended)

```typescript
build: {
  chunkSizeWarningLimit: 700
}
```

## Testing

- Run `npm run build` and verify bundle sizes
- Test app to ensure lazy-loaded components work correctly
- Measure initial load time before/after
