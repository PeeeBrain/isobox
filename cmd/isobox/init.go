package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"isobox/internal/projectpolicy"
)

// isoboxTasksIgnoreEntry is the .gitignore line isobox init adds so a
// project's Project Task Store is excluded from source control.
const isoboxTasksIgnoreEntry = ".isobox/tasks/"

// initCmd runs `isobox init [path]` to create project-owned Tool-Call Sandbox
// policy at <path>/.isobox/config.yaml. With no path it operates on the
// current working directory.
func initCmd(args []string) error {
	target, err := resolveInitTarget(args)
	if err != nil {
		return err
	}

	if !isInsideGitRepository(target) {
		return fmt.Errorf("isobox init requires a Git repository at %s; the first Tool-Call Sandbox milestone uses the Git repository root as the Workspace Source", target)
	}

	configPath := filepath.Join(target, ".isobox", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("isobox init refused: project policy already exists at %s; remove it before re-initializing", configPath)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s: %w", configPath, err)
	}

	rendered, err := projectpolicy.Default().Render()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(configPath), err)
	}

	if err := os.WriteFile(configPath, []byte(rendered), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}

	if err := ensureGitignoreEntry(filepath.Join(target, ".gitignore"), isoboxTasksIgnoreEntry); err != nil {
		return err
	}

	fmt.Printf("created project policy at %s\n", configPath)
	return nil
}

// ensureGitignoreEntry adds the given entry to a .gitignore file, creating
// the file if it does not already exist. The entry is appended on its own
// line; pre-existing duplicate entries are not modified.
func ensureGitignoreEntry(gitignorePath, entry string) error {
	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", gitignorePath, err)
	}

	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", gitignorePath, err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		if _, err := writer.WriteString("\n"); err != nil {
			return fmt.Errorf("write %s: %w", gitignorePath, err)
		}
	}
	if _, err := writer.WriteString(entry + "\n"); err != nil {
		return fmt.Errorf("write %s: %w", gitignorePath, err)
	}
	return writer.Flush()
}

// resolveInitTarget returns the directory `isobox init` should initialize.
// A positional argument is used as the target; otherwise the current working
// directory is used.
func resolveInitTarget(args []string) (string, error) {
	switch len(args) {
	case 0:
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
		return cwd, nil
	case 1:
		target := args[0]
		abs, err := filepath.Abs(target)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", target, err)
		}
		info, err := os.Stat(abs)
		if err != nil {
			return "", fmt.Errorf("inspect %s: %w", abs, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("isobox init target %s is not a directory", abs)
		}
		return abs, nil
	default:
		return "", fmt.Errorf("usage: isobox init [path]")
	}
}

// isInsideGitRepository reports whether the given directory is inside a Git
// working tree, by asking Git for its own toplevel. A missing or non-Git
// directory returns false.
func isInsideGitRepository(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}
