# Changelog

User-visible changes to isobox are tracked here.

This project follows semantic version tags such as `v0.1.0`. Keep new entries
under `Unreleased` until a human-approved release is created.

## Unreleased

No unreleased changes.

## v0.1.1 - 2026-06-28

### Added

- GitHub Release build pipeline for Linux `amd64` and `arm64` binaries.
- Platform-detecting `install.sh` installer that downloads the latest matching
  GitHub Release asset and verifies checksums.
- `isobox version` command for release metadata.
- Richer top-level and per-command help: `isobox --help`, `isobox -h`,
  and `isobox help` now explain what isobox is, list every command with
  a short purpose, and use the project glossary terms (Task, Workspace,
  Sandbox, Task Record, Task Result, Promotion, Workload Command).
  `isobox <command> --help` for `init`, `run`, `tool`, `promote`,
  `version`, `doctor`, and `update` prints command-specific usage and
  examples.
- `isobox doctor [path]` read-only diagnostic command. It reports global
  readiness checks, discovers the target Git repository root, diagnoses
  missing or malformed project policy, accumulates unsupported first-
  milestone Tool-Call Sandbox policy findings, checks `.isobox/tasks/`
  gitignore coverage, warns about dirty trusted repositories, verifies
  task-store writability without creating project files, and exits with
  status 1 only when any `Doctor Finding` has severity `error`.
- Global `isobox doctor` checks: version metadata, `git` on PATH
  (error when missing), `bubblewrap (bwrap)` on PATH (warning with
  Tool-Call Sandbox consequence when missing, project readiness error when
  required by parsed policy), `isobox` on PATH (warning when missing), and
  multiple `isobox` binaries on PATH (warning listing the active binary plus
  duplicates). The checks never call the network or evaluate self-update
  eligibility.
- `isobox update --check` observability-only update check. The
  command reports the current version, the latest stable GitHub
  Release (drafts and prereleases are ignored), the selected Update
  Target resolved from the first `isobox` executable on the host
  PATH, any duplicate `isobox` binaries as warnings, and refuses
  `dev` builds and clearly package-manager-managed Update Targets
  before doing any I/O. The release metadata source is an injectable
  client so the integration tests do not depend on live GitHub.
  Power users with unusual system-managed install locations can
  teach the updater about additional managed prefixes by exporting
  `ISOBOX_UPDATE_MANAGED_PATH_PREFIXES` (one path per line) before
  running the check.

## Release process

See [docs/releasing.md](docs/releasing.md). Creating, moving, deleting, or
pushing release tags is a human-in-the-loop action and requires an explicit
release request or human approval.
