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

**Task Artifact**:
A durable file captured as part of a Task Record for review, audit, or Promotion.
_Avoid_: log blob, result attachment

**Project Task Store**:
The project-local storage location for Task Records and Task Artifacts.
_Avoid_: global task cache, committed task history

**Task Result**:
The captured reviewable output of a Task Attempt.
_Avoid_: workspace, final state

**Agent Feedback**:
The command output, errors, and reviewable file changes returned to an Agent after a Task Attempt so it can continue its work.
_Avoid_: logs, response

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

**Cooperative Tool Call**:
A shell action that an external Agent voluntarily routes through isobox.
_Avoid_: intercepted command, governed shell call

**Cooperative Safe Mode**:
An Agent Skill operating convention where an Agent routes shell actions through isobox by default.
_Avoid_: enforced safe mode, global shell protection

**Direct Shell Escape**:
A human-approved shell action that an Agent runs without routing through isobox.
_Avoid_: safe bypass, untracked sandbox command

**Risky Shell Action**:
A shell action that may modify trusted assets, execute untrusted code, access network resources, expose credentials, or alter persistence.
_Avoid_: unsafe command, dangerous command

**Preflight Rule**:
An advisory policy check that may reject an action before it enters a Sandbox but is not the primary containment mechanism.
_Avoid_: enforcement rule, command firewall

**Host Agent Reuse**:
Launching an Agent in a Sandbox using the user's existing host-installed agent binary, configuration, and related local setup where explicitly exposed.
_Avoid_: safe agent install, inherited agent

**Reuse Input**:
A host binary, path, environment variable, credential reference, or local integration explicitly exposed to a Sandbox for Host Agent Reuse.
_Avoid_: automatic mount, inherited home, implicit config

**Environment Input**:
A non-secret environment variable explicitly exposed to a Sandbox.
_Avoid_: inherited environment, ambient env

**Credential Policy**:
The Sandbox Policy section that controls whether secret-bearing values or credential references are exposed to a Sandbox.
_Avoid_: secret passthrough, inherited credentials

**Network Policy**:
The Sandbox Policy section that controls whether a Sandbox may access network resources.
_Avoid_: internet access, connectivity setting

**Inherited Network Access**:
Network access granted to a Sandbox according to the Runtime Backend's ordinary network behavior.
_Avoid_: network allowlist, unrestricted network

**Filesystem Policy**:
The Sandbox Policy section that controls which host filesystem assets a Sandbox may directly access.
_Avoid_: mount config, volume setting

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

**Bubblewrap Backend**:
A Runtime Backend that uses bubblewrap to create a local filesystem containment boundary for a Sandbox.
_Avoid_: bubblewrap env, bwrap mode

**Containment Boundary**:
The boundary that prevents actions inside a Sandbox from directly affecting trusted host assets except through explicit Export or reviewed Promotion.
_Avoid_: quarantine, safety mode

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

**Promotion Approval**:
A human approval that authorizes Promotion after reviewing a Task Result.
_Avoid_: auto-merge, agent approval

**Promotion Confirmation**:
The CLI confirmation mode used when Promotion is invoked.
_Avoid_: approval proof, approver identity

**Agent Skill**:
A reusable instruction package that teaches cooperative Agents how to use isobox.
_Avoid_: managed integration, enforced adapter

**Promotion Report**:
A structured changed-file summary generated from a Task Result that flags high-risk categories for focused review before explicit Promotion. It is informational and never gates or auto-applies Promotion.
_Avoid_: risk report, review verdict

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
- A **Task Record** may contain one or more **Task Artifacts**
- A **Project Task Store** contains **Task Records** for one project
- A **Task Attempt** has exactly one **Task Attempt Outcome** after it ends
- A **Task Attempt** may produce exactly one **Task Result**
- A **Task Result** for a **Repository Workspace** includes tracked changes and reviewable untracked files
- A **Task Result** may exist even when the **Workload Command** exits non-zero
- A **Task Attempt** may produce **Agent Feedback**
- A **Cooperative Tool Call** creates a full **Task Record**
- A **Task Attempt** has exactly one **Sandbox**
- A **Sandbox** is created by exactly one **Runtime Backend**
- A **Sandbox** has exactly one **Containment Boundary**
- A **Runtime Backend** enforces the **Containment Boundary** to the extent its capabilities allow
- A first-milestone **Tool-Call Sandbox** requires a **Bubblewrap Backend**
- A **Sandbox** runs exactly one primary **Workload Command** for a **Development Workload**
- A **Development Workload** is usually an **Agent**
- A **Workload Command** may launch an **Agent**
- A **Session Sandbox** is the primary product workflow
- A **Tool-Call Sandbox** is a secondary workflow built on the same execution model
- A **Tool-Call Sandbox** is created only for a **Cooperative Tool Call**
- A **Risky Shell Action** should be routed through a **Tool-Call Sandbox**
- **Cooperative Safe Mode** routes shell actions through **Tool-Call Sandbox** by default
- **Cooperative Safe Mode** may allow a **Direct Shell Escape** after explicit human approval
- Project **Sandbox Policy** may disable **Tool-Call Sandbox** creation
- A **Preflight Rule** may reject a **Risky Shell Action** before Sandbox creation
- A **Preflight Rule** does not replace a **Containment Boundary**
- A **Preflight Rule** rejects a first-milestone **Tool-Call Sandbox** when the trusted repository has uncommitted changes
- **Host Agent Reuse** preserves existing Agent configuration but lowers isolation assurance
- **Host Agent Reuse** exposes one or more **Reuse Inputs**
- A **Reuse Input** is explicit, not discovered implicitly
- An **Environment Input** is explicit, not inherited implicitly
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
- A **Task Artifact** outlives its **Workspace**
- A **Task Result** outlives its **Workspace**
- Review is based on a **Task Result**, not a live **Workspace**
- Reviewable untracked files in a **Repository Workspace** belong to the **Task Result**
- A **Retained Workspace** exists only by explicit user or policy choice
- **Promotion** applies a reviewed **Task Result** from a **Repository Workspace**
- **Promotion** applies only to a **Repository Workspace**
- **Promotion** applies tracked changes and reviewable untracked files from a **Task Result**
- **Promotion** requires **Promotion Approval**
- **Promotion** records a **Promotion Confirmation**
- An **Agent Skill** may instruct an **Agent** to request **Promotion Approval** before running Promotion
- Human-facing Promotion asks for confirmation by default
- A **Promotion Report** summarizes a **Task Result** before explicit **Promotion** but never gates or auto-applies it
- **Export** moves a **Task Result** out of isobox-managed storage
- A **Containment Boundary** allows results to leave a **Sandbox** only through explicit **Export** or reviewed **Promotion**
- A **Task** has exactly one **Effective Policy**
- An **Effective Policy** is captured in the **Task Record**
- **Reuse Inputs** are captured in the **Effective Policy**
- **Reuse Inputs** are visible in the **Task Record**
- A **Sandbox Policy** has exactly one **Credential Policy**
- A **Sandbox Policy** has exactly one **Network Policy**
- A **Sandbox Policy** has exactly one **Filesystem Policy**
- A **Network Policy** may deny network access or allow **Inherited Network Access**
- A first-milestone **Tool-Call Sandbox** denies credential access
- A first-milestone **Tool-Call Sandbox** starts from a clean **Workspace Source**
- A first-milestone **Tool-Call Sandbox** uses manual **Promotion**
- **Agent Feedback** includes command output without interpreting credential-related command failures
- **Agent Feedback** includes early Task metadata, live command output, and a completion summary
- A **Task Record** records whether the **Credential Policy** exposed credentials

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
- Environment inheritance could silently leak host state or secrets — resolved: **Environment Inputs** are explicit, and credential-shaped values require credential policy rather than ordinary environment pass-through.
- Credential access would expand the first **Tool-Call Sandbox** from containment into secret brokering — resolved: first-milestone **Tool-Call Sandbox** denies credential access, and scoped credential exposure is a later explicit feature.
- Private dependency installation can fail for many reasons that isobox cannot reliably classify — resolved: command output is preserved as **Agent Feedback**, while the **Task Record** reports that no credentials were exposed.
- `PATH` determines available tooling rather than ordinary process context — resolved: `PATH` belongs to the **Development Environment** and starts with a backend-default path mode rather than normal **Environment Input** allowlisting.
- The first Tool-Call Development Environment should avoid premature environment design — resolved: `development_environment.path_mode: backend_default` is the only initial path mode.
- First-milestone **Tool-Call Sandbox** should avoid ambient environment inheritance — resolved: no ordinary environment variables are exposed beyond the backend-default `PATH`.
- "quarantine" was overloaded between blocking actions and containing their effects — resolved: **Containment Boundary** is the domain term, while quarantine may describe the user-facing safety posture.
- Replacing every Agent's built-in shell tool is not a realistic adoption path — resolved: Agents can voluntarily route shell actions through a **Tool-Call Sandbox** using normal shell access and project instructions.
- Project instructions can teach cooperative Agents when and how to use isobox, but cannot by themselves compel arbitrary Agent harnesses to use it — resolved: mandatory enforcement requires isobox to be placed in the execution path by an external wrapper, integration, or environment policy.
- Shell actions run without invoking isobox are outside the cooperative boundary — resolved: isobox creates no **Task Record**, tracks no effects, and makes no containment claim for direct shell tool calls.
- Seemingly read-only shell commands can still trigger unwanted behavior through aliases, shell hooks, wrappers, or project-local tooling — resolved: when the **Agent Skill** is invoked for **Cooperative Safe Mode**, it should instruct the Agent to route all shell actions through isobox by default rather than only obviously risky commands.
- Direct shell execution gives isobox no reliable event to audit — resolved: first-milestone **Direct Shell Escape** approval is conversation-level guidance from the **Agent Skill**, not an isobox **Task Record**.
- Some commands may be impractical to route through isobox in the first milestone — resolved: the **Agent Skill** may allow a **Direct Shell Escape** only after fresh human approval and a stated reason, with no isobox containment claim.
- Cooperative tool-call sandboxing is a meaningful product workflow expansion with a real adoption-versus-enforcement trade-off — resolved: record it in an ADR after the remaining boundary decisions are settled.
- Requiring every project to copy isobox usage instructions into README or Agent-specific markdown files creates adoption friction — resolved: an independently installed reusable Agent skill should be the primary instruction surface for teaching Agents how to invoke isobox.
- isobox should not manage Agent skill installation itself — resolved: users install the reusable Agent skill through the skill distribution channel, while isobox documentation may link to installation instructions.
- `isobox promote` is a CLI command and cannot distinguish whether a human or Agent invoked it through shell access — resolved: the cooperative **Agent Skill** should instruct Agents to seek human **Promotion Approval**, but that guidance is not an enforcement guarantee.
- Accidental Promotion should be harder in human-facing terminal use without blocking approved Agent workflows — resolved: `isobox promote` asks for confirmation by default and requires an explicit non-interactive confirmation flag for scripted or Agent-initiated Promotion.
- Non-interactive Promotion should be audit-visible without overclaiming human approval — resolved: Promotion records whether confirmation was interactive or explicit non-interactive.
- Project initialization should create only project-owned isobox policy files — resolved: `isobox init` creates `.isobox` configuration and does not edit Agent instructions, README files, or install reusable Agent skills.
- Running a **Tool-Call Sandbox** without project policy would make isobox choose security defaults on the user's behalf — resolved: `isobox tool` requires project configuration and fails preflight when `.isobox/config.yaml` is missing.
- Project policy should not be bypassed by the same caller it is meant to govern — resolved: when project **Sandbox Policy** disables Tool-Call Sandbox creation, `isobox tool` fails without a CLI force override.
- Project initialization should start from a restrictive posture — resolved: generated policy denies broad access by default and expects the user to explicitly loosen it for the project.
- Generated project policy should be understandable without becoming embedded documentation — resolved: `isobox init` may include sparse comments that explain security consequences, especially for denied network or credential access.
- First Tool-Call policy needs to be explicit without pretending every backend can enforce every control — resolved: the minimal project policy names tool-call enablement, runtime backend, development environment, workspace source, network policy, filesystem policy, credential policy, preflight rules, and promotion mode.
- Network allowlists imply enforcement precision that the first milestone cannot honestly guarantee — resolved: first-milestone **Network Policy** supports denied network access and explicit **Inherited Network Access**, but not host or domain allowlists.
- Preflight policy should use named checks before custom command matching — resolved: first-class **Preflight Rules** refer to built-in checks rather than user-authored regex as the main interface.
- Large command output and untracked file contents would make `record.json` hard to review and evolve — resolved: a **Task Record** is a directory containing metadata plus **Task Artifacts** such as stdout, stderr, patch data, and preserved untracked files.
- Tool-Call tasks are governed by project policy and promoted back to the same project — resolved: first-milestone **Project Task Store** lives under the project's `.isobox/tasks`, while generated ignore rules keep Task Records out of source control.
- **Tool-Call Sandbox** should use the same Task lifecycle as **Session Sandbox** — resolved: a tool call loads policy, resolves an **Effective Policy**, creates a **Workspace**, runs a **Workload Command**, captures a **Task Result**, and disposes or retains the **Workspace** according to policy.
- A **Tool-Call Sandbox** still creates a full **Task Record** — resolved: quick tool executions use the same durable audit model as longer **Session Sandbox** work instead of introducing a lighter command record.
- **Cooperative Safe Mode** can produce many low-change traces, but classifying commands as harmless is unreliable — resolved: every **Cooperative Tool Call** creates a full **Task Record** in the first milestone.
- **Tool-Call Sandbox** starts from a clean **Workspace Source** in the first milestone — resolved: pre-existing uncommitted changes are rejected so the **Task Result** remains attributable to the contained action.
- Dirty trusted repositories can make the Agent misunderstand what entered the Sandbox — resolved: first-milestone **Tool-Call Sandbox** preflight rejects tracked modifications and untracked, non-ignored files before Sandbox creation.
- **Dirty Source Snapshot** is useful but broadens attribution and promotion semantics — resolved: first-milestone **Tool-Call Sandbox** has no dirty-source override.
- **Tool-Call Sandbox** uses simple project root discovery in the first milestone — resolved: `.isobox/config.yaml` is found by walking upward from the current directory, and its directory must match the Git repository root.
- **Tool-Call Sandbox** preserves the caller's relative working directory — resolved: commands run inside the **Workspace** at the same relative path from the project root that the Agent invoked from.
- A successful **Tool-Call Sandbox** does not imply trusted output — resolved: successful execution produces a **Task Result** for review, while applying repository changes still requires **Promotion**.
- Automated command success is not sufficient approval to change the trusted repository — resolved: first-milestone **Tool-Call Sandbox** never auto-promotes changes; **Promotion** requires human review and **Promotion Approval**.
- Reviewable untracked files must survive the promotion path — resolved: Promotion uses patch application and/or Git plumbing rather than relying only on a tracked-file diff that omits untracked files.
- **Tool-Call Sandbox** must preserve the Agent feedback loop — resolved: command stdout, stderr, exit status, and reviewable diffs should be returned to the Agent while trusted repository changes remain behind **Promotion**.
- **Agent Feedback** should not require keeping live execution state — resolved: stdout, stderr, exit status, and diffs are returned by default, while a **Retained Workspace** remains an explicit policy or user choice.
- isobox should not manage an Agent's context window by default — resolved: **Agent Feedback** should not be truncated unless the user or policy explicitly requests output limits.
- **Agent Feedback** should be live when possible — resolved: stdout and stderr should stream to the Agent during execution while also being captured in the **Task Record**.
- **Tool-Call Sandbox** must expose Task identity without corrupting command stdout — resolved: isobox prints a short metadata prelude and compact completion summary on stderr while preserving wrapped command stdout for command output.
- **Tool-Call Sandbox** should preserve familiar command exit behavior — resolved: if the wrapped command runs, `isobox tool` exits with that command's exit code; if isobox fails before or around execution, it exits with status code 1 and records the precise outcome in the **Task Record**.
- Non-zero command exit can still leave useful reviewable changes — resolved: **Task Result** capture is attempted after wrapped command exit, and **Agent Feedback** reports both the non-zero exit and any captured changes.
- **Tool-Call Sandbox** is for non-interactive commands — resolved: interactive command flows are out of scope for the first tool-call workflow and belong in **Session Sandbox** or a future explicitly interactive mode.
- Tool calls commonly create new reviewable files — resolved: **Task Result** for a **Repository Workspace** captures untracked, non-ignored Workspace files as well as tracked changes.
- Sandbox Policy may express intent before every Runtime Backend can enforce it — resolved: backend enforcement gaps are reported in **Agent Feedback** and captured in the **Task Record** rather than hidden.
- The first stronger local Runtime Backend should prioritize low startup latency — resolved: investigate `bubblewrap` before Docker so Tool-Call Sandbox remains lightweight for Agent feedback loops.
- A Tool-Call Sandbox without filesystem isolation does not satisfy the milestone's purpose — resolved: first-milestone **Tool-Call Sandbox** requires an initial **Bubblewrap Backend**, and `host_process` is not sufficient for this workflow.
- The first stronger Runtime Backend should prove filesystem containment before broader controls — resolved: a command should see and mutate the **Workspace** copy without direct access to the trusted repository or host home.
- Runtime Backends should present stable internal paths when possible — resolved: commands should run from a normalized Workspace path such as `/workspace` when the backend supports it, while host paths remain internal metadata.
