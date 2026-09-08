# setupmjr - Cross-platform Setup Utility

[![Test Installation Methods](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-installation.yml)
[![Test Suite](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/test-suite.yml)
[![ShellCheck](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml/badge.svg)](https://github.com/MiguelRodo/setupmjr/actions/workflows/shellcheck.yml)

`setupmjr` is a cross-platform Go setup CLI utility that automates the setup of HPC, Bash, R, Git, Slurm, Apptainer, agent, and GitHub Project environments.

## Installation

Install via the provided local script or from source.

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

To configure Codex so that explicitly authorised work can be delegated to Gemini through `agy`:

```bash
setupmjr agent --subagent-agy
```

To install the shared `github-project-admin` skill for universal agents at user scope:

```bash
setupmjr project --gh-skill
```

## Commands and capabilities

- `setupmjr hpc` — Master HPC setup, with options for `scratch`, `apptainer`, `slurm`, `login git`, `git`, and `r`.
- `setupmjr bash` — Manage Bash environments (`rc.d`, `login`).
- `setupmjr r` — Set up R environments (e.g., `radian`).
- `setupmjr git` — Manage Git configuration and authentication.
- `setupmjr repo` — Manage repositories (e.g., `readme`, `devcontainer`, `action`, `install repos`).
- `setupmjr agent --subagent-agy` — Add bounded Codex guidance for opt-in `agy` delegation and the narrow Codex execution rule required to invoke it. Existing Codex instructions are preserved outside the managed block.
- `setupmjr project --gh-skill` — Run `gh skill install MiguelRodo/projects github-project-admin --agent universal --scope user`.
