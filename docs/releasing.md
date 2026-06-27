# Releasing isobox

This document is the release runbook for isobox. Follow it when a human asks
you to create a new release.

## Human approval requirement

Creating a release is a human-in-the-loop process.

An agent **MUST NOT** create, move, delete, or push a version tag unless one of
these is true:

1. The user explicitly prompts the agent to create a new release.
2. The agent recommends creating a new release and a human explicitly approves
   that recommendation before tag creation.

If the release was only implied by feature work, documentation work, CI work, or
installation work, stop before tag creation and ask for human approval.

## Release model

isobox releases are built from Git tags by GitHub Actions and GoReleaser.

```text
git tag v0.1.0
git push origin v0.1.0
        |
        v
GitHub Actions release workflow runs
        |
        v
GoReleaser builds Linux binaries
        |
        v
GitHub Release gets assets:
  isobox_linux_amd64.tar.gz
  isobox_linux_arm64.tar.gz
  checksums.txt
```

The public install command must stay platform-neutral:

```sh
curl -fsSL https://raw.githubusercontent.com/PeeeBrain/isobox/main/install.sh | bash
```

Do not point public install docs, landing pages, or release notes directly at a
single binary asset. `install.sh` detects the user's platform and downloads the
matching latest GitHub Release asset.

## Before tagging

1. Confirm the user has explicitly requested a release or has approved an
   agent-recommended release.
2. Confirm the working tree only contains intended release changes:

   ```sh
   git status --short
   ```

3. Run the test suite:

   ```sh
   go test ./...
   ```

4. Verify Linux release builds still compile:

   ```sh
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/isobox-linux-amd64 ./cmd/isobox
   CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /tmp/isobox-linux-arm64 ./cmd/isobox
   ```

5. If GoReleaser is installed locally, run a snapshot build:

   ```sh
   goreleaser release --snapshot --clean
   ```

   If GoReleaser is not installed, say so in the release handoff. The GitHub
   Actions workflow still runs GoReleaser during the tagged release.

## Create the release

Only run these commands after the human approval requirement is satisfied.

Choose the next semantic version tag, then create and push it:

```sh
git tag v0.x.0
git push origin v0.x.0
```

Replace `v0.x.0` with the approved version.

Pushing the tag triggers `.github/workflows/release.yml`. The workflow runs
tests, invokes GoReleaser, and publishes the GitHub Release assets.

## Verify the release

After the GitHub Actions release workflow completes, verify the latest release
assets exist:

```sh
curl -I https://github.com/PeeeBrain/isobox/releases/latest/download/isobox_linux_amd64.tar.gz
curl -I https://github.com/PeeeBrain/isobox/releases/latest/download/isobox_linux_arm64.tar.gz
curl -I https://github.com/PeeeBrain/isobox/releases/latest/download/checksums.txt
```

Then verify the installer path:

```sh
curl -fsSL https://raw.githubusercontent.com/PeeeBrain/isobox/main/install.sh | bash
isobox version
```

If `$HOME/.local/bin` is not on `PATH`, either run the installed binary by its
full path or set `INSTALL_DIR` to a directory already on `PATH`.

## If a release fails

Do not delete or move a pushed tag without explicit human approval.

Report:

- the tag name
- the failed workflow run
- the failed step
- the proposed fix

After the fix lands, ask the human whether to reuse the same version by moving
the tag or to create a new patch version. Moving a release tag is also a
human-in-the-loop action and requires explicit approval.
