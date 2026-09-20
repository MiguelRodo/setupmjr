# setupmjr

A cross-platform setup utility for configuring development environments, repositories, coding agents, and HPC systems.

## Installation

See [install.qmd](install.qmd) for all supported installation methods.

## Agent configuration

Use `setupmjr agent` for coding-agent provider, credential and optional subagent setup:

```bash
setupmjr agent --config
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

- `setupmjr hpc` — Master HPC setup, with options for `scratch`, `apptainer`, `slurm`, `git`, and `r`.
- `setupmjr bash` — Manage Bash environments (`rc.d`, `login`).
- `setupmjr r` — Set up R environments (e.g., `radian`).
- `setupmjr git` — Manage Git configuration and authentication.
- `setupmjr repo` — Configure repositories with `readme`, `devcontainer`, and `action`.
- `setupmjr install [--repos] [--pj] [--gh-skill]` — Install or refresh setupmjr-managed external tools. At least one target flag is required.
- `setupmjr agent` — Inspect or configure coding agents: `-a` / `--auth <endpoint>` stores endpoint credentials without changing providers, `-p` / `--copilot-provider` switches Copilot between GitHub-managed models and DeepSeek, `-c` / `--codex-provider` switches the primary Codex provider (`openai` or `deepseek`), `-s` / `--subagent` configures the opt-in subagent (`agy` or `deepseek`), and `--config` / `--list` inspect the current or available configuration.
