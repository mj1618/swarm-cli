#!/usr/bin/env bash
# validate.sh - Helper script to run patchboard validation.
#
# Handles Python venv creation and dependency installation automatically.
# Designed to work in both local development and CI (GitHub Actions).
#
# Usage:
#   bash .patchboard/tooling/validate.sh [--verbose]
#
# The venv is created at .patchboard-local/venv/ (gitignored by convention).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VENV_DIR="$REPO_ROOT/.patchboard-local/venv"

# Find Python 3
PYTHON=""
for candidate in python3 python; do
    if command -v "$candidate" &>/dev/null; then
        version=$("$candidate" -c "import sys; print(sys.version_info.major)")
        if [ "$version" -ge 3 ]; then
            PYTHON="$candidate"
            break
        fi
    fi
done

if [ -z "$PYTHON" ]; then
    echo "ERROR: Python 3 is required but not found." >&2
    exit 1
fi

# Create or reuse venv
if [ ! -f "$VENV_DIR/bin/activate" ]; then
    echo "Creating validation venv at $VENV_DIR..."
    mkdir -p "$(dirname "$VENV_DIR")"
    "$PYTHON" -m venv "$VENV_DIR"
fi

# Install/ensure deps (quiet to reduce CI noise)
"$VENV_DIR/bin/pip" install -q pyyaml jsonschema python-dateutil

# Run validation, forwarding any arguments (e.g. --verbose)
"$VENV_DIR/bin/python" "$SCRIPT_DIR/patchboard.py" validate "$@"
