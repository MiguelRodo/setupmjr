#!/usr/bin/env bash

dotfiles_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ "$CODESPACES" = "true" ]; then
  echo "Setting up dotfiles for Codespaces"
  "$dotfiles_dir/install-env.sh" codespace
else
  echo "Setting up dotfiles for devcontainer environment"
  "$dotfiles_dir/install-env.sh" dev
fi

