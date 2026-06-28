# isobox

`isobox` manages policy-bound execution of coding-agent work while keeping
trusted host assets outside the execution boundary. It clones a Git repository
into a disposable private Workspace, runs a Workload Command inside a Sandbox,
captures the result as a durable Task Record, and promotes reviewed changes back
to the trusted source repository.

isobox is not a safer shell. It is a safety boundary for Agent autonomy: the
product keeps a trusted repository out of an Agent's reach until a human has
reviewed a captured Task Result and explicitly promoted it. The boundary is
built from a policy-shaped workflow, durable audit records, and a review-gated
Promotion boundary, rather than from any single runtime isolation guarantee.

> [!IMPORTANT]
> The current milestone ships a **host Runtime Backend** and a filesystem-contained **bubblewrap Runtime Backend**. The host Runtime Backend runs a Workload Command as a normal host process with the current user's privileges, environment, and filesystem access; it does **not** provide strong isolation, and isobox does not claim to. The bubblewrap Runtime Backend provides filesystem containment for Cooperative Tool Calls but does **not** provide strong resource or network isolation. See [Policy intent versus enforcement](#policy-intent-versus-enforcement) for what is recorded today and what remains unenforced.

The current implementation proves the basic product loop:

1. Clone a Git repository into a disposable private Workspace.
2. Run an opaque Workload Command inside that Workspace.
3. Capture stdout, stderr, exit status, policy metadata, and the Git diff.
4. Store the result as a durable Task Record.
5. Apply the reviewed diff back to the source repository on explicit promotion.

## How isobox works

isobox models a single unit of work as a **Task**. A Task runs in a disposable
**Sandbox** created by a **Runtime Backend**. The Sandbox holds exactly one
**Workspace**: a private repository copy derived from a trusted **Workspace
Source**, never the trusted source itself. A **Workload Command** runs from the
root of the Workspace and is usually, but not always, a coding **Agent**.

The boundary is policy-shaped, not just runtime-shaped:

- **Sandbox Policy** describes the capabilities and limits requested for a
  Task, such as resource limits, network intent, and explicit Reuse Inputs.
- **Effective Policy** is the resolved policy actually used for a Task. It is
  captured in the **Task Record** so every execution is auditable and never
  implies stronger isolation than the selected Runtime Backend provides.

Because the **Workspace** is disposable by default, the **Task Record** and
captured **Task Result** outlive the Workspace. Review is based on the Task
Result, not on a live Workspace. Only a reviewed Task Result from a
**Repository Workspace** can be **Promoted** back into the trusted source
repository.

### Policy intent versus enforcement

isobox records **policy intent** and reports **enforcement status**
separately. Both the host Runtime Backend and the bubblewrap Runtime Backend are lower-assurance and do not enforce resource and network limits in this milestone. The Task Record honestly records this gap so it never overstates containment.

The Effective Policy captures the following intent and enforcement status:

| Policy category | Recorded intent | Host Runtime Backend enforcement | Bubblewrap Runtime Backend enforcement |
| --- | --- | --- | --- |
| **Network** | deny-by-default with optional allow rules | **not enforced** — Workload Commands retain host network access | **not enforced** — Workload Commands retain host network access |
| **Resource limits** | resolved defaults (no explicit limit in this milestone) | **not enforced** — time, output size, CPU, memory, process, disk, and file descriptor limits are not enforced | **not enforced** — time, output size, CPU, memory, process, disk, and file descriptor limits are not enforced |
| **Reuse Inputs** | explicit host assets exposed for Host Agent Reuse | **declared and recorded only** — referenced host assets are not mounted or brokered | **not supported** — `isobox tool` does not support declaring Reuse Inputs |

In other words: the runtime backends record what *should* happen (the policy intent) and record that they are **not enforced** yet. Stronger enforcement depends on future Runtime Backends. Until then, the safety boundary rests on the disposable Workspace, the durable Task Record, the filesystem containment for Tool-Call Sandboxes, and the review-gated Promotion boundary — not on host-process or complete network containment.

### Disposable Workspaces, Task Records, and Promotion

- **Workspace**: the private repository copy where task work occurs. A
  Workspace is derived from a trusted Workspace Source, never receives direct
  access to that source, and is disposed after the Task Attempt by default. A
  Workspace may be retained only by an explicit `--retain-workspace` choice.
- **Task Record**: the durable audit and result history for a Task. It
  outlives the Workspace and records the schema version, Effective Policy
  (including declared Reuse Inputs and backend enforcement limitations), the
  Workspace source type and commit, the Task Attempt Outcome, captured output,
  and the Git diff.
- **Task Result**: the captured reviewable output of a Task Attempt. Review is
  based on the Task Result, not on a live Workspace, so disposable cleanup does
  not lose what a reviewer needs. Task Results include tracked changes and
  reviewable untracked files.
- **Promotion**: the review-gated movement of a Task Result from a Repository
  Workspace into its trusted repository. Promotion applies only to a Repository
  Workspace and only after a human has reviewed the Task Result. It is the
  explicit step that moves reviewed output across the safety boundary.

## Requirements

- Go 1.26 or newer
- Git
- Linux or WSL2

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/PeeeBrain/isobox/main/install.sh | bash
```

The installer detects Linux `amd64` or `arm64`, downloads the matching asset
from the latest GitHub Release, verifies `checksums.txt`, and installs the
binary to `${HOME}/.local/bin` unless `INSTALL_DIR` is set.

## Build

```sh
go build -o bin/isobox ./cmd/isobox
```

Release builds are created by tagging a version:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The release workflow uses GoReleaser to publish Linux `amd64` and `arm64`
archives plus `checksums.txt` to GitHub Releases.

See [Releasing isobox](docs/releasing.md) for the release runbook. Creating or
pushing release tags is a human-in-the-loop process and requires an explicit
release request or human approval.

Project changes are tracked in [CHANGELOG.md](CHANGELOG.md). Development
workflow notes live in [CONTRIBUTING.md](CONTRIBUTING.md).

## Run A Workload

The source must be a local Git repository. The safest form to paste is a
single-line command:

```sh
./bin/isobox run --source /path/to/repository --records /tmp/isobox-records -- sh -c 'printf "changed\n" > README.md'
```

The equivalent multiline form is:

```sh
./bin/isobox run \
  --source /path/to/repository \
  --records /tmp/isobox-records \
  -- \
  sh -c 'printf "changed\n" > README.md'
```

Do not insert blank lines after continuation backslashes. A backslash only
continues onto the immediately following line. If the continuation is broken,
the shell may execute the Workload Command directly in the current directory.

Everything after `--` is the Workload Command. The command runs from the root
of the private Workspace.

If `--records` is omitted, Task Records are stored under
`.isobox/tasks` relative to the current directory.

The Workspace Source must be clean. `isobox run` rejects staged, unstaged, or
untracked changes so that only explicitly committed content enters the
Workspace. Dirty Source Snapshots are not supported in this POC.

### Retain The Workspace For Debugging

By default, the private Workspace is disposed after the Task Attempt. To keep
it for review or debugging, pass `--retain-workspace`:

```sh
./bin/isobox run \
  --source /path/to/repository \
  --records /tmp/isobox-records \
  --retain-workspace \
  -- \
  sh -c 'printf "changed\n" > README.md'
```

The CLI prints the retained path and the Task Record stores it under
`workspace.path`. That path is the retained repository Workspace root, so it
contains the files exactly as the Workload Command left them. Retained
Workspaces are a debugging aid; review should still be based on the captured
Task Result.

### Declare Reuse Inputs For Host Agent Reuse

Host Agent Reuse exposes explicit host assets to a Sandbox so a Workload
Command can reuse an existing Agent installation, configuration, or local
integration. Reuse Inputs are always explicit; isobox never silently inherits
broad host state.

Declare each exposed asset with `--reuse-input kind=value`, repeating the flag
for multiple inputs. Supported kinds are `host_binary`, `path`, `env_var`,
`credential_ref`, and `local_integration`:

```sh
./bin/isobox run \
  --source /path/to/repository \
  --records /tmp/isobox-records \
  --reuse-input host_binary=/usr/local/bin/codex \
  --reuse-input path=/home/user/.codex \
  --reuse-input local_integration=filesystem-mcp \
  -- \
  codex
```

Each declared Reuse Input is recorded in the Effective Policy so the Task
Record makes Host Agent Reuse exposure visible. This POC declares and records
Reuse Inputs only; it does not mount or broker the referenced host assets, and
Host Agent Reuse lowers isolation assurance compared with a more isolated
Development Environment.

## Initialize A Project Policy

To use the **Tool-Call Sandbox**, you must first initialize project policy in the Git repository root of your project:

```sh
./bin/isobox init
```

Or to initialize a specific directory's Git repository:

```sh
./bin/isobox init /path/to/project
```

This creates a restrictive default project policy at `.isobox/config.yaml` and adds `.isobox/tasks/` to `.gitignore`. You can edit `.isobox/config.yaml` by hand to adjust settings.

## Run A Tool-Call Sandbox

Once initialized, cooperative Agents can enter **Cooperative Safe Mode** by routing shell actions through the project-local Tool-Call Sandbox workflow. The command shape is:

```sh
./bin/isobox tool -- <command>
```

For example:

```sh
./bin/isobox tool -- sh -c 'printf "changed\n" > README.md'
```

Everything after `--` is the Workload Command. The command executes with a disposable copy of the workspace repository mounted at `/workspace` (with PID namespace isolated, env cleared except for a default `PATH` of `/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin`, and chdirs to `/workspace`).

The first Tool-Call Sandbox milestone requires `bubblewrap` (`bwrap`) and a project policy with `runtime_backend: bubblewrap`. `host_process` is insufficient for this workflow because it runs as an ordinary host process and cannot provide the filesystem Containment Boundary required by ADR 0003.

Before the Sandbox is created, `isobox tool` runs **Preflight Rules** to verify:
1. A project policy exists at the repository root.
2. `tool_call.enabled` is `true` in `.isobox/config.yaml`.
3. The policy shape matches the supported configuration (manual promotion, etc.).
4. The trusted repository is clean (no uncommitted tracked or untracked changes).
5. `bwrap` (bubblewrap) is installed on the host `PATH`.

If any check fails, the preflight rejects the execution before launching any processes or creating the Sandbox. The command exits with an `isobox tool preflight` error and does not create a Task Record.

Successful launches stream command stdout and stderr back to the caller as **Agent Feedback** while also capturing them for review. `isobox` writes a short task metadata prelude and completion summary on stderr, preserves wrapped command stdout for command output, captures patch data from the Workspace, and stores the Task Record in the project-owned **Project Task Store** under `.isobox/tasks/`.

## Review A Task Result

Each execution creates a Task Record directory such as:

```text
.isobox/tasks/task-0123456789abcdef/
├── record.json
└── artifacts/
    ├── stdout.txt
    ├── stderr.txt
    └── diff.patch
```

Inspect the record before promotion:

```sh
cat .isobox/tasks/task-0123456789abcdef/record.json
```

`record.json` is intentionally metadata-first. Large review data is stored as **Task Artifacts** referenced from the record:

- the Task Record schema version
- the Effective Policy, including the Workspace Source, Workload Command,
  selected Runtime Backend, retention mode, resource limits and their
  enforcement status, network policy and its enforcement status, declared
  Reuse Inputs, and known backend limitations
- the Workspace source type (`repository`), source commit, retention state,
  and retained Workspace path, when requested
- the Task Attempt Outcome, distinguishing success, preparation failure,
  launch failure, Workload Command exit, and result-capture failure
- stdout and stderr artifact references, plus process exit status
- patch data in `artifacts/diff.patch`, including tracked changes and preserved reviewable untracked file content from the Workspace
- the Promotion Report, a structured changed-file summary that flags
  high-risk categories (scripts, hooks, dependency manifests, CI workflows,
  large files, and binary-looking changes) so review can focus before
  explicit Promotion

The Effective Policy records both **intent** and **enforcement status**, so
the record shows what was requested and whether the selected Runtime Backend
enforced it. Where a category is `not_enforced`, the record says so explicitly
rather than implying containment.

## Promote A Result

After reviewing the Task Result, apply its changes to the trusted source
repository:

```sh
./bin/isobox promote .isobox/tasks/task-0123456789abcdef
```

To run promotion non-interactively (e.g., in automated script flows or CI tasks where approval has already been gathered), pass the `--yes` flag to bypass the interactive confirmation prompt:

```sh
./bin/isobox promote --yes .isobox/tasks/task-0123456789abcdef
```

Promotion is a review-gated movement from a Repository Workspace Task Result
into the trusted repository. It loads and validates the Task Record,
then checks all of the following before applying the captured changes:

- the Task Attempt Outcome is `success` or `workload_command_exit`
- the Workspace source type is `repository`
- the recorded Workspace Source commit matches the current HEAD of the
  trusted source repository
- the task result has changes to promote (a non-empty captured diff patch and/or captured untracked files)

If any check fails, the source repository is left unchanged and a clear error
is reported.

Before applying the changes, `isobox promote` prints the Promotion Report captured
in the Task Record. The report lists changed files and any high-risk categories
that apply, so review is focused at the moment of Promotion. The report is
informational only: it never gates or auto-applies Promotion. The user remains
the review gate by running `isobox promote` explicitly.

Upon human confirmation (or when `--yes` is specified), `isobox promote` applies the captured diff with `git apply --whitespace=nowarn` and copies any reviewable untracked files from the task result artifacts back into their target locations in the trusted source repository.

The report is generated from the captured patch data, so it reflects what the
Task Result exposes, including reviewable untracked files preserved in the
diff artifact.

## Cooperative Boundary And Direct Shell Escape

ADR 0003 defines the first Tool-Call Sandbox as a cooperative routing workflow, not enforced shell interception. Cooperative Safe Mode means an Agent voluntarily routes shell actions through `isobox tool -- <command>` by default. Project instructions or an Agent Skill can teach that convention, but isobox does not automatically intercept arbitrary shell calls made outside its CLI.

A **Direct Shell Escape** is a shell action the Agent runs without routing through isobox after fresh conversation-level human approval and a stated reason. A Direct Shell Escape creates no Task Record, receives no containment claim, produces no isobox Agent Feedback, and is not eligible for isobox Promotion unless its effects are later captured by a separate isobox Task. Use this only when the first milestone cannot practically route the command through `isobox tool`.

## Development

Run the integration tests:

```sh
go test ./...
```

The tests exercise the CLI through its public interface using temporary Git
repositories.

## Run A Diagnostic

`isobox doctor [path]` runs read-only Doctor Checks and reports Doctor
Findings with severities `ok`, `warning`, or `error`. The command is the
recommended first step on a fresh install: it confirms the isobox binary is
on `PATH`, surfaces obvious setup problems, and exits with status 1 only when
a finding has severity `error` so warnings do not break normal development.

```sh
isobox doctor
isobox doctor /path/to/repository
```

The grouped output separates `Global checks` from conditional `Project checks`.
Global checks run on every invocation: version metadata, `git` on `PATH`
(error when missing), `bubblewrap (bwrap)` on `PATH` (warning because
Tool-Call Sandboxes via `isobox tool` are unavailable), `isobox` on `PATH`,
and multiple isobox binaries on PATH (duplicate `isobox` binaries are listed).

When the target directory is inside a Git repository, project checks use the
repository root. They report a missing `.isobox/config.yaml` as a single
initialization warning, parse existing policy, surface unsupported first-
milestone policy fields, check `.isobox/tasks/` gitignore coverage, warn when
tracked or untracked repository changes would make `isobox tool` preflight
reject the repository, and verify task-store writability without creating
`.isobox/tasks/`. `ok` means ready, `warning` means an affected workflow may be
blocked but doctor still exits 0, and `error` means doctor exits 1.

`isobox doctor` is read-only: it does not run `isobox init`, create or modify
`.isobox/tasks/`, `.gitignore`, policy files, task records, or any other
project file. It also does not call the network, check update availability
(use `isobox update --check` for that), evaluate self-update eligibility, run a
platform support check, run a bubblewrap self-test, or automatically remediate
findings.

## Check For Updates

`isobox update --check` is the observability-only entry point for
keeping an installed release current. The check path uses the GitHub
Releases API to identify the latest stable release (drafts and
prereleases are ignored), compares it to the running version, and
reports the selected **Update Target** resolved from the first
`isobox` executable on your `PATH`. Additional `isobox` binaries on
`PATH` are listed as warnings without changing the selected target.

```sh
isobox update --check
```

The check never downloads or replaces anything. It refuses `dev`
builds and clearly package-manager-managed Update Targets (e.g.
`/usr/bin`, `/opt/homebrew`, `/snap`, `/var/lib/dpkg`,
`/var/lib/rpm`, `/var/lib/pacman`, `/nix/store`) with guidance to
use the package manager or move the binary to a writable manual-style
directory such as `${HOME}/.local/bin` or `/usr/local/bin`. The
release metadata source is injectable so the tests do not depend on
live GitHub.

Run `isobox --help` for the full command list and `isobox <command> --help`
for per-command usage and examples.

## Current Limitations

- The host Runtime Backend does **not** provide strong isolation. The bubblewrap Runtime Backend provides filesystem containment but does not provide strong resource or network isolation.
- No Dirty Source Snapshot support
- No explicit Reuse Input brokering; Reuse Inputs are declared and recorded
  only, not mounted or brokered
- No network, credential, resource, or process policy **enforcement** —
  network intent (deny-by-default plus allow rules), resource limits, and
  Reuse Inputs are recorded in the Effective Policy, but the Runtime Backends
  do not enforce them in this milestone
- Repository Workspaces only; Directory Workspaces are not implemented
- Promotion Report detection is limited to changes present in the captured
  patch data

## Agent Skill

isobox ships an Agent Skill that teaches coding agents how to use `isobox tool`
correctly — routing shell actions through the Tool-Call Sandbox, handling
preflight failures, requesting human Promotion Approval, and managing Direct
Shell Escapes.

Install it into any project with:

```sh
npx skills add PeeeBrain/isobox
```

This installs the `isobox-agent-guide` skill into your project's `.agents/skills/`
directory. Agents that support the Skills convention (Claude Code, Cursor, Codex,
and others) will automatically discover and follow the skill when you ask them to
run shell commands or test work in safe mode.

## Project Documents

- [Product requirements](PRD.md)
- [Domain language](CONTEXT.md)
- [Daemonless MVP decision](docs/adr/0001-daemonless-mvp.md)
- [Host Agent Reuse decision](docs/adr/0002-host-agent-reuse-for-developer-preview.md)
- [Cooperative Tool Call Sandboxing decision](docs/adr/0003-cooperative-tool-call-sandboxing.md)
