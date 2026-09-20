# setupmjr - Cross-platform Setup Utility

[![Test Installation Methods](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml)
[![Test Suite](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml)
[![ShellCheck](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml)

`setupmjr` is a cross-platform Go CLI for configuring shell, R, Git, repository, HPC and coding-agent environments. It also provides installation glue for the external tools it integrates with.

## Installation

Use the native package path where available:

```bash
# Debian / Ubuntu, after adding apt-miguelrodo
sudo apt-get update
sudo apt-get install -y setupmjr

# macOS
brew tap MiguelRodo/tap
brew trust MiguelRodo/tap
brew install MiguelRodo/tap/setupmjr
```

```powershell
# Windows
scoop bucket add MiguelRodo https://github.com/MiguelRodo/scoop-bucket
scoop install setupmjr
```

For a user-local Linux or macOS install without sudo:

```bash
git clone https://github.com/MiguelRodo/setupmjr.git
cd setupmjr
bash install-local.sh
```

See [Installation](install.qmd) for repository setup, Windows installer usage, source builds, release assets and the Python/R wrappers.

## Quick start

```bash
setupmjr --help
setupmjr hpc
setupmjr git
setupmjr repo devcontainer
setupmjr agent --list
```

Install external tools through the top-level `install` command:

```bash
setupmjr install --repos --pj --gh-skill
```

`repos`, `pj`, `MiguelRodo/actions` and `github-projects-skill` keep ownership of their own behaviour. `setupmjr` owns the setup and integration glue around them.

## Command groups

- `setupmjr hpc` configures Linux HPC environments and exposes focused `scratch`, `apptainer`, `slurm`, `git` and `r` setup steps.
- `setupmjr shell` configures Bash or Zsh startup files; `setupmjr bash` is shorthand for Bash.
- `setupmjr r` configures `radian`, `lintr` and the optional project-local `switch_r` helper.
- `setupmjr git` configures Git identity and GitHub authentication.
- `setupmjr repo` scaffolds repository files, devcontainers and released workflow examples.
- `setupmjr install` installs or refreshes setupmjr-managed external tools.
- `setupmjr agent` configures coding-agent credentials, providers and opt-in subagents.

`setupmjr --help` is the authoritative top-level syntax. `setupmjr agent --help` gives the detailed agent options.

## Guides

- [Installation](install.qmd)
- [HPC](hpc.qmd)
- [Shell](shell.qmd)
- [R setup](r.qmd)
- [Git](git.qmd)
- [Repository setup](repo.qmd)
- [Agent configuration](agent.qmd)
