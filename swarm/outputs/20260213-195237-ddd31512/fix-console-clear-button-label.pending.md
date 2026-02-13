# Fix Console Panel "Clear" Button Label

## Issue
In `ConsolePanel.tsx`, the "Clear" button doesn't actually clear the console content. Instead, it:
- For the "console" tab: resets `logContents` to empty and calls `fetchLogFiles()` to refresh
- For individual log tabs: calls `fetchLogContent(activeTab)` to refresh

The button should be labeled "Refresh" to accurately describe its behavior, or it should actually clear the content if "Clear" is the intended action.

## Current Behavior
```tsx
<button
  onClick={() => {
    if (activeTab === 'console') {
      setLogContents({})
      fetchLogFiles()
    } else {
      fetchLogContent(activeTab)
    }
  }}
  // ...
>
  Clear
</button>
```

## Solution Options
1. **Rename to "Refresh"** - Change the label to accurately describe the refresh behavior
2. **Implement actual clear** - Make the button actually clear the displayed content without refetching

Option 1 is recommended as it aligns with the existing implementation.

## Files to modify
- `electron/src/renderer/components/ConsolePanel.tsx`

## Dependencies
None
