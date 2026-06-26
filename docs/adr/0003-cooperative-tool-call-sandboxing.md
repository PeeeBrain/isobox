# Adopt cooperative tool-call sandboxing with bubblewrap containment

The first Tool-Call Sandbox milestone will support cooperative Agents that route shell actions through `isobox tool -- <command>` while keeping direct shell calls outside isobox tracking and containment claims. This workflow requires an initial Bubblewrap Backend because a host-process-only implementation would be a glorified worktree with traces rather than a meaningful Containment Boundary; project policy, Task Records, Task Artifacts, and explicit Promotion provide the audit and review path around that backend.

## Consequences

- The reusable Agent Skill defines Cooperative Safe Mode: when active, Agents should route all shell actions through `isobox tool` by default.
- Direct Shell Escape remains possible only as conversation-level human-approved guidance and does not create an isobox Task Record.
- `host_process` remains useful for earlier developer-preview workflows, but it is not sufficient for the first Tool-Call Sandbox milestone.
- Every Cooperative Tool Call creates a full Task Record, even when the command appears read-only.
