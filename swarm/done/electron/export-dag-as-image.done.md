# Task: Export DAG as Image

**Phase:** 5 - Polish (Enhancement)
**Priority:** Low

## Goal

Add the ability to export the DAG canvas as a PNG or SVG image for documentation, sharing, and debugging purposes. This allows users to capture their pipeline visualization without taking screenshots.

## Files to Modify

- `electron/src/renderer/components/DagCanvas.tsx` — Add export button to toolbar and implement export functionality using React Flow's built-in methods
- `electron/src/main/index.ts` — Add IPC handler for save dialog with image file filters
- `electron/src/preload/index.ts` — Expose image save dialog if needed

## Dependencies

- DAG canvas with React Flow (completed)
- Existing save dialog infrastructure (completed for log export)

## Implementation Notes

### React Flow Export API

React Flow provides methods to export the viewport:
1. `getNodes()` and `getEdges()` for data export
2. Use `toPng` or `toSvg` from `@xyflow/react` or implement using `html2canvas`/`dom-to-image`

### Export Button

Add a small export button (camera/download icon) to the DAG toolbar, next to existing controls:
- Position: Top-right of DAG canvas, near "Reset Layout" or "+ Add Task" buttons
- Icon: Camera or download icon
- Dropdown options: "Export as PNG", "Export as SVG"

### Export Flow

1. User clicks export button
2. Show dropdown with format options (PNG/SVG)
3. Capture the current viewport (or optionally fit-all-nodes first)
4. Open native save dialog with appropriate file extension filter
5. Save the image to user-selected location
6. Show toast notification on success/failure

### Styling Considerations

- Ensure export captures the dark theme background
- Include all visible nodes and edges
- Option to include padding around the graph
- Consider adding a watermark or title (optional)

## Acceptance Criteria

1. Export button visible in DAG canvas toolbar
2. Clicking export shows format options (PNG, SVG)
3. Selecting a format opens native save dialog
4. Image is saved with correct format and includes all visible nodes/edges
5. Toast notification confirms successful export
6. Export works correctly with various graph sizes (small and large DAGs)
7. App builds successfully with `npm run build`

## Notes

This feature complements the existing log export functionality and provides a complete documentation workflow. Users can export both their pipeline visualization and execution logs.

---

## Completion Notes

**Completed by:** f97f9233
**Date:** 2026-02-13

### Implementation Summary

1. **Added `html-to-image` package** for capturing the React Flow viewport as PNG/SVG

2. **Added new IPC handler in main process** (`dialog:saveImage`) that:
   - Accepts data URL, default filename, and format (png/svg)
   - Opens native save dialog with appropriate file filters
   - Decodes and saves the image data (base64 for PNG, UTF-8 for SVG)

3. **Updated preload** to expose `dialog.saveImage()` API with TypeScript types

4. **Added export functionality to DagCanvas**:
   - Export dropdown button in top-right Panel (next to Reset Layout)
   - Format options: PNG (2x pixel ratio for high DPI) and SVG
   - Uses `getNodesBounds()` and `getViewportForBounds()` from React Flow to calculate proper export dimensions
   - Respects theme setting (dark/light background)
   - Shows loading spinner while exporting
   - Toast notifications for success/error states

### All acceptance criteria met:
- Export button is visible in DAG toolbar
- Clicking shows PNG/SVG dropdown options
- Native save dialog opens with correct file filters
- Images saved with all nodes/edges and proper padding
- Toast notifications on success/failure
- Works with various graph sizes
- App builds successfully
