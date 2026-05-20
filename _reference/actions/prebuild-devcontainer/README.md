# Pre-build Dev Container Action

**Pre-build Dev Container** is a composite GitHub Action that speeds up Dev Container startup times. It builds your `.devcontainer/Dockerfile`, tags it with a git tag (and automatically creates SemVer alias tags), pushes it to a container registry, and optionally generates a `prebuild/devcontainer.json` for instant loads.

## Quick Start

Copy this into `.github/workflows/prebuild-devcontainer.yml`:

```yaml
name: 'Pre-build Dev Container'

on:
  push:
    tags:
      - 'v*'
      - '*-v*'
  workflow_dispatch:
    inputs:
      tag:
        description: 'Tag to build (e.g. v1.2.3 or main-v1.2.3)'
        required: true

jobs:
  build:
    runs-on: ubuntu-latest
    concurrency:
      group: ${{ github.workflow }}-${{ github.ref }}
      cancel-in-progress: true
    permissions:
      contents: write
      packages: write

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4

      - name: Run Dev Container Prebuild
        uses: MiguelRodo/actions/prebuild-devcontainer@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          tag: ${{ github.event.inputs.tag }}
```

## Outputs

| Output | Description |
| --- | --- |
| `image_name` | Full image name without tag (e.g. `ghcr.io/owner/repo-main`). |
| `image_tag` | Primary image tag that was built and pushed (e.g. `v1.2.3`). |
| `image_ref` | Full image reference including tag (e.g. `ghcr.io/owner/repo-main:v1.2.3`). |
| `alias_tags` | Comma-separated list of SemVer alias tags also pushed (e.g. `v1.2,v1`). |

## Permissions

The calling workflow needs the following permissions:

| Permission | Why it is needed |
| --- | --- |
| `contents: write` | Push the updated `prebuild/devcontainer.json` back to the repository. |
| `packages: write` | Push the built container image to the GitHub Container Registry (GHCR). |

```yaml
permissions:
  contents: write
  packages: write
```

> **Version pinning:** For stricter supply-chain security, pin to a specific commit SHA instead of a floating tag:
> ```yaml
> uses: MiguelRodo/actions/prebuild-devcontainer@<full-commit-sha>
> ```

## Inputs

| Input | Description | Required | Default |
| --- | --- | --- | --- |
| `github_token` | Token for logging into the container registry and pushing commits. | **Yes** | — |
| `no_cache` | Disable Docker cache during build (`true`/`false`). | No | `false` |
| `create_prebuild_json` | Generate and commit a `prebuild/devcontainer.json` (`true`/`false`). | No | `true` |
| `devcontainer_path` | Path to the `.devcontainer` directory, relative to the repo root. | No | `.devcontainer` |
| `image_name` | Full image name without tag (e.g. `ghcr.io/myorg/myimage`). Defaults to `{registry}/{repo}-{branch}` where `{branch}` is the current branch name (e.g. `ghcr.io/owner/myrepo-main`). The branch name is always used for the image name even when tagging with SemVer. | No | `{registry}/{repo}-{branch}` |
| `tag` | Git tag used as the primary container image tag (e.g. `v1.2.3` or `main-v1.2.3`). Auto-detected from `GITHUB_REF` on tag push. Falls back to `latest` if not set and not a tag push. | No | `""` |
| `registry` | Container registry URL. | No | `ghcr.io` |
| `registry_username` | Username for registry login. | No | Repository owner |
| `version_force` | When `'true'`, skip the version progression check and push the specified version as-is. Useful when jumping more than one increment at a time or when no previous image exists and you want an explicit override. | No | `false` |

## SemVer Alias Tags

When the tag matches a semantic versioning pattern, the action automatically creates and pushes additional alias tags:

| Tag format | Primary tag | Alias tags created |
| --- | --- | --- |
| `vX.Y.Z` (e.g. `v1.2.3`) | `v1.2.3` | `v1.2`, `v1` |
| `{prefix}-vX.Y.Z` (e.g. `main-v1.2.3`) | `main-v1.2.3` | `main-v1.2`, `main-v1` |
| Any other format (e.g. `latest`) | as-is | _(none)_ |

This allows callers to pin to a specific patch (`v1.2.3`), minor (`v1.2`), or major (`v1`) version.

## Version Progression Check

When the image tag is a SemVer tag and the registry is `ghcr.io`, the action queries the registry for existing image versions and verifies that the new version is exactly **one major, minor, or patch increment** ahead of the previous one. This prevents accidental large version jumps or downgrades.

The check is automatically **skipped** when:
- The tag is not a SemVer tag (e.g., `latest`).
- No previous image exists in the registry yet.
- The registry is not `ghcr.io`.

Set `version_force: 'true'` to bypass the check entirely.

## Examples

### Standard tag push workflow

Trigger automatically when a tag like `v1.2.3` or `main-v1.2.3` is pushed:

```yaml
on:
  push:
    tags:
      - 'v*'
      - '*-v*'
```

### Manual trigger with a custom tag

```yaml
      - name: Run Dev Container Prebuild
        uses: MiguelRodo/actions/prebuild-devcontainer@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          tag: ${{ github.event.inputs.tag }}
```

### Custom image name

```yaml
      - name: Run Dev Container Prebuild
        uses: MiguelRodo/actions/prebuild-devcontainer@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          image_name: 'ghcr.io/myorg/my-devcontainer'
```

### Non-default devcontainer path

```yaml
      - name: Run Dev Container Prebuild
        uses: MiguelRodo/actions/prebuild-devcontainer@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          devcontainer_path: 'src/.devcontainer'
```

### Using a non-GitHub container registry

When using a custom `image_name`, the `registry` input is only used for login—it is not automatically prepended to the image name. Ensure they match.

```yaml
      - name: Run Dev Container Prebuild
        uses: MiguelRodo/actions/prebuild-devcontainer@v2
        with:
          github_token: ${{ secrets.REGISTRY_TOKEN }}
          registry: 'registry.example.com'
          registry_username: 'my-username'
          image_name: 'registry.example.com/myorg/my-devcontainer'
```
