# isobox Product Requirements Document

**Status:** Draft  
**Version:** 0.1  
**Date:** 2026-06-06  
**Working name:** `isobox`  
**Primary implementation language:** Go  
**Document purpose:** Product and implementation context for human contributors and coding agents

---

## 1. Executive Summary

`isobox` is a local-first, agent-independent sandbox supervisor for safely running autonomous coding agents and untrusted code against software repositories.

The product is not intended to replace containers, microVMs, kernel sandboxes, or cloud execution platforms. Instead, `isobox` acts as the policy, orchestration, auditing, and result-promotion layer above one or more existing isolation backends.

The core value proposition is:

> Run a coding agent with broad freedom inside a disposable environment while strictly limiting what it can access, what credentials it can use, where it can connect, how many resources it can consume, and what changes can be promoted back to the trusted repository.

Initial integrations should target command-line coding agents such as Codex and OpenCode. Agents are launched as normal processes inside the isolated environment. `isobox` remains outside the sandbox and controls the lifecycle, policy, repository boundaries, network access, credential access, auditing, and result review.

The MVP should focus on Linux and WSL2, private Git clone workflows, deny-by-default network controls, structured audit logs, resource limits, and safe patch or commit export.

---

## 2. Problem Statement

Coding agents increasingly execute shell commands, edit source code, install dependencies, run tests, start services, inspect environment variables, and interact with remote developer platforms.

This creates a security problem:

- A repository may contain malicious or misleading instructions.
- Dependencies may execute installation scripts.
- An agent may accidentally delete or modify files outside the project.
- Shell access can bypass application-level permission systems.
- Long-lived credentials may be exposed to arbitrary child processes.
- Network access can be used for data exfiltration.
- A generated patch may contain dangerous automation, hooks, or configuration.
- Persistent development environments can accumulate poisoned state.
- Existing sandboxes usually focus on machine isolation rather than agent intent, policy portability, provenance, and safe result promotion.

Developers need a way to give coding agents enough freedom to work effectively without granting them ambient access to the host machine, user credentials, trusted repository, or unrestricted network.

---

## 3. Product Vision

`isobox` should become an open, local-first control plane for executing coding agents safely across multiple isolation backends.

A user should be able to run:

```bash
isobox run codex --repo . --task "Fix issue #123 and run the tests"
```

or:

```bash
isobox run opencode --repo .
```

and receive:

- A disposable isolated environment.
- A private writable copy of the repository.
- Enforced CPU, memory, disk, process, and time limits.
- Deny-by-default network access.
- No raw host credentials inside the sandbox.
- A complete task event history.
- A final patch, commit, or artifact bundle.
- A security-focused promotion report before any result reaches the trusted repository.

The product should make the safe path the easiest path.

---

## 4. Goals

### 4.1 Primary Goals

1. Safely run coding agents inside existing isolation backends.
2. Provide one policy format independent of the selected agent and runtime.
3. Prevent direct writes to the trusted host repository by default.
4. Restrict network access with deny-by-default rules.
5. Avoid exposing long-lived raw credentials inside sandboxes.
6. Capture a structured audit trail of agent activity.
7. Inspect outputs before promotion to the trusted environment.
8. Support both interactive and non-interactive agent sessions.
9. Work well on Linux and WSL2.
10. Provide a CLI-first experience suitable for automation and coding-agent use.

### 4.2 Secondary Goals

1. Support multiple concurrent tasks in separate sandboxes.
2. Allow resumable task checkpoints without weakening final verification.
3. Provide a stable Go SDK or internal package boundary for future integrations.
4. Support local and remote runtime adapters through a common interface.
5. Produce machine-readable events for terminal UIs, desktop applications, and CI systems.
6. Enable agent comparison and security evaluation using normalized task records.

---

## 5. Non-Goals

The first versions of `isobox` will not:

1. Build a new container runtime, hypervisor, kernel sandbox, or microVM implementation.
2. Claim perfect isolation or zero-risk execution.
3. Rely on shell-command parsing as the main security boundary.
4. Provide a full cloud compute platform.
5. Replace Git, Docker, CI systems, or existing coding agents.
6. Implement a general-purpose secrets manager.
7. Provide native Windows isolation in the MVP.
8. Build a large graphical dashboard before the CLI and policy model are stable.
9. Automatically promote agent output without an explicit policy decision.
10. Treat application-level tool permissions as sufficient containment.
11. Include protocol-based agent tool integrations; initial agent support will use direct process execution and native event interfaces where available.

---

## 6. Target Users

### 6.1 Individual Developer

A developer wants to run a coding agent against an unfamiliar repository without exposing the rest of the workstation.

Needs:

- Simple commands.
- Local execution.
- Safe repository handling.
- Clear diffs.
- Minimal setup.
- Predictable cleanup.

### 6.2 Maintainer or Open-Source Contributor

A maintainer wants multiple agents to work on independent issues or pull requests.

Needs:

- One sandbox per task.
- Branch and worktree isolation.
- Reproducible test runs.
- Reviewable output.
- Task history.
- Concurrency controls.

### 6.3 CI or Platform Engineer

An engineer wants to execute agent-generated changes or untrusted contributions in controlled environments.

Needs:

- Non-interactive operation.
- Machine-readable policy decisions.
- Strict resource limits.
- Ephemeral credentials.
- Stable exit codes.
- Artifact export.
- Full audit records.

### 6.4 Security-Conscious AI Tool Builder

A tool builder wants to use autonomous agents but does not want security to depend on each agent's internal permission system.

Needs:

- Agent-independent enforcement.
- Runtime adapters.
- Network and credential brokers.
- Policy versioning.
- Evidence for each decision.
- Fail-closed behavior.

---

## 7. Core Product Principles

### 7.1 The Repository Is Untrusted

Repository contents may contain malicious instructions, package scripts, generated files, hooks, CI workflows, editor tasks, or other executable configuration.

The sandbox must treat all repository content as untrusted input.

### 7.2 The Agent Is Untrusted

The agent may make mistakes, misunderstand instructions, execute unsafe commands, or follow malicious repository content.

The product must not depend on the agent voluntarily respecting boundaries.

### 7.3 The Supervisor Is Not the Isolation Boundary

The Go daemon coordinates policy and lifecycle. It must delegate low-level containment to an existing sandbox backend.

### 7.4 Host State Is Trusted and Protected

The user's original repository, credentials, SSH agents, Docker daemon, home directory, browser sessions, and unrelated files must be inaccessible by default.

### 7.5 Outputs Are Untrusted Until Promoted

A successful test run does not imply a safe patch. Generated outputs must be inspected before entering the trusted environment.

### 7.6 Policies Describe Capabilities, Not Command Strings

The system should express what a task may access or accomplish, rather than attempting to enumerate every safe shell command.

### 7.7 Unsafe Failure Modes Must Fail Closed

When policy evaluation, backend enforcement, credential brokering, or auditing fails, execution should stop unless an explicit unsafe override is supplied.

---

## 8. Primary Use Cases

### 8.1 Interactive Coding Session

A developer launches an agent in a sandbox and interacts with it through the terminal.

```bash
isobox run opencode --repo .
```

Expected result:

- An isolated session starts.
- The repository is cloned privately.
- OpenCode runs inside the sandbox.
- Network and resource policies are enforced.
- Changes remain isolated until reviewed.

### 8.2 Non-Interactive Coding Task

```bash
isobox run codex \
  --repo . \
  --task "Add pagination to the users endpoint and update tests"
```

Expected result:

- The task runs without direct host access.
- Progress events are streamed.
- Tests are executed.
- A patch and task report are produced.
- The user explicitly approves or rejects promotion.

### 8.3 Untrusted Repository Inspection

```bash
isobox inspect https://github.com/example/untrusted-repo
```

Expected result:

- The repository is cloned inside a disposable environment.
- Install scripts and setup commands are prevented from touching the host.
- Network access follows inspection policy.
- The final report includes processes, domains, file changes, and suspicious configuration.

### 8.4 Parallel Issue Resolution

```bash
isobox task create --agent codex --issue 101
isobox task create --agent codex --issue 102
isobox task create --agent opencode --issue 103
isobox task list
```

Each task runs in an independent sandbox and produces independent reviewable output.

### 8.5 CI Verification

```bash
isobox verify \
  --repo . \
  --ref refs/pull/123/head \
  --policy policies/ci-untrusted.yaml
```

Expected result:

- A clean environment is created.
- The target ref is fetched.
- Tests run under strict network and resource policy.
- The command exits with a stable machine-readable status.
- Artifacts and audit logs are exported.

### 8.6 Package or Plugin Evaluation

```bash
isobox inspect-package npm some-package
```

Expected result:

- Package installation occurs only inside the sandbox.
- Installation scripts are observed.
- File modifications and outbound connections are recorded.
- No host package manager state is changed.

---

## 9. User Experience

### 9.1 First-Run Flow

```bash
isobox init
```

The command should:

1. Detect Linux or WSL2.
2. Detect available runtime backends.
3. Validate required dependencies.
4. Create the user configuration directory.
5. Generate a conservative default policy.
6. Run a harmless sandbox self-test.
7. Print remediation steps for failed checks.

### 9.2 Minimal Run Flow

```bash
isobox run codex --repo .
```

Default behavior:

1. Resolve the trusted repository path.
2. Create a task record.
3. Create a disposable sandbox.
4. clone the repository into the sandbox.
5. Launch the selected agent.
6. Stream normalized events.
7. Stop the sandbox when the session exits.
8. Generate a promotion report.
9. Ask the user whether to export the result.
10. Destroy the sandbox after completion unless retained explicitly.

### 9.3 Review Flow

```bash
isobox diff <task-id>
isobox report <task-id>
isobox approve <task-id>
```

Approval should never directly overwrite the repository without showing:

- Changed files.
- Added executable files.
- Modified dependency manifests.
- Modified lockfiles.
- Modified CI workflows.
- Modified Git hooks.
- Modified editor or task automation.
- Modified agent instruction files.
- New binaries or large generated files.
- Test results.
- Network destinations contacted.
- Credentials or capabilities requested.

### 9.4 Destruction Flow

```bash
isobox destroy <task-id>
```

The command must:

- Stop all sandbox processes.
- Revoke task-scoped capabilities.
- Remove writable overlays.
- Delete temporary repository copies.
- Preserve only the configured audit and result artifacts.
- Return a non-zero exit code if cleanup is incomplete.

---

## 10. CLI Requirements

### 10.1 Core Commands

```text
isobox init
isobox doctor
isobox run
isobox task create
isobox task list
isobox task show
isobox task stop
isobox task resume
isobox task delete
isobox exec
isobox logs
isobox events
isobox diff
isobox report
isobox approve
isobox reject
isobox export
isobox verify
isobox inspect
isobox destroy
isobox policy validate
isobox backend list
isobox backend doctor
```

### 10.2 Example Commands

```bash
isobox run codex --repo . --task "Fix failing tests"
isobox run opencode --repo . --interactive
isobox exec <task-id> -- npm test
isobox logs <task-id> --follow
isobox events <task-id> --format jsonl
isobox report <task-id> --format markdown
isobox export <task-id> --format patch
isobox approve <task-id> --apply-to .
```

### 10.3 Global Flags

```text
--config <path>
--policy <path>
--backend <name>
--log-level <level>
--output <text|json>
--no-color
--yes
--unsafe
```

The `--unsafe` flag must:

- Be explicit.
- Print a prominent warning.
- Record the override in the audit log.
- Require confirmation in interactive mode.
- Never be enabled by configuration defaults.

### 10.4 Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General failure |
| 2 | Invalid arguments or configuration |
| 3 | Policy denied |
| 4 | Sandbox creation failed |
| 5 | Agent execution failed |
| 6 | Verification failed |
| 7 | Promotion blocked |
| 8 | Cleanup incomplete |
| 9 | Backend unavailable |
| 10 | Unsafe override required |

---

## 11. Functional Requirements

### 11.1 Task Lifecycle

The system must support the following task states:

```text
created
preparing
running
waiting_for_approval
stopping
stopped
verifying
completed
failed
rejected
destroying
destroyed
```

Requirements:

- State transitions must be persisted.
- Invalid transitions must be rejected.
- Every transition must emit an event.
- Interrupted daemon restarts must recover known tasks where possible.
- Tasks with unknown backend state must be marked for reconciliation.

### 11.2 Repository Handling

Default mode: `private-clone`.

Requirements:

- The original host repository is never writable by the sandbox.
- The sandbox receives a private clone or copied worktree.
- Uncommitted host changes require an explicit policy:
  - reject,
  - include as patch,
  - or snapshot.
- Submodules must follow separate network and credential rules.
- Git hooks must not execute during clone, checkout, export, or promotion unless explicitly allowed.
- The final output may be exported as:
  - unified patch,
  - Git commit,
  - Git bundle,
  - tar archive,
  - selected artifacts.

### 11.3 Process Execution

Requirements:

- Commands execute inside the backend sandbox.
- Process trees are tracked.
- Standard output and standard error are streamed.
- Timeouts are enforced outside the sandboxed process.
- Background processes are recorded.
- Public port binding is denied by default.
- Privileged execution is denied by default.
- Host PID, IPC, network, user, and mount namespaces must not be shared unless the selected backend documents equivalent isolation.

### 11.4 Resource Controls

Configurable limits:

- CPU count or quota.
- Memory.
- Disk.
- Process count.
- Open files.
- Maximum task duration.
- Maximum individual command duration.
- Maximum output size.
- Maximum artifact size.
- Maximum concurrent tasks.

The system must stop tasks that exceed hard limits and record the reason.

### 11.5 Network Controls

Default policy: deny all outbound and inbound network access.

Requirements:

- Network rules are defined per task.
- Domain allowlists must be resolved and enforced by the backend or proxy.
- DNS events should be recorded where practical.
- Redirects to unapproved hosts must be blocked.
- Public listener exposure must be denied by default.
- Network policy changes during a task require an approval event.
- Network denials must be visible to the user and agent logs.
- The system should distinguish read-oriented registry access from publish or mutation operations when a capable proxy is configured.

### 11.6 Credential Brokering

Requirements:

- Raw long-lived credentials must not be copied into the sandbox by default.
- Credential use must be task-scoped and time-limited.
- A credential capability must identify:
  - provider,
  - target resource,
  - allowed operations,
  - expiry,
  - task identifier.
- Credential requests and uses must be audited.
- Capabilities must be revoked when the task stops.
- If a backend cannot support secure brokering, the policy must reject the task or require an unsafe override.

Initial credential support may be limited to read-only Git cloning through a host-side helper or short-lived token mechanism.

### 11.7 Agent Adapters

An agent adapter must define:

```go
type AgentAdapter interface {
    Name() string
    Detect(ctx context.Context) (DetectionResult, error)
    Prepare(ctx context.Context, task Task, sandbox Sandbox) error
    Command(ctx context.Context, task Task) (CommandSpec, error)
    ParseEvent(ctx context.Context, line []byte) ([]Event, error)
    CollectResult(ctx context.Context, task Task, sandbox Sandbox) (AgentResult, error)
}
```

Initial adapters:

- Codex
- OpenCode
- Generic command adapter

Agent adapters are responsible for:

- Preparing configuration.
- Constructing launch commands.
- Enabling native machine-readable output where available.
- Parsing agent-specific events.
- Collecting final status.
- Never acting as the isolation boundary.

### 11.8 Runtime Backends

A runtime backend must define:

```go
type Runtime interface {
    Name() string
    Capabilities(ctx context.Context) (RuntimeCapabilities, error)
    Create(ctx context.Context, spec SandboxSpec) (Sandbox, error)
    Attach(ctx context.Context, sandboxID string) (Sandbox, error)
    Restore(ctx context.Context, snapshotID string) (Sandbox, error)
    Destroy(ctx context.Context, sandboxID string) error
}
```

A sandbox handle must support:

```go
type Sandbox interface {
    ID() string
    Exec(ctx context.Context, req ExecRequest) (Execution, error)
    Upload(ctx context.Context, src, dst string) error
    Download(ctx context.Context, src, dst string) error
    Snapshot(ctx context.Context) (Snapshot, error)
    Inspect(ctx context.Context) (SandboxStatus, error)
    Stop(ctx context.Context) error
    Destroy(ctx context.Context) error
}
```

MVP backend strategy:

1. Implement one strongly isolated local backend adapter.
2. Implement a mock backend for tests.
3. Add a remote backend only after the policy and task lifecycle are stable.

The project must not silently downgrade to a weaker backend.

### 11.9 Structured Event System

Events should use JSON Lines for streaming and storage.

Example:

```json
{
  "version": "1",
  "event_id": "evt_01J...",
  "task_id": "task_01J...",
  "timestamp": "2026-06-06T12:00:00Z",
  "source": "runtime",
  "type": "process.started",
  "severity": "info",
  "data": {
    "executable": "/usr/bin/npm",
    "arguments": ["test"],
    "working_directory": "/workspace"
  }
}
```

Required event categories:

- Task lifecycle.
- Sandbox lifecycle.
- Agent lifecycle.
- Process execution.
- Filesystem changes.
- Network requests and denials.
- Credential requests and uses.
- Policy decisions.
- User approvals.
- Resource-limit events.
- Verification.
- Promotion.
- Cleanup.

### 11.10 Audit Storage

The default local layout should resemble:

```text
~/.local/share/isobox/
  tasks/
    <task-id>/
      task.json
      policy.snapshot.yaml
      events.jsonl
      stdout.log
      stderr.log
      diff.patch
      report.md
      artifacts/
      verification/
```

Requirements:

- Policy snapshots must be immutable after task start.
- Events must be append-only.
- Sensitive values must be redacted before persistence.
- Audit retention must be configurable.
- Deleted tasks should support secure removal where the filesystem permits it.
- Logs must include enough information to explain a policy decision.

### 11.11 Promotion Gate

The promotion gate inspects task output before export.

Required checks:

- Git diff size.
- Binary additions.
- Executable bit changes.
- Dependency manifest changes.
- Lockfile changes.
- Package installation scripts.
- CI workflow changes.
- Git hook changes.
- Editor task or launch configuration changes.
- Agent instruction file changes.
- Secret-like material.
- Symlink additions or target changes.
- Submodule changes.
- Large generated files.
- Test results.
- Verification environment status.

Risk levels:

```text
low
medium
high
blocked
```

Promotion decisions:

```text
allow
allow_with_warning
require_approval
deny
```

The initial implementation may use deterministic rules only. Learned classifiers should not be required for the MVP.

### 11.12 Fresh Verification

Before promotion, the system should optionally:

1. Create a fresh sandbox.
2. Clone the original trusted revision.
3. Apply the generated patch.
4. Install dependencies under verification policy.
5. Run configured test commands.
6. Compare expected artifacts.
7. Record results separately from the development sandbox.

A successful development sandbox run must not substitute for fresh verification when policy requires it.

---

## 12. Policy Model

### 12.1 Example Policy

```yaml
apiVersion: isobox.dev/v1alpha1
kind: SandboxPolicy

metadata:
  name: safe-coding

runtime:
  backend: local-isolated
  lifecycle: ephemeral
  maxDuration: 30m

resources:
  cpu: 4
  memory: 4GiB
  disk: 10GiB
  processes: 256
  openFiles: 4096

workspace:
  mode: private-clone
  includeUncommittedChanges: false
  writable:
    - src/**
    - tests/**
    - docs/**
  protected:
    - .git/hooks/**
    - .github/workflows/**
    - .vscode/tasks.json
    - "**/.env*"

network:
  default: deny
  allow:
    - host: registry.npmjs.org
      ports: [443]
    - host: github.com
      ports: [443]
    - host: api.github.com
      ports: [443]

credentials:
  default: deny
  capabilities:
    - name: source-repository-read
      provider: git
      operations:
        - clone
        - fetch
      expiresAfter: 20m

execution:
  privileged: false
  publicPorts: deny
  nestedContainers: deny
  shell: allowed-inside-sandbox

promotion:
  output: patch
  requireFreshVerification: true
  requireUserApproval: true
  block:
    - secrets
    - unsafe-symlinks
  warn:
    - dependencies
    - ci
    - git-hooks
    - editor-automation
    - agent-instructions

verification:
  commands:
    - npm ci
    - npm test
```

### 12.2 Policy Requirements

- Policies must be versioned.
- Unknown fields must produce validation errors by default.
- Policies must support reusable named profiles.
- CLI flags may tighten a policy.
- CLI flags must not weaken a policy unless an explicit unsafe override is used.
- The effective policy must be stored with every task.
- Backend capabilities must be checked against policy requirements before launch.
- Unsupported required controls must cause task rejection.

---

## 13. System Architecture

```text
┌───────────────────────────────────────────────┐
│                User or CI Client              │
│         CLI, TUI, script, or future UI        │
└──────────────────────┬────────────────────────┘
                       │ local RPC
┌──────────────────────▼────────────────────────┐
│                 isobox daemon                 │
│                                               │
│  Task manager        Policy engine            │
│  Agent adapters      Runtime adapters         │
│  Workspace manager   Event recorder           │
│  Network broker      Credential broker        │
│  Verification        Promotion gate           │
└──────────────────────┬────────────────────────┘
                       │ runtime API
┌──────────────────────▼────────────────────────┐
│             Isolation backend                │
│   container, microVM, or remote sandbox       │
└──────────────────────┬────────────────────────┘
                       │
┌──────────────────────▼────────────────────────┐
│   Private repository + coding agent process  │
└───────────────────────────────────────────────┘
```

### 13.1 Recommended Components

```text
cmd/isobox              CLI entry point
cmd/isoboxd             daemon entry point
internal/task           task lifecycle
internal/policy         policy parsing and evaluation
internal/runtime        runtime interfaces
internal/runtime/mock   mock runtime
internal/agent          agent interfaces
internal/agent/codex    Codex adapter
internal/agent/opencode OpenCode adapter
internal/workspace      Git and repository handling
internal/events         event schema and storage
internal/audit          audit persistence and redaction
internal/network        network policy coordination
internal/credentials    credential capability broker
internal/promotion      output inspection
internal/verification   fresh verification
internal/config         user and project configuration
internal/rpc            CLI-to-daemon protocol
pkg/api                 optional stable public Go types
```

### 13.2 Daemon Model

The daemon should:

- Run as the current user.
- Use a local Unix domain socket.
- Require same-user access by default.
- Persist task state.
- Supervise long-running sandboxes.
- Recover after restart.
- Avoid listening on public network interfaces.
- Separate privileged helpers into narrowly scoped processes only when required.

### 13.3 CLI-to-Daemon Communication

Recommended initial options:

1. ConnectRPC or gRPC over Unix domain sockets.
2. A small versioned JSON-RPC protocol over Unix domain sockets.

Requirements:

- Version negotiation.
- Streaming events.
- Cancellation.
- Authentication through filesystem permissions and peer identity.
- No public TCP listener by default.

---

## 14. Data Model

### 14.1 Task

```go
type Task struct {
    ID              string
    Name            string
    State           TaskState
    Agent           AgentSpec
    Runtime         RuntimeSpec
    Repository      RepositorySpec
    PolicyDigest    string
    CreatedAt       time.Time
    StartedAt       *time.Time
    FinishedAt      *time.Time
    SandboxID       string
    BaseRevision    string
    ResultRevision  string
    FailureReason   string
}
```

### 14.2 Action

```go
type Action struct {
    TaskID      string
    Actor       string
    Type        string
    Resource    string
    Operation   string
    Attributes  map[string]string
    RequestedAt time.Time
}
```

### 14.3 Policy Decision

```go
type Decision struct {
    Result       DecisionResult
    RuleID       string
    Reason       string
    RequiresUser bool
    ExpiresAt    *time.Time
}
```

### 14.4 Promotion Report

```go
type PromotionReport struct {
    TaskID            string
    Risk               RiskLevel
    Decision           PromotionDecision
    Findings           []Finding
    ChangedFiles       []FileChange
    Verification       VerificationResult
    NetworkSummary     NetworkSummary
    CredentialSummary  CredentialSummary
}
```

---

## 15. Threat Model

### 15.1 Protected Assets

- Host filesystem.
- Trusted repository.
- SSH keys and agents.
- Cloud credentials.
- API tokens.
- Browser and desktop session data.
- Host Docker daemon.
- Local network services.
- Other repositories.
- User identity and developer accounts.
- CI secrets.
- Audit integrity.

### 15.2 Adversaries

- Malicious repository author.
- Compromised dependency.
- Prompt-injected coding agent.
- Incorrect agent-generated command.
- Malicious package installation script.
- Compromised agent binary.
- Untrusted pull request.
- Curious or compromised child process.
- Network service attempting redirect or data collection.

### 15.3 Key Threats

1. Host filesystem escape.
2. Writing directly into the trusted repository.
3. Credential theft.
4. Network exfiltration.
5. Host Docker socket access.
6. Privilege escalation.
7. Resource exhaustion.
8. Persistent malware or poisoned caches.
9. Dangerous generated configuration.
10. Audit log tampering.
11. Symlink-based path escape.
12. Secret leakage into logs.
13. Sandbox backend downgrade.
14. Reuse of stale capabilities.
15. Promotion of unverified artifacts.

### 15.4 Security Invariants

- The trusted repository is not writable from the sandbox.
- No host credential is present in sandbox environment variables by default.
- Required policy controls cannot be silently ignored.
- The sandbox cannot access the host Docker socket.
- Task-scoped credentials expire and are revoked.
- The final patch is treated as untrusted.
- Verification uses a clean environment when required.
- Every unsafe override is recorded.
- Cleanup failure is visible and non-successful.
- Agent-level approvals do not replace backend enforcement.

---

## 16. Security Requirements

1. Use existing isolation primitives rather than custom security mechanisms.
2. Use least-privilege filesystem mounts.
3. Resolve and validate paths against symlink traversal.
4. Redact secret-like values from logs.
5. Avoid command construction through shell interpolation where direct process APIs are available.
6. Store configuration with restrictive filesystem permissions.
7. Validate backend versions and capabilities.
8. Sign or hash immutable task metadata and policy snapshots.
9. Provide reproducible security self-tests.
10. Maintain a documented insecure-development mode separate from normal operation.
11. Never auto-enable privileged containers.
12. Never mount the host home directory by default.
13. Never mount the host Docker socket.
14. Never expose the daemon publicly by default.
15. Refuse to run when mandatory controls cannot be enforced.

---

## 17. Configuration

### 17.1 User Configuration

```text
~/.config/isobox/config.yaml
```

Example:

```yaml
defaultPolicy: ~/.config/isobox/policies/safe-coding.yaml
defaultBackend: local-isolated
dataDir: ~/.local/share/isobox
maxConcurrentTasks: 2

agents:
  codex:
    binary: codex
  opencode:
    binary: opencode

logging:
  level: info
  format: text
  retentionDays: 30
```

### 17.2 Project Configuration

```text
.isobox.yaml
```

Example:

```yaml
policy: .isobox/policy.yaml

verification:
  commands:
    - go test ./...
    - go vet ./...

promotion:
  expectedPaths:
    - cmd/**
    - internal/**
    - docs/**
```

Project configuration must not be able to weaken user security defaults automatically. Since the repository is untrusted, project configuration should be treated as a request that requires validation against the user's trusted policy.

### 17.3 Configuration Precedence

From strongest to weakest:

1. Hard-coded security invariants.
2. User-enforced policy constraints.
3. Explicit CLI tightening.
4. User default policy.
5. Project configuration.
6. Agent requests.

Unsafe overrides sit outside normal precedence and must be explicit and audited.

---

## 18. Observability

### 18.1 Logs

- Human-readable default logs.
- JSON option for automation.
- Correlation by task ID.
- Separate agent output from supervisor output.
- Redaction before persistence.
- Configurable verbosity.

### 18.2 Metrics

Optional local metrics:

- Task count by state.
- Sandbox startup failures.
- Policy denials.
- Network denials.
- Average task duration.
- Resource-limit terminations.
- Promotion findings.
- Verification pass rate.
- Cleanup failures.

Metrics must not include source code, prompts, file contents, secrets, or personally identifying data by default.

### 18.3 Diagnostics

```bash
isobox doctor
isobox backend doctor <backend>
isobox task diagnose <task-id>
```

Diagnostics should identify:

- Missing runtime dependencies.
- Unsupported backend features.
- Permission problems.
- Stale task state.
- Orphaned sandboxes.
- Cleanup failures.
- Policy incompatibilities.

---

## 19. MVP Scope

### 19.1 MVP Inclusions

- Go CLI.
- Go daemon.
- Linux and WSL2.
- One local strongly isolated runtime backend.
- Mock runtime backend.
- Codex adapter.
- OpenCode adapter.
- Generic command adapter.
- Private clone workspace mode.
- Ephemeral tasks.
- CPU, memory, disk, process, and duration limits.
- Deny-by-default network configuration.
- Explicit domain allowlist.
- Structured JSONL events.
- Local audit storage.
- Patch export.
- Deterministic promotion checks.
- Fresh verification.
- Manual approval before applying changes.
- `doctor`, `run`, `logs`, `diff`, `report`, `approve`, and `destroy`.

### 19.2 MVP Exclusions

- Native Windows backend.
- macOS backend.
- Large web dashboard.
- Multi-user server mode.
- Team RBAC.
- Hosted control plane.
- Arbitrary secrets-manager integrations.
- Automated pull-request creation.
- Learned security classifiers.
- Full packet capture.
- Transparent TLS inspection.
- Dozens of runtime adapters.
- Background marketplace or plugin ecosystem.

---

## 20. Milestones

### Milestone 0: Repository and Design Foundation

Deliverables:

- Go module.
- CLI skeleton.
- Daemon skeleton.
- Task and event types.
- Policy schema.
- Mock runtime.
- Architecture decision records.
- Threat model.
- Basic unit-test setup.
- Static analysis and CI.

Exit criteria:

- `isobox doctor` runs.
- A mock task can move through the full lifecycle.
- Events are persisted as JSONL.
- Policy validation works.

### Milestone 1: Local Sandbox Execution

Deliverables:

- First real runtime adapter.
- Sandbox creation and destruction.
- Command execution.
- Resource limits.
- File upload and download.
- Failure cleanup.
- Backend capability negotiation.

Exit criteria:

- A generic command runs in isolation.
- The sandbox cannot write to the host repository.
- Resource-limit tests pass.
- Orphan cleanup is detectable.

### Milestone 2: Git-Native Workspace Isolation

Deliverables:

- Private clone workflow.
- Base revision capture.
- Patch generation.
- Uncommitted change policy.
- Hook suppression.
- Symlink safety checks.

Exit criteria:

- A sandbox task can modify a private repository copy.
- A unified patch is generated.
- No direct host repository writes occur.
- Unsafe symlink test cases are blocked.

### Milestone 3: Coding Agent Adapters

Deliverables:

- Codex adapter.
- OpenCode adapter.
- Generic agent adapter.
- Interactive session support.
- Non-interactive task support.
- Normalized agent events.

Exit criteria:

- Both initial agents can complete a simple repository task.
- Agent output is streamed.
- Agent exit status is captured.
- Agent configuration remains inside the sandbox.

### Milestone 4: Network and Credential Controls

Deliverables:

- Deny-by-default network policy.
- Domain allowlist.
- Network event records.
- Initial read-only repository credential capability.
- Capability expiry and revocation.

Exit criteria:

- Disallowed egress is blocked.
- Allowed dependency access succeeds.
- No raw long-lived credential is stored in the sandbox.
- Capability use appears in the audit report.

### Milestone 5: Promotion and Fresh Verification

Deliverables:

- File-change classification.
- Sensitive-path checks.
- Secret scanning.
- Fresh verification sandbox.
- Markdown task report.
- Approval and patch application flow.

Exit criteria:

- High-risk changes are flagged.
- Fresh verification runs against a clean environment.
- Promotion is blocked when required checks fail.
- Approved patches can be applied to the trusted repository.

### Milestone 6: Reliability and Developer Preview

Deliverables:

- Daemon restart recovery.
- Task reconciliation.
- Improved diagnostics.
- Retention and cleanup controls.
- Documentation.
- Example policies.
- Security test suite.
- Release packaging.

Exit criteria:

- Interrupted tasks recover or fail clearly.
- Known orphan scenarios are handled.
- A new user can complete the documented quickstart.
- Security invariants have automated tests.

---

## 21. Acceptance Criteria

The MVP is acceptable when all of the following are true:

1. A user can initialize `isobox` on Linux or WSL2.
2. A user can run Codex and OpenCode inside an isolated backend.
3. The host repository remains unchanged until explicit approval.
4. The sandbox cannot access unrelated host files under the default policy.
5. The sandbox cannot access the public internet under the default policy.
6. Allowed domains can be enabled explicitly.
7. CPU, memory, process, disk, and time limits are enforced.
8. Task state survives daemon restart.
9. Every task has a policy snapshot and structured event log.
10. Agent output and supervisor logs are distinguishable.
11. Raw long-lived credentials are not placed inside the sandbox by default.
12. The final patch receives deterministic risk checks.
13. A fresh verification environment can apply and test the patch.
14. Promotion fails closed when required verification fails.
15. Cleanup failures return a non-success status.
16. Unsafe overrides are explicit and auditable.
17. The system never silently substitutes a weaker backend.
18. Core lifecycle, policy, workspace, and promotion logic have automated tests.
19. The CLI supports both human-readable and JSON output.
20. Documentation clearly states the security boundaries and limitations.

---

## 22. Testing Strategy

### 22.1 Unit Tests

- Policy parsing.
- Policy precedence.
- State transitions.
- Path validation.
- Redaction.
- Risk classification.
- Event serialization.
- Backend capability matching.
- Agent event parsing.

### 22.2 Integration Tests

- Sandbox create, exec, stop, and destroy.
- Repository clone and patch export.
- Resource exhaustion.
- Network denial.
- Allowed network access.
- Daemon restart recovery.
- Agent cancellation.
- Fresh verification.
- Cleanup after process crash.

### 22.3 Security Tests

- Symlink path escape.
- `../` traversal.
- Attempted host home access.
- Host Docker socket access.
- Privileged execution request.
- Fork bomb or process explosion.
- Memory exhaustion.
- Disk exhaustion.
- Secret output redaction.
- Redirect from allowed to disallowed domain.
- Git hook execution.
- Malicious package installation script.
- CI workflow modification.
- Editor task modification.
- Binary artifact promotion.
- Stale credential reuse.
- Policy downgrade attempt.
- Project configuration attempting to weaken user policy.

### 22.4 End-to-End Tests

A test repository should contain controlled adversarial cases and verify that:

- The agent can complete ordinary coding work.
- The agent cannot escape required boundaries.
- Dangerous output is identified.
- Approved output applies cleanly.
- Rejected output never reaches the trusted repository.

---

## 23. Coding-Agent Implementation Guidance

This PRD is intended to be used as context for coding agents. Agents working on `isobox` should follow these rules:

1. Do not invent security guarantees that are not enforced by the selected backend.
2. Do not weaken a failing security check to make tests pass.
3. Do not introduce privileged execution without an architecture decision record.
4. Keep runtime-specific code behind interfaces.
5. Keep agent-specific code behind adapters.
6. Prefer deterministic policy behavior.
7. Return typed errors with actionable context.
8. Use `context.Context` for all blocking and external operations.
9. Avoid global mutable state.
10. Use direct process execution rather than shell strings where possible.
11. Treat filesystem paths as hostile input.
12. Keep audit redaction centralized.
13. Add tests for every security-sensitive bug.
14. Preserve backward compatibility for stored event and policy versions.
15. Document any operation that crosses the sandbox boundary.
16. Never place secrets in fixtures, logs, snapshots, or examples.
17. Do not silently continue after partial cleanup.
18. Keep the initial implementation small enough to audit.

### Suggested Go Practices

- Standard library first.
- Cobra is acceptable for the CLI.
- Use structured logging.
- Use interfaces at runtime and agent boundaries, not everywhere.
- Use table-driven tests.
- Use temporary directories in tests.
- Run `go test -race ./...`.
- Run `go vet ./...`.
- Use static analysis in CI.
- Keep packages cohesive and avoid circular dependencies.
- Avoid reflection-heavy configuration mechanisms.
- Use explicit schema versions.
- Prefer append-only event records.
- Make task IDs sortable and collision-resistant.

---

## 24. Architecture Decision Records

The repository should maintain ADRs for decisions including:

1. Selected local isolation backend.
2. CLI-to-daemon protocol.
3. Policy schema and versioning.
4. Task persistence format.
5. Event schema.
6. Repository clone and promotion strategy.
7. Credential brokering strategy.
8. Network enforcement strategy.
9. Daemon recovery model.
10. Security boundaries and unsupported environments.

Example path:

```text
docs/adr/0001-runtime-backend.md
```

Each ADR should include:

- Context.
- Decision.
- Alternatives considered.
- Security consequences.
- Operational consequences.
- Migration strategy.

---

## 25. Open Questions

1. Which local isolation backend provides the strongest practical Linux and WSL2 developer experience?
2. Should the first daemon persistence layer use SQLite or filesystem records?
3. Should task events be stored only as JSONL or indexed in SQLite as well?
4. How should uncommitted host changes be included safely?
5. Which network-control mechanism is reliable across Linux distributions and WSL2?
6. What is the minimum viable credential broker for private Git clones?
7. How should interactive terminal resize and signal forwarding work?
8. Should promotion apply patches directly or always create a new branch?
9. How should background services be exposed to the user without public host ports?
10. Which changes should be blocked versus warned by default?
11. How should backend capability attestations be represented?
12. What recovery guarantees are possible after daemon or host crashes?
13. How should caches be shared without creating persistence or poisoning risks?
14. Should verification use the same backend or require a fresh backend instance?
15. How should large monorepositories be cloned or copied efficiently?
16. What audit data should be retained by default?
17. How should task-level encryption be handled for sensitive logs or artifacts?
18. What is the safest behavior when agent-native event output is unavailable?
19. How should `isobox` detect and report backend security regressions?
20. What guarantees can be tested automatically on WSL2?

---

## 26. Future Directions

These are explicitly post-MVP possibilities:

- Additional local and remote runtime adapters.
- Native macOS support.
- Native Windows support.
- Team policy distribution.
- Signed policy bundles.
- CI provider integrations.
- Pull-request generation.
- Centralized audit storage.
- Web or desktop task dashboard.
- Remote task workers.
- Hardware acceleration policies.
- Reusable dependency caches with integrity controls.
- Sandboxed data-science execution.
- Student code grading.
- Agent benchmark and comparison suites.
- Policy simulation before task launch.
- Organization-wide credential capabilities.
- Artifact signing and provenance attestations.

---

## 27. Success Metrics

Early product success should be measured by:

- Percentage of tasks completing without unsafe overrides.
- Sandbox startup success rate.
- Cleanup success rate.
- Number of prevented direct host writes.
- Number of blocked unauthorized network requests.
- Percentage of tasks with complete audit records.
- Fresh verification pass rate.
- Frequency of high-risk promotion findings.
- Median setup steps for a new user.
- Number of coding agents supported through direct adapters.
- Reproducibility of repeated task execution.
- Security test coverage and regression count.

Avoid optimizing primarily for raw agent speed until safety, correctness, and cleanup reliability are established.

---

## 28. Release Readiness Checklist

Before a public preview:

- [ ] Threat model reviewed.
- [ ] Security invariants documented.
- [ ] Default policy is deny-oriented.
- [ ] Host repository is protected by default.
- [ ] No raw credentials are injected by default.
- [ ] Network is denied by default.
- [ ] Resource limits are tested.
- [ ] Cleanup is reliable and observable.
- [ ] Runtime downgrade is impossible without explicit override.
- [ ] Codex adapter passes end-to-end tests.
- [ ] OpenCode adapter passes end-to-end tests.
- [ ] Fresh verification works.
- [ ] Promotion checks identify dangerous file classes.
- [ ] Logs redact known secret formats.
- [ ] `go test -race ./...` passes.
- [ ] Static analysis passes.
- [ ] WSL2 quickstart is tested.
- [ ] Failure modes are documented.
- [ ] Unsafe flags are clearly labeled and audited.
- [ ] Example policies are included.
- [ ] Security limitations are stated without marketing exaggeration.

---

## 29. Initial Repository Structure

```text
isobox/
├── cmd/
│   ├── isobox/
│   │   └── main.go
│   └── isoboxd/
│       └── main.go
├── internal/
│   ├── agent/
│   │   ├── agent.go
│   │   ├── codex/
│   │   ├── opencode/
│   │   └── generic/
│   ├── audit/
│   ├── config/
│   ├── credentials/
│   ├── events/
│   ├── network/
│   ├── policy/
│   ├── promotion/
│   ├── rpc/
│   ├── runtime/
│   │   ├── runtime.go
│   │   ├── mock/
│   │   └── local/
│   ├── task/
│   ├── verification/
│   └── workspace/
├── pkg/
│   └── api/
├── policies/
│   ├── safe-coding.yaml
│   ├── offline.yaml
│   └── ci-untrusted.yaml
├── docs/
│   ├── architecture.md
│   ├── threat-model.md
│   ├── security.md
│   └── adr/
├── testdata/
│   └── adversarial-repos/
├── .github/
│   └── workflows/
├── go.mod
├── go.sum
├── LICENSE
├── README.md
└── PRD.md
```

---

## 30. First Implementation Slice

The first coding slice should intentionally avoid real coding-agent integration.

Build a vertical path that:

1. Parses a policy.
2. Creates a task.
3. Uses the mock runtime.
4. Executes a generic command.
5. Emits task and process events.
6. Writes an audit directory.
7. Produces a fake repository diff.
8. Runs deterministic promotion checks.
9. Requires approval.
10. Destroys the task.

Only after this lifecycle is stable should a real runtime backend be added. Only after the runtime backend is reliable should Codex and OpenCode adapters be added.

This order keeps the product architecture testable and prevents agent-specific behavior from defining the core design.

---

## 31. Product Summary

`isobox` is a Go-based sandbox supervisor for safely running coding agents and untrusted development tasks.

Its defining characteristics are:

- Agent-independent.
- Runtime-independent.
- Git-native.
- Local-first.
- Deny-by-default.
- No direct trusted-repository writes.
- No ambient raw credentials.
- Structured auditing.
- Fresh verification.
- Explicit result promotion.
- Built above existing isolation backends rather than replacing them.

The central product promise is:

> Give coding agents freedom inside the box, while keeping the developer, repository, credentials, and wider system outside the blast radius.
