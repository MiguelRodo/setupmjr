#!/usr/bin/env bash
set -euo pipefail

scenario="${1:?scenario required}"
bin="${2:?setupmjr binary required}"

root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT
export HOME="$root/home"
export XDG_CONFIG_HOME="$HOME/.config"
export GIT_CONFIG_NOSYSTEM=1
mkdir -p "$HOME" "$root/bin"
export PATH="$(dirname "$bin"):$root/bin:$PATH"

secret_in_managed_files() {
  local token="$1"
  local path
  for path in "$HOME/.gitconfig" "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.bashrc.d/login.sh" "$HOME/.zshrc.d/login.sh"; do
    if [ -f "$path" ] && grep -Fq "$token" "$path"; then
      echo "secret leaked into $path" >&2
      return 0
    fi
  done
  return 1
}

assert_token_fallback() {
  local token="$1"
  local token_file="$HOME/.config/setupmjr/auth/github.token"

  test -f "$token_file"
  test "$(cat "$token_file")" = "$token"
  if [ "$(uname -s)" != "MINGW" ]; then
    test "$(stat -c '%a' "$HOME/.config/setupmjr/auth")" = "700"
    test "$(stat -c '%a' "$token_file")" = "600"
  fi

  if secret_in_managed_files "$token"; then
    exit 1
  fi

  unset GH_TOKEN GITHUB_TOKEN
  creds="$(printf 'protocol=https\nhost=github.com\n\n' | git credential fill)"
  grep -Fq 'username=x-access-token' <<<"$creds"
  grep -Fq "password=$token" <<<"$creds"

  bash -c 'source "$HOME/.bashrc"; test "$GH_TOKEN" = "'"$token"'"'
  zsh_path="$(command -v zsh || true)"
  if [ -n "$zsh_path" ]; then
    "$zsh_path" -c 'source "$HOME/.zshrc"; test "$GH_TOKEN" = "'"$token"'"'
  fi
}

case "$scenario" in
  token-env)
    token='ghp_setupmjr_actions_env_token'
    export GH_TOKEN="$token"
    unset GITHUB_TOKEN || true
    "$bin" git auth </dev/null
    assert_token_fallback "$token"

    # Re-running must not duplicate managed shell blocks or mutate the secret.
    export GH_TOKEN="$token"
    "$bin" git auth </dev/null
    test "$(grep -c '^# setupmjr-github-token:start$' "$HOME/.bashrc")" = "1"
    test "$(grep -c '^# setupmjr-github-token:start$' "$HOME/.zshrc")" = "1"
    test "$(cat "$HOME/.config/setupmjr/auth/github.token")" = "$token"
    ;;

  existing-token)
    token='ghp_setupmjr_actions_existing_token'
    mkdir -p "$HOME/.config/setupmjr/auth"
    chmod 700 "$HOME/.config/setupmjr/auth"
    printf '%s\n' "$token" > "$HOME/.config/setupmjr/auth/github.token"
    chmod 600 "$HOME/.config/setupmjr/auth/github.token"
    unset GH_TOKEN GITHUB_TOKEN || true
    "$bin" git auth </dev/null
    assert_token_fallback "$token"
    ;;

  gh-helper)
    token='ghp_setupmjr_actions_gh_token'
    export FAKE_GH_LOG="$root/gh.log"
    export FAKE_GH_TOKEN="$token"
    cat > "$root/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$FAKE_GH_LOG"
if [ "$1 $2" = "auth status" ]; then
  exit 0
fi
if [ "$1 $2" = "auth setup-git" ]; then
  git config --global credential.https://github.com.helper '!gh auth git-credential'
  exit 0
fi
if [ "$1 $2" = "auth git-credential" ]; then
  operation="${3:-}"
  cat >/dev/null
  if [ "$operation" = "get" ]; then
    printf 'username=oauth2\npassword=%s\n' "$FAKE_GH_TOKEN"
  fi
  exit 0
fi
exit 2
EOF
    chmod +x "$root/bin/gh"
    unset GH_TOKEN GITHUB_TOKEN || true

    "$bin" git auth </dev/null
    grep -Fq 'auth status --hostname github.com' "$FAKE_GH_LOG"
    grep -Fq 'auth setup-git --hostname github.com' "$FAKE_GH_LOG"
    test ! -e "$HOME/.config/setupmjr/auth/github.token"

    creds="$(printf 'protocol=https\nhost=github.com\n\n' | git credential fill)"
    grep -Fq 'username=oauth2' <<<"$creds"
    grep -Fq "password=$token" <<<"$creds"
    ;;

  gh-unavailable-piped-token)
    token='ghp_setupmjr_actions_piped_token'
    # Shadow gh with an unauthenticated implementation and supply a token on stdin.
    cat > "$root/bin/gh" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-} ${2:-}" = "auth status" ]; then
  exit 1
fi
exit 2
EOF
    chmod +x "$root/bin/gh"
    unset GH_TOKEN GITHUB_TOKEN || true
    printf '%s\n' "$token" | "$bin" git auth
    assert_token_fallback "$token"
    ;;

  *)
    echo "unknown scenario: $scenario" >&2
    exit 2
    ;;
esac
