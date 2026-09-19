#!/usr/bin/env bash

# Optional login-shell environment variables managed by setupmjr.
#
# GitHub credentials do not belong in this file. Use `setupmjr git auth`;
# setupmjr will either reuse a persistent GitHub CLI login or keep one private
# token file under ~/.config/setupmjr/auth/.

# Uncomment and set a Hugging Face token only if this shell-level fallback is
# required for your environment.
# HF_PAT="your_huggingface_pat"

export_if_set() {
    local export_name="$1"
    local export_value="$2"

    if [ -n "$export_value" ]; then
        export "$export_name"="$export_value"
    fi
}

export_if_set "HF_PAT" "$HF_PAT"
export_if_set "HF_TOKEN" "$HF_PAT"
