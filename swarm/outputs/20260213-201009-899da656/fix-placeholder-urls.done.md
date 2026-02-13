# Fix Placeholder URLs in Electron App

## Summary

The Help menu and About dialog contain placeholder URLs that need to be updated to point to the actual repository.

## Files Affected

- `electron/src/main/index.ts` (lines 387, 391)
- `electron/src/renderer/components/AboutDialog.tsx` (lines 86, 93)

## Current (Broken)

```typescript
'https://github.com/your-org/swarm-cli#readme'
'https://github.com/your-org/swarm-cli/issues'
'https://github.com/your-org/swarm-cli'
```

## Expected

The URLs should point to the actual repository. Either:
1. Update to the real repository URL
2. Or make them configurable via environment/config

## Priority

Low - cosmetic issue that doesn't affect functionality

## Dependencies

None
