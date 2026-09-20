package repo

var actionWorkflows = map[string]string{
	"prebuild-devcontainer": `name: 'Pre-build Dev Container'

on:
  push:
    tags:
      - 'v*'
      - '*-v*'
  workflow_dispatch:
    inputs:
      version:
        description: 'Exact image version (e.g. v1.2.3). Leave blank to bump a component.'
        required: false
        type: string
      bump_type:
        description: 'Component to bump. Choose none when entering an exact version.'
        required: false
        type: choice
        default: none
        options:
          - none
          - patch
          - minor
          - major
      version_force:
        description: 'Allow a non-sequential version, such as a downgrade or skipped increment.'
        required: false
        type: boolean
        default: false

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
          version: ${{ inputs.version }}
          bump_type: ${{ inputs.bump_type != 'none' && inputs.bump_type || '' }}
          version_force: ${{ inputs.version_force }}
`,
	"version-release": `name: Version and Release

on:
  push:
    tags:
      - 'v[0-9]+.[0-9]+.[0-9]+'
  workflow_dispatch:
    inputs:
      version:
        description: 'Exact release version in X.Y.Z form (e.g. 1.2.3). Leave blank to bump a component.'
        required: false
        type: string
      bump_type:
        description: 'Component to bump. Choose none when entering an exact version.'
        required: false
        type: choice
        default: none
        options:
          - none
          - patch
          - minor
          - major
      version_force:
        description: 'Allow a non-sequential version, such as a downgrade or skipped increment.'
        required: false
        type: boolean
        default: false
      python_version:
        description: 'Optional exact Python version in X.Y.Z form; overrides the release version.'
        required: false
        type: string
      r_version:
        description: 'Optional exact R version in X.Y.Z form; overrides the release version.'
        required: false
        type: string

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
          bump_type: ${{ inputs.bump_type != 'none' && inputs.bump_type || '' }}
          version_force: ${{ inputs.version_force }}
          python_version: ${{ inputs.python_version }}
          r_version: ${{ inputs.r_version }}
`,
	"go-version-release": `name: Go Version and Release

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:
    inputs:
      version:
        description: 'Exact release version in X.Y.Z form (e.g. 1.2.3). Leave blank to bump a component.'
        required: false
        type: string
      bump_type:
        description: 'Component to bump. Choose none when entering an exact version.'
        required: false
        type: choice
        default: none
        options:
          - none
          - patch
          - minor
          - major
      version_force:
        description: 'Allow a non-sequential version, such as a downgrade or skipped increment.'
        required: false
        type: boolean
        default: false
      go_version:
        description: 'Go version to install (e.g. 1.22).'
        required: false
        type: string
      goreleaser_config:
        description: 'Optional path to the GoReleaser config file.'
        required: false
        type: string
      apt_repo:
        description: 'Optional target GitHub repository in owner/name form for publishing generated .deb artifacts.'
        required: false
        type: string

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
          apt_signing_key: ${{ secrets.APT_SIGNING_KEY }}
          apt_signing_key_passphrase: ${{ secrets.APT_SIGNING_KEY_PASSPHRASE }}
          scoop_token: ${{ secrets.SCOOP_TAP_TOKEN }}
          homebrew_token: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          version: ${{ inputs.version }}
          bump_type: ${{ inputs.bump_type != 'none' && inputs.bump_type || '' }}
          version_force: ${{ inputs.version_force }}
          go_version: ${{ inputs.go_version }}
          goreleaser_config: ${{ inputs.goreleaser_config }}
          apt_repo: ${{ inputs.apt_repo }}
`,
	"r-version-release": `name: R Version and Release

on:
  push:
    tags:
      - 'v[0-9]+.[0-9]+.[0-9]+'
  workflow_dispatch:
    inputs:
      version:
        description: 'Exact release version in X.Y.Z form (e.g. 1.2.3). Leave blank to bump a component.'
        required: false
        type: string
      bump_type:
        description: 'Component to bump. Choose none when entering an exact version.'
        required: false
        type: choice
        default: none
        options:
          - none
          - patch
          - minor
          - major
      version_force:
        description: 'Allow a non-sequential version, such as a downgrade or skipped increment.'
        required: false
        type: boolean
        default: false

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
          bump_type: ${{ inputs.bump_type != 'none' && inputs.bump_type || '' }}
          version_force: ${{ inputs.version_force }}
`,
	"apt-repo-prune": `name: Prune APT Repository

on:
  workflow_dispatch:
    inputs:
      retention:
        description: 'Versions to retain for each package and architecture.'
        required: false
        type: choice
        default: latest-per-major
        options:
          - latest-per-major
          - latest-per-minor
          - latest
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
`,
	"publish-quarto-site": `name: Publish Quarto Site

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
`,
}
