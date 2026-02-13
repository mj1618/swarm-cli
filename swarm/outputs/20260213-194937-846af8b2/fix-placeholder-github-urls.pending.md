# Fix Placeholder GitHub URLs

## Description

The Help menu and About dialog contain placeholder URLs that point to a non-existent repository:
- `https://github.com/your-org/swarm-cli#readme`
- `https://github.com/your-org/swarm-cli/issues`
- `https://github.com/your-org/swarm-cli`

These links will 404 when users click them.

## Files Affected

- `electron/src/main/index.ts` (lines 349, 353)
- `electron/src/renderer/components/AboutDialog.tsx` (lines 86, 93)

## Solution

Either:
1. Replace with the actual swarm-cli repository URL if it exists publicly
2. Remove the links or disable them if no public repository exists
3. Replace with documentation links that do exist

## Priority

Low - Cosmetic issue, doesn't affect core functionality

## Dependencies

None
