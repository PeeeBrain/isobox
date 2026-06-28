# Changelog

User-visible changes to isobox are tracked here.

This project follows semantic version tags such as `v0.1.0`. Keep new entries
under `Unreleased` until a human-approved release is created.

## Unreleased

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
- `isobox doctor [path]` read-only diagnostic command. The 0.1.1 first
  slice reports isobox version metadata as a `Doctor Finding` with
  severity `ok` and exits with status 1 only when any finding has
  severity `error`. The grouped output distinguishes global checks
  from project checks so richer checks can land in follow-up slices
  without changing the CLI shape.
- Global `isobox doctor` checks: version metadata, `git` on PATH
  (error when missing), `bubblewrap (bwrap)` on PATH (warning with
  Tool-Call Sandbox consequence when missing), `isobox` on PATH
  (warning when missing), and multiple `isobox` binaries on PATH
  (warning listing the active binary plus duplicates). The checks
  use an injectable PATH lookup so they are unit-tested without
  depending on the host's actual dependency state, and they never
  call the network or evaluate self-update eligibility.
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
