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

install_fake_gh() {
  cat > "$root/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "${FAKE_GH_LOG:?}"
if [ "${1:-} ${2:-}" = "auth status" ]; then
  [ "${FAKE_GH_AUTHENTICATED:-0}" = "1" ]
  exit $?
fi
if [ "${1:-} ${2:-}" = "auth setup-git" ]; then
  [ "${FAKE_GH_AUTHENTICATED:-0}" = "1" ] || exit 1
  git config --global credential.https://github.com.helper '!gh auth git-credential'
  exit 0
fi
if [ "${1:-} ${2:-}" = "auth git-credential" ]; then
  operation="${3:-}"
  cat >/dev/null
  if [ "$operation" = "get" ]; then
    printf 'username=oauth2\npassword=%s\n' "${FAKE_GH_TOKEN:?}"
  fi
  exit 0
fi
exit 2
EOF
  chmod +x "$root/bin/gh"
  export FAKE_GH_LOG="$root/gh.log"
  : > "$FAKE_GH_LOG"
}

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
  test "$(stat -c '%a' "$HOME/.config/setupmjr/auth")" = "700"
  test "$(stat -c '%a' "$token_file")" = "600"

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
    install_fake_gh
    export FAKE_GH_AUTHENTICATED=0
    export FAKE_GH_TOKEN='unused'
    token='ghp_setupmjr_actions_env_token'
    export GH_TOKEN="$token"
    unset GITHUB_TOKEN || true
    "$bin" git auth </dev/null
    assert_token_fallback "$token"

    export GH_TOKEN="$token"
    "$bin" git auth </dev/null
    test "$(grep -c '^# setupmjr-github-token:start$' "$HOME/.bashrc")" = "1"
    test "$(grep -c '^# setupmjr-github-token:start$' "$HOME/.zshrc")" = "1"
    test "$(cat "$HOME/.config/setupmjr/auth/github.token")" = "$token"
    ;;

  existing-token)
    install_fake_gh
    export FAKE_GH_AUTHENTICATED=0
    export FAKE_GH_TOKEN='unused'
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
    install_fake_gh
    token='ghp_setupmjr_actions_gh_token'
    export FAKE_GH_AUTHENTICATED=1
    export FAKE_GH_TOKEN="$token"
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
    install_fake_gh
    export FAKE_GH_AUTHENTICATED=0
    export FAKE_GH_TOKEN='unused'
    token='ghp_setupmjr_actions_piped_token'
    unset GH_TOKEN GITHUB_TOKEN || true
    printf '%s\n' "$token" | "$bin" git auth
    assert_token_fallback "$token"
    ;;

  legacy-login-migration)
    install_fake_gh
    export FAKE_GH_AUTHENTICATED=0
    export FAKE_GH_TOKEN='unused'
    old_token='ghp_setupmjr_legacy_leaked_token'
    new_token='ghp_setupmjr_actions_migrated_token'
    mkdir -p "$HOME/.bashrc.d" "$HOME/.zshrc.d"
    for path in "$HOME/.bashrc.d/login.sh" "$HOME/.zshrc.d/login.sh"; do
      cat > "$path" <<EOF
KEEP_ME=yes
GITHUB_USERNAME="miguel"
GH_TOKEN="$old_token"
HF_TOKEN="keep-huggingface-alone"
EOF
      chmod 755 "$path"
    done
    export GH_TOKEN="$new_token"
    "$bin" git auth </dev/null
    assert_token_fallback "$new_token"
    for path in "$HOME/.bashrc.d/login.sh" "$HOME/.zshrc.d/login.sh"; do
      ! grep -Fq "$old_token" "$path"
      ! grep -q '^GH_TOKEN=' "$path"
      ! grep -q '^GITHUB_USERNAME=' "$path"
      grep -Fq 'KEEP_ME=yes' "$path"
      grep -Fq 'HF_TOKEN="keep-huggingface-alone"' "$path"
    done
    ;;

  fallback-to-gh)
    install_fake_gh
    fallback_token='ghp_setupmjr_actions_fallback_token'
    gh_token='ghp_setupmjr_actions_switched_gh_token'
    export FAKE_GH_AUTHENTICATED=0
    export FAKE_GH_TOKEN="$gh_token"
    export GH_TOKEN="$fallback_token"
    "$bin" git auth </dev/null
    assert_token_fallback "$fallback_token"

    unset GH_TOKEN GITHUB_TOKEN || true
    export FAKE_GH_AUTHENTICATED=1
    : > "$FAKE_GH_LOG"
    "$bin" git auth </dev/null
    grep -Fq 'auth setup-git --hostname github.com' "$FAKE_GH_LOG"
    ! grep -q '^# setupmjr-github-token:start$' "$HOME/.bashrc"
    ! grep -q '^# setupmjr-github-token:start$' "$HOME/.zshrc"
    test "$(cat "$HOME/.config/setupmjr/auth/github.token")" = "$fallback_token"

    creds="$(printf 'protocol=https\nhost=github.com\n\n' | git credential fill)"
    grep -Fq 'username=oauth2' <<<"$creds"
    grep -Fq "password=$gh_token" <<<"$creds"
    ;;

  *)
    echo "unknown scenario: $scenario" >&2
    exit 2
    ;;
esac
