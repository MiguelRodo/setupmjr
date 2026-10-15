#!/usr/bin/env bash
# Print the release bump implied by conventional commits since the latest
# vX.Y.Z tag: major for breaking changes, minor for feat, patch for fix/perf.
# Prints nothing when no commit warrants a release (docs, ci, chore, ...).
set -euo pipefail
last_tag="$(git describe --tags --abbrev=0 --match 'v[0-9]*.[0-9]*.[0-9]*' 2>/dev/null || true)"
range="${last_tag:+$last_tag..}HEAD"
bump=""
while IFS= read -r -d '' message; do
  subject="${message%%$'\n'*}"
  if [[ "$subject" =~ ^[a-z]+(\([^\)]*\))?!: ]] || grep -q '^BREAKING[ -]CHANGE:' <<<"$message"; then
    echo major; exit 0
  elif [[ "$subject" =~ ^feat(\([^\)]*\))?: ]]; then
    bump=minor
  elif [[ "$subject" =~ ^(fix|perf)(\([^\)]*\))?: && -z "$bump" ]]; then
    bump=patch
  fi
done < <(git log -z --format=%B "$range")
echo "$bump"
