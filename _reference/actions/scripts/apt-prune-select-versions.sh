#!/bin/bash
# apt-prune-select-versions.sh
#
# Reads .deb files under <repo-dir>/pool and writes to stdout the relative
# paths (from <repo-dir>) of files that should be REMOVED based on the
# chosen retention policy.
#
# Usage: apt-prune-select-versions.sh <retention> <repo-dir>
#
#   retention  One of (case-insensitive):
#              latest            – keep only the newest version per (package, arch) pair
#              latest-per-minor  – keep the highest patch for each major.minor per (package, arch)
#              latest-per-major  – keep the highest minor.patch for each major per (package, arch)
#   repo-dir   Root of the apt repository (the pool/ subdirectory is searched).
#
# The script exits 0 in all normal cases and prints nothing when there is
# nothing to remove.

set -euo pipefail

RETENTION="${1:?Usage: apt-prune-select-versions.sh <retention> <repo-dir>}"
REPO_DIR="${2:?Usage: apt-prune-select-versions.sh <retention> <repo-dir>}"

# Normalize to lowercase so the argument is case-agnostic.
RETENTION="${RETENTION,,}"

case "$RETENTION" in
  latest | latest-per-minor | latest-per-major) ;;
  *)
    echo "Error: retention must be one of: latest, latest-per-minor, latest-per-major." >&2
    exit 1
    ;;
esac

# ── Build package inventory ──────────────────────────────────────────────────
# Each line: <pkg>\t<version>\t<arch>\t<relpath>
TMP_DATA="$(mktemp)"
cleanup() {
  rm -f "$TMP_DATA"
}
trap cleanup EXIT

while IFS= read -r -d '' FILE; do
  # Make path relative to repo root
  REL_PATH="${FILE#"${REPO_DIR}/"}"

  PKG="$(dpkg-deb -f "$FILE" Package 2>/dev/null | tr -d '\n')" || continue
  VER="$(dpkg-deb -f "$FILE" Version 2>/dev/null | tr -d '\n')" || continue
  ARCH="$(dpkg-deb -f "$FILE" Architecture 2>/dev/null | tr -d '\n')" || continue

  [ -n "$PKG" ]  || continue
  [ -n "$VER" ]  || continue
  [ -n "$ARCH" ] || continue

  # Strip epoch (e.g. "1:2.3.4" → "2.3.4") and debian revision (e.g. "2.3.4-1" → "2.3.4")
  VER="${VER##*:}"
  VER="${VER%%-*}"
  # Strip optional leading 'v'
  VER="${VER#v}"

  # Only handle plain semver X.Y.Z
  [[ "$VER" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || continue

  printf '%s\t%s\t%s\t%s\n' "$PKG" "$VER" "$ARCH" "$REL_PATH" >> "$TMP_DATA"
done < <(find "${REPO_DIR}/pool" -type f -name '*.deb' -print0 2>/dev/null | sort -z)

[ -s "$TMP_DATA" ] || exit 0

# ── Process each (pkg, arch) group ──────────────────────────────────────────

while IFS=$'\t' read -r COMBO_PKG COMBO_ARCH; do
  # Versions for this group, sorted from oldest to newest
  mapfile -t SORTED_VERS < <(
    awk -F'\t' -v p="$COMBO_PKG" -v a="$COMBO_ARCH" \
      '$1==p && $3==a { print $2 }' "$TMP_DATA" | sort -V | uniq
  )

  [ "${#SORTED_VERS[@]}" -eq 0 ] && continue

  # Determine which versions to keep
  declare -a _KEEP=()

  case "$RETENTION" in
    latest)
      _KEEP=("${SORTED_VERS[-1]}")
      ;;

    latest-per-minor)
      # For each MAJOR.MINOR keep the highest PATCH.
      # Iterating in ascending version order means the last assignment wins.
      declare -A _MINOR_BEST=()
      for V in "${SORTED_VERS[@]}"; do
        _MINOR_BEST["${V%.*}"]="$V"
      done
      mapfile -t _KEEP < <(printf '%s\n' "${_MINOR_BEST[@]}")
      unset _MINOR_BEST
      ;;

    latest-per-major)
      # For each MAJOR keep the highest MINOR.PATCH.
      declare -A _MAJOR_BEST=()
      for V in "${SORTED_VERS[@]}"; do
        _MAJOR_BEST["${V%%.*}"]="$V"
      done
      mapfile -t _KEEP < <(printf '%s\n' "${_MAJOR_BEST[@]}")
      unset _MAJOR_BEST
      ;;
  esac

  # Build a quick-lookup set of kept versions
  declare -A _KEEP_SET=()
  for V in "${_KEEP[@]}"; do
    _KEEP_SET["$V"]=1
  done

  # Output paths of files whose version is NOT in the keep-set
  while IFS=$'\t' read -r _P _V _A _PATH; do
    [ "$_P" = "$COMBO_PKG" ]  || continue
    [ "$_A" = "$COMBO_ARCH" ] || continue
    if [ -z "${_KEEP_SET["$_V"]+x}" ]; then
      printf '%s\n' "$_PATH"
    fi
  done < "$TMP_DATA"

  unset _KEEP _KEEP_SET

done < <(awk -F'\t' '{ print $1 "\t" $3 }' "$TMP_DATA" | sort -u)
