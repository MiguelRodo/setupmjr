#!/usr/bin/env bash
# Exercise the workflow's consumer check without network access or publication.
set -euo pipefail
cd "$(dirname "$0")/../.."
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT
export TEST_DIR="$test_dir"
awk '
  /- name: Verify published APT version/ { step = 1 }
  step && /        run: \|/ { body = 1; next }
  body { sub(/^          /, ""); print }
' .github/workflows/go-version-release.yml > "$test_dir/verify.sh"
test -s "$test_dir/verify.sh"

# Keep the real shell control flow and filesystem operations; mock external tools.
curl() { echo public-key; }
gpg() { cat; }
sudo() { "$@"; }
dpkg() { return 0; }
sleep() { :; }
apt-get() {
  echo update >> "$TEST_DIR/updates"
  [[ "$SCENARIO" != bad-signature ]] || return 100
  if [[ "$SCENARIO" == delayed && $(wc -l < "$TEST_DIR/updates") -lt 2 ]]; then
    return 100
  fi
}
apt-cache() {
  local package="${*: -1}"
  echo "$package" >> "$TEST_DIR/queries"
  [[ "$package" == "setupmjr:amd64=0.8.0" || "$package" == "setupmjr:arm64=0.8.0" ]] || return 100
  [[ "$SCENARIO" != stale ]] || return 100
  [[ "$SCENARIO" != missing-arm64 || "$package" != *arm64* ]] || return 100
}
export -f curl gpg sudo dpkg sleep apt-get apt-cache
export VERSION=0.8.0 APT_REPO=MiguelRodo/apt-miguelrodo
export GITHUB_STEP_SUMMARY="$test_dir/summary"
for SCENARIO in fresh delayed stale missing-arm64 bad-signature; do
  export SCENARIO
  : > "$test_dir/updates"
  : > "$test_dir/queries"
  : > "$test_dir/summary"
  result=0
  bash "$test_dir/verify.sh" > "$test_dir/output" 2>&1 || result=$?
  case "$SCENARIO" in
    fresh|delayed)
      test "$result" -eq 0
      test -s "$test_dir/summary"
      grep -q 'setupmjr:arm64=0.8.0' "$test_dir/queries"
      ;;
    *)
      test "$result" -ne 0
      test ! -s "$test_dir/summary"
      test "$(wc -l < "$test_dir/updates")" -eq 20
      grep -q '::error::' "$test_dir/output"
      ;;
  esac
  echo "PASS: $SCENARIO"
done

