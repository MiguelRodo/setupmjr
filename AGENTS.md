# Agent contract

This repository is setup and integration glue. Keep it small.

<!-- github-projects:start -->
## GitHub issues and Projects

For GitHub issue or Project administration, use
`.agents/skills/github-projects/SKILL.md` and read
`.projects/project.md` before acting.
<!-- github-projects:end -->

## Default working style

- Apply the vendored Ponytail skill at `.agents/skills/ponytail/SKILL.md` in **full** mode for coding work. Use **ultra** only when explicitly requested.
- Understand the real execution path before editing it, then make the smallest correct change.
- Prefer deletion, reuse, the Go standard library, native platform behaviour, and existing repository patterns over new abstractions or dependencies.
- Do not add frameworks, generic configuration systems, one-implementation interfaces, factories, or speculative extension points unless the task demonstrably requires them.
- Security, validation, error handling that prevents data loss, and explicitly requested behaviour are not places to cut corners.

## Ownership boundaries

`setupmjr` owns installation, configuration, environment setup, and the glue needed to connect external tools. It should not become a second implementation of tools owned elsewhere.

- `MiguelRodo/pj` owns `pj` behaviour and its installer/release contract.
- `MiguelRodo/repos` owns `repos` behaviour and its installer/release contract.
- `MiguelRodo/actions` owns reusable GitHub Actions and canonical workflow examples.
- `MiguelRodo/github-projects-skill` owns shared GitHub Projects semantics.

Before adding or copying behaviour from one of those projects, inspect the canonical owner and prefer delegation or consumption of its supported interface. Preserve existing release/channel semantics unless the task explicitly changes them.

## User environment changes

- Treat `$HOME`, shell startup files, Git configuration, credentials, HPC configuration, and generated project files as user-owned state.
- Managed edits must be idempotent and preserve unrelated user content.
- Store secrets once. Never copy literal credentials into shell startup files, Git config, repository files, Dockerfiles, or devcontainer configuration when a protected source plus standard consumer mechanism will work.
- Prefer existing `rc.d` conventions for shell snippets rather than adding unrelated setup directly to `.bashrc` or `.zshrc`.
- Keep platform-specific behaviour explicit. Do not pretend Linux/HPC, macOS, and Windows have identical capabilities.

## Testing

- Non-trivial behaviour needs the smallest focused regression test that would fail if the change broke.
- Prefer extending the existing Go or GitHub Actions test paths over introducing a new test framework.
- For security-sensitive, installer, shell, or cross-platform changes, test the relevant boundary rather than only a helper function.
- Do not add coverage for its own sake.

## Change discipline

- Fix root causes in the shared path instead of patching individual callers.
- Keep diffs narrow and avoid unrelated cleanup.
- Update documentation when user-visible commands, defaults, storage locations, or ownership boundaries change.
- Do not silently broaden `setupmjr`'s responsibilities to solve a neighbouring repository's problem.
