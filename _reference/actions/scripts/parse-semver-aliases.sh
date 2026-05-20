#!/bin/bash
# Usage: parse-semver-aliases.sh <TAG>
#
# Given a container image tag, prints the comma-separated list of SemVer alias
# tags to also push.
#
# Standard SemVer:         vX.Y.Z[...]       → "vX.Y,vX"
# Branch-prefixed SemVer:  {prefix}-vX.Y.Z[...] → "{prefix}-vX.Y,{prefix}-vX"
# Any other tag:           prints an empty string (not an error)
#
# Exit status: always 0.

set -euo pipefail

TAG="${1:?Usage: parse-semver-aliases.sh <TAG>}"
ALIAS_TAGS=""

if [[ "$TAG" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
  MAJOR="${BASH_REMATCH[1]}"
  MINOR="${BASH_REMATCH[2]}"
  ALIAS_TAGS="v${MAJOR}.${MINOR},v${MAJOR}"
elif [[ "$TAG" =~ ^(.+)-v([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
  PREFIX="${BASH_REMATCH[1]}"
  MAJOR="${BASH_REMATCH[2]}"
  MINOR="${BASH_REMATCH[3]}"
  ALIAS_TAGS="${PREFIX}-v${MAJOR}.${MINOR},${PREFIX}-v${MAJOR}"
fi

echo "$ALIAS_TAGS"
