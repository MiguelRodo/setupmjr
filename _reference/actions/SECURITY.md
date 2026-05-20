# Security Policy

## Supported versions

Security fixes are applied to the latest release only. We recommend always using the most recent tag.

| Version | Supported |
| --- | --- |
| Latest (`v2`) | ✅ |
| Older major versions | ❌ |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report vulnerabilities by using [GitHub's private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability):

1. Go to the **Security** tab of this repository.
2. Click **Report a vulnerability**.
3. Fill in the details and submit.

You can expect an initial response within **72 hours**. If a fix is warranted, a patched release will be published and you will be credited in the release notes (unless you prefer to remain anonymous).

## Security considerations for users

### Token scopes

Each action documents the minimum token scopes it requires. Never grant broader scopes than necessary:

- `prebuild-devcontainer` — `contents: write`, `packages: write`
- `add-issues-to-project` — PAT with `repo`, `project`, `read:org` (org projects only)
- `version-release` — `contents: write`
- `publish-quarto-site` — `contents: write`, `pages: write`

### Version pinning

Floating tags (e.g. `@v2`) are convenient but can be updated at any time. For environments with strict supply-chain requirements, pin to a specific commit SHA:

```yaml
uses: MiguelRodo/actions/prebuild-devcontainer@<full-commit-sha>
```

Find the SHA for any release on the [Releases page](../../releases) or by running:

```bash
git ls-remote https://github.com/MiguelRodo/actions refs/tags/v2
```
