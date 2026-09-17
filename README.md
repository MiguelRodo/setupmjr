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

To install or update the `pj` operator launcher from its floating `v0` release line:

```bash
setupmjr project --pj
```

To install the shared `github-projects` skill for universal agents at user scope:

```bash
setupmjr project --gh-skill
```

Both may be requested together:

```bash
setupmjr project --pj --gh-skill
```

See [project.qmd](project.qmd) for the release and ownership details.

## Commands and capabilities

- `setupmjr hpc` — Master HPC setup, with options for `scratch`, `apptainer`, `slurm`, `login git`, `git`, and `r`.
- `setupmjr bash` — Manage Bash environments (`rc.d`, `login`).
- `setupmjr r` — Set up R environments (e.g., `radian`).
- `setupmjr git` — Manage Git configuration and authentication.
- `setupmjr repo` — Manage repositories (e.g., `readme`, `devcontainer`, `action`, `install repos`).
- `setupmjr agent` — Inspect or configure coding agents: `-a` / `--auth <endpoint>` stores endpoint credentials without changing providers, `-p` / `--copilot-provider` switches Copilot between GitHub-managed models and DeepSeek, `-c` / `--codex-provider` switches the primary Codex provider (`openai` or `deepseek`), `-s` / `--subagent` configures the opt-in subagent (`agy` or `deepseek`), and `--config` / `--list` inspect the current or available configuration.
- `setupmjr project --pj` — Install or update `pj` from `MiguelRodo/pj`'s floating `v0` release tag. It does not install from `pj/main`.
- `setupmjr project --gh-skill` — Run `gh skill install MiguelRodo/github-projects-skill github-projects --agent universal --scope user --force --pin main`.
