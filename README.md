# setupmjr - Cross-platform Setup Utility

[![Test Installation Methods](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml)
[![Test Suite](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml)
[![ShellCheck](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml)

`setupmjr` is a cross-platform Go setup CLI utility that automates the setup of HPC, Bash, R, Git, Slurm, Apptainer, agent, and GitHub Project environments.

## Installation

On Debian or Ubuntu, after configuring the [`apt-miguelrodo`](https://github.com/MiguelRodo/apt-miguelrodo) repository, install the current packaged release with:

```bash
sudo apt-get update
sudo apt-get install -y setupmjr
```

For development, use the local installer or build from source:

```bash
# Local installation script
./install-local.sh

# Go (from source)
go build ./cmd/setupmjr
```

### Installation CI

Pull requests test Debian-package, local Linux, Windows, and source installations using `setupmjr` artefacts built from the PR. Installer tests provide the `repos` dependency locally so an unrelated release or network failure cannot make PR validation fail.

The already-published APT repository is a separate integration check. Run the `Test Installation Methods` workflow manually when validating repository availability and published package dependencies.

## Quick Start

You can use `setupmjr` to configure specific environments quickly. For instance, to set up the master HPC environment:

```bash
setupmjr hpc
```

To inspect agent choices, switch Codex providers, or configure an opt-in subagent:

```bash
setupmjr agent --list
setupmjr agent --auth deepseek  # Store/reuse DeepSeek API credential only
setupmjr agent -p d             # Copilot -> DeepSeek
setupmjr agent -p               # Copilot -> GitHub-managed models
setupmjr agent -c d             # Codex -> DeepSeek
setupmjr agent -c               # Codex -> OpenAI/ChatGPT
setupmjr agent -s               # agy subagent (default)
setupmjr agent -s deepseek      # DeepSeek subagent
```

See [agent.qmd](agent.qmd) for provider switching, snapshots, credentials and subagent behaviour.

Use the top-level `install` command for external tools that `setupmjr` installs or refreshes:

```bash
setupmjr install --repos
setupmjr install --pj
setupmjr install --gh-skill
```

Multiple targets may be installed together:

```bash
setupmjr install --repos --pj --gh-skill
```

`--pj` follows `MiguelRodo/pj`'s floating `v0` release line. `--gh-skill` installs `MiguelRodo/github-projects-skill` for universal agents at user scope from `main`. After installing `repos`, invoke `repos` directly for repository orchestration rather than routing its commands through `setupmjr`.

See [install.qmd](install.qmd) for installation and ownership details.

## Commands and capabilities

- `setupmjr hpc` — Master HPC setup, with options for `scratch`, `apptainer`, `slurm`, `login git`, `git`, and `r`.
- `setupmjr bash` — Manage Bash environments (`rc.d`, `login`).
- `setupmjr r` — Set up R environments (e.g., `radian`).
- `setupmjr git` — Manage Git configuration and authentication.
- `setupmjr repo` — Configure repositories with `readme`, `devcontainer`, and `action`.
- `setupmjr install [--repos] [--pj] [--gh-skill]` — Install or refresh setupmjr-managed external tools. At least one target flag is required.
- `setupmjr agent` — Inspect or configure coding agents: `-a` / `--auth <endpoint>` stores endpoint credentials without changing providers, `-p` / `--copilot-provider` switches Copilot between GitHub-managed models and DeepSeek, `-c` / `--codex-provider` switches the primary Codex provider (`openai` or `deepseek`), `-s` / `--subagent` configures the opt-in subagent (`agy` or `deepseek`), and `--config` / `--list` inspect the current or available configuration.
