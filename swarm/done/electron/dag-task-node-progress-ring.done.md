# Task: Add Progress Ring Around Running DAG Task Nodes

**Phase:** 5 - Polish (DAG Live Execution Overlay enhancement)

## Goal

Replace the thin linear progress bar on running DAG task nodes with a circular SVG progress ring that wraps around the entire node border. The ELECTRON_PLAN.md explicitly states: "Progress ring around running tasks showing iteration progress."

This provides a much more visible and intuitive indication of agent progress when viewing the DAG.

## Files

- `electron/src/renderer/components/TaskNode.tsx` — Replace the linear progress bar with an SVG progress ring rendered around the node border. The ring should:
  - Use an SVG `<circle>` with `stroke-dasharray` and `stroke-dashoffset` to show progress
  - Animate smoothly between progress values (CSS transition on stroke-dashoffset)
  - Use blue color (`stroke: #3b82f6`) matching the existing running-state styling
  - Only appear when `agentStatus === 'running'` and `agentProgress.total > 0`
  - Be layered behind the node content using absolute positioning
  - The ring should be subtle (2px stroke) so it doesn't visually overwhelm the node

## Dependencies

- `dag-live-execution-overlay.done.md` (completed) — provides the `agentStatus` and `agentProgress` data flowing into TaskNode

## Acceptance Criteria

1. When a task is running with known progress (iterations > 0), a circular progress ring is visible around the node border
2. The ring fills proportionally: e.g., 3/20 iterations = ~15% of the ring filled
3. The ring smoothly animates when progress changes (CSS transition)
4. The ring disappears when the task is no longer running (succeeded/failed/pending)
5. The existing thin linear progress bar is removed (replaced by the ring)
6. The ring uses rounded corners at its ends (`stroke-linecap: round`)
7. The implementation uses only inline SVG — no new dependencies

## Notes

- The SVG ring can be positioned absolutely around the node container using `inset: -N px` to sit just outside the border
- Use `stroke-dasharray: circumference` and `stroke-dashoffset: circumference * (1 - progress)` for the fill effect
- The background track of the ring should be a faint gray (`stroke: rgba(255,255,255,0.1)`)
- Since TaskNode already receives `agentProgress` with `{ current, total }`, no data changes are needed
- Keep the iteration text label (`iter 3/20`) inside the node as-is for exact numbers

## Completion Notes

**Completed by agent 12a6b339 on iteration 12.**

Implemented a `ProgressRing` component in `TaskNode.tsx` that renders an SVG rounded-rectangle ring around running DAG task nodes:

- Uses SVG `<rect>` with `rx`/`ry` rounded corners matching the node's `rounded-lg` border radius
- Background track: `rgba(255,255,255,0.1)` subtle gray
- Progress fill: `#3b82f6` blue, 2px stroke width
- Uses `pathLength={100}` trick with `strokeDasharray`/`strokeDashoffset` for clean percentage-based fill
- Smooth 0.5s CSS transition on `stroke-dashoffset`
- Positioned absolutely with `inset: -4px` to wrap just outside the node border
- `pointer-events: none` so it doesn't interfere with node interactions
- Removed the old linear progress bar (`h-0.5 bg-muted` div)
- Removed `overflow-hidden` from node container to allow the ring to render outside bounds
- Iteration text label (`iter 3/20`) preserved inside the node
