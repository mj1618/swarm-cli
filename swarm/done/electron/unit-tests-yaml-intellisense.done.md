# Add Unit Tests for yamlIntellisense Module

## Goal

Add comprehensive unit tests for `electron/src/renderer/lib/yamlIntellisense.ts`. This module provides YAML IntelliSense features for the Monaco editor including autocomplete, hover documentation, and validation.

## Files

- **Create**: `electron/src/renderer/lib/__tests__/yamlIntellisense.test.ts`
- **Reference**: `electron/src/renderer/lib/yamlIntellisense.ts`

## Dependencies

- Vitest is already configured (see `electron/vitest.config.ts`)
- Test setup exists at `electron/src/test/setup.ts`
- Similar test patterns exist in `yamlParser.test.ts` and `yamlWriter.test.ts`

## Acceptance Criteria

1. Test file created at `electron/src/renderer/lib/__tests__/yamlIntellisense.test.ts`
2. Tests cover `isSwarmYaml()` function:
   - Returns true for paths ending in `swarm.yaml` or `swarm.yml`
   - Returns false for other YAML files and non-YAML files
3. Tests cover `extractTaskNames()` (internal function - may need export or inline testing):
   - Extracts task names from valid YAML content
   - Returns empty array for content with no tasks
   - Handles malformed or edge-case content gracefully
4. Tests cover `validateSwarmYaml()` with mocked Monaco:
   - Detects unknown top-level keys
   - Detects unknown task keys
   - Validates condition values (success/failure/any/always)
   - Validates task references in depends_on
   - Validates numeric values for iterations/parallelism
5. Tests cover hover provider logic for known keys
6. All tests pass when running `npm test` in electron directory

## Notes

- The module has Monaco type dependencies - use type-only imports and mock the monaco instance for validation tests
- `extractTaskNames()` is not exported but is used internally by other functions - you may need to either:
  - Export it for testing, or
  - Test it indirectly through `validateSwarmYaml()` behavior
- Follow the existing test patterns in the `__tests__` directory
- The completion provider tests may be more complex due to async nature - prioritize testing the pure/synchronous functions first

## Completion Notes

**Completed by agent 4d2a7221 on iteration 12**

Implemented 33 unit tests covering:
- `isSwarmYaml()` - 4 tests for path matching
- `extractTaskNames()` - 8 tests including edge cases (exported function for testing)
- `validateSwarmYaml()` - 13 tests for validation logic with mocked Monaco
- `createHoverProvider()` - 8 tests for hover documentation

Changes made:
1. Created `electron/src/renderer/lib/__tests__/yamlIntellisense.test.ts`
2. Exported `extractTaskNames()` in `yamlIntellisense.ts` for direct testing

All tests pass and app builds successfully.
