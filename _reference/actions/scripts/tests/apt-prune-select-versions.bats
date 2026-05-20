#!/usr/bin/env bats

# Unit tests for scripts/apt-prune-select-versions.sh

SCRIPT="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)/apt-prune-select-versions.sh"

# ---------------------------------------------------------------------------
# Test helpers
# ---------------------------------------------------------------------------

setup() {
  REPO_DIR="$(mktemp -d)"
  MOCK_BIN="$(mktemp -d)"

  # Mock dpkg-deb that extracts Package / Version / Architecture from the
  # filename convention used in these tests: <name>_<version>_<arch>.deb
  cat > "$MOCK_BIN/dpkg-deb" << 'EOF'
#!/usr/bin/env bash
# Called as: dpkg-deb -f <filepath> <field>
FILEPATH="$2"
FIELD="$3"
BASENAME="$(basename "$FILEPATH" .deb)"
IFS='_' read -r NAME VERSION ARCH REST <<< "$BASENAME"
case "$FIELD" in
  Package)      printf '%s\n' "$NAME"    ;;
  Version)      printf '%s\n' "$VERSION" ;;
  Architecture) printf '%s\n' "$ARCH"    ;;
esac
EOF
  chmod +x "$MOCK_BIN/dpkg-deb"
  export PATH="$MOCK_BIN:$PATH"

  mkdir -p "$REPO_DIR/pool/main/m"
}

teardown() {
  rm -rf "$REPO_DIR" "$MOCK_BIN"
}

# Create an empty .deb stub at the correct pool path
mk_deb() {
  local name="$1" version="$2" arch="$3"
  local bucket="${name:0:1}"
  mkdir -p "$REPO_DIR/pool/main/$bucket"
  touch "$REPO_DIR/pool/main/$bucket/${name}_${version}_${arch}.deb"
}

# Run the script and collect output lines into array OUT_LINES
run_select() {
  run bash "$SCRIPT" "$1" "$REPO_DIR"
}

# ---------------------------------------------------------------------------
# Argument validation
# ---------------------------------------------------------------------------

@test "fails with no arguments" {
  run bash "$SCRIPT"
  [ "$status" -ne 0 ]
}

@test "fails with unknown retention policy" {
  run bash "$SCRIPT" bogus "$REPO_DIR"
  [ "$status" -ne 0 ]
  [[ "$output" == *"retention must be one of"* ]]
}

@test "accepts policy name case-insensitively" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 2.0.0 amd64

  run bash "$SCRIPT" "LATEST" "$REPO_DIR"
  [ "$status" -eq 0 ]

  run bash "$SCRIPT" "Latest-Per-Major" "$REPO_DIR"
  [ "$status" -eq 0 ]
}

@test "exits 0 with no output when pool is empty" {
  run_select latest
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

# ---------------------------------------------------------------------------
# latest – keep only the absolute most-recent version
# ---------------------------------------------------------------------------

@test "latest: keeps only the newest version, removes all older ones" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 1.1.0 amd64
  mk_deb myapp 1.2.0 amd64

  run_select latest
  [ "$status" -eq 0 ]
  [[ "$output" == *"myapp_1.0.0_amd64.deb"* ]]
  [[ "$output" == *"myapp_1.1.0_amd64.deb"* ]]
  [[ "$output" != *"myapp_1.2.0_amd64.deb"* ]]
}

@test "latest: keeps single version without removing anything" {
  mk_deb myapp 2.3.4 amd64

  run_select latest
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "latest: removes all arches of old versions" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 1.0.0 arm64
  mk_deb myapp 2.0.0 amd64
  mk_deb myapp 2.0.0 arm64

  run_select latest
  [ "$status" -eq 0 ]
  [[ "$output" == *"myapp_1.0.0_amd64.deb"* ]]
  [[ "$output" == *"myapp_1.0.0_arm64.deb"* ]]
  [[ "$output" != *"myapp_2.0.0_amd64.deb"* ]]
  [[ "$output" != *"myapp_2.0.0_arm64.deb"* ]]
}

# ---------------------------------------------------------------------------
# latest-per-minor – keep the highest patch for each major.minor
# ---------------------------------------------------------------------------

@test "latest-per-minor: keeps newest patch per minor series" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 1.0.1 amd64
  mk_deb myapp 1.0.2 amd64
  mk_deb myapp 1.1.0 amd64
  mk_deb myapp 1.1.3 amd64

  run_select latest-per-minor
  [ "$status" -eq 0 ]
  # Old 1.0.x patches removed
  [[ "$output" == *"myapp_1.0.0_amd64.deb"* ]]
  [[ "$output" == *"myapp_1.0.1_amd64.deb"* ]]
  # Latest patches kept
  [[ "$output" != *"myapp_1.0.2_amd64.deb"* ]]
  [[ "$output" != *"myapp_1.1.3_amd64.deb"* ]]
  # Intermediate 1.1.x removed
  [[ "$output" == *"myapp_1.1.0_amd64.deb"* ]]
}

@test "latest-per-minor: keeps single version per minor without removing anything" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 2.0.0 amd64

  run_select latest-per-minor
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "latest-per-minor: handles multiple majors" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 1.0.1 amd64
  mk_deb myapp 2.0.0 amd64
  mk_deb myapp 2.0.1 amd64

  run_select latest-per-minor
  [ "$status" -eq 0 ]
  [[ "$output" == *"myapp_1.0.0_amd64.deb"* ]]
  [[ "$output" == *"myapp_2.0.0_amd64.deb"* ]]
  [[ "$output" != *"myapp_1.0.1_amd64.deb"* ]]
  [[ "$output" != *"myapp_2.0.1_amd64.deb"* ]]
}

# ---------------------------------------------------------------------------
# latest-per-major – keep the highest minor.patch for each major
# ---------------------------------------------------------------------------

@test "latest-per-major: keeps only the newest minor series per major" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 1.1.0 amd64
  mk_deb myapp 1.2.0 amd64
  mk_deb myapp 2.0.0 amd64
  mk_deb myapp 2.1.0 amd64

  run_select latest-per-major
  [ "$status" -eq 0 ]
  [[ "$output" == *"myapp_1.0.0_amd64.deb"* ]]
  [[ "$output" == *"myapp_1.1.0_amd64.deb"* ]]
  [[ "$output" == *"myapp_2.0.0_amd64.deb"* ]]
  [[ "$output" != *"myapp_1.2.0_amd64.deb"* ]]
  [[ "$output" != *"myapp_2.1.0_amd64.deb"* ]]
}

@test "latest-per-major: keeps single major version without removing anything" {
  mk_deb myapp 1.5.3 amd64

  run_select latest-per-major
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "latest-per-major: handles patches within the kept minor series" {
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 1.0.1 amd64
  mk_deb myapp 1.1.0 amd64
  mk_deb myapp 1.1.2 amd64

  run_select latest-per-major
  [ "$status" -eq 0 ]
  # Old minor series removed entirely
  [[ "$output" == *"myapp_1.0.0_amd64.deb"* ]]
  [[ "$output" == *"myapp_1.0.1_amd64.deb"* ]]
  # Older patch of kept series also removed
  [[ "$output" == *"myapp_1.1.0_amd64.deb"* ]]
  # Latest kept
  [[ "$output" != *"myapp_1.1.2_amd64.deb"* ]]
}

# ---------------------------------------------------------------------------
# Multiple packages
# ---------------------------------------------------------------------------

@test "handles multiple distinct packages independently" {
  mk_deb alpha 1.0.0 amd64
  mk_deb alpha 2.0.0 amd64
  mk_deb beta  1.0.0 amd64
  mk_deb beta  1.1.0 amd64

  run_select latest
  [ "$status" -eq 0 ]
  [[ "$output" == *"alpha_1.0.0_amd64.deb"* ]]
  [[ "$output" != *"alpha_2.0.0_amd64.deb"* ]]
  [[ "$output" == *"beta_1.0.0_amd64.deb"* ]]
  [[ "$output" != *"beta_1.1.0_amd64.deb"* ]]
}

# ---------------------------------------------------------------------------
# Architecture independence
# ---------------------------------------------------------------------------

@test "treats each arch independently within the same package" {
  # amd64 has two versions; arm64 has one
  mk_deb myapp 1.0.0 amd64
  mk_deb myapp 2.0.0 amd64
  mk_deb myapp 2.0.0 arm64

  run_select latest
  [ "$status" -eq 0 ]
  [[ "$output" == *"myapp_1.0.0_amd64.deb"* ]]
  [[ "$output" != *"myapp_2.0.0_amd64.deb"* ]]
  [[ "$output" != *"myapp_2.0.0_arm64.deb"* ]]
}
