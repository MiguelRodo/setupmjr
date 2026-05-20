#!/usr/bin/env bash
# install-env.sh — unified installer for hpc, wsl, dev, linux

set -euo pipefail

# -----------------------------------------------------------------------------
# Usage and argument parsing
# -----------------------------------------------------------------------------
usage() {
  cat <<EOF
Usage: $0 [env]

[env] (optional): One of:
  hpc       - HPC setup
  linux     - Linux setup (not in a container)
  wsl       - WSL setup (not in a container)
  dev       - Devcontainer setup inside Linux/WSL
  codespace - Devcontainer setup inside Codespace
  mac       - MacOS setup

If [env] is omitted:
  - If run inside a GitHub Codespace (CODESPACES=true), defaults to: codespace
  - Otherwise, defaults to: dev

Examples:
  $0 hpc
  $0 linux
  $0 wsl
  $0 dev
  $0 codespace
  $0 mac
  $0            # Uses default logic (see above)

EOF
  exit 1
}

parse_args() {
  echo "Parsing arguments…"
  if [[ $# -eq 0 ]]; then
    if [[ "${CODESPACES:-}" == "true" ]]; then
      echo "No environment argument supplied, but running in a GitHub Codespace (CODESPACES=true)."
      dotfiles_env="codespace"
    else
      echo "No environment argument supplied. Defaulting to 'dev' environment."
      dotfiles_env="dev"
    fi
  else
    dotfiles_env="$1"
  fi
  case "$dotfiles_env" in
    hpc|linux|wsl|dev|codespace|codespaces|mac) ;;
    *)
      echo "Error: invalid environment '$dotfiles_env'." >&2
      usage
      ;;
  esac
  if [[ "$dotfiles_env" == "codespaces" ]]; then
    dotfiles_env="codespace"
  fi
  echo "Arguments parsed successfully. Environment set to '$dotfiles_env'."
}


# -----------------------------------------------------------------------------
# Ensure ~/.bashrc will source ~/.bashrc.d/*.sh fragments
# -----------------------------------------------------------------------------
ensure_shell_rc_sourcing() {
  echo "Ensuring shell rc files source fragment files…"

  local rc fragment_dir fragment_line rc_type
  fragment_dir="$(get_fragment_dir)"

  if [[ "$dotfiles_env" == "mac" ]]; then
    rc="$HOME/.zshrc"
    fragment_line='for i in $HOME/.zshrc.d/*; do [ -r "$i" ] && source "$i"; done'
    rc_type="~/.zshrc"
  else
    rc="$HOME/.bashrc"
    fragment_line='for i in $HOME/.bashrc.d/*; do [ -r "$i" ] && source "$i"; done'
    rc_type="~/.bashrc"
  fi

  [[ -e "$rc" ]] || touch "$rc"
  mkdir -p "$fragment_dir"

  if ! grep -Fq "$fragment_dir" "$rc"; then
    echo "$fragment_line" >> "$rc"
    echo "Added $fragment_dir sourcing to $rc"
  fi

  echo "$rc_type is set up to source $fragment_dir/* fragments."
}

# -----------------------------------------------------------------------------
# Create required directories
# -----------------------------------------------------------------------------
prepare_directories() {
  mkdir -p "$(get_fragment_dir)" "$HOME/.local/bin"
}

get_fragment_dir() {
  if [[ "$dotfiles_env" == "mac" ]]; then
    echo "$HOME/.zshrc.d"
  else
    echo "$HOME/.bashrc.d"
  fi
}

# -----------------------------------------------------------------------------
# Convert CRLF→LF and chmod +x in dotfiles
# -----------------------------------------------------------------------------
normalise_dotfiles() {
  echo "Normalising line endings and permissions in dotfiles…"
  local dotfiles_dir fragment_dir
  dotfiles_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  fragment_dir="$(get_fragment_dir)"

  if [[ "$dotfiles_env" == "mac" ]]; then
    echo "Skipping dos2unix on macOS, setting permissions only…"
    find "$dotfiles_dir/scripts" "$fragment_dir" -type f -exec chmod +x {} +
    echo "Permissions set on macOS."
    return
  fi

  if command -v dos2unix &>/dev/null; then
    echo "Converting line endings to LF using dos2unix…"
    find "$dotfiles_dir/scripts" "$fragment_dir" -type f -exec dos2unix {} +
  else
    echo "dos2unix not found, skipping line ending conversion."
  fi

  find "$dotfiles_dir/scripts" "$fragment_dir" -type f -exec chmod +x {} +
  echo "Permissions set."
  echo "Normalisation complete."
}


# -----------------------------------------------------------------------------
# Copy scripts into ~/.local/bin, skipping per-env patterns
# -----------------------------------------------------------------------------
copy_scripts() {
  echo "Copying scripts to ~/.local/bin…"
  shopt -s nullglob
  local dotfiles_dir
  dotfiles_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  for script in "$dotfiles_dir/scripts/"*; do
    name=$(basename "$script")
    case "$dotfiles_env" in
      hpc)
        cp "$script" "$HOME/.local/bin/" ;;
      wsl)
        [[ "$name" == slurm* ]] && { echo "  Skipping $name"; continue; }
        cp "$script" "$HOME/.local/bin/" ;;
      linux)
        [[ "$name" == slurm* ]] && { echo "  Skipping $name"; continue; }
        cp "$script" "$HOME/.local/bin/" ;;
      dev|mac)
        if [[ "$name" == slurm* || "$name" == apptainer-* ]]; then
          echo "  Skipping $name"; continue
        fi
        cp "$script" "$HOME/.local/bin/" ;;
      codespace)
        if [[ "$name" == slurm* || "$name" == apptainer-* || "$name" == "dotfiles-update" ]]; then
          echo "  Skipping $name"; continue
        fi
        cp "$script" "$HOME/.local/bin/" ;;
    esac
  done
  shopt -u nullglob
  echo "Scripts copied to ~/.local/bin."
}

# -----------------------------------------------------------------------------
# Copy [ba][z]shrc.d fragments, skipping hpc-* outside of hpc, and handle login.sh
# -----------------------------------------------------------------------------
copy_shell_fragments() {
  echo "Copying shell rc fragments…"
  local dotfiles_dir fragment_dir
  dotfiles_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  fragment_dir="$(get_fragment_dir)"
  shopt -s nullglob

  for file in "$dotfiles_dir/bashrc.d/"*; do
    local filename
    filename=$(basename "$file")

    # Skip HPC-only fragments in non-hpc envs
    if [[ "$dotfiles_env" != hpc && "$filename" == hpc-* ]]; then
      echo "  Skipping $filename (HPC-specific)"
      continue
    fi

    # Use correct fragment dir (bashrc.d or zshrc.d)
    local dest_file="$fragment_dir/$filename"
    # Skip if login.sh already exists (do not overwrite)
    if [[ "$filename" == login.sh && -e "$dest_file" ]]; then
      echo "  Skipping $filename (already exists)"
      continue
    fi

    cp "$file" "$dest_file"

    # Only prompt for creds if not in codespace/dev/mac (i.e., interactive setups)
    if [[ "$filename" == login.sh && "$dotfiles_env" != codespace ]]; then
      echo "  Configuring $filename"
      configure_login
    fi
  done
  shopt -u nullglob
  echo "Shell rc fragments copied."
}

# -----------------------------------------------------------------------------
# Prompt to inject GitHub & HuggingFace creds into login.sh
# -----------------------------------------------------------------------------
configure_login() {
  echo "Configuring GitHub & Hugging Face credentials in login.sh…"
  local login_file
  login_file="$(get_fragment_dir)/login.sh"
  [[ -f "$login_file" ]] || return

  # Do not prompt in dev, codespace, or mac environments
  if [[ "$dotfiles_env" == "codespace" ]]; then
    return
  fi

  echo
  echo "Configure GitHub & Hugging Face credentials in login.sh (leave blank to skip):"
  read -p "  GitHub username: " gh_user
  read -p "  GitHub token:    " gh_token
  read -p "  HuggingFace PAT: " hf_pat

  inject_var() {
    local var="$1" val="$2"
    # Replace only commented-out or existing assignments
    sed -i -E "s|^#?\s*${var}=.*|${var}=\"${val}\"|" "$login_file"
  }

  [[ -n "$gh_user" ]] && { inject_var GITHUB_USERNAME "$gh_user"; inject_var GITHUB_USER "$gh_user"; }
  if [[ -n "$gh_token" ]]; then
    inject_var GH_TOKEN "$gh_token"
    inject_var GITHUB_TOKEN "$gh_token"
    inject_var GITHUB_PAT "$gh_token"
  fi
  [[ -n "$hf_pat" ]] && { inject_var HF_PAT "$hf_pat"; inject_var HF_TOKEN "$hf_pat"; }

  echo "login.sh updated."
}

# -----------------------------------------------------------------------------
# Copy hidden config files, sanitising .Renviron for non-hpc environments
# -----------------------------------------------------------------------------
copy_hidden_configs_r() {
  echo "Copying hidden config files…"

  # If in dev or codespace, and R not available, skip copying R configs
  if [[ "$dotfiles_env" == "dev" || "$dotfiles_env" == "codespace" ]]; then
    if ! command -v R &> /dev/null; then
      echo "R not found, skipping all R config files in $dotfiles_env environment."
      return
    fi
  fi

  local files=( .Renviron .lintr .radian_profile )
  local dotfiles_dir dest_dir
  dotfiles_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  dest_dir="$HOME"

  for file in "${files[@]}"; do
    local src="$dotfiles_dir/r/$file" dest="$dest_dir/$file"
    [[ -e "$src" ]] || { echo "  $file not found, skipping"; continue; }

    if [[ ! -e "$dest" ]] || ! cmp -s "$src" "$dest"; then
      echo
      if [[ -e "$dest" ]]; then
        echo "$file exists. Showing diff:"
        diff "$dest" "$src"
        action="overwrite"
      else
        echo "$file does not exist locally."
        action="copy"
      fi

      if auto_approve; then
        echo "Auto-approving $action of $file"
        yn="y"
      else
        while true; do
          read -p "Do you want to $action $file (likely say yes if unsure)? [y/n] " yn
          case "${yn,,}" in
            y|yes|n|no) break ;;
            *) echo "  Please answer y or n." ;;
          esac
        done
      fi

      if [[ "${yn,,}" == "y" || "${yn,,}" == "yes" ]]; then
        # .Renviron special handling
        if [[ "$file" == ".Renviron" && "$dotfiles_env" != "hpc" ]]; then
          sed -E \
            '/^(RENV_PATHS_LIBRARY_ROOT|RENV_PATHS_CACHE|RENV_PATHS_ROOT|R_LIBS)=/d' \
            "$src" > "$dest"
          echo "  Sanitised and copied $file"
        else
          cp "$src" "$dest"
          echo "  Copied $file"
        fi
      else
        echo "  Skipped $file"
      fi
    else
      echo "  $file is up to date."
    fi
  done
  echo "Hidden config files copied."
}


auto_approve() {
  # automatically copy config across for codespaces, 
  # as prompting is not possible in the codespace setup.
  [[ "$dotfiles_env" == "codespace" ]]
}

# -----------------------------------------------------------------------------
# Prompt to set up Git user.name & user.email if missing
# -----------------------------------------------------------------------------
configure_git() {
  echo "Configuring Git line endings, user.name and user.email…"
  echo "Setting Git core.autocrlf to 'input' for better cross-platform compatibility."
  git config --global core.autocrlf input
  local name email def_email username

  name="$(git config --global user.name || git config --system user.name || echo "")"
  email="$(git config --global user.email || git config --system user.email || echo "")"
  username="${GITHUB_USER:-${USER:-$(id -un)}}"

  # Choose default email domain by environment
  case "$dotfiles_env" in
    hpc)       def_email="$username@hpc.auto" ;;
    wsl)       def_email="$username@wsl.local" ;;
    dev)       def_email="$username@dev.local" ;;
    codespace) def_email="$username@codespace.local" ;;
    mac)       def_email="$username@mac.local" ;;
    linux|*)   def_email="$username@linux.local" ;;
  esac

  if auto_approve; then
    echo "Auto-approving Git user configuration if unset"
    # Use sensible defaults without prompting
    [[ -z "$name" ]]  && git config --global user.name  "$username"
    [[ -z "$email" ]] && git config --global user.email "$def_email"
  else
    echo "Prompting for Git user configuration if unset"
    prompt_for() {
      local prompt_text="$1" varname="$2" default="$3"
      read -p "$prompt_text" answer
      answer=${answer:-$default}
      [[ -n "$answer" ]] && git config --global "$varname" "$answer"
    }

    [[ -z "$name" ]]  && prompt_for "Enter Git user.name: "  "user.name"  "$username"
    [[ -z "$email" ]] && prompt_for "Enter Git user.email: " "user.email" "$def_email"
  fi
  echo "Git configuration complete."
}

# -----------------------------------------------------------------------------
# Ensure dotfiles-update knows where the dotfiles repo is
# -----------------------------------------------------------------------------
ensure_dotfiles_dir() {
  local file="${HOME}/.local/bin/dotfiles-update"
  local dirval="${BASH_SOURCE[0]}"

  # Sanity check
  if [[ ! -f "$file" ]]; then
    echo "Error: $file does not exist." >&2
    return 0
  fi

  # Backup in case you need to roll back
  cp "$file" "${file}.bak"

  # Insert the DOTFILES_DIR line immediately after `set -euo pipefail`
  # but only if the next non-blank line does *not* already begin with DOTFILES_DIR=
  sed -i -e '/^set -euo pipefail$/ {
    :loop
    n
    # if this line is blank, skip it
    /^[[:space:]]*$/b loop
    # if it already starts with DOTFILES_DIR=, do nothing and quit
    /^[[:space:]]*DOTFILES_DIR=/b end
    # otherwise, insert our line before it
    i\
DOTFILES_DIR="'"$dirval"'"
    :end
  }' "$file"

  echo "Patched DOTFILES_DIR in $file (backup at ${file}.bak)."
}



# -----------------------------------------------------------------------------
# Print final success message
# -----------------------------------------------------------------------------
print_completion() {
  case "$dotfiles_env" in
    hpc)    echo "HPC setup complete." ;;
    linux)  echo "Linux setup complete." ;;
    wsl)    echo "WSL setup complete." ;;
    dev)    echo "Devcontainer setup complete." ;;
    mac)    echo "Mac setup complete." ;;
    codespace) echo "Codespace setup complete." ;;
  esac
}

# -----------------------------------------------------------------------------
# Unset the dotfiles_env variable to avoid conflicts
# -----------------------------------------------------------------------------
unset_dotfiles_env() {
  unset dotfiles_env
}

# -----------------------------------------------------------------------------
# Main workflow
# -----------------------------------------------------------------------------
main() {
  parse_args "$@"
  ensure_shell_rc_sourcing
  prepare_directories
  normalise_dotfiles
  copy_scripts
  copy_shell_fragments
  copy_hidden_configs_r
  configure_git
  ensure_dotfiles_dir
  print_completion
  unset_dotfiles_env
}

main "$@"
