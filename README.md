# isobox

`isobox` is an early proof of concept for running coding-agent workloads in a
private repository Workspace, capturing their result, and promoting reviewed
changes back to the trusted source repository.

The current implementation proves the basic product loop:

1. Clone a Git repository into a disposable private Workspace.
2. Run an opaque Workload Command inside that Workspace.
3. Capture stdout, stderr, exit status, policy metadata, and the Git diff.
4. Store the result as a durable Task Record.
5. Apply the reviewed diff back to the source repository on explicit promotion.

> [!WARNING]
> This POC does not provide a security sandbox yet. The Workload Command is a
> normal host process and can access anything allowed by the current user. The
> private Workspace prevents direct repository writes, but it is not an
> isolation boundary.

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
Reuse Inputs only; it does not mount or broker the referenced host assets.

## Review A Task Result

Each execution creates a directory such as:

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
  selected Runtime Backend, retention mode, declared Reuse Inputs, and known
  backend limitations
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

- Host Runtime Backend only; it does not provide strong isolation
- No Dirty Source Snapshot support
- No interactive review prompt
- No explicit Reuse Input brokering; Reuse Inputs are declared and recorded only, not mounted
- No network, credential, resource, or process policy enforcement
- Repository Workspaces only; Directory Workspaces are not implemented
- Promotion Report detection is limited to changes present in the captured
  Git diff; new untracked files are not yet captured, so newly added
  high-risk files may not appear in the report

## Project Documents

- [Product requirements](PRD.md)
- [Domain language](CONTEXT.md)
- [Daemonless MVP decision](docs/adr/0001-daemonless-mvp.md)
- [Host Agent Reuse decision](docs/adr/0002-host-agent-reuse-for-developer-preview.md)
