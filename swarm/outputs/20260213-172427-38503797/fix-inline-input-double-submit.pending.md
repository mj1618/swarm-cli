# Task: Fix InlineInput Double Submit on Enter + Blur

## Goal

Fix a race condition in the FileTreeItem `InlineInput` component where pressing Enter to submit a rename/create triggers both `onKeyDown` (Enter) and `onBlur` handlers, potentially causing double submission.

## Files

- **Modify**: `electron/src/renderer/components/FileTreeItem.tsx` — Add a submitted flag ref to the InlineInput component to prevent double-firing of onSubmit

## Bug Details

In `FileTreeItem.tsx` lines 110-128, the `InlineInput` component has both:
- `onKeyDown` handler that calls `onSubmit(val)` when Enter is pressed
- `onBlur` handler that also calls `onSubmit(val)` if the value changed

When the user presses Enter, the input loses focus (blur fires after the keydown), so `onSubmit` may be called twice — once from Enter and once from blur. This could create duplicate files/folders or show duplicate error toasts.

## Fix

Add a `useRef(false)` flag (`submittedRef`) that is set to `true` in the `onKeyDown` handler after calling `onSubmit`. In the `onBlur` handler, check `submittedRef.current` before calling `onSubmit`.

## Acceptance Criteria

1. Pressing Enter in rename/create input only triggers onSubmit once
2. Clicking away (blur without Enter) still triggers onSubmit correctly
3. Pressing Escape still cancels correctly
