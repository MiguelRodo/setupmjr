# Custom GitHub Actions

Reusable composite GitHub Actions for common CI/CD tasks: pre-building Dev Containers, syncing issues to GitHub Projects, automating version bumps and releases, and publishing Quarto sites.

> **Full documentation:** [https://miguelrodo.github.io/actions](https://miguelrodo.github.io/actions)

## Actions

### [Pre-build Dev Container](./prebuild-devcontainer)

Builds your Dev Container image, pushes it to a container registry (GHCR by default), and optionally generates a `prebuild/devcontainer.json` for instant environment loads.

<details>
<summary>Minimal workflow</summary>

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
      - uses: actions/checkout@v4
      - uses: MiguelRodo/actions/prebuild-devcontainer@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          tag: ${{ github.event.inputs.tag }}
```

</details>

See the [action README](./prebuild-devcontainer/README.md) for all inputs, including custom image names, non-default devcontainer paths, and alternative container registries.

---

### [Add Issues to Project](./add-issues-to-project)

Syncs issues from a repository to a GitHub Project (V2) board with built-in duplicate detection.

**Requires:** A PAT with `repo`, `project`, and `read:org` scopes saved as the `ADD_ISSUES_TO_PROJECT_TOKEN` secret.

<details>
<summary>Minimal workflow</summary>

```yaml
name: Sync Issues to Project

on:
  workflow_dispatch:
  issues:
    types: [opened, reopened]

jobs:
  add-to-project:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: MiguelRodo/actions/add-issues-to-project@v2
        with:
          ADD_ISSUES_TO_PROJECT_TOKEN: ${{ secrets.ADD_ISSUES_TO_PROJECT_TOKEN }}
          # project_name: "My Custom Project Board"
          # is_project_owner_org: "true"
```

</details>

See the [action README](./add-issues-to-project/README.md) for all inputs and advanced usage.

---

### [Version and Release](./version-release)

Bumps versions in Python (`pyproject.toml`) and/or R (`DESCRIPTION`) packages, creates a versioned git tag with floating major/minor aliases, and publishes a GitHub Release.

<details>
<summary>Minimal workflow</summary>

```yaml
name: Version and Release

on:
  push:
    tags:
      - 'v[0-9]+.[0-9]+.[0-9]+'
  workflow_dispatch:
    inputs:
      version:
        description: 'Exact version (e.g. 1.2.3). Cannot be used with bump_type.'
        required: false
      bump_type:
        description: 'Component to bump: major | minor | patch. Cannot be used with version.'
        required: false
      python_version:
        description: 'Override: exact version for the Python package.'
        required: false
      r_version:
        description: 'Override: exact version for the R package.'
        required: false

jobs:
  version-release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: MiguelRodo/actions/version-release@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          version: ${{ inputs.version }}
          bump_type: ${{ inputs.bump_type }}
          python_version: ${{ inputs.python_version }}
          r_version: ${{ inputs.r_version }}
```

</details>

See the [action README](./version-release/README.md) for all inputs, outputs, and version-precedence rules.

---

### [Go Version and Release](./go-version-release)

Creates a Go release by resolving a semantic version (explicit or bumped), optionally validating version progression, pushing the git tag, updating floating tags (`vX`, `vX.Y`), running GoReleaser native publishing, and optionally publishing generated `.deb` artifacts to a separate apt repository.

<details>
<summary>Minimal workflow</summary>

```yaml
name: Go Version and Release

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:
    inputs:
      version:
        description: 'Exact version (e.g. 1.2.3). Cannot be used with bump_type.'
        required: false
      bump_type:
        description: 'Component to bump: major | minor | patch. Cannot be used with version.'
        required: false
      go_version:
        description: 'Go version to install (defaults to 1.22).'
        required: false
      goreleaser_config:
        description: 'Optional path to the GoReleaser config file.'
        required: false
      apt_repo:
        description: 'Optional target GitHub repository in owner/name form for publishing generated .deb artifacts.'
        required: false
      apt_repo_token:
        description: 'Optional token for apt_repo access when publishing to a different repository.'
        required: false

jobs:
  release:
    runs-on: ubuntu-latest # Linux runner required
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: MiguelRodo/actions/go-version-release@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          apt_repo_token: ${{ secrets.APT_REPO_TOKEN }}
          version: ${{ inputs.version }}
          bump_type: ${{ inputs.bump_type }}
          go_version: ${{ inputs.go_version }}
          goreleaser_config: ${{ inputs.goreleaser_config }}
          apt_repo: ${{ inputs.apt_repo }}
```

</details>

See the [action README](./go-version-release/README.md) for all inputs, outputs, and behavior details, including apt repository management/signing and how to configure Homebrew and Scoop publishing in `.goreleaser.yml`.

---

### [R Version and Release](./r-version-release)

Bumps the version in an R `DESCRIPTION` file, builds the R package as a tarball, creates a versioned git tag with floating major/minor aliases, and publishes a GitHub Release containing the `.tar.gz` package artifact.

<details>
<summary>Minimal workflow</summary>

```yaml
name: R Version and Release

on:
  push:
    tags:
      - 'v[0-9]+.[0-9]+.[0-9]+'
  workflow_dispatch:
    inputs:
      version:
        description: 'Exact version (e.g. v1.2.3). Cannot be used with bump_type.'
        required: false
      bump_type:
        description: 'Component to bump: major | minor | patch. Cannot be used with version.'
        required: false
      version_force:
        description: 'When true, skip strict version progression checks.'
        required: false
        type: boolean

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: MiguelRodo/actions/r-version-release@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          version: ${{ inputs.version }}
          bump_type: ${{ inputs.bump_type }}
          version_force: ${{ inputs.version_force }}
```

</details>

See the [action README](./r-version-release/README.md) for all inputs, outputs, and behavior details.

---

### [APT Repository Prune](./apt-repo-prune)

Prunes superseded `.deb` package versions from a GitHub-hosted apt repository by rewriting Git history, then regenerates the apt metadata and force-pushes the result.

<details>
<summary>Minimal workflow</summary>

```yaml
name: Prune APT Repository

on:
  workflow_dispatch:
    inputs:
      retention:
        description: 'Retention policy: latest | latest-per-minor | latest-per-major'
        required: false
        default: latest-per-major
  schedule:
    - cron: '0 3 * * 0'   # weekly on Sunday at 03:00 UTC

jobs:
  prune:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: MiguelRodo/actions/apt-repo-prune@v2
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          retention: ${{ inputs.retention || 'latest-per-major' }}
          apt_signing_key: ${{ secrets.APT_SIGNING_KEY }}
          apt_signing_key_passphrase: ${{ secrets.APT_SIGNING_KEY_PASSPHRASE }}
```

</details>

See the [action README](./apt-repo-prune/README.md) for all inputs, retention policies, and how the history rewrite works.

---

### [Publish Quarto Site](./publish-quarto-site)

Publishes a Quarto site to the `gh-pages` branch, creating the branch automatically if it does not already exist.

<details>
<summary>Minimal workflow</summary>

```yaml
name: Publish Quarto Site

on:
  push:
    branches: [main]

permissions:
  contents: write
  pages: write

jobs:
  build-and-publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: MiguelRodo/actions/publish-quarto-site@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

</details>

See the [action README](./publish-quarto-site/README.md) for all inputs and troubleshooting tips.
