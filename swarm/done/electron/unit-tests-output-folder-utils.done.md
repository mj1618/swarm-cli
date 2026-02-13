# Unit Tests for outputFolderUtils.ts

## Goal

Add comprehensive unit tests for `electron/src/renderer/lib/outputFolderUtils.ts` which contains utility functions for parsing output folder names and formatting timestamps in the file tree.

## Files

- **Create**: `electron/src/renderer/lib/__tests__/outputFolderUtils.test.ts`

## Dependencies

- None (unit test infrastructure already exists via `unit-test-setup-vitest.done.md`)

## Acceptance Criteria

1. Test file created at `electron/src/renderer/lib/__tests__/outputFolderUtils.test.ts`
2. Tests cover all exported functions:
   - `parseOutputFolderName()` - parsing folder names like "20260213-142305-abc12345"
   - `isInOutputsDirectory()` - checking if path contains /outputs/
   - `formatOutputTimestamp()` - formatting dates as "Today 2:43 PM", "Yesterday", etc.
   - `getOutputFolderDisplay()` - combining parse + format for display
3. Tests include edge cases:
   - Invalid folder name patterns (wrong format, letters in date, etc.)
   - Invalid dates (Feb 30, month 13, etc.)
   - Path edge cases (path without outputs, path with outputs elsewhere)
   - Timestamp boundaries (today at midnight, yesterday at 11:59 PM)
   - Year boundary handling (this year vs last year)
   - Different hash lengths
4. All tests pass: `npm test` in electron/ directory
5. Tests follow existing patterns in `yamlParser.test.ts` and `dagValidation.test.ts`

## Notes

The `outputFolderUtils.ts` module handles formatting of output folder names in the file tree. Output folders use the pattern `YYYYMMDD-HHMMSS-[hash]`.

Key test scenarios:
1. Valid folder name parsing extracts correct date components and hash
2. Invalid patterns return null (not throw)
3. `formatOutputTimestamp` handles "Today", "Yesterday", same year, and different year
4. `isInOutputsDirectory` correctly identifies paths with /outputs/ segment
5. `getOutputFolderDisplay` returns null for non-output paths or invalid names

Test patterns from existing tests:

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  parseOutputFolderName,
  isInOutputsDirectory,
  formatOutputTimestamp,
  getOutputFolderDisplay,
} from '../outputFolderUtils'
```

For testing `formatOutputTimestamp`, use `vi.useFakeTimers()` to control "now":

```typescript
beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date(2026, 1, 13, 14, 30, 0)) // Feb 13, 2026 2:30 PM
})

afterEach(() => {
  vi.useRealTimers()
})
```

---

## Completion Notes

**Completed by**: Agent 6c3ca75b
**Date**: Feb 13, 2026

### Implementation Summary

Created comprehensive test suite with 62 tests covering all four exported functions:

#### `parseOutputFolderName` (27 tests)
- Valid folder names: standard, uppercase/mixed case hex, short/long hashes, midnight/end-of-day timestamps
- Invalid folder names: empty string, missing parts, wrong formats, invalid hex characters, extra prefix/suffix, wrong separators
- Date validation: handles JavaScript Date rollover behavior for invalid dates (Feb 30, month 13, etc.)

#### `isInOutputsDirectory` (10 tests)
- Correct detection of `/outputs/` path segment
- Edge cases: empty path, "outputs" without slashes, Windows-style paths, multiple outputs segments

#### `formatOutputTimestamp` (14 tests)
- Today formatting: morning, midnight, noon, 11:59 PM
- Yesterday formatting: various times
- Same year formatting: earlier dates
- Different year formatting: last year, several years ago, future dates
- Year boundary handling: yesterday across year boundary
- Edge case: current time is midnight

#### `getOutputFolderDisplay` (11 tests)
- Valid output folders with correct timestamp and hash formatting
- Null returns for invalid paths, invalid names, empty inputs
- Nested outputs directories

### Verification

- All 62 tests pass
- App builds successfully (`npm run build`)
- All 153 electron tests pass (`npm test`)
