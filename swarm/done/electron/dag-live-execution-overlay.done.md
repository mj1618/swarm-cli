# Task: DAG Live Execution Overlay

## Goal

Add real-time agent status visualization to the DAG canvas. When agents are running, their corresponding task nodes in the DAG should display live status badges, iteration progress, and visual styling changes based on agent state. This connects the existing agent monitoring system (Panel 3) with the DAG visualization (Panel 2).

## Phase

Phase 2: DAG Visualization — "Live Execution Overlay" (the last uncompleted item in this phase)

## Files to Modify

1. **`electron/src/renderer/components/TaskNode.tsx`** — Add status badge rendering, progress ring/bar, and conditional styling based on agent state
2. **`electron/src/renderer/components/DagCanvas.tsx`** — Accept `agents` prop, match agents to task nodes by name, and pass agent state data into node data
3. **`electron/src/renderer/App.tsx`** — Pass the existing `agents` state array down to `DagCanvas`
4. **`electron/src/renderer/lib/yamlParser.ts`** — Extend `TaskNodeData` type to include optional agent status fields

## Dependencies

- Agent state watching already works (state:read, state:watch, state:onChanged)
- DagCanvas and TaskNode components already exist and render correctly
- AgentState type is defined in preload/index.ts

## Implementation Details

### 1. Extend TaskNodeData (yamlParser.ts)

Add optional fields to the `TaskNodeData` interface:
- `agentStatus?: 'pending' | 'running' | 'paused' | 'succeeded' | 'failed' | 'skipped'`
- `agentProgress?: { current: number; total: number }`
- `agentCost?: number`
- `agentPaused?: boolean`

### 2. DagCanvas: Match agents to task nodes

- Accept an `agents: AgentState[]` prop
- After computing nodes from YAML, enrich each node's data by finding a matching agent (match by `agent.name === taskName` or `agent.labels?.task_id === taskName`)
- Map agent status to a simplified display status:
  - `running` + `paused` → `'paused'`
  - `running` → `'running'`
  - `terminated` + successful iterations = total → `'succeeded'`
  - `terminated` + exit_reason `crashed`/`killed` → `'failed'`
  - No matching agent → no status (static display, as today)

### 3. TaskNode: Visual status indicators

Per the ELECTRON_PLAN.md spec:

| Status    | Visual                              |
|-----------|-------------------------------------|
| Pending   | Gray dot (no agent matched)         |
| Running   | Blue pulsing dot + progress bar     |
| Paused    | Yellow dot + "paused" label         |
| Succeeded | Green checkmark                     |
| Failed    | Red X                               |

- Show a small status indicator dot in the node header (left of task name)
- When running: show a thin progress bar below the header (current_iteration / iterations)
- When running: show cost in the node body (`$0.42`)
- Add a subtle animated pulse class for the running state (CSS animation)

### 4. App.tsx: Pass agents to DagCanvas

Simply pass the already-available `agents` state as a prop to `<DagCanvas agents={agents} .../>`.

## Acceptance Criteria

1. When an agent is running with a name matching a task in the DAG, the corresponding task node shows a blue pulsing indicator and iteration progress bar
2. When an agent is paused, its node shows a yellow indicator
3. When an agent completes successfully (terminated, all iterations done), its node shows a green checkmark
4. When an agent fails (terminated with crash/kill), its node shows a red X
5. Tasks with no matching agent display normally (no status indicator)
6. The status updates in real-time as agent state changes (already handled by state watcher → App re-render → DagCanvas re-render)
7. No regressions: clicking nodes still opens the task drawer, drag/connections still work

## Notes

- The agent name field corresponds to the task name in swarm.yaml (this is how swarm-cli names agents when running compose tasks)
- Keep the status dot small and unobtrusive — it should enhance, not clutter the node
- Use Tailwind's `animate-pulse` for the running indicator
- The progress bar should be a thin (2px) bar spanning the node width, using primary color
- Cost display uses the same formatting as AgentCard (`$X.XX`)

## Completion Notes

Implemented by agent 3b8608b3. All 4 files modified as specified:

1. **yamlParser.ts**: Added `AgentDisplayStatus` type and optional `agentStatus`, `agentProgress`, `agentCost` fields to `TaskNodeData`
2. **DagCanvas.tsx**: Added `agents` prop, `resolveAgentStatus()` helper, and `enrichedNodes` useMemo that matches agents to task nodes by name/labels/current_task
3. **TaskNode.tsx**: Added `StatusIndicator` component with blue pulsing dot (running), yellow dot (paused), green checkmark (succeeded), red X (failed). Added thin progress bar below header, iteration counter, cost display, and "paused" label
4. **App.tsx**: Passed `agents={agents}` prop to `<DagCanvas>`

Status updates flow reactively: state watcher -> App re-render -> DagCanvas enrichedNodes recalculation -> TaskNode re-render
