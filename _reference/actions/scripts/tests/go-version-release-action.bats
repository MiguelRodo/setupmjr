#!/usr/bin/env bats

ACTION_FILE="$(cd "$(dirname "$BATS_TEST_FILENAME")/../.." && pwd)/go-version-release/action.yml"
ACTION_README="$(cd "$(dirname "$BATS_TEST_FILENAME")/../.." && pwd)/go-version-release/README.md"

@test "go-version-release action exists and is a composite action" {
  [ -f "$ACTION_FILE" ]
  run grep -F 'using: "composite"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'if [ "${{ runner.os }}" != "Linux" ]; then' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'go-version-release currently supports Linux runners only' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "go-version-release inputs include github_token, apt inputs, and go_version defaults" {
  run grep -F 'github_token:' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'apt_repo_token:' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run awk '
    /^  apt_repo_token:$/ { in_block=1; next }
    in_block && /^    default: ""$/ { found=1; exit 0 }
    in_block && /^  [^ ]/ { exit 1 }
    END { exit found ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'go_version:' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'default: "1.22"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'apt_repo:' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run awk '
    /^  apt_repo:$/ { in_block=1; next }
    in_block && /^    default: ""$/ { found=1; exit 0 }
    in_block && /^  [^ ]/ { exit 1 }
    END { exit found ? 0 : 1 }
  ' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'scoop_repo:' "$ACTION_FILE"
  [ "$status" -ne 0 ]

  run grep -F 'homebrew_tap:' "$ACTION_FILE"
  [ "$status" -ne 0 ]
}

@test "determine version logic enforces mutual exclusivity and uses apply-version-bump script" {
  run grep -F "cannot set both 'version' and 'bump_type'" "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F '$GITHUB_ACTION_PATH/../scripts/apply-version-bump.sh' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "version progression step uses check-version-progression script" {
  run grep -F "if: inputs.version_force != 'true'" "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F '$GITHUB_ACTION_PATH/../scripts/check-version-progression.sh' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "action sets up Go and runs GoReleaser native publishing" {
  run grep -F 'uses: actions/setup-go@v5' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'go-version: ${{ inputs.go_version }}' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'uses: goreleaser/goreleaser-action@v5' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'args: release --clean --skip=announce' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F '--skip=publish' "$ACTION_FILE"
  [ "$status" -ne 0 ]

  run grep -F 'Collect packaged release assets' "$ACTION_FILE"
  [ "$status" -ne 0 ]

  run grep -F 'softprops/action-gh-release@v2' "$ACTION_FILE"
  [ "$status" -ne 0 ]
}

@test "apt publishing is optional and generates structured multi-arch apt metadata from dist debs" {
  run grep -F "if: inputs.apt_repo != ''" "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'apt_repo must be in owner/name format' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F "required command '\$REQUIRED_COMMAND' is not available on the runner" "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F "find dist -type f -name '*.deb'" "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'export GIT_ASKPASS="$ASKPASS_SCRIPT"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'APT_PUSH_TOKEN="${APT_REPO_TOKEN}"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'APT_PUSH_TOKEN="${GITHUB_TOKEN_INPUT}"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'export GIT_TOKEN_FOR_ASKPASS="$APT_PUSH_TOKEN"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F '${GITHUB_SERVER_URL}/${APT_REPO_INPUT}.git' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'publish_deb() {' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'find "$APT_REPO_DIR" \( -path "$APT_REPO_DIR/.git" -prune \) -o \( -type f -name '\''*.deb'\'' -print0 \)' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'DEST_DIR="$APT_REPO_DIR/pool/main/$BUCKET"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'dpkg-scanpackages --multiversion -a "$ARCH" pool /dev/null > "$BINARY_DIR/Packages"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'gzip -9c "$BINARY_DIR/Packages" > "$BINARY_DIR/Packages.gz"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'apt-ftparchive \' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'release dists/stable > dists/stable/Release' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'git push origin HEAD:main' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "apt publishing signs Release when apt_signing_key is provided, otherwise removes stale signed metadata" {
  run grep -F 'APT_SIGNING_KEY: ${{ inputs.apt_signing_key }}' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'APT_SIGNING_KEY_PASSPHRASE: ${{ inputs.apt_signing_key_passphrase }}' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'SIGNING_KEY_FINGERPRINT=""' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'GNUPGHOME=""' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'chmod 700 "$GNUPGHOME"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'export GNUPGHOME' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F "could not extract fingerprint from apt_signing_key" "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F "printf '%s\n' \"\$APT_SIGNING_KEY\" | gpg --batch --import" "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F -- '--passphrase-file "$GPG_PASSPHRASE_FILE"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'rm -f "$GPG_PASSPHRASE_FILE"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F -- '--default-key "$SIGNING_KEY_FINGERPRINT"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F -- '--armor --detach-sign -o dists/stable/Release.gpg dists/stable/Release' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F -- '--clearsign -o dists/stable/InRelease dists/stable/Release' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'rm -f InRelease Release.gpg dists/stable/InRelease dists/stable/Release.gpg' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'rm -rf "$GNUPGHOME"' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "scoop and homebrew custom publishing steps are removed" {
  run grep -F 'Publish Scoop manifest to scoop repository' "$ACTION_FILE"
  [ "$status" -ne 0 ]

  run grep -F 'Publish Homebrew formula to tap repository' "$ACTION_FILE"
  [ "$status" -ne 0 ]

  run grep -F 'scoop_manifest_source' "$ACTION_FILE"
  [ "$status" -ne 0 ]

  run grep -F 'homebrew_formula_source' "$ACTION_FILE"
  [ "$status" -ne 0 ]
}

@test "apt publishing README documents apt_signing_key input and signed metadata behavior" {
  run grep -F '`apt_signing_key`' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '`apt_signing_key_passphrase`' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F 'InRelease' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F 'Release.gpg' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F 'APT_SIGNING_KEY' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '/etc/apt/keyrings/' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F 'signed-by=' "$ACTION_README"
  [ "$status" -eq 0 ]
}

@test "tag handling validates existing tag points to HEAD" {
  run grep -F 'git fetch --no-tags origin "refs/tags/${TAG}:refs/tags/${TAG}"' "$ACTION_FILE"
  [ "$status" -eq 0 ]

  run grep -F 'already exists on origin but points to' "$ACTION_FILE"
  [ "$status" -eq 0 ]
}

@test "go-version-release has dedicated README with checkout guidance" {
  [ -f "$ACTION_README" ]

  run grep -F 'fetch-depth: 0' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F 'supports Linux runners only' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '`apt_repo`' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '`apt_repo_token`' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F "push:" "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F "tags:" "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '`goreleaser_config`' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F 'apt_repo: ${{ inputs.apt_repo }}' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F 'apt_repo_token: ${{ secrets.APT_REPO_TOKEN }}' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '`goreleaser/goreleaser-action`' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '`vX` and `vX.Y`' "$ACTION_README"
  [ "$status" -eq 0 ]

  run grep -F '`brews:` and `scoops:` blocks' "$ACTION_README"
  [ "$status" -eq 0 ]
}
