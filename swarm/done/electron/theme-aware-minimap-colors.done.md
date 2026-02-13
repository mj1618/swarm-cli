# Task: Add Theme-Aware MiniMap Colors

**Phase:** 5 - Polish
**Priority:** Low

## Goal

Update the DAG canvas MiniMap component to use theme-aware colors that match the current appearance mode (dark/light). Currently, the MiniMap uses hardcoded dark theme colors even when the app is in light mode, causing visual inconsistency.

## Current Behavior

In `DagCanvas.tsx`, the MiniMap has hardcoded dark theme colors:

```tsx
<MiniMap
  nodeColor="hsl(217 91% 60%)"
  maskColor="hsl(222 84% 5% / 0.7)"
  bgColor="hsl(222 84% 5%)"
  style={{ borderRadius: 8, border: '1px solid hsl(217 33% 17%)' }}
/>
```

When the user switches to light mode, the MiniMap still displays with dark colors, creating visual inconsistency with the rest of the app.

## Files to Modify

- `electron/src/renderer/components/DagCanvas.tsx` — Update MiniMap props to use theme-conditional colors

## Dependencies

- Theme toggle feature (completed: `theme-toggle-dark-light-mode.done.md`)
- DAG canvas foundation (completed)

## Implementation Notes

The `DagCanvas` component already receives a `theme` prop of type `EffectiveTheme` ('dark' | 'light'). Use this to conditionally set MiniMap colors:

```tsx
const minimapColors = theme === 'light' 
  ? {
      nodeColor: 'hsl(217 91% 45%)',      // Slightly darker blue for visibility on light bg
      maskColor: 'hsl(220 14% 96% / 0.7)', // Light gray mask
      bgColor: 'hsl(220 14% 96%)',         // Light background (matches slate-100)
      borderColor: 'hsl(214 32% 91%)',     // Light border (matches slate-200)
    }
  : {
      nodeColor: 'hsl(217 91% 60%)',       // Current blue
      maskColor: 'hsl(222 84% 5% / 0.7)',  // Dark mask
      bgColor: 'hsl(222 84% 5%)',          // Dark background
      borderColor: 'hsl(217 33% 17%)',     // Dark border
    }
```

Apply these colors to the MiniMap component:

```tsx
<MiniMap
  nodeColor={minimapColors.nodeColor}
  maskColor={minimapColors.maskColor}
  bgColor={minimapColors.bgColor}
  style={{ borderRadius: 8, border: `1px solid ${minimapColors.borderColor}` }}
/>
```

## Acceptance Criteria

1. When app is in dark mode, MiniMap displays with dark background and appropriate contrast
2. When app is in light mode, MiniMap displays with light background that matches the app's light theme
3. MiniMap colors transition smoothly when theme is toggled
4. Node colors in MiniMap remain visible and distinguishable in both themes
5. MiniMap border color matches the theme
6. App builds successfully with `npm run build`

## Notes

- The `theme` prop is already passed to `DagCanvas` from `App.tsx`
- Consider using Tailwind's color palette values for consistency (slate-100, slate-200 for light mode)
- The MiniMap should have sufficient contrast in both modes for usability

---

## Completion Notes

**Completed by:** Agent 8b2d77dc
**Date:** 2026-02-13

### What was implemented

Updated the MiniMap component in `DagCanvas.tsx` to use theme-conditional colors:

- **Dark mode:** Keeps original colors (dark background `hsl(222 84% 5%)`, blue nodes `hsl(217 91% 60%)`, dark border)
- **Light mode:** Uses light colors (light gray background `hsl(220 14% 96%)`, darker blue nodes `hsl(217 91% 45%)` for better visibility, light border)

The implementation uses inline ternary expressions on the MiniMap props based on the existing `theme` prop, which keeps the code concise and maintains existing dark mode behavior while adding proper light mode support.

### Verification

- Build passes successfully with `npm run build`
- All acceptance criteria met
