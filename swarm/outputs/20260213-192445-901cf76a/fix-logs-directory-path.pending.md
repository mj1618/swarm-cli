# Fix Logs Directory Path in Electron App

## Problem

The Electron app is looking for log files in the wrong directory.

**Current (incorrect):**
```typescript
// electron/src/main/index.ts line 520
const logsDir = path.join(os.homedir(), 'swarm', 'logs')  // ~/swarm/logs/
```

**Expected (correct):**
```typescript
const logsDir = path.join(os.homedir(), '.swarm', 'logs')  // ~/.swarm/logs/
```

## Evidence

The swarm CLI stores logs in `~/.swarm/logs/` as shown in:
- `internal/detach/detach.go` line 17: `logsDir := filepath.Join(homeDir, ".swarm", "logs")`
- `cmd/doctor.go` line 275: `logsDir := filepath.Join(homeDir, ".swarm", "logs")`

## Impact

- Console panel log viewing will not work because it can't find log files
- Log file listing will always return empty
- The "View Log" feature in agent detail view will fail

## Fix

Change line 520 in `electron/src/main/index.ts` from:
```typescript
const logsDir = path.join(os.homedir(), 'swarm', 'logs')
```

To:
```typescript
const logsDir = path.join(os.homedir(), '.swarm', 'logs')
```

Also update the comment on line 519 from:
```
// Logs directory IPC handlers — read log files from ~/swarm/logs/
```
To:
```
// Logs directory IPC handlers — read log files from ~/.swarm/logs/
```

## Dependencies

None

## Testing

1. Run `swarm run -s "Hello" -n 1 -d` to create a log file
2. Open the Electron app
3. Verify that log files appear in the Console panel
4. Verify that clicking "View Log" on an agent works
