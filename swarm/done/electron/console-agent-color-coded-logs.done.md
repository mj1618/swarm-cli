# Task: Color-coded agent logs in Console Panel

## Goal

Add color-coding to the Console Panel so that log lines are visually distinguished by agent name. The ELECTRON_PLAN.md specifies "Color-coded by agent" as a feature of the Bottom Panel / Console. Currently, LogView renders all lines as plain monospace text with no color differentiation.

## Files

- `electron/src/renderer/components/LogView.tsx` — Add agent-name detection and per-agent color styling to log lines
- `electron/src/renderer/components/ConsolePanel.tsx` — Minor updates if needed to pass agent color mapping

## Dependencies

None — the Console Panel and LogView already exist and display log content.

## Acceptance Criteria

1. Log lines that contain an agent identifier pattern (e.g., `[planner]`, `[coder]`, `[evaluator]`) are color-coded with a unique color per agent name
2. The color assignment is consistent: the same agent name always gets the same color within a session
3. The agent name tag itself (e.g., `[planner]`) is displayed in the assigned color; the rest of the line remains the default text color
4. Colors are chosen from a visually distinct palette that works well on the dark background (e.g., cyan, green, yellow, pink, orange, purple)
5. The highlight search feature still works correctly when color-coding is active
6. The filter mode still works correctly — filtered lines maintain their color-coding
7. Lines without an agent identifier render normally (no color)

## Notes

- From ELECTRON_PLAN.md, the Console section states: "Color-coded by agent"
- The log format typically includes lines like: `2:30:15 [planner] Starting iteration 3...`
- A simple approach: parse each line for a `[name]` pattern, maintain a Map<string, color> that assigns colors deterministically (e.g., hash the agent name to pick from a fixed palette)
- This is a Phase 4/5 polish feature — all core console functionality (tabs, search, filter, export, streaming) already works

## Completion Notes

Implemented in `LogView.tsx` only (no changes needed to `ConsolePanel.tsx`):

- Added a 10-color palette of Tailwind classes (cyan, green, yellow, pink, orange, purple, blue, emerald, rose, teal) all at 400 shade for dark background visibility
- `parseAgentTag()` uses a regex to detect `[agentName]` patterns in log lines
- `getAgentColor()` uses a deterministic string hash to assign consistent colors per agent name via a module-level Map cache
- `renderLineContent()` wraps the agent tag `<span>` with the assigned color class, while leaving the rest of the line in its default classification color (error/tool/normal)
- Search highlighting and filter mode continue to work correctly — `renderLineContent` composes with `highlightMatches` when a query is active
- Lines without agent tags render exactly as before
