# Electron Release Workflow

## Goal

Create a GitHub Actions workflow that builds and publishes Electron desktop app packages (DMG for macOS, NSIS installer for Windows, AppImage for Linux) when a new release is created. This enables users to download pre-built binaries of Swarm Desktop.

## Files

- **Create**: `.github/workflows/electron-release.yml`
- **Modify**: None

## Dependencies

- All ELECTRON_PLAN.md phases are complete
- `electron-ci.yml` workflow exists and passes
- `electron-builder` is already configured in `electron/package.json`

## Acceptance Criteria

1. New workflow file `.github/workflows/electron-release.yml` exists
2. Workflow triggers on:
   - GitHub release published (to match existing release cadence)
   - Manual workflow dispatch (for testing)
3. Build matrix runs on:
   - `macos-latest` for DMG
   - `windows-latest` for NSIS installer
   - `ubuntu-latest` for AppImage
4. Workflow steps include:
   - Checkout code
   - Setup Node.js 20.x with npm caching
   - Install dependencies (`npm ci`)
   - Build the app (`npm run build && npm run build:electron`)
   - Package with electron-builder (`npm run package`)
   - Upload artifacts to the GitHub release
5. Built packages are attached to the release as downloadable assets

## Notes

- Use `electron-builder` which is already configured in `package.json` with:
  - `mac.target: "dmg"`
  - `win.target: "nsis"`  
  - `linux.target: "AppImage"`
- Output directory is `electron/release/`
- Use `softprops/action-gh-release@v1` or `actions/upload-release-asset` to attach artifacts
- Consider code signing in the future (not required for this task)
- The existing `release.yml` handles Go CLI releases via GoReleaser - keep these separate
- May need to add `GH_TOKEN` to electron-builder for auto-update support later

---

## Completion Notes

**Completed by:** Agent 35ab7d4d  
**Date:** 2026-02-13

### What was implemented:

1. Created `.github/workflows/electron-release.yml` with:
   - Trigger on GitHub release published events
   - Manual workflow dispatch with optional release tag input
   - Build matrix for macOS (DMG), Windows (NSIS), and Linux (AppImage)
   - Node.js 20 setup with npm caching
   - Full build pipeline: `npm ci` → `npm run build` → `npm run build:electron` → `npm run package`
   - Artifact upload to GitHub Actions for debugging
   - Release asset upload using `softprops/action-gh-release@v2`

2. Verified the electron app builds successfully locally:
   - `npm run build` ✅
   - `npm run build:electron` ✅

### All acceptance criteria met:
- ✅ Workflow file exists
- ✅ Triggers on release published and workflow dispatch
- ✅ Build matrix for all three platforms
- ✅ All required workflow steps included
- ✅ Artifacts attached to releases
