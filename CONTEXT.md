# isobox

isobox manages policy-bound execution of coding agents and untrusted development work while keeping trusted host assets outside the execution boundary.

## Language

**Task**:
A single user- or system-requested unit of work that is executed, audited, reviewed, and either promoted or rejected.
_Avoid_: job, run

**Task Attempt**:
One execution attempt for a Task, bound to a single Sandbox.
_Avoid_: retry, run

**Task Attempt Outcome**:
The reason a Task Attempt ended. Implemented values are `success`, `preparation_failure`, `launch_failure`, `workload_command_exit`, and `result_capture_failure`. `interruption` is reserved for future use.
_Avoid_: task status, exit status

**Task Record**:
The durable audit and result history for a Task.
_Avoid_: task folder, task storage

**Task Result**:
The captured reviewable output of a Task Attempt.
_Avoid_: workspace, final state

**Agent**:
A local coding assistant process launched inside a Sandbox to work on a Task.
_Avoid_: Codex, OpenCode

**Workload Command**:
An opaque local command configuration executed inside a Sandbox to perform a Task.
_Avoid_: agent command, generic agent, harness

**Development Workload**:
A repository-oriented command run inside a Sandbox, usually a coding Agent but not necessarily one.
_Avoid_: generic workload, job

**Session Sandbox**:
A Sandbox whose primary Workload Command is a coding Agent session.
_Avoid_: agent wrapper, sandboxed shell

**Tool-Call Sandbox**:
A Sandbox used by an external Agent to run a specific tool command.
_Avoid_: command sandbox, shell wrapper

**Host Agent Reuse**:
Launching an Agent in a Sandbox using the user's existing host-installed agent binary, configuration, and related local setup where explicitly exposed.
_Avoid_: safe agent install, inherited agent

**Reuse Input**:
A host binary, path, environment variable, credential reference, or local integration explicitly exposed to a Sandbox for Host Agent Reuse.
_Avoid_: automatic mount, inherited home, implicit config

**Development Environment**:
The tools, runtimes, package managers, Agent installation, Agent configuration, and local integrations available inside a Sandbox.
_Avoid_: sandbox, image

**User-Provided Environment**:
A Development Environment supplied by the user for creating Sandboxes.
_Avoid_: custom image, bring-your-own image

**isobox-Managed Environment**:
A Development Environment provisioned by isobox.
_Avoid_: fresh agent install, clean mode, safe mode

**Sandbox**:
A disposable, policy-bound execution environment managed by isobox for one Task.
_Avoid_: isolated environment, task environment, backend sandbox

**Runtime Backend**:
An isolation provider that creates and enforces the low-level boundary for a Sandbox.
_Avoid_: sandbox backend, runtime, backend

**Workspace**:
The private repository copy inside a Sandbox where task work occurs.
_Avoid_: checkout, worktree, repository copy

**Retained Workspace**:
A Workspace kept after its Task Attempt for explicit review or debugging.
_Avoid_: preserved sandbox, saved workspace

**Workspace Source**:
The trusted host path or remote source used to create a Workspace.
_Avoid_: source, input

**Repository Workspace**:
A Workspace derived from a Git repository.
_Avoid_: git workspace, cloned workspace

**Dirty Source Snapshot**:
A Workspace derived from a Repository Workspace Source that includes uncommitted source changes by explicit choice.
_Avoid_: dirty clone, patch overlay

**Directory Workspace**:
A Workspace derived from a non-Git directory.
_Avoid_: folder workspace, plain workspace

**Promotion**:
The review-gated movement of output from a Repository Workspace into its trusted repository.
_Avoid_: export, apply

**Export**:
The movement of task output or artifacts out of a Sandbox without applying them to a trusted repository.
_Avoid_: promotion

**Sandbox Policy**:
A versioned description of the capabilities and limits requested for a Task.
_Avoid_: config, settings

**Effective Policy**:
The resolved Sandbox Policy actually used for a Task.
_Avoid_: merged config, final settings

## Relationships

- A **Task** has one or more **Task Attempts**
- A **Task** has exactly one **Task Record**
- A **Task Attempt** has exactly one **Task Attempt Outcome** after it ends
- A **Task Attempt** may produce exactly one **Task Result**
- A **Task Attempt** has exactly one **Sandbox**
- A **Sandbox** is created by exactly one **Runtime Backend**
- A **Sandbox** runs exactly one primary **Workload Command** for a **Development Workload**
- A **Development Workload** is usually an **Agent**
- A **Workload Command** may launch an **Agent**
- A **Session Sandbox** is the primary product workflow
- A **Tool-Call Sandbox** is a secondary workflow built on the same execution model
- **Host Agent Reuse** preserves existing Agent configuration but lowers isolation assurance
- **Host Agent Reuse** exposes one or more **Reuse Inputs**
- A **Reuse Input** is explicit, not discovered implicitly
- A **Sandbox** has exactly one **Development Environment**
- A **Development Environment** may be assembled through **Host Agent Reuse**, supplied as a **User-Provided Environment**, or provisioned as an **isobox-Managed Environment**
- A **Sandbox** contains exactly one **Workspace**
- A **Workspace** is derived from exactly one **Workspace Source**
- A **Workspace** is either a **Repository Workspace** or a **Directory Workspace**
- A **Repository Workspace** is derived from a trusted repository but is not the trusted repository
- A **Dirty Source Snapshot** is a kind of **Repository Workspace**
- A **Sandbox** never receives direct access to the **Workspace Source**
- A **Workspace** is disposable by default
- A **Task Record** outlives its **Workspace**
- A **Task Result** outlives its **Workspace**
- Review is based on a **Task Result**, not a live **Workspace**
- A **Retained Workspace** exists only by explicit user or policy choice
- **Promotion** applies a reviewed **Task Result** from a **Repository Workspace**
- **Promotion** applies only to a **Repository Workspace**
- **Export** moves a **Task Result** out of isobox-managed storage
- A **Task** has exactly one **Effective Policy**
- An **Effective Policy** is captured in the **Task Record**
- **Reuse Inputs** are captured in the **Effective Policy**
- **Reuse Inputs** are visible in the **Task Record**

## Example dialogue

> **Dev:** "Can the agent write directly to my repository during a Task?"
> **Domain expert:** "No. The agent writes to the **Workspace** inside the **Sandbox**; only reviewed output can be promoted back to the trusted repository."

## Flagged ambiguities

- "sandbox" was used for both the isobox-managed environment and the lower-level isolation mechanism — resolved: **Sandbox** is the isobox-managed environment, and **Runtime Backend** is the isolation provider.
- Specific coding agents were used as examples — resolved: **Agent** means any local coding assistant process that can be launched inside a Sandbox.
- "agent adapter" implied maintained, agent-specific integrations in the MVP — resolved: **Workload Command** is the MVP mechanism for executing opaque local commands, including agents.
- "development workload" is broader than the target workflow — resolved: coding agents are the primary product focus, while non-agent commands are supported as secondary uses of the same execution model.
- "tool-call sandbox" could make isobox a generic command sandbox — resolved: **Session Sandbox** is the product focus, while **Tool-Call Sandbox** is a secondary mode.
- "global installation" hides an important tradeoff — resolved: **Host Agent Reuse** preserves real developer agent setup while lowering isolation assurance, and stricter execution depends on a more isolated **Development Environment**.
- "sandbox" and "development environment" were conflated — resolved: a **Sandbox** is the execution boundary, while a **Development Environment** is the tool/runtime setup inside that boundary.
- "workspace" implied Git-only behavior — resolved: **Repository Workspace** supports Git-native promotion semantics, while **Directory Workspace** is a lower-capability mode for non-Git folders.
- "promotion" and "export" were easy to conflate — resolved: **Promotion** applies reviewed output to a trusted repository, while **Export** only moves task output out of a Sandbox.
- Uncommitted repository changes can accidentally broaden what enters a Sandbox — resolved: a **Dirty Source Snapshot** exists only by explicit user or policy choice.
- Host Agent Reuse could silently inherit broad host state — resolved: Host Agent Reuse exposes explicit **Reuse Inputs**, and those inputs are captured in the **Effective Policy** and **Task Record**.
