#!/usr/bin/env bash

###############################################################################
# Script Name    : hpc-scratch.sh
# Description    : Configures environment variables to utilize the /scratch
#                  directory for XDG data, cache, Singularity, Apptainer,
#                  renv, and R libraries. This setup is particularly useful
#                  in high-performance computing (HPC) environments where
#                  /scratch provides temporary storage for user data.
#
# Requirements    :
#   - Bash shell
#   - /scratch directory exists and is writable
#   - Hostname matches the pattern ^srvrochpc[0-9]+
#
# Environment Variables Set:
#   - XDG_DATA_HOME        : /scratch/$USER/.local/share
#   - XDG_CACHE_HOME       : /scratch/$USER/.cache
#   - SINGULARITY_CACHE_DIR: /scratch/$USER/.cache/singularity
#   - APPTAINER_CACHE_DIR  : /scratch/$USER/.cache/apptainer
#   - RENV_PATHS_ROOT      : /scratch/$USER/.local/renv
#   - R_LIBS               : /scratch/$USER/.local/lib/R
#
# Author         : Miguel Rodo
# Contact        : miguel.rodo@uct.ac.za
# License        : MIT License
# Version        : 1.0
# Last Modified  : 2024 Nov 07
#
# Notes          :
#   - Ensure that /scratch has sufficient space and appropriate permissions.
#   - This script is intended for use on specific HPC systems as indicated
#     by the hostname pattern.
#   - Modify the hostname pattern in the script if your environment differs.
###############################################################################

create_dir() {
    mkdir -p "$1" || {
        echo "Failed to create directory: $1"
        exit 1
    }
}

use_scratch() {
    # Use XDG relative directories, but
    # relative to /scratch/$USER
    export XDG_DATA_HOME="/scratch/$USER/.local/share"
    export XDG_CACHE_HOME="/scratch/$USER/.cache"

    # Set Apptainer and Singularity
    # Cache directories explicitly to cache
    # (Apptainer, at least, may ignore the XDG env vars)
    export SINGULARITY_CACHE_DIR="/scratch/$USER/.cache/singularity"
    export APPTAINER_CACHE_DIR="/scratch/$USER/.cache/apptainer"
    export APPTAINER_TMPDIR="/scratch/$USER/.cache/apptainer"
    create_dir "$SINGULARITY_CACHE_DIR"
    create_dir "$APPTAINER_CACHE_DIR"
    create_dir "$APPTAINER_TMPDIR"

    # Force renv to use /scratch
    export RENV_CONFIG_PAK_ENABLED=false
    export RENV_CONFIG_SANDBOX_ENABLED=false
    export RENV_PATHS_LIBRARY_ROOT="/scratch/$USER/.local/lib/R/library"
    export RENV_PATHS_CACHE="/scratch/$USER/renv/cache:/renv/cache"
    export RENV_PATHS_ROOT="/scratch/$USER/.local/lib/R/library"
    create_dir "$RENV_PATHS_LIBRARY_ROOT"
    create_dir  "$RENV_PATHS_CACHE"
    create_dir "$RENV_PATHS_ROOT"

    # Force R to use scratch
    export R_LIBS="/scratch/$USER/.local/lib/R"
    create_dir "$R_LIBS"
}

# Don't run if if /scratch/$USER directory does not exist
if [[ -d "/scratch/$USER" ]]; then
    use_scratch
fi
