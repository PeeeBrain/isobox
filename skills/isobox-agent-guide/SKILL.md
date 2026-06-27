---
name: isobox-agent-guide
description: Route Agent shell actions through isobox Tool-Call Sandboxes by default, with explicit human approval gates for Promotion and direct shell escape. Invoke this skill when the user asks to run bash or shell commands, test out work in safe mode, execute commands in a sandbox, or when the agent needs to run any shell action in a project that has isobox initialized.
---

# Cooperative Safe Mode

Cooperative Safe Mode is an Agent Skill operating convention for using isobox
with a host Agent. When this mode is active, route shell actions through
`isobox tool` by default so each Cooperative Tool Call creates an isobox Task
Record and uses the configured Tool-Call Sandbox.

This is cooperative behavior, not shell interception. isobox records and
contains only shell actions that you actually invoke through isobox.

## Prerequisites

Before `isobox tool` can be used, the project must be initialized:

1. Run `isobox init` in the project directory (or pass the project path) to
   create `.isobox/config.yaml` with restrictive defaults.
2. Commit the generated `.isobox/config.yaml` and `.gitignore` changes so the
   trusted repository is clean before the first tool call.

The tool-call workflow requires `bubblewrap` (`bwrap`) on the host `PATH`.

## Default Shell Routing

For shell actions, use:

```sh
isobox tool -- <command>
```

Apply this default to read, test, build, formatting, code-generation, package
manager, Git-inspection, and repository-editing commands. The command after
`--` is the Workload Command that runs in the Tool-Call Sandbox.

Do not silently fall back to a direct shell command when `isobox tool` fails
preflight. Stop and report the exact reason instead.

## Sandbox Environment

The Workload Command executes inside a bubblewrap filesystem boundary with
these properties:

- The repository workspace is mounted at `/workspace`. The command's working
  directory is `/workspace` (or a subdirectory matching the caller's relative
  position within the project).
- The environment is cleared. Only a default `PATH` of
  `/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin` is set.
  No host environment variables, credentials, or ambient state are inherited.
- The workspace is always disposable. It is removed after the task completes.
- Host paths outside the workspace (including the trusted repository, home
  directory, and credential stores) are not exposed.

## Exit Codes And Streaming

`isobox tool` propagates the Workload Command's exit code. If the wrapped
command exits with a non-zero status, `isobox tool` exits with the same code.

Command stdout and stderr are streamed back to the caller live as Agent
Feedback while also being captured for the Task Record. isobox emits its
own task lifecycle messages on stderr (e.g., `isobox task <id>: starting
tool call` and `isobox task <id>: completed outcome=success`).

## Task Records

Each `isobox tool` invocation creates an artifact-backed Task Record under
`.isobox/tasks/task-<id>/` in the project. The record includes:

- `record.json` with the Effective Policy, Workspace Source commit, outcome,
  and Promotion Report
- `artifacts/stdout.txt`, `artifacts/stderr.txt`, and `artifacts/diff.patch`

The Task Record is the basis for review and Promotion.

## Preflight Failures

If `isobox tool` reports any preflight failure, stop and report the exact
reason to the human before taking further shell action. Do this for:

- missing project policy or missing `.isobox/config.yaml`
- tool calls are disabled by project policy
- dirty trusted repository state, including staged, unstaged, or untracked
  changes
- bubblewrap is missing from the host `PATH`
- unsupported policy shape or unsupported policy setting

Do not retry the command directly unless the human gives fresh Direct Shell
Escape approval after seeing the exact failure reason.

### Commit Before Next Tool Call

After promoting a Task Result, the trusted repository will have unstaged
changes from the applied diff. You must commit (or stash) those changes
before the next `isobox tool` call, because preflight rejects a dirty trusted
repository. A typical post-promotion workflow is:

```sh
git add -A && git commit -m "<describe promoted changes>"
isobox tool -- <next command>
```

## Direct Shell Escape

Direct Shell Escape is a human-approved shell action that runs without routing
through isobox. Use it only after fresh human approval for the specific action
or tightly scoped command group, and state the reason the escape is needed.

Before using Direct Shell Escape, tell the human:

- the command or command group you want to run
- why `isobox tool` is not being used
- that the Direct Shell Escape creates no isobox Task Record
- that the Direct Shell Escape does not make an isobox containment claim

Approval must be fresh in the current conversation context. Prior approvals,
general preferences, or convenience are not enough.

## Promotion Approval

Do not initiate `isobox promote --yes` on your own authority. Before running
`isobox promote --yes`, request human Promotion Approval for the specific Task
Result that will be promoted.

Your request must identify the Task Result path or identifier and make clear
that Promotion moves reviewed output into the trusted repository. Run
`isobox promote --yes <task-result>` only after the human explicitly approves
Promotion for that specific Task Result.

## Reporting Results

After a Cooperative Tool Call completes, summarize the relevant Agent Feedback
from isobox and continue from the returned Task Result. If the Task Attempt
fails, report the Task Attempt Outcome and the command output that matters for
the next decision.

Keep the safety language precise: Cooperative Tool Calls create Task Records;
Direct Shell Escape does not. Tool-Call Sandboxes provide the containment
claimed by the configured Runtime Backend; direct shell calls do not receive an
isobox containment claim.
