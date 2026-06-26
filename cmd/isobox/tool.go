package main

import (
	"fmt"

	"isobox/internal/preflight"
)

// toolCmd runs `isobox tool -- <command>` to enter a Tool-Call Sandbox.
//
// The first milestone implements the preflight boundary only: the command
// runs the named preflight checks and refuses to create a Sandbox until
// they all pass. Actual Workspace creation and Workload Command execution
// arrive in later milestones.
func toolCmd(args []string) error {
	if _, err := parseToolCommand(args); err != nil {
		return err
	}

	if err := preflight.Run("."); err != nil {
		return err
	}

	return fmt.Errorf("isobox tool: tool-call execution is not implemented in this milestone")
}

// parseToolCommand extracts the Workload Command from `isobox tool -- <cmd>`.
// The command portion is required so a Cooperative Tool Call that lacks a
// workload is rejected before any preflight work runs.
func parseToolCommand(args []string) ([]string, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			cmd := args[i+1:]
			if len(cmd) == 0 {
				return nil, fmt.Errorf("isobox tool: workload command is required after `--`")
			}
			return cmd, nil
		}
	}
	return nil, fmt.Errorf("isobox tool: usage: isobox tool -- <command> [args...]")
}
