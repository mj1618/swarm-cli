# Task: Add Configurable Sound Alerts

## Goal

Add configurable sound alerts to the notifications system. The ELECTRON_PLAN.md (line 321) specifies "Sound alerts (configurable)" as part of the notifications feature set. Currently, the app has toast notifications and system notifications, but no audio feedback when agents complete or fail.

## Files

### Create
- `electron/src/renderer/assets/sounds/success.mp3` — short success chime
- `electron/src/renderer/assets/sounds/failure.mp3` — short error/failure tone
- `electron/src/renderer/lib/soundManager.ts` — utility to play sounds with volume control and enable/disable

### Modify
- `electron/src/renderer/components/SettingsPanel.tsx` — add sound alerts toggle and volume slider
- `electron/src/renderer/App.tsx` — trigger sounds on agent completion/failure (near the existing notification logic around agent state transitions)

## Dependencies

- `notifications-system` (done) — sound alerts augment the existing notification flow
- `settings-panel` (done) — settings UI already exists, just needs new controls

## Acceptance Criteria

1. A short sound plays when an agent completes successfully
2. A different short sound plays when an agent fails
3. Settings panel has a "Sound alerts" toggle (on/off) that persists to localStorage
4. Settings panel has a volume slider (0-100%) for sound alert volume
5. Sounds respect the toggle — no audio when disabled
6. Sounds are generated programmatically using Web Audio API (no external audio files needed) — a pleasant short chime for success and a low tone for failure
7. Sound preference persists across app restarts via localStorage

## Notes

- Use the Web Audio API to generate tones programmatically rather than bundling audio files. This avoids asset management and keeps the bundle small.
- The success sound should be a short ascending two-tone chime (~200ms). The failure sound should be a short descending two-tone (~200ms).
- Hook into the same agent state transition logic in App.tsx that already triggers toast and system notifications.
- Store settings in localStorage alongside the existing `swarm-system-notifications` key (e.g., `swarm-sound-alerts-enabled`, `swarm-sound-alerts-volume`).
- Do NOT create actual audio files — update the "Create" list above to just create `soundManager.ts` with Web Audio API synthesis.

## Completion Notes

Implemented by agent cd6dedf1. All acceptance criteria met:

- Created `electron/src/renderer/lib/soundManager.ts` with Web Audio API synthesis (no audio files)
- Success sound: ascending two-tone C5→E5 chime (~200ms)
- Failure sound: descending two-tone E4→C4 (~200ms)
- Settings stored in localStorage (`swarm-sound-alerts-enabled`, `swarm-sound-alerts-volume`)
- Toggle and volume slider added to SettingsPanel under "Sound Alerts" section
- Sounds trigger in App.tsx on agent state transitions (success on completion, failure on crash, nothing on manual kill)
- Preview sound plays when enabling the toggle
- Volume slider only shown when sound alerts are enabled
- Build passes cleanly
