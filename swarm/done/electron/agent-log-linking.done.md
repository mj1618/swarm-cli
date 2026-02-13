# Task: Add Agent-to-Log Linking in Console Panel

**Phase:** 5 - Polish
**Priority:** Medium-High

## Goal

Add a "View Log" button in the AgentDetailView component that navigates the user to the corresponding agent's log tab in the ConsolePanel. Currently, the AgentDetailView displays the `log_file` path as static text, but there's no way to jump to that log in the console. This feature connects the agent panel to the console panel for a seamless monitoring workflow.

The ELECTRON_PLAN.md specifies that the console should have per-agent tabs (line 248-258) and that agents show log file paths — but there's no linkage between them.

## Files

### Modify

1. **`electron/src/renderer/App.tsx`**
   - Lift the console panel's active tab state up to App level (or use a simple event emitter / Zustand store slice)
   - Add a `consoleActiveTab` state and `setConsoleActiveTab` callback
   - Pass these down to both `ConsolePanel` and `AgentPanel`
   - Ensure the console auto-expands (un-collapses) when a log tab is selected from the agent panel

2. **`electron/src/renderer/components/ConsolePanel.tsx`**
   - Accept optional `activeTab` and `onActiveTabChange` props
   - Use controlled state when props are provided, internal state otherwise (backwards compatible)
   - When `activeTab` prop changes externally, switch to that tab

3. **`electron/src/renderer/components/AgentDetailView.tsx`**
   - Replace the static log file path display with a clickable "View Log" button
   - The button should call a new `onViewLog?: (logPath: string) => void` prop
   - Keep the path text visible but make it clickable

4. **`electron/src/renderer/components/AgentPanel.tsx`**
   - Thread the `onViewLog` callback from App through to AgentDetailView

## Dependencies

- ConsolePanel component exists (done)
- AgentDetailView component exists (done)
- Agent panel with detail view exists (done)

## Implementation Approach

### Option A: Lift State (Recommended)
1. In `App.tsx`, add state: `const [consoleActiveTab, setConsoleActiveTab] = useState<string>('console')`
2. Pass `activeTab={consoleActiveTab}` and `onActiveTabChange={setConsoleActiveTab}` to `ConsolePanel`
3. Pass `onViewLog` callback to `AgentPanel` → `AgentDetailView` that:
   - Sets `consoleActiveTab` to the matching log file path
   - Sets `consoleCollapsed` to `false` (to ensure the console is visible)
4. In `ConsolePanel`, use the prop-controlled tab when provided
5. In `AgentDetailView`, add a button next to the log file path

### UI for the View Log button
```tsx
{agent.log_file && (
  <Section label="Log File">
    <div className="flex items-center gap-2">
      <p className="text-[10px] font-mono text-muted-foreground break-all select-all flex-1">
        {agent.log_file}
      </p>
      <button
        onClick={() => onViewLog?.(agent.log_file!)}
        className="shrink-0 px-2 py-0.5 text-[10px] bg-primary/10 text-primary rounded hover:bg-primary/20 transition-colors"
      >
        View Log
      </button>
    </div>
  </Section>
)}
```

## Acceptance Criteria

1. AgentDetailView shows a "View Log" button next to the log file path
2. Clicking "View Log" switches the ConsolePanel to the corresponding agent's log tab
3. If the console panel is collapsed, it automatically expands when "View Log" is clicked
4. The ConsolePanel's own tab clicking still works normally
5. If the agent's log file doesn't exist in the log file list yet, the console stays on the current tab (no crash)
6. TypeScript compiles without errors
7. App builds successfully with `npm run build`

## Notes

- The ConsolePanel currently manages its own `activeTab` state internally (line 14). This needs to be lifted to App or made controllable via props.
- The log file paths in agent state (e.g., `~/.swarm/logs/abc12345.log`) should match the paths returned by `window.logs.list()`.
- Keep the implementation simple — no need for a full event bus or complex state management. Prop drilling through AgentPanel is fine for this case.

## Completion Notes

**Completed by agent babbc5f4**

Implemented Option A (Lift State) as recommended:

1. **ConsolePanel.tsx**: Added `ConsolePanelProps` interface with optional `activeTab` and `onActiveTabChange` props. Uses controlled state when props are provided, falls back to internal state otherwise (backwards compatible).

2. **AgentDetailView.tsx**: Added `onViewLog?: (logPath: string) => void` prop. The log file section now shows the path alongside a "View Log" button that triggers the callback.

3. **AgentPanel.tsx**: Added `AgentPanelProps` interface with `onViewLog` prop, threaded down to `AgentDetailView`.

4. **App.tsx**: Added `consoleActiveTab` state lifted to App level. Created `handleViewLog` callback that sets the active tab and auto-expands the console if collapsed. Passes controlled tab props to `ConsolePanel` and `onViewLog` to `AgentPanel`.

All acceptance criteria met. Build passes cleanly (`npm run build`).
