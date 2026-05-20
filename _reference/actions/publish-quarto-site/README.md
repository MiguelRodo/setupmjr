# Publish Quarto Site Action

A composite GitHub Action that publishes a Quarto site to the `gh-pages` branch of your repository. It automatically creates the `gh-pages` branch if it does not already exist.

## 📋 TL;DR

Copy the following to `.github/workflows/publish-quarto-site.yml`:

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

      - name: Publish Quarto Site
        uses: MiguelRodo/actions/publish-quarto-site@v2
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

---

## 🔐 Permissions

The calling workflow needs the following permissions:

| Permission | Why it is needed |
| --- | --- |
| `contents: write` | Push rendered output to the `gh-pages` branch. |
| `pages: write` | Required when deploying through GitHub Pages environments. |

```yaml
permissions:
  contents: write
  pages: write
```

> **Version pinning:** For stricter supply-chain security, pin to a specific commit SHA instead of a floating tag:
> ```yaml
> uses: MiguelRodo/actions/publish-quarto-site@<full-commit-sha>
> ```

---

## 🔧 Inputs

| Input | Description | Required | Default |
| --- | --- | --- | --- |
| `github_token` | GitHub token used to push to the `gh-pages` branch. Use `${{ secrets.GITHUB_TOKEN }}`. | **Yes** | — |

---

## ⚙️ How It Works

1. Configures git with the `github-actions[bot]` identity.
2. Checks whether the `gh-pages` branch already exists on the remote. If not, creates an empty orphan branch and pushes it.
3. Installs Quarto via [`quarto-dev/quarto-actions/setup@v2`](https://github.com/quarto-dev/quarto-actions).
4. Publishes the rendered site to `gh-pages` via [`quarto-dev/quarto-actions/publish@v2`](https://github.com/quarto-dev/quarto-actions).

---

## 🐞 Troubleshooting

* **Permission denied pushing to gh-pages:** Ensure the calling workflow has `permissions: contents: write` (and `pages: write` if using GitHub Pages deployments).
* **Quarto render errors:** The action delegates rendering entirely to `quarto-dev/quarto-actions/publish`. Consult the [quarto-actions documentation](https://github.com/quarto-dev/quarto-actions) for render-related issues.
