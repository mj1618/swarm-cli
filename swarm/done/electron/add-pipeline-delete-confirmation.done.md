# Add Confirmation Dialog for Pipeline Deletion

## Goal

Add a confirmation dialog before deleting a pipeline in the PipelinePanel component. Currently, clicking the "Delete" button immediately deletes the pipeline without any confirmation, which is inconsistent with task deletion (which does show a confirmation) and could lead to accidental data loss.

## Files

- `electron/src/renderer/components/PipelinePanel.tsx` - Add confirmation state and dialog UI

## Dependencies

None - this is a standalone UX improvement.

## Acceptance Criteria

1. Clicking the "Delete" button in PipelinePanel shows a confirmation dialog
2. The dialog displays the pipeline name being deleted
3. The dialog has "Cancel" and "Delete" buttons
4. Only clicking "Delete" in the dialog actually removes the pipeline
5. Pressing Escape closes the dialog without deleting
6. The dialog styling matches the existing delete confirmation dialogs (see DagCanvas.tsx lines 582-608 and FileTree.tsx lines 423-444 for reference)

## Notes

The existing delete button is at line 222-227 in PipelinePanel.tsx. The `handleDelete` function at lines 102-105 should be updated to show confirmation first instead of directly calling `onDelete`.

Reference implementation patterns:
- `DagCanvas.tsx` uses `deleteConfirm` state with `{ taskName: string } | null`
- `FileTree.tsx` uses `confirmDelete` state with `DirEntry | null`

Follow the same pattern: add a `showDeleteConfirm` boolean state, show a modal/dialog when true, and only call `onDelete(pipelineName)` when user confirms.

---

## Completion Notes

**Completed by agent 65dede25 on iteration 5**

Implemented the confirmation dialog for pipeline deletion in `PipelinePanel.tsx`:

1. Added `showDeleteConfirm` state variable
2. Modified `handleDelete` to show the confirmation dialog instead of directly deleting
3. Added `handleConfirmDelete` and `handleCancelDelete` callbacks
4. Updated the Escape key handler to close the dialog first if open
5. Added the confirmation dialog UI matching the style from DagCanvas.tsx

All acceptance criteria met:
- Delete button now shows confirmation dialog
- Dialog displays the pipeline name being deleted
- Dialog has Cancel and Delete buttons
- Only Delete button actually removes the pipeline
- Escape closes the dialog without deleting
- Styling matches existing delete confirmation dialogs
