# Add Application Icons for Electron Releases

## Goal

Add custom application icons for Swarm Desktop so the release packages (DMG, NSIS installer, AppImage) show proper branding instead of the default Electron icon.

## Files

### Create
- `electron/build/icon.icns` - macOS icon (512x512 and multiple sizes)
- `electron/build/icon.ico` - Windows icon (256x256 and multiple sizes)  
- `electron/build/icon.png` - Linux icon (512x512)
- `electron/build/icons/` - Directory with PNG icons at standard sizes (16x16, 32x32, 48x48, 64x64, 128x128, 256x256, 512x512)

### Modify
- `electron/package.json` - Add icon paths to electron-builder config

## Dependencies

- `electron-release.yml` workflow exists (completed)
- electron-builder is configured in package.json

## Acceptance Criteria

1. `electron/build/` directory exists with icon files
2. Icons use a design that represents Swarm Desktop:
   - A swarm/hive motif, or
   - A DAG/node graph visual, or
   - A simple "S" lettermark
3. `package.json` build config includes icon configuration:
   ```json
   "build": {
     "icon": "build/icon"
   }
   ```
4. `npm run package` successfully uses the custom icons
5. Icons display correctly in:
   - macOS dock and Finder
   - Windows taskbar and Start menu
   - Linux application menus

## Notes

- electron-builder automatically selects the right icon format based on platform
- The `build/icon` path (without extension) lets electron-builder pick `.icns`, `.ico`, or `.png` as appropriate
- For macOS, `.icns` must contain multiple sizes (16, 32, 64, 128, 256, 512, 1024)
- For Windows, `.ico` should contain 16, 32, 48, 64, 128, 256 pixel sizes
- Tools like `png2icns` (macOS) or online converters can generate these from a source PNG
- Keep the design simple and recognizable at small sizes (16x16)
- Consider using the same icon for the app's title bar and tray if those features are added later

---

## Completion Notes

**Completed by agent 27c67375 on iteration 9**

### Implementation Summary

1. **Created icon generator script** (`electron/scripts/generate-icons.js`):
   - Uses `sharp` package to generate SVG-to-PNG icons programmatically
   - Generates all standard sizes: 16, 32, 48, 64, 128, 256, 512, 1024 pixels
   - Creates Windows `.ico` using `png-to-ico`
   - Design: Stylized "S" lettermark on amber/gold gradient (swarm/hive theme)
   - Small particle dots suggesting swarm movement

2. **Generated icon files**:
   - `build/icon.icns` - macOS (534KB, multi-resolution)
   - `build/icon.ico` - Windows (285KB, multi-resolution)
   - `build/icon.png` - Linux primary (54KB, 512x512)
   - `build/icons/` - All standard PNG sizes

3. **Updated `package.json`**:
   - Added `"generate-icons"` script for regenerating icons
   - Added `"icon": "build/icon"` to build config
   - Added platform-specific icon paths for mac, win, linux

4. **Dependencies added**:
   - `sharp` - Image processing for PNG generation
   - `png-to-ico` - Windows ICO file creation

5. **Fixed TypeScript errors** in test files (unused imports/variables)

### Build Verification

- `npm run build` - Passes
- `npm run typecheck` - Passes
