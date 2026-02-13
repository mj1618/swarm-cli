# Fix Settings Config File Path

## Problem

The Electron app is reading/writing settings from the wrong config file path.

**Current (incorrect):**
```typescript
// electron/src/main/index.ts line 604-606
function getConfigFilePath() {
  return path.join(swarmRoot, '.swarm.toml')  // = swarm/.swarm.toml
}
```

**Expected (correct):**
```typescript
function getConfigFilePath() {
  return path.join(swarmRoot, 'swarm.toml')  // = swarm/swarm.toml
}
```

## Evidence

The swarm CLI uses the following config paths:
- Project config: `swarm/swarm.toml` (from `internal/config/config.go` line 216)
- Global config: `~/.config/swarm/config.toml` (from `internal/config/config.go` line 211)

The electron app is incorrectly using `swarm/.swarm.toml` (with a leading dot), which doesn't match the CLI's expected location.

## Secondary Issue

The settings panel also shows incorrect default `logsDir` on line 620-621:
```typescript
logsDir: path.join(os.homedir(), 'swarm', 'logs'),  // = ~/swarm/logs
```

Should be:
```typescript
logsDir: path.join(os.homedir(), '.swarm', 'logs'),  // = ~/.swarm/logs
```

This is the same bug as documented in `fix-logs-directory-path.pending.md`.

## Impact

- Settings changes made in the electron app are saved to the wrong file
- Settings shown in the app won't reflect the actual CLI configuration
- Users will be confused when settings don't persist

## Fix

1. Change line 605 from `'.swarm.toml'` to `'swarm.toml'`
2. Update the comment on line 602
3. Fix the logsDir default on line 620

## Dependencies

None

## Testing

1. Create or modify `swarm/swarm.toml` with specific backend/model settings
2. Open the Electron app settings panel
3. Verify settings match what's in the config file
4. Change a setting and save
5. Verify the change is written to `swarm/swarm.toml`
6. Run `swarm doctor` or `swarm run` to verify CLI picks up the change
