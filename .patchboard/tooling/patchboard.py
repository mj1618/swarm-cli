#!/usr/bin/env python3
"""
patchboard.py — repo-local CLI for repo-native task management.

This tool intentionally keeps the repo as the system of record:
- tasks are markdown with YAML frontmatter under /.patchboard/tasks/
- locks are JSON lease files under /.patchboard/state/locks/
- board config is YAML under /.patchboard/planning/boards/

Typical usage:

  python .patchboard/tooling/patchboard.py validate
  python .patchboard/tooling/patchboard.py index

Lock operations (claim/renew/release) are NOT available via CLI.
Agents must use PR/issue comments which trigger the lock broker workflow:

  /patchboard claim T-0001
  /patchboard renew T-0001
  /patchboard release T-0001 --status review

See lock_broker.py and the patchboard-lock-broker GitHub Action.
"""
from __future__ import annotations

import argparse
import dataclasses
import json
import os
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone, timedelta
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

import yaml
from dateutil.parser import isoparse
from jsonschema import Draft202012Validator

FM_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)

ALLOWED_STATUSES = {"todo", "ready", "in_progress", "blocked", "review", "done"}


@dataclass
class Task:
    id: str
    path: Path
    frontmatter: Dict[str, Any]
    body: str

    @property
    def status(self) -> str:
        return str(self.frontmatter.get("status", "")).strip()

    @property
    def title(self) -> str:
        return str(self.frontmatter.get("title", "")).strip()

    @property
    def depends_on(self) -> List[str]:
        deps = self.frontmatter.get("depends_on") or []
        return list(deps)

    @property
    def owner(self) -> Optional[str]:
        v = self.frontmatter.get("owner")
        return None if v in (None, "", "null") else str(v)

    @property
    def task_type(self) -> str:
        return str(self.frontmatter.get("type", "task")).strip()

    @property
    def parent_epic(self) -> Optional[str]:
        v = self.frontmatter.get("parent_epic")
        return None if v in (None, "", "null") else str(v)

    @property
    def children(self) -> List[str]:
        kids = self.frontmatter.get("children") or []
        return list(kids)

    def with_updates(self, **updates: Any) -> "Task":
        fm = dict(self.frontmatter)
        fm.update(updates)
        return Task(id=self.id, path=self.path, frontmatter=fm, body=self.body)


@dataclass
class Lock:
    task_id: str
    path: Path
    data: Dict[str, Any]

    @property
    def claimed_by(self) -> str:
        return str(self.data.get("claimed_by", "")).strip()

    @property
    def lease_expires_at(self) -> datetime:
        return isoparse(str(self.data["lease_expires_at"]))

    @property
    def is_expired(self) -> bool:
        try:
            return self.lease_expires_at <= now_utc()
        except Exception:
            return True


def now_utc() -> datetime:
    return datetime.now(timezone.utc)


def repo_root_from_here() -> Path:
    # .patchboard/tooling/patchboard.py -> parents: [tooling, .patchboard, repo]
    return Path(__file__).resolve().parents[2]


def read_text(p: Path) -> str:
    return p.read_text(encoding="utf-8")


def write_text(p: Path, content: str) -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content.rstrip() + "\n", encoding="utf-8")


def load_json(p: Path) -> Dict[str, Any]:
    return json.loads(read_text(p))


def dump_json(p: Path, obj: Any) -> None:
    write_text(p, json.dumps(obj, indent=2, sort_keys=False))


def load_yaml(p: Path) -> Any:
    return yaml.safe_load(read_text(p))


def dump_yaml_str(obj: Any) -> str:
    # Keep task frontmatter human-readable.
    return yaml.safe_dump(obj, sort_keys=False).strip()


def parse_frontmatter(md: str) -> Tuple[Dict[str, Any], str]:
    m = FM_RE.match(md)
    if not m:
        raise ValueError("missing YAML frontmatter (expected leading '--- ... ---')")
    fm_raw = m.group(1)
    fm = yaml.safe_load(fm_raw) or {}
    body = md[m.end():]
    if not isinstance(fm, dict):
        raise ValueError("frontmatter must be a YAML mapping/object")
    return fm, body


def render_task_markdown(frontmatter: Dict[str, Any], body: str) -> str:
    return f"---\n{dump_yaml_str(frontmatter)}\n---\n\n{body.lstrip()}"


def task_dir(repo_root: Path) -> Path:
    return repo_root / ".patchboard" / "tasks"


def locks_dir(repo_root: Path) -> Path:
    return repo_root / ".patchboard" / "state" / "locks"


def board_path(repo_root: Path) -> Path:
    return repo_root / ".patchboard" / "planning" / "boards" / "kanban.yaml"


def schemas_dir(repo_root: Path) -> Path:
    return repo_root / ".patchboard" / "tooling" / "schemas"


def load_schema(repo_root: Path, name: str) -> Draft202012Validator:
    schema = load_json(schemas_dir(repo_root) / name)
    return Draft202012Validator(schema)


def git_current_branch(repo_root: Path) -> Optional[str]:
    try:
        out = subprocess.check_output(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"], cwd=str(repo_root), stderr=subprocess.DEVNULL
        ).decode("utf-8", errors="replace").strip()
        return out or None
    except Exception:
        return None


def discover_tasks(repo_root: Path) -> Dict[str, Task]:
    tasks: Dict[str, Task] = {}
    base = task_dir(repo_root)
    if not base.exists():
        return tasks

    for task_folder in sorted(base.iterdir()):
        if not task_folder.is_dir():
            continue
        # Skip template folders (starting with _) and hidden folders (starting with .)
        if task_folder.name.startswith("_") or task_folder.name.startswith("."):
            continue
        task_md = task_folder / "task.md"
        if not task_md.exists():
            continue
        raw = read_text(task_md)
        fm, body = parse_frontmatter(raw)
        tid = str(fm.get("id") or task_folder.name).strip()
        tasks[tid] = Task(id=tid, path=task_md, frontmatter=fm, body=body)
    return tasks


def discover_archived_tasks(repo_root: Path) -> Dict[str, Task]:
    """Discover tasks in the .archived folder."""
    tasks: Dict[str, Task] = {}
    archived_dir = task_dir(repo_root) / ".archived"
    if not archived_dir.exists():
        return tasks

    for task_folder in sorted(archived_dir.iterdir()):
        if not task_folder.is_dir():
            continue
        task_md = task_folder / "task.md"
        if not task_md.exists():
            continue
        raw = read_text(task_md)
        fm, body = parse_frontmatter(raw)
        tid = str(fm.get("id") or task_folder.name).strip()
        tasks[tid] = Task(id=tid, path=task_md, frontmatter=fm, body=body)
    return tasks


def discover_all_tasks(repo_root: Path) -> Dict[str, Task]:
    """Discover all tasks including archived ones."""
    tasks = discover_tasks(repo_root)
    tasks.update(discover_archived_tasks(repo_root))
    return tasks


def load_lock(repo_root: Path, task_id: str) -> Optional[Lock]:
    p = locks_dir(repo_root) / f"{task_id}.lock.json"
    if not p.exists():
        return None
    data = load_json(p)
    return Lock(task_id=task_id, path=p, data=data)


def is_lock_unexpired(lock: Lock) -> bool:
    try:
        return not lock.is_expired
    except Exception:
        return False


def human_dt(dt: datetime) -> str:
    return dt.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")


def validate(repo_root: Path, *, verbose: bool = False) -> int:
    """Return exit code."""
    errors: List[str] = []
    warnings: List[str] = []

    tasks = discover_tasks(repo_root)
    # Include archived tasks for reference validation (deps, parent_epic can reference archived tasks)
    all_tasks = discover_all_tasks(repo_root)

    # Board config
    bpath = board_path(repo_root)
    if not bpath.exists():
        errors.append(f"Missing board config: {bpath}")
        board = None
    else:
        try:
            board = load_yaml(bpath)
            load_schema(repo_root, "board.schema.json").validate(board)
        except Exception as e:
            errors.append(f"Invalid board config {bpath}: {e}")
            board = None

    # Schemas
    task_validator = load_schema(repo_root, "task.schema.json")

    # Validate tasks
    for tid, task in sorted(tasks.items(), key=lambda kv: kv[0]):
        # Folder name should match id by convention
        folder = task.path.parent.name
        if folder != tid:
            warnings.append(f"{tid}: folder name '{folder}' does not match frontmatter id '{tid}'")

        try:
            task_validator.validate(task.frontmatter)
        except Exception as e:
            errors.append(f"{tid}: invalid frontmatter: {e}")

        st = task.status
        if st not in ALLOWED_STATUSES:
            errors.append(f"{tid}: invalid status '{st}'")

        # Dependencies must exist (can reference archived tasks)
        for dep in task.depends_on:
            if dep not in all_tasks:
                errors.append(f"{tid}: depends_on references missing task '{dep}'")

    # Dependency rule (baseline): deps must be done before ready/in_progress
    # Note: archived tasks are considered "done" for dependency purposes
    for tid, task in sorted(tasks.items(), key=lambda kv: kv[0]):
        if task.status in {"ready", "in_progress"}:
            for dep in task.depends_on:
                dep_task = all_tasks.get(dep)
                # Archived tasks (not in active tasks) are considered satisfied
                if dep_task and dep in tasks and dep_task.status != "done":
                    errors.append(
                        f"{tid}: status '{task.status}' requires deps done, but '{dep}' is '{dep_task.status}'"
                    )

    # Note: Task locking is now done via PRs (see policies.md), not lock files

    # Epic relationship validation
    for tid, task in sorted(tasks.items(), key=lambda kv: kv[0]):
        # Validate parent_epic references (can reference archived epics)
        if task.parent_epic:
            if task.parent_epic not in all_tasks:
                errors.append(f"{tid}: parent_epic references missing task '{task.parent_epic}'")
            elif task.parent_epic in tasks:
                # Only check type if epic is not archived
                parent = tasks[task.parent_epic]
                if parent.task_type != "epic":
                    warnings.append(
                        f"{tid}: parent_epic '{task.parent_epic}' has type '{parent.task_type}', expected 'epic'"
                    )

        # Validate children references (for epics) - can reference archived tasks
        for child_id in task.children:
            if child_id not in all_tasks:
                errors.append(f"{tid}: children references missing task '{child_id}'")

    # Check bidirectional consistency (children <-> parent_epic)
    for tid, task in sorted(tasks.items(), key=lambda kv: kv[0]):
        if task.task_type == "epic":
            # Check that children have parent_epic pointing back
            for child_id in task.children:
                if child_id in tasks:
                    child = tasks[child_id]
                    if child.parent_epic != tid:
                        warnings.append(
                            f"{tid}: child '{child_id}' has parent_epic='{child.parent_epic}', expected '{tid}'"
                        )
            # Check for tasks with parent_epic pointing here but not in children list
            for other_id, other_task in tasks.items():
                if other_task.parent_epic == tid and other_id not in task.children:
                    warnings.append(
                        f"{tid}: task '{other_id}' has parent_epic='{tid}' but is not in children list"
                    )

    # Detect circular epic references (epic A -> epic B -> epic A)
    def find_epic_cycle(start_id: str, visited: set, path: list) -> Optional[List[str]]:
        if start_id in visited:
            cycle_start = path.index(start_id)
            return path[cycle_start:] + [start_id]
        if start_id not in tasks:
            return None
        task = tasks[start_id]
        if task.task_type != "epic":
            return None
        visited.add(start_id)
        path.append(start_id)
        for child_id in task.children:
            if child_id in tasks and tasks[child_id].task_type == "epic":
                cycle = find_epic_cycle(child_id, visited.copy(), path.copy())
                if cycle:
                    return cycle
        return None

    for tid, task in sorted(tasks.items(), key=lambda kv: kv[0]):
        if task.task_type == "epic":
            cycle = find_epic_cycle(tid, set(), [])
            if cycle:
                cycle_str = " -> ".join(cycle)
                errors.append(f"{tid}: circular epic reference detected: {cycle_str}")
                break  # Report only one cycle to avoid duplicates

    if verbose or warnings:
        for w in warnings:
            print(f"WARNING: {w}", file=sys.stderr)

    if errors:
        for e in errors:
            print(f"ERROR: {e}", file=sys.stderr)
        return 1

    if verbose:
        print("OK: validation passed")
    return 0


def claim(repo_root: Path, task_id: str, actor: str, lease_seconds: int, *, allow_steal_expired: bool = True) -> int:
    tasks = discover_tasks(repo_root)
    if task_id not in tasks:
        print(f"ERROR: unknown task '{task_id}'", file=sys.stderr)
        return 1
    task = tasks[task_id]
    if task.status == "done":
        print(f"ERROR: task '{task_id}' is done; refusing to claim", file=sys.stderr)
        return 1

    # Dependency rule baseline
    for dep in task.depends_on:
        dep_task = tasks.get(dep)
        if dep_task and dep_task.status != "done":
            print(
                f"ERROR: cannot claim '{task_id}': dependency '{dep}' is '{dep_task.status}' (must be done)",
                file=sys.stderr,
            )
            return 1

    existing = load_lock(repo_root, task_id)
    if existing and is_lock_unexpired(existing):
        if existing.claimed_by == actor:
            print(f"ERROR: '{task_id}' is already locked by you; use renew instead", file=sys.stderr)
        else:
            print(
                f"ERROR: '{task_id}' is already locked by '{existing.claimed_by}' until {human_dt(existing.lease_expires_at)}",
                file=sys.stderr,
            )
        return 1

    if existing and (not is_lock_unexpired(existing)) and (not allow_steal_expired):
        print(f"ERROR: '{task_id}' has an expired lock; stealing disabled", file=sys.stderr)
        return 1

    now = now_utc()
    expires = now + timedelta(seconds=lease_seconds)
    lock_data = {
        "task_id": task_id,
        "claimed_by": actor,
        "claimed_at": human_dt(now),
        "lease_seconds": int(lease_seconds),
        "lease_expires_at": human_dt(expires),
        "branch": git_current_branch(repo_root),
        "last_renewed_at": human_dt(now),
    }
    lock_path = locks_dir(repo_root) / f"{task_id}.lock.json"
    dump_json(lock_path, lock_data)

    # Update task frontmatter
    updated = task.with_updates(
        status="in_progress",
        owner=actor,
        updated_at=now.date().isoformat(),
    )
    md = render_task_markdown(updated.frontmatter, updated.body)
    write_text(task.path, md)

    print(f"Claimed {task_id} as {actor}")
    print(f"  Lock: {lock_path}")
    print(f"  Expires: {human_dt(expires)}")
    print("")
    print("Next:")
    print(f"  git add {lock_path.as_posix()} {task.path.as_posix()}")
    print(f'  git commit -m "Claim {task_id}"')
    return 0


def renew(repo_root: Path, task_id: str, actor: str, lease_seconds: int) -> int:
    lock = load_lock(repo_root, task_id)
    if not lock:
        print(f"ERROR: no lock exists for '{task_id}' (claim first)", file=sys.stderr)
        return 1
    if lock.claimed_by != actor:
        print(f"ERROR: lock for '{task_id}' is owned by '{lock.claimed_by}', not '{actor}'", file=sys.stderr)
        return 1
    if lock.is_expired:
        print(f"ERROR: lock for '{task_id}' is expired; re-claim instead of renew", file=sys.stderr)
        return 1

    now = now_utc()
    expires = now + timedelta(seconds=lease_seconds)
    lock.data["lease_seconds"] = int(lease_seconds)
    lock.data["lease_expires_at"] = human_dt(expires)
    lock.data["last_renewed_at"] = human_dt(now)
    dump_json(lock.path, lock.data)

    print(f"Renewed {task_id} as {actor}")
    print(f"  Expires: {human_dt(expires)}")
    print("")
    print("Next:")
    print(f"  git add {lock.path.as_posix()}")
    print(f'  git commit -m "Renew {task_id} lock"')
    return 0


def release(repo_root: Path, task_id: str, actor: str, *, force: bool = False, new_status: Optional[str] = None) -> int:
    lock = load_lock(repo_root, task_id)
    if not lock:
        print(f"ERROR: no lock exists for '{task_id}'", file=sys.stderr)
        return 1
    if (lock.claimed_by != actor) and (not force):
        print(
            f"ERROR: lock for '{task_id}' is owned by '{lock.claimed_by}', not '{actor}' (use --force to override)",
            file=sys.stderr,
        )
        return 1

    # Delete lock
    try:
        lock.path.unlink()
    except Exception as e:
        print(f"ERROR: failed to delete lock file: {e}", file=sys.stderr)
        return 1

    # Optionally update task status
    tasks = discover_tasks(repo_root)
    task_path = task_dir(repo_root) / task_id / "task.md"
    if task_id in tasks:
        task = tasks[task_id]
        updates: Dict[str, Any] = {"updated_at": now_utc().date().isoformat()}
        if new_status:
            if new_status not in ALLOWED_STATUSES:
                print(f"ERROR: invalid status '{new_status}'", file=sys.stderr)
                return 1
            updates["status"] = new_status
            # If marking done, keep owner (audit) or clear? We'll keep owner.
        updated = task.with_updates(**updates)
        md = render_task_markdown(updated.frontmatter, updated.body)
        write_text(task.path, md)

    print(f"Released lock for {task_id}")
    print("")
    print("Next:")
    if new_status and task_path.exists():
        print(f"  git add {lock.path.as_posix()} {task_path.as_posix()}")
        print(f'  git commit -m "Release {task_id} lock ({new_status})"')
    else:
        print(f"  git add {lock.path.as_posix()}")
        print(f'  git commit -m "Release {task_id} lock"')
    return 0


def generate_index(repo_root: Path) -> int:
    tasks = discover_tasks(repo_root)
    out_tasks: List[Dict[str, Any]] = []
    for tid, task in sorted(tasks.items(), key=lambda kv: kv[0]):
        lock = load_lock(repo_root, tid)
        lock_info = None
        if lock:
            try:
                lock_info = {
                    "claimed_by": lock.claimed_by,
                    "lease_expires_at": lock.data.get("lease_expires_at"),
                    "expired": bool(lock.is_expired),
                }
            except Exception:
                lock_info = {"claimed_by": lock.data.get("claimed_by"), "lease_expires_at": lock.data.get("lease_expires_at"), "expired": True}

        out_tasks.append(
            {
                "id": tid,
                "title": task.title,
                "type": task.frontmatter.get("type"),
                "status": task.status,
                "priority": task.frontmatter.get("priority"),
                "owner": task.owner,
                "labels": task.frontmatter.get("labels") or [],
                "depends_on": task.depends_on,
                "acceptance": task.frontmatter.get("acceptance") or [],
                "path": str(task.path.relative_to(repo_root)),
                "lock": lock_info,
            }
        )

    out = {
        "generated_at": human_dt(now_utc()),
        "tasks": out_tasks,
    }
    out_path = repo_root / ".patchboard" / "state" / "index.json"
    dump_json(out_path, out)

    print(f"Wrote index: {out_path}")
    return 0


def parse_body_sections(body: str) -> Dict[str, str]:
    """Parse markdown body into sections (Context, Plan, Notes).

    Args:
        body: The markdown body content after frontmatter

    Returns:
        Dict with keys 'context', 'plan', 'notes' containing section content
    """
    sections: Dict[str, str] = {"context": "", "plan": "", "notes": ""}
    current_section: Optional[str] = None
    current_content: List[str] = []

    for line in body.split("\n"):
        # Check for section headers (case-insensitive)
        line_lower = line.lower().strip()
        if line_lower.startswith("## context"):
            if current_section:
                sections[current_section] = "\n".join(current_content).strip()
            current_section = "context"
            current_content = []
        elif line_lower.startswith("## plan"):
            if current_section:
                sections[current_section] = "\n".join(current_content).strip()
            current_section = "plan"
            current_content = []
        elif line_lower.startswith("## notes"):
            if current_section:
                sections[current_section] = "\n".join(current_content).strip()
            current_section = "notes"
            current_content = []
        elif current_section:
            current_content.append(line)

    # Don't forget the last section
    if current_section:
        sections[current_section] = "\n".join(current_content).strip()

    return sections


def _extract_match_context(text: str, query: str, context_chars: int = 50) -> Optional[str]:
    """Extract a snippet of text around the first match of query.

    Args:
        text: The text to search in
        query: The search query (case-insensitive)
        context_chars: Number of characters to include before and after match

    Returns:
        A snippet with the match and surrounding context, or None if no match
    """
    if not text or not query:
        return None

    text_lower = text.lower()
    query_lower = query.lower()
    pos = text_lower.find(query_lower)

    if pos == -1:
        return None

    # Calculate start and end positions with context
    start = max(0, pos - context_chars)
    end = min(len(text), pos + len(query) + context_chars)

    snippet = text[start:end]

    # Add ellipsis if truncated
    if start > 0:
        snippet = "..." + snippet
    if end < len(text):
        snippet = snippet + "..."

    return snippet


def search_tasks(
    repo_root: Path,
    query: str,
    status: Optional[str] = None,
    priority: Optional[str] = None,
    owner: Optional[str] = None,
    label: Optional[str] = None,
    limit: int = 20,
) -> List[Dict[str, Any]]:
    """Search tasks by query string with optional filters.

    Searches across task titles, descriptions (context/plan/notes), and acceptance criteria.
    Search is case-insensitive and supports partial matches.

    Args:
        repo_root: Path to the repository root
        query: Search query string
        status: Optional filter by status
        priority: Optional filter by priority
        owner: Optional filter by owner
        label: Optional filter by label
        limit: Maximum number of results to return (default 20)

    Returns:
        List of matching tasks with match context
    """
    tasks = discover_tasks(repo_root)
    results: List[Dict[str, Any]] = []

    query_lower = query.lower() if query else ""

    for tid, task in sorted(tasks.items(), key=lambda kv: kv[0]):
        # Apply filters first
        if status and task.status != status:
            continue
        if priority and task.frontmatter.get("priority") != priority:
            continue
        if owner and task.owner != owner:
            continue
        if label and label not in (task.frontmatter.get("labels") or []):
            continue

        # Parse body sections for search
        body_sections = parse_body_sections(task.body)

        # If no query, include all filtered tasks
        if not query_lower:
            results.append({
                "id": tid,
                "title": task.title,
                "status": task.status,
                "priority": task.frontmatter.get("priority"),
                "owner": task.owner,
                "labels": task.frontmatter.get("labels") or [],
                "path": str(task.path.relative_to(repo_root)),
                "match_context": None,
                "match_field": None,
            })
            if len(results) >= limit:
                break
            continue

        # Search in various fields
        match_context: Optional[str] = None
        match_field: Optional[str] = None

        # Search in title
        if query_lower in task.title.lower():
            match_context = task.title
            match_field = "title"

        # Search in context (body)
        if not match_context:
            context = body_sections.get("context", "")
            ctx = _extract_match_context(context, query)
            if ctx:
                match_context = ctx
                match_field = "context"

        # Search in plan (body)
        if not match_context:
            plan = body_sections.get("plan", "")
            ctx = _extract_match_context(plan, query)
            if ctx:
                match_context = ctx
                match_field = "plan"

        # Search in notes (body)
        if not match_context:
            notes = body_sections.get("notes", "")
            ctx = _extract_match_context(notes, query)
            if ctx:
                match_context = ctx
                match_field = "notes"

        # Search in acceptance criteria
        if not match_context:
            acceptance = task.frontmatter.get("acceptance") or []
            for criterion in acceptance:
                if query_lower in criterion.lower():
                    match_context = criterion
                    match_field = "acceptance"
                    break

        # If match found, add to results
        if match_context:
            results.append({
                "id": tid,
                "title": task.title,
                "status": task.status,
                "priority": task.frontmatter.get("priority"),
                "owner": task.owner,
                "labels": task.frontmatter.get("labels") or [],
                "path": str(task.path.relative_to(repo_root)),
                "match_context": match_context,
                "match_field": match_field,
            })
            if len(results) >= limit:
                break

    return results


def search(
    repo_root: Path,
    query: str,
    status: Optional[str] = None,
    priority: Optional[str] = None,
    owner: Optional[str] = None,
    label: Optional[str] = None,
    limit: int = 20,
) -> int:
    """Search tasks and print results."""
    results = search_tasks(
        repo_root,
        query=query,
        status=status,
        priority=priority,
        owner=owner,
        label=label,
        limit=limit,
    )

    if not results:
        print("No matching tasks found.")
        return 0

    print(f"Found {len(results)} matching task(s):\n")

    for result in results:
        task_id = result["id"]
        title = result["title"] or "Untitled"
        task_status = result["status"] or "unknown"
        task_priority = result["priority"] or "-"
        task_owner = result["owner"] or "-"
        labels = ", ".join(result["labels"]) if result["labels"] else "-"

        print(f"  {task_id}: {title}")
        print(f"    Status: {task_status}  Priority: {task_priority}  Owner: {task_owner}")
        if result["labels"]:
            print(f"    Labels: {labels}")
        if result["match_context"] and result["match_field"]:
            print(f"    Match in {result['match_field']}: {result['match_context']}")
        print()

    return 0


def archived_dir(repo_root: Path) -> Path:
    """Return the path to the archived tasks directory."""
    return task_dir(repo_root) / ".archived"


def archive_task(repo_root: Path, task_id: str) -> int:
    """Archive a task by moving it to .patchboard/tasks/.archived/.

    Args:
        repo_root: Path to the repository root
        task_id: The task ID to archive (e.g., "T-0001")

    Returns:
        Exit code (0 on success, 1 on error)
    """
    tasks = discover_tasks(repo_root)
    if task_id not in tasks:
        # Check if already archived
        arch_path = archived_dir(repo_root) / task_id
        if arch_path.exists():
            print(f"ERROR: task '{task_id}' is already archived", file=sys.stderr)
            return 1
        print(f"ERROR: unknown task '{task_id}'", file=sys.stderr)
        return 1

    task = tasks[task_id]
    task_folder = task.path.parent  # T-XXXX folder containing task.md

    # Check if task has an active lock
    lock = load_lock(repo_root, task_id)
    if lock and not lock.is_expired:
        print(
            f"ERROR: cannot archive '{task_id}': task has active lock by '{lock.claimed_by}' until {human_dt(lock.lease_expires_at)}",
            file=sys.stderr,
        )
        return 1

    # Create .archived directory if it doesn't exist
    arch_dir = archived_dir(repo_root)
    arch_dir.mkdir(parents=True, exist_ok=True)

    # Destination path
    dest_path = arch_dir / task_id

    if dest_path.exists():
        print(f"ERROR: destination '{dest_path}' already exists", file=sys.stderr)
        return 1

    # Move the task folder
    try:
        shutil.move(str(task_folder), str(dest_path))
    except Exception as e:
        print(f"ERROR: failed to move task folder: {e}", file=sys.stderr)
        return 1

    # Remove lock file if it exists (expired locks)
    lock_path = locks_dir(repo_root) / f"{task_id}.lock.json"
    if lock_path.exists():
        try:
            lock_path.unlink()
        except Exception as e:
            print(f"WARNING: failed to remove lock file: {e}", file=sys.stderr)

    print(f"Archived {task_id}")
    print(f"  From: {task_folder}")
    print(f"  To: {dest_path}")
    print("")
    print("Next:")
    print(f"  git add -A .patchboard/tasks/")
    print(f'  git commit -m "Archive {task_id}"')
    return 0


def unarchive_task(repo_root: Path, task_id: str) -> int:
    """Unarchive a task by moving it from .archived/ back to tasks/.

    Args:
        repo_root: Path to the repository root
        task_id: The task ID to unarchive (e.g., "T-0001")

    Returns:
        Exit code (0 on success, 1 on error)
    """
    # Check if task exists in archive
    arch_path = archived_dir(repo_root) / task_id
    if not arch_path.exists():
        # Check if task is already in active tasks
        tasks = discover_tasks(repo_root)
        if task_id in tasks:
            print(f"ERROR: task '{task_id}' is not archived (it exists in active tasks)", file=sys.stderr)
            return 1
        print(f"ERROR: archived task '{task_id}' not found", file=sys.stderr)
        return 1

    # Destination path
    dest_path = task_dir(repo_root) / task_id

    if dest_path.exists():
        print(f"ERROR: destination '{dest_path}' already exists", file=sys.stderr)
        return 1

    # Move the task folder
    try:
        shutil.move(str(arch_path), str(dest_path))
    except Exception as e:
        print(f"ERROR: failed to move task folder: {e}", file=sys.stderr)
        return 1

    print(f"Unarchived {task_id}")
    print(f"  From: {arch_path}")
    print(f"  To: {dest_path}")
    print("")
    print("Next:")
    print(f"  git add -A .patchboard/tasks/")
    print(f'  git commit -m "Unarchive {task_id}"')
    return 0


def main(argv: Optional[List[str]] = None) -> int:
    """
    CLI entry point for patchboard utilities.

    NOTE: Lock operations (claim/renew/release) are intentionally NOT exposed here.
    Agents must use PR/issue comments (/patchboard claim T-XXXX) which trigger
    the lock broker GitHub Action. See lock_broker.py for that workflow.
    """
    argv = argv if argv is not None else sys.argv[1:]
    repo_root = repo_root_from_here()

    parser = argparse.ArgumentParser(prog="patchboard.py")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_val = sub.add_parser("validate", help="Validate /.patchboard tasks/locks/board invariants")
    p_val.add_argument("--verbose", action="store_true")

    p_idx = sub.add_parser("index", help="Generate /.patchboard/state/index.json")
    # no args

    p_search = sub.add_parser("search", help="Search tasks by query string")
    p_search.add_argument("query", nargs="?", default="", help="Search query (searches title, context, plan, notes, acceptance)")
    p_search.add_argument("--status", help="Filter by status (todo, ready, in_progress, blocked, review, done)")
    p_search.add_argument("--priority", help="Filter by priority (P0, P1, P2, P3, P4)")
    p_search.add_argument("--owner", help="Filter by owner")
    p_search.add_argument("--label", help="Filter by label")
    p_search.add_argument("--limit", type=int, default=20, help="Max results (default: 20)")

    p_archive = sub.add_parser("archive", help="Archive a task (move to .archived/ folder)")
    p_archive.add_argument("task_id", help="Task ID to archive (e.g., T-0001)")

    p_unarchive = sub.add_parser("unarchive", help="Unarchive a task (restore from .archived/ folder)")
    p_unarchive.add_argument("task_id", help="Task ID to unarchive (e.g., T-0001)")

    args = parser.parse_args(argv)

    if args.cmd == "validate":
        return validate(repo_root, verbose=args.verbose)
    if args.cmd == "index":
        return generate_index(repo_root)
    if args.cmd == "search":
        return search(
            repo_root,
            query=args.query,
            status=args.status,
            priority=args.priority,
            owner=args.owner,
            label=args.label,
            limit=args.limit,
        )
    if args.cmd == "archive":
        return archive_task(repo_root, args.task_id)
    if args.cmd == "unarchive":
        return unarchive_task(repo_root, args.task_id)

    print("ERROR: unknown command", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
