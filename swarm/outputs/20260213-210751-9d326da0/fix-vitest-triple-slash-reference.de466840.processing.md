# Fix Triple Slash Reference Lint Error in vitest.d.ts

## Problem

The linter reports an error in `electron/src/test/vitest.d.ts`:

```
src/test/vitest.d.ts
  2:1  error  Do not use a triple slash reference for @testing-library/jest-dom, use `import` style instead  @typescript-eslint/triple-slash-reference
```

This is the only linter **error** (not warning) in the codebase and may cause CI failures.

## Current Code

```typescript
/// <reference types="vitest/globals" />
/// <reference types="@testing-library/jest-dom" />

// Augment vitest's Assertion interface with jest-dom matchers
import '@testing-library/jest-dom'
```

## Fix

Replace the triple-slash reference with an import. The file already imports `@testing-library/jest-dom` at line 5, so the triple-slash reference on line 2 is redundant.

Proposed fix - remove line 2:

```typescript
/// <reference types="vitest/globals" />

// Augment vitest's Assertion interface with jest-dom matchers
import '@testing-library/jest-dom'
```

Or if the reference directive is needed for type augmentation, disable the rule for that specific line:

```typescript
/// <reference types="vitest/globals" />
// eslint-disable-next-line @typescript-eslint/triple-slash-reference
/// <reference types="@testing-library/jest-dom" />

// Augment vitest's Assertion interface with jest-dom matchers
import '@testing-library/jest-dom'
```

## Verification

1. Run `npm run lint` - should have 0 errors (warnings are acceptable)
2. Run `npm run test` - all tests should still pass
3. Run `npm run typecheck` - should pass

## Dependencies

None

## Notes

This is a minor code quality issue. The triple-slash reference may be redundant since there's already an import statement for the same module.
