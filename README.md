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
> The current milestone ships only a **host Runtime Backend**. The host Runtime
> Backend runs a Workload Command as a normal host process with the current
> user's privileges, environment, and filesystem access. It does **not**
> provide strong isolation, and isobox does not claim to. The host backend's
> role is to route the existing execution behavior through the same Backend
> contract that stronger Runtime Backends will use later. See
> [Policy intent versus enforcement](#policy-intent-versus-enforcement) for what
> is recorded today and what remains unenforced.

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
separately. The current host Runtime Backend is lower-assurance and does not
enforce most of the recorded intent. The Task Record honestly records this gap
so it never overstates containment.

The Effective Policy captures the following intent and enforcement status:

| Policy category | Recorded intent | Host Runtime Backend enforcement |
| --- | --- | --- |
| **Network** | deny-by-default with optional allow rules | **not enforced** — Workload Commands retain host network access |
| **Resource limits** | resolved defaults (no explicit limit in this milestone) | **not enforced** — time, output size, CPU, memory, process, disk, and file descriptor limits are not enforced |
| **Reuse Inputs** | explicit host assets exposed for Host Agent Reuse | **declared and recorded only** — referenced host assets are not mounted or brokered |

In other words: in this milestone the host Runtime Backend records what *should*
happen (the policy intent) and records that it *does not* enforce it yet.
Stronger enforcement depends on future Runtime Backends. Until then, the safety
boundary rests on the disposable Workspace, the durable Task Record, and the
review-gated Promotion boundary — not on host-process containment.

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
  not lose what a reviewer needs.
- **Promotion**: the review-gated movement of a Task Result from a Repository
  Workspace into its trusted repository. Promotion applies only to a Repository
  Workspace and only after a human has reviewed the Task Result. It is the
  explicit step that moves reviewed output across the safety boundary.

## Requirements

- Go 1.26 or newer
- Git
- Linux or WSL2

## Build

```sh
go build -o bin/isobox ./cmd/isobox
```

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

## Review A Task Result

Each execution creates a Task Record directory such as:

```text
/tmp/isobox-records/task-0123456789abcdef/
└── record.json
```

Inspect the record before promotion:

```sh
cat /tmp/isobox-records/task-0123456789abcdef/record.json
```

The record contains:

- the Task Record schema version
- the Effective Policy, including the Workspace Source, Workload Command,
  selected Runtime Backend, retention mode, resource limits and their
  enforcement status, network policy and its enforcement status, declared
  Reuse Inputs, and known backend limitations
- the Workspace source type (`repository`), source commit, retention state,
  and retained Workspace path, when requested
- the Task Attempt Outcome, distinguishing success, preparation failure,
  launch failure, Workload Command exit, and result-capture failure
- stdout, stderr, and process exit status
- the captured Git diff
- the Promotion Report, a structured changed-file summary that flags
  high-risk categories (scripts, hooks, dependency manifests, CI workflows,
  large files, and binary-looking changes) so review can focus before
  explicit Promotion

The Effective Policy records both **intent** and **enforcement status**, so
the record shows what was requested and whether the host Runtime Backend
enforced it. Where a category is `not_enforced`, the record says so explicitly
rather than implying containment.

## Promote A Result

After reviewing the Task Result, apply its diff to the trusted source
repository:

```sh
./bin/isobox promote /tmp/isobox-records/task-0123456789abcdef
```

Promotion is a review-gated movement from a Repository Workspace Task Result
into the trusted source repository. It loads and validates the Task Record,
then checks all of the following before applying the captured diff with
`git apply`:

- the Task Attempt Outcome is `success`
- the Workspace source type is `repository`
- the recorded Workspace Source commit matches the current HEAD of the
trusted source repository
- the captured diff is non-empty

If any check fails, the source repository is left unchanged and a clear error
is reported.

Before applying the diff, `isobox promote` prints the Promotion Report captured
in the Task Record. The report lists changed files and any high-risk categories
that apply, so review is focused at the moment of Promotion. The report is
informational only: it never gates or auto-applies Promotion. The user remains
the review gate by running `isobox promote` explicitly.

The report is generated from the captured Git diff, so it reflects only what
the diff exposes. New untracked files are not included in the current POC diff;
high-risk detection for newly added files is therefore limited until the Task
Result captures untracked changes.

## Development

Run the integration tests:

```sh
go test ./...
```

The tests exercise the CLI through its public interface using temporary Git
repositories.

## Current Limitations

- Host Runtime Backend only; it does **not** provide strong isolation
- No Dirty Source Snapshot support
- No interactive review prompt
- No explicit Reuse Input brokering; Reuse Inputs are declared and recorded
  only, not mounted or brokered
- No network, credential, resource, or process policy **enforcement** —
  network intent (deny-by-default plus allow rules), resource limits, and
  Reuse Inputs are recorded in the Effective Policy, but the host Runtime
  Backend does not enforce them in this milestone
- Repository Workspaces only; Directory Workspaces are not implemented
- Promotion Report detection is limited to changes present in the captured
  Git diff; new untracked files are not yet captured, so newly added
  high-risk files may not appear in the report

## Project Documents

- [Product requirements](PRD.md)
- [Domain language](CONTEXT.md)
- [Daemonless MVP decision](docs/adr/0001-daemonless-mvp.md)
- [Host Agent Reuse decision](docs/adr/0002-host-agent-reuse-for-developer-preview.md)