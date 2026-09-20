# setupmjr - Cross-platform Setup Utility

[![Test Installation Methods](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml)
[![Test Suite](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml)
[![ShellCheck](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml)

`setupmjr` is a cross-platform Go CLI for configuring shell, R, Git, repository, HPC and coding-agent environments, plus installing the external tools it integrates with.

## Installation

On Debian or Ubuntu, after configuring the [`apt-miguelrodo`](https://github.com/MiguelRodo/apt-miguelrodo) repository:

```bash
sudo apt-get update
sudo apt-get install -y setupmjr
```

For a user-local release install on Linux or macOS:

```bash
git clone https://github.com/MiguelRodo/setupmjr.git
cd setupmjr
bash install-local.sh
```

For development:

```bash
go build ./cmd/setupmjr
```

See [install.qmd](install.qmd) for the supported APT, Scoop, release-installer and source paths.

### Installation CI

Pull requests test Debian-package, local Linux, Windows and source installations using artefacts built from the PR. The release-installer tests separately verify that a fresh installer dispatches the `repos` dependency through `setupmjr install --repos`, then exercise the real PR-built setupmjr binary without depending on the published `repos` release.

The published APT repository is a separate integration check and is exercised by the manually triggered `Test Installation Methods` workflow.

## Quick start

Run the Linux-only master HPC setup with:

```bash
setupmjr hpc
```

Inspect or configure coding agents with:

```bash
setupmjr agent --list
setupmjr agent --auth deepseek
setupmjr agent -p d
setupmjr agent -c d
setupmjr agent -s deepseek
```

See [agent.qmd](agent.qmd) for provider switching, credentials, snapshots and subagent behaviour.

Use the top-level `install` command for external tools managed by setupmjr's installation glue:

```bash
setupmjr install --repos
setupmjr install --pj
setupmjr install --gh-skill
```

Flags may be combined:

```bash
setupmjr install --repos --pj --gh-skill
```

`--pj` follows `MiguelRodo/pj`'s floating `v0` release line. `--gh-skill` installs `MiguelRodo/github-projects-skill` for universal agents at user scope from `main`. After installing `repos`, invoke `repos` directly for repository orchestration.

## Commands and capabilities

- `setupmjr hpc` - Run the Linux-only master HPC setup, or use `scratch`, `apptainer`, `slurm`, `git` and `r` independently.
- `setupmjr shell <shell>` - Configure `rc.d`, executable paths and login/profile integration. `setupmjr bash` is shorthand for `setupmjr shell bash`.
- `setupmjr r` - Configure `radian`, `lintr` and optionally the project-local `switch_r` helper.
- `setupmjr git` - Configure Git identity and GitHub authentication.
- `setupmjr repo` - Configure repository README, devcontainer and released workflow examples.
- `setupmjr install [--repos] [--pj] [--gh-skill]` - Install or refresh setupmjr-managed external tools. At least one target flag is required.
- `setupmjr agent` - Inspect or configure coding-agent authentication, providers and the opt-in subagent.

Run `setupmjr --help` for the complete command syntax, or use the Quarto guides linked above for details.
