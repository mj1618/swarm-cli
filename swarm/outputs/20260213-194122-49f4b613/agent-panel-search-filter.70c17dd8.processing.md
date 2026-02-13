# Task: Add Agent Panel Search and Filter

**Phase:** 5 - Polish (enhancement)
**Priority:** Low

## Goal

Add search and filter capabilities to the Agent Panel, allowing users to quickly find agents by name, task, or status when the history list grows large.

## Files to Modify

### electron/src/renderer/components/AgentPanel.tsx

Add a search input and status filter dropdown in the header section.

## Implementation Details

### 1. Add State Variables

```typescript
const [searchQuery, setSearchQuery] = useState('')
const [statusFilter, setStatusFilter] = useState<'all' | 'running' | 'terminated'>('all')
```

### 2. Add Search/Filter UI in Header

After the existing header button, add:

```tsx
<div className="p-2 border-b border-border flex items-center gap-2">
  <input
    type="text"
    value={searchQuery}
    onChange={(e) => setSearchQuery(e.target.value)}
    placeholder="Search agents..."
    className="flex-1 h-7 rounded border border-border bg-background px-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
  />
  <select
    value={statusFilter}
    onChange={(e) => setStatusFilter(e.target.value as 'all' | 'running' | 'terminated')}
    className="h-7 rounded border border-border bg-background px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
  >
    <option value="all">All</option>
    <option value="running">Running</option>
    <option value="terminated">History</option>
  </select>
</div>
```

### 3. Add Filtering Logic

```typescript
const filteredAgents = useMemo(() => {
  let result = agents
  
  // Apply status filter
  if (statusFilter === 'running') {
    result = result.filter(a => a.status === 'running')
  } else if (statusFilter === 'terminated') {
    result = result.filter(a => a.status !== 'running')
  }
  
  // Apply search query
  if (searchQuery.trim()) {
    const query = searchQuery.toLowerCase()
    result = result.filter(a => 
      a.name?.toLowerCase().includes(query) ||
      a.id.toLowerCase().includes(query) ||
      a.model?.toLowerCase().includes(query) ||
      a.current_task?.toLowerCase().includes(query)
    )
  }
  
  return result
}, [agents, searchQuery, statusFilter])
```

### 4. Update the Rendering Logic

Use `filteredAgents` instead of `agents` when splitting into running/history sections:

```typescript
const runningAgents = filteredAgents.filter(a => a.status === 'running')
const historyAgents = filteredAgents
  .filter(a => a.status !== 'running')
  .sort((a, b) => {
    const aTime = a.terminated_at || a.started_at
    const bTime = b.terminated_at || b.started_at
    return new Date(bTime).getTime() - new Date(aTime).getTime()
  })
```

### 5. Show "No results" Message

When filtered results are empty but original list has agents:

```tsx
{filteredAgents.length === 0 && agents.length > 0 && (
  <div className="text-sm text-muted-foreground p-2">
    No agents match your search
  </div>
)}
```

## Acceptance Criteria

1. Search input filters agents by name, ID, model, or current task
2. Status dropdown filters to show "All", "Running", or "History (terminated)"
3. Filtering is case-insensitive
4. "No agents match your search" message shown when filter yields no results
5. Clearing the search shows all agents again
6. Filtering persists while viewing agent details (when returning to list)
7. App builds successfully with `npm run build`

## Notes

- Search should be debounced for performance if agent lists get very large (optional enhancement)
- Consider adding a "clear search" button (X icon) in the search input
- The filter state could optionally be persisted to localStorage
