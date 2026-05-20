#!/bin/bash
# Usage: check-version-progression.sh <NEW_VERSION> <PREV_VERSION>
#
# Validates that NEW_VERSION is exactly one semver increment (major, minor, or
# patch) ahead of PREV_VERSION.  Both arguments are in X.Y.Z format (without a
# leading 'v').
#
# Exit status: 0 when the progression is valid, 1 otherwise.

set -euo pipefail

NEW_VERSION="${1:?Usage: check-version-progression.sh <NEW_VERSION> <PREV_VERSION>}"
PREV_VERSION="${2:?Usage: check-version-progression.sh <NEW_VERSION> <PREV_VERSION>}"

IFS='.' read -r new_major new_minor new_patch <<< "$NEW_VERSION"
new_major="${new_major:-0}"; new_minor="${new_minor:-0}"; new_patch="${new_patch:-0}"

IFS='.' read -r prev_major prev_minor prev_patch <<< "$PREV_VERSION"
prev_major="${prev_major:-0}"; prev_minor="${prev_minor:-0}"; prev_patch="${prev_patch:-0}"

next_major=$((prev_major + 1))
next_minor=$((prev_minor + 1))
next_patch=$((prev_patch + 1))

# Reject versions that are not strictly ahead of the previous release
if [ "$new_major" -lt "$prev_major" ] || \
   { [ "$new_major" -eq "$prev_major" ] && [ "$new_minor" -lt "$prev_minor" ]; } || \
   { [ "$new_major" -eq "$prev_major" ] && [ "$new_minor" -eq "$prev_minor" ] && [ "$new_patch" -le "$prev_patch" ]; }; then
  echo "Error: version '$NEW_VERSION' is not ahead of the previous release '$PREV_VERSION'." >&2
  echo "  Set version_force: 'true' to override this guard." >&2
  exit 1
fi

# Accept only a single major, minor, or patch increment
VALID=false
if [ "$new_major" -eq "$next_major" ] && [ "$new_minor" -eq 0 ] && [ "$new_patch" -eq 0 ]; then
  VALID=true
elif [ "$new_major" -eq "$prev_major" ] && [ "$new_minor" -eq "$next_minor" ] && [ "$new_patch" -eq 0 ]; then
  VALID=true
elif [ "$new_major" -eq "$prev_major" ] && [ "$new_minor" -eq "$prev_minor" ] && [ "$new_patch" -eq "$next_patch" ]; then
  VALID=true
fi

if [ "$VALID" = false ]; then
  echo "Error: version '$NEW_VERSION' is more than one increment ahead of '$PREV_VERSION'." >&2
  echo "  Valid next versions: ${next_major}.0.0, ${prev_major}.${next_minor}.0, ${prev_major}.${prev_minor}.${next_patch}" >&2
  echo "  Set version_force: 'true' to override this guard." >&2
  exit 1
fi

echo "Version check passed: '$PREV_VERSION' → '$NEW_VERSION'"
