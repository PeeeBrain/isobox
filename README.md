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

Only committed source content is cloned into the Workspace. Uncommitted source
changes are not included in this POC.

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

- the Workspace Source path
- the Workload Command
- stdout and stderr
- process exit status
- the captured Git diff

## Promote A Result

After reviewing the Task Result, apply its diff to the trusted source
repository:

```sh
./bin/isobox promote /tmp/isobox-records/task-0123456789abcdef
```

Promotion uses `git apply`. It fails if the source repository has changed in a
way that prevents the captured patch from applying.

## Development

Run the integration tests:

```sh
go test ./...
```

The tests exercise the CLI through its public interface using temporary Git
repositories.

## Current Limitations

- No Runtime Backend or security isolation
- No dirty-source rejection or Dirty Source Snapshot support
- No interactive review prompt
- No Task Record schema version
- No explicit Reuse Input support
- No network, credential, resource, or process policy enforcement
- Repository Workspaces only; Directory Workspaces are not implemented

## Project Documents

- [Product requirements](PRD.md)
- [Domain language](CONTEXT.md)
- [Daemonless MVP decision](docs/adr/0001-daemonless-mvp.md)
- [Host Agent Reuse decision](docs/adr/0002-host-agent-reuse-for-developer-preview.md)
