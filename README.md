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

See [Installation](install.qmd) for repository setup, Windows installer usage, source builds and release assets.

## Quick start

```bash
setupmjr --help
setupmjr hpc
setupmjr git
setupmjr repo devcontainer
setupmjr agent --list
setupmjr agent doctor
```

Install external tools through the top-level `install` command:

```bash
setupmjr install --repos --pj --gh-skill
setupmjr install agents
setupmjr install claude
setupmjr install codex
setupmjr install copilot
```

`setupmjr install agents` installs all three coding-agent CLIs through their native vendor installers, skipping those already on `PATH`. It reports each result, continues after failures and returns an error if any installation fails. Authentication, API credentials and configured model providers are unchanged.

`repos`, `pj`, `MiguelRodo/actions` and `github-projects-skill` keep ownership of their own behaviour. `setupmjr` owns the setup and integration glue around them.

## Command groups

- `setupmjr hpc` configures Linux HPC environments and exposes focused `scratch`, `apptainer`, `slurm`, `git` and `r` setup steps.
- `setupmjr shell` configures Bash or Zsh startup files; `setupmjr bash` is shorthand for Bash.
- `setupmjr r` configures `radian`, `lintr` and the optional project-local `switch_r` helper.
- `setupmjr git` configures Git identity and GitHub authentication.
- `setupmjr repo` scaffolds repository files, devcontainers and released workflow examples.
- `setupmjr install` installs or refreshes setupmjr-managed external tools and can invoke the official Claude Code, Codex and Copilot CLI installers.
- `setupmjr agent` configures coding-agent credentials, providers and opt-in subagents. `setupmjr agent update <claude|codex|copilot>` updates one installed CLI; `setupmjr agent update --all` attempts every supported CLI on PATH. See [supported update paths](agent.qmd#update-installed-clis).

`setupmjr --help` is the authoritative top-level syntax. `setupmjr agent --help` gives the detailed agent options.

## Guides

- [Installation](install.qmd)
- [HPC](hpc.qmd)
- [Shell](shell.qmd)
- [R setup](r.qmd)
- [Git](git.qmd)
- [Repository setup](repo.qmd)
- [Agent configuration](agent.qmd)

## Releases

Conventional commits on `main` update a Release Please PR; merging that PR approves
release timing. Ordinary merges do not publish packages. `fix:` proposes a patch,
`feat:` a minor, and a breaking change (`!` or `BREAKING CHANGE:`) a major, including
before 1.0. The manifest starts at the existing `v0.7.5` release. Review the proposed
version and changelog before merging; do not enable automatic merging for release PRs.

[Go Version and Release](.github/workflows/go-version-release.yml) passes the approved
version and exact release commit to the pinned `MiguelRodo/actions/go-version-release`
action. That action owns tag validation, GoReleaser, package artefacts, signing,
APT publication and floating tags. The existing GoReleaser configuration is unchanged.
The workflow then uses APT against only the published repository, checking signatures,
index hashes and the exact `setupmjr` version for both `amd64` and `arm64`. It waits up
to 20 times at 15-second intervals for GitHub Pages, with a ten-minute step timeout. A green workflow and its verification summary are
the publication signal: the GitHub release can exist before packaging or APT succeeds.
After verification, users with the repository configured can run
`sudo apt update && sudo apt upgrade` without editing either repository.

Repository setup (no workflows need to be dispatched for configuration):

- Allow GitHub Actions to create pull requests. The proposal job needs contents,
  issues and pull-request write permissions. Optionally provide `RELEASE_PLEASE_TOKEN`
  (a scoped PAT or GitHub App token) so generated PRs trigger required CI checks;
  PRs created with the fallback `GITHUB_TOKEN` do not trigger those checks. See the
  [Release Please token guidance](https://github.com/googleapis/release-please-action#other-actions-on-release-please-prs).
- Keep `APT_REPO_TOKEN` with write access to `MiguelRodo/apt-miguelrodo` and permission
  to trigger its Pages publication. Keep `GPG_PRIVATE_KEY` matching the public
  `KEY.gpg` served there; optionally set `GPG_PASSPHRASE` for an encrypted key.
  Missing signing credentials fail the release job rather than publishing unsigned metadata.
- Retain `SCOOP_TAP_TOKEN` and `HOMEBREW_TAP_TOKEN` for the existing package channels.
  The APT repository must already serve its signed `stable/main` distribution via
  GitHub Pages; this workflow consumes that existing publication setup.

For recovery, inspect the failing step (tagging, GoReleaser, signing, push or APT
verification). Fix the credentials or external publication failure and **re-run failed
jobs** on that run. Re-running the entire workflow can skip publication because
Release Please has already created the release. The deliberate `workflow_dispatch`
path remains available: select the existing exact `vX.Y.Z` tag for a retry, set
`version` to `X.Y.Z`, and retain the APT/config defaults. A fresh manual release can
instead use an exact `version` or `bump_type`, never both. Use `version_force` only
for an intentional progression override; it does not permit moving an existing tag.
Do not retry an old version after a newer release without reviewing progression
and floating-tag effects. Keep the manifest aligned with any manual release before
the next release PR. Failed partial uploads may require inspection of the existing
GitHub release assets before retrying; do not delete or move published version tags.

The repository-local APT step is a consumer smoke check, not a second publisher.
A follow-up in `MiguelRodo/actions` should make signed publication mandatory when
requested, and offer post-publication verification (public base URL, package,
version, architectures, bounded retries), completing before floating tags advance.
It should also define recovery for partially uploaded release assets. Until then,
setupmjr checks the public indexes after the existing action returns; floating tags
may already have advanced when a Pages verification fails.
