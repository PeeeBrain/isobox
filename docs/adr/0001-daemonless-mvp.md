# Daemonless MVP

MVP1 will run as a single CLI process instead of introducing an `isoboxd` daemon. The daemon model remains a future option, but the first implementation needs to prove the core loop of creating a Task, materializing a private Workspace, running a Workload Command in a Sandbox, capturing a Task Result, and cleaning up before adding background supervision, socket security, restart reconciliation, or concurrent task management.

The rough POC should validate this loop through a Session Sandbox command that creates a private Workspace, launches an opaque Workload Command, captures stdout/stderr, exit status, Effective Policy metadata, and a repository diff as the initial Task Result, then cleans up disposable execution state. The POC should also exercise the Promotion boundary by allowing a reviewed patch to be applied back to the trusted repository, rather than stopping at patch generation only.
