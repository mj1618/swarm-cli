# Add TypeScript Type-Checking and ESLint to CI

## Goal

Enhance the Electron CI pipeline to catch type errors and code quality issues before merge. Currently, the CI only runs `tsc && vite build` and `tsc -p tsconfig.main.json`, which compile the code but may not catch all type errors (especially with `skipLibCheck` or loose config). Add explicit type-checking and introduce ESLint for code quality.

## Files

### Modify
- `.github/workflows/electron-ci.yml` - Add type-check and lint steps
- `electron/package.json` - Add `lint` and `typecheck` scripts, add eslint dependencies

### Create
- `electron/.eslintrc.cjs` - ESLint configuration for React/TypeScript
- `electron/eslint.config.js` - (Alternative: flat config if using ESLint 9+)

## Dependencies

- None (CI workflow already exists)

## Implementation Steps

1. Add ESLint and TypeScript ESLint dependencies to package.json:
   - `eslint`
   - `@typescript-eslint/parser`
   - `@typescript-eslint/eslint-plugin`
   - `eslint-plugin-react`
   - `eslint-plugin-react-hooks`

2. Create ESLint config with rules for:
   - TypeScript strict type checking
   - React hooks rules
   - No unused variables
   - Consistent code style

3. Add npm scripts to package.json:
   - `"typecheck": "tsc --noEmit && tsc -p tsconfig.main.json --noEmit"`
   - `"lint": "eslint src/ --ext .ts,.tsx"`

4. Update `.github/workflows/electron-ci.yml` to add steps:
   - Run `npm run typecheck` after install
   - Run `npm run lint` after typecheck
   - Keep existing build steps

## Acceptance Criteria

- [ ] `npm run typecheck` passes locally with no errors
- [ ] `npm run lint` passes locally with no errors (or only warnings)
- [ ] CI workflow runs type-check and lint before build
- [ ] CI fails if there are type errors or lint errors
- [ ] ESLint config includes React hooks rules

## Notes

- Use ESLint flat config (eslint.config.js) if the installed ESLint version is 9+
- Consider adding `--max-warnings 0` to lint script to fail on warnings
- The existing build already compiles TypeScript, but explicit `--noEmit` type-check catches errors earlier and gives clearer error messages
- May need to fix existing lint/type issues before CI will pass

## Completion Note

Implemented ESLint and TypeScript type-checking for the Electron app:

1. Added ESLint 9 with flat config (`eslint.config.js`) including:
   - TypeScript ESLint for type-aware linting
   - React and React Hooks plugins
   - Sensible defaults for unused vars, no-explicit-any, no-console

2. Added npm scripts:
   - `typecheck`: Runs `tsc --noEmit` for both renderer and main process configs
   - `lint`: Runs ESLint on `src/` directory

3. Updated CI workflow to run typecheck and lint before build

4. Fixed several bugs found during linting:
   - React hooks called after early returns in `LogView.tsx` and `DagCanvas.tsx` (rules of hooks violations)
   - Unused variable in `AgentPanel.tsx`
   - Unnecessary escape characters in regex patterns
   - Regex spaces issue in `yamlIntellisense.ts`

All type errors and lint errors are now resolved. Only warnings remain (mostly `@typescript-eslint/no-explicit-any` for IPC handlers).
