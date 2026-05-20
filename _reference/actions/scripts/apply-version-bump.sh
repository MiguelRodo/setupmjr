#!/bin/bash
# Usage: apply-version-bump.sh <BUMP_TYPE> <CURRENT_VERSION>
#
# Computes and prints the next version string (X.Y.Z) obtained by applying
# BUMP_TYPE to CURRENT_VERSION.
#
# BUMP_TYPE:       major | minor | patch
# CURRENT_VERSION: X.Y.Z  (leading 'v' is stripped automatically)
#
# Exit status: 0 on success, 1 on invalid BUMP_TYPE.

set -euo pipefail

BUMP_TYPE="${1:?Usage: apply-version-bump.sh <BUMP_TYPE> <CURRENT_VERSION>}"
CURRENT_VERSION="${2:?Usage: apply-version-bump.sh <BUMP_TYPE> <CURRENT_VERSION>}"

# Strip optional leading 'v'
CURRENT_VERSION="${CURRENT_VERSION#v}"

IFS='.' read -r cur_major cur_minor cur_patch <<< "$CURRENT_VERSION"
cur_major="${cur_major:-0}"
cur_minor="${cur_minor:-0}"
cur_patch="${cur_patch:-0}"

case "$BUMP_TYPE" in
  major) echo "$((cur_major + 1)).0.0" ;;
  minor) echo "${cur_major}.$((cur_minor + 1)).0" ;;
  patch) echo "${cur_major}.${cur_minor}.$((cur_patch + 1))" ;;
  *)
    echo "Error: BUMP_TYPE must be major, minor, or patch. Got: '${BUMP_TYPE}'" >&2
    exit 1
    ;;
esac
