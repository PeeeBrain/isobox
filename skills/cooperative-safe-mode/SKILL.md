---
name: cooperative-safe-mode
description: Route Agent shell actions through isobox Tool-Call Sandboxes by default, with explicit human approval gates for Promotion and direct shell escape.
---

# Cooperative Safe Mode

Cooperative Safe Mode is an Agent Skill operating convention for using isobox
with a host Agent. When this mode is active, route shell actions through
`isobox tool` by default so each Cooperative Tool Call creates an isobox Task
Record and uses the configured Tool-Call Sandbox.

This is cooperative behavior, not shell interception. isobox records and
contains only shell actions that you actually invoke through isobox.

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
