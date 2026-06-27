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

## Release process

See [docs/releasing.md](docs/releasing.md). Creating, moving, deleting, or
pushing release tags is a human-in-the-loop action and requires an explicit
release request or human approval.
