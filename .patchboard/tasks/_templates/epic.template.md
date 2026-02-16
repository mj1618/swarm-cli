---
id: E-XXXX
title: "<epic title>"
type: epic
status: todo            # todo | in_progress | done (explicit, not computed)
priority: P2            # P0..P3
owner: null             # human handle or agent id; advisory only
labels:
  - epic
depends_on: []          # other epics or tasks this epic depends on
children: []            # list of child task IDs (T-XXXX) - keep in sync with tasks' parent_epic
acceptance:
  - "<epic-level acceptance criteria - high-level outcomes>"
created_at: YYYY-MM-DD
updated_at: YYYY-MM-DD
---

## Context

Why does this epic exist? What problem are we solving? Link to vision/design docs.

## Scope

What's included in this epic? List the major deliverables or phases:

1. **Phase/Area 1** (T-XXXX) - Brief description
2. **Phase/Area 2** (T-XXXX, T-XXXX) - Brief description
3. **Phase/Area 3** (T-XXXX) - Brief description

## Out of scope

What explicitly will NOT be addressed by this epic (to avoid scope creep):

- Item 1 - may be addressed in future epic
- Item 2 - out of scope for this project

## Notes

<!-- Usage guidance:
- Children field: List all task IDs that belong to this epic
- Each child task should have parent_epic: E-XXXX pointing back
- Epic status is set manually (not computed from children)
- Progress (X of Y done) is calculated at display time from child statuses
- Use depends_on for epic-level dependencies (blocking other epics/tasks)
-->
