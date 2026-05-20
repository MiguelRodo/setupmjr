#!/bin/bash
# Usage: determine-bump-type.sh <NEW_VERSION> <PREV_VERSION>
#
# Prints "major", "minor", or "patch" depending on which component changed
# between PREV_VERSION and NEW_VERSION.  Both arguments are in X.Y.Z format
# (without a leading 'v').
#
# Exit status: always 0.

set -euo pipefail

NEW_VERSION="${1:?Usage: determine-bump-type.sh <NEW_VERSION> <PREV_VERSION>}"
PREV_VERSION="${2:?Usage: determine-bump-type.sh <NEW_VERSION> <PREV_VERSION>}"

IFS='.' read -r new_major new_minor new_patch <<< "$NEW_VERSION"
new_major="${new_major:-0}"; new_minor="${new_minor:-0}"; new_patch="${new_patch:-0}"

IFS='.' read -r prev_major prev_minor prev_patch <<< "$PREV_VERSION"
prev_major="${prev_major:-0}"; prev_minor="${prev_minor:-0}"; prev_patch="${prev_patch:-0}"

if [ "$new_major" -gt "$prev_major" ]; then
  echo "major"
elif [ "$new_minor" -gt "$prev_minor" ]; then
  echo "minor"
else
  echo "patch"
fi
