package main

import (
	"fmt"
	"strings"
)

// helpTopLevel returns the rich top-level help text printed by
// `isobox --help`, `isobox -h`, and `isobox help`. It explains what
// isobox does, lists the available commands with their short purposes,
// points users toward per-command help, and uses the project glossary
// terms so the CLI copy matches the documentation.
func helpTopLevel() string {
	var b strings.Builder
	b.WriteString("isobox manages policy-bound execution of coding agents and untrusted\n")
	b.WriteString("development work while keeping trusted host assets outside the\n")
	b.WriteString("execution boundary.\n")
	b.WriteString("\n")
	b.WriteString("Usage:\n")
	b.WriteString("  isobox <command> [options] [arguments]\n")
	b.WriteString("\n")
	b.WriteString("Commands:\n")
	b.WriteString("  init      Initialize a project policy in the current Git repository.\n")
	b.WriteString("  run       Run a Workload Command in a Session Sandbox with a Repository Workspace.\n")
	b.WriteString("  tool      Run a Cooperative Tool Call in a Tool-Call Sandbox (requires project policy).\n")
	b.WriteString("  promote   Apply a reviewed Task Result to the trusted repository.\n")
	b.WriteString("  version   Print the isobox version, commit, and build date.\n")
	b.WriteString("  doctor    Run read-only Doctor Checks and report Doctor Findings.\n")
	b.WriteString("\n")
	b.WriteString("Glossary terms used throughout this help: Task, Workspace, Sandbox,\n")
	b.WriteString("Task Record, Task Result, Promotion, and Workload Command.\n")
	b.WriteString("\n")
	b.WriteString("Run `isobox <command> --help` for command-specific usage and examples.\n")
	return b.String()
}

// helpForCommand returns the per-command help text for the given command
// name. The second return value reports whether a help entry exists for
// the command. Per-command help includes the usage line, a short purpose,
// and one or more examples so users can copy a minimal command and adapt
// it. The text uses the same project glossary terms as the top-level help
// so the CLI copy stays consistent.
func helpForCommand(name string) (string, bool) {
	switch name {
	case "init":
		return helpInit(), true
	case "run":
		return helpRun(), true
	case "tool":
		return helpTool(), true
	case "promote":
		return helpPromote(), true
	case "version":
		return helpVersion(), true
	case "doctor":
		return helpDoctor(), true
	default:
		return "", false
	}
}

// helpUnknownCommand returns the concise actionable usage shown when the
// user passes a command name that isobox does not recognize. The message
// must remain short: the rich top-level help is one flag away, and a long
// unknown-command message hides the next step.
func helpUnknownCommand() string {
	return "usage: isobox <init|run|tool|promote|version|doctor>\n" +
		"run `isobox --help` for the full command list.\n"
}

// helpMissingCommand returns the concise actionable usage shown when the
// user runs `isobox` with no arguments at all.
func helpMissingCommand() string {
	return "usage: isobox <init|run|tool|promote|version|doctor>\n" +
		"run `isobox --help` for the full command list.\n"
}

func helpInit() string {
	var b strings.Builder
	b.WriteString("isobox init - Initialize a project policy\n")
	b.WriteString("\n")
	b.WriteString("Usage:\n")
	b.WriteString("  isobox init [path]\n")
	b.WriteString("\n")
	b.WriteString("Generates a restrictive default project policy in the Git repository\n")
	b.WriteString("that contains the given directory (default: current working directory).\n")
	b.WriteString("The policy is written at the Git root as .isobox/config.yaml and\n")
	b.WriteString(".isobox/tasks/ is added to .gitignore. A generated project policy is\n")
	b.WriteString("required before Cooperative Tool Calls (isobox tool) can run.\n")
	b.WriteString("\n")
	b.WriteString("Examples:\n")
	b.WriteString("  isobox init\n")
	b.WriteString("  isobox init /path/to/repository\n")
	b.WriteString("\n")
	b.WriteString("Related: Task, Workspace, Sandbox, Tool-Call Sandbox, Project Policy.\n")
	return b.String()
}

func helpRun() string {
	var b strings.Builder
	b.WriteString("isobox run - Run a Workload Command in a Session Sandbox\n")
	b.WriteString("\n")
	b.WriteString("Usage:\n")
	b.WriteString("  isobox run --source <path> --records <path> [--retain-workspace]\n")
	b.WriteString("           [--reuse-input kind=value]... -- <command> [args...]\n")
	b.WriteString("\n")
	b.WriteString("Clones a Git repository into a disposable private Workspace, runs the\n")
	b.WriteString("Workload Command inside that Workspace, and captures a Task Record\n")
	b.WriteString("with the Task Result for later review and Promotion.\n")
	b.WriteString("\n")
	b.WriteString("Examples:\n")
	b.WriteString("  isobox run --source /path/to/repository --records /tmp/tasks \\\n")
	b.WriteString("      -- sh -c 'printf changed > README.md'\n")
	b.WriteString("  isobox run --source /path/to/repository --records /tmp/tasks \\\n")
	b.WriteString("      --reuse-input host_binary=/usr/local/bin/codex -- codex\n")
	b.WriteString("\n")
	b.WriteString("Related: Task, Task Record, Task Result, Workspace, Promotion,\n")
	b.WriteString("Workload Command, Host Agent Reuse.\n")
	return b.String()
}

func helpTool() string {
	var b strings.Builder
	b.WriteString("isobox tool - Run a Cooperative Tool Call in a Tool-Call Sandbox\n")
	b.WriteString("\n")
	b.WriteString("Usage:\n")
	b.WriteString("  isobox tool -- <command> [args...]\n")
	b.WriteString("\n")
	b.WriteString("Runs the Workload Command inside a bubblewrap Tool-Call Sandbox built\n")
	b.WriteString("from the current project's Repository Workspace. Requires an initialized\n")
	b.WriteString("project policy (.isobox/config.yaml), Preflight Rules pass, and\n")
	b.WriteString("bubblewrap on the host PATH. The -- separator is required and the\n")
	b.WriteString("command after -- becomes the Workload Command.\n")
	b.WriteString("\n")
	b.WriteString("Examples:\n")
	b.WriteString("  isobox tool -- sh -c 'printf changed > README.md'\n")
	b.WriteString("  isobox tool -- ls -la\n")
	b.WriteString("\n")
	b.WriteString("Related: Task, Task Record, Task Result, Sandbox, Workload Command,\n")
	b.WriteString("Cooperative Tool Call, Project Policy, Preflight Rules, Promotion.\n")
	return b.String()
}

func helpPromote() string {
	var b strings.Builder
	b.WriteString("isobox promote - Apply a reviewed Task Result to the trusted repository\n")
	b.WriteString("\n")
	b.WriteString("Usage:\n")
	b.WriteString("  isobox promote [--yes] <task-record-dir>\n")
	b.WriteString("\n")
	b.WriteString("Loads a Task Record, validates that the Task Attempt Outcome can be\n")
	b.WriteString("promoted, prints the Promotion Report for review, asks for human\n")
	b.WriteString("confirmation (or accepts --yes for explicit non-interactive Promotion),\n")
	b.WriteString("and then applies the captured Task Result to the trusted repository.\n")
	b.WriteString("\n")
	b.WriteString("Examples:\n")
	b.WriteString("  isobox promote .isobox/tasks/task-0123456789abcdef\n")
	b.WriteString("  isobox promote --yes .isobox/tasks/task-0123456789abcdef\n")
	b.WriteString("\n")
	b.WriteString("Related: Task Record, Task Result, Promotion, Promotion Approval,\n")
	b.WriteString("Promotion Confirmation, Promotion Report, Workspace.\n")
	return b.String()
}

func helpVersion() string {
	var b strings.Builder
	b.WriteString("isobox version - Print the isobox version\n")
	b.WriteString("\n")
	b.WriteString("Usage:\n")
	b.WriteString("  isobox version\n")
	b.WriteString("\n")
	b.WriteString("Prints the isobox version, source commit, and build date recorded at\n")
	b.WriteString("compile time. Useful when reporting issues or confirming which release\n")
	b.WriteString("is currently on the Update Target.\n")
	b.WriteString("\n")
	b.WriteString("Related: Update Target.\n")
	return b.String()
}

func helpDoctor() string {
	var b strings.Builder
	b.WriteString("isobox doctor - Run read-only Doctor Checks\n")
	b.WriteString("\n")
	b.WriteString("Usage:\n")
	b.WriteString("  isobox doctor [path]\n")
	b.WriteString("\n")
	b.WriteString("Runs read-only Doctor Checks and reports Doctor Findings with severities\n")
	b.WriteString("ok, warning, or error. Exits with status 1 only when any finding has\n")
	b.WriteString("severity error. With no path argument, doctor runs global checks; with\n")
	b.WriteString("a directory path inside a Git repository, doctor also runs project\n")
	b.WriteString("checks. doctor never creates or modifies host or project state.\n")
	b.WriteString("\n")
	b.WriteString("Examples:\n")
	b.WriteString("  isobox doctor\n")
	b.WriteString("  isobox doctor /path/to/repository\n")
	b.WriteString("\n")
	b.WriteString("Related: Doctor Check, Doctor Finding, Update Target, Sandbox,\n")
	b.WriteString("Workload Command.\n")
	return b.String()
}

// printTopLevelHelp writes the top-level help text to stdout.
func printTopLevelHelp() {
	fmt.Print(helpTopLevel())
}

// printCommandHelp writes the per-command help text for the given command
// name to stdout. When the command is unknown, it falls back to the rich
// top-level help so the user is never left without a next step.
func printCommandHelp(name string) {
	if text, ok := helpForCommand(name); ok {
		fmt.Print(text)
		return
	}
	fmt.Print(helpTopLevel())
}
