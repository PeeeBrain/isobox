// Package workspace implements the Repository Workspace lifecycle.
//
// A Repository Workspace is a private copy of a trusted Git repository used for
// a single Task Attempt. It is created from a clean Workspace Source, runs a
// Workload Command from its own root, and captures a reviewable diff before
// being disposed of by default.
package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrDirtyWorkspaceSource indicates that the Workspace Source has uncommitted
// changes and cannot be used to create a clean Repository Workspace.
var ErrDirtyWorkspaceSource = errors.New("Workspace Source has uncommitted changes; commit them before running isobox")

// RepositoryWorkspace is a private repository copy where a Workload Command runs.
type RepositoryWorkspace struct {
	root         string
	source       string
	retain       bool
	sourceCommit string
}

// CreateRepository creates a new Repository Workspace from a clean Git Workspace
// Source. It rejects Workspace Sources with uncommitted changes.
func CreateRepository(source string) (*RepositoryWorkspace, error) {
	if err := assertClean(source); err != nil {
		return nil, err
	}

	commit, err := HeadCommit(source)
	if err != nil {
		return nil, err
	}

	root, err := os.MkdirTemp("", "isobox-workspace-*")
	if err != nil {
		return nil, fmt.Errorf("create workspace root: %w", err)
	}

	workspace := &RepositoryWorkspace{root: root, source: source, sourceCommit: commit}
	if err := workspace.materialize(); err != nil {
		_ = workspace.Close()
		return nil, err
	}
	return workspace, nil
}

// Root returns the directory from which the Workload Command should run.
func (w *RepositoryWorkspace) Root() string {
	return filepath.Join(w.root, "workspace")
}

// Diff captures the current changes in the Repository Workspace as a reviewable
// diff.
func (w *RepositoryWorkspace) Diff() (string, error) {
	var buf bytes.Buffer
	cmd := gitCommand(w.Root(), "diff", "--no-ext-diff")
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("capture diff: %w", err)
	}
	return buf.String(), nil
}

// Close disposes of the private Workspace. It is safe to call more than once.
func (w *RepositoryWorkspace) Close() error {
	if w.root == "" {
		return nil
	}
	if w.retain {
		return nil
	}
	err := os.RemoveAll(w.root)
	w.root = ""
	return err
}

// Retain prevents Close from removing the private Workspace and returns the
// retained Repository Workspace path for review or debugging.
func (w *RepositoryWorkspace) Retain() string {
	w.retain = true
	return w.Root()
}

// SourceCommit returns the HEAD commit hash of the Workspace Source that this
// Repository Workspace was created from.
func (w *RepositoryWorkspace) SourceCommit() string {
	return w.sourceCommit
}

// HeadCommit returns the current HEAD commit hash of the Git repository at source.
func HeadCommit(source string) (string, error) {
	out, err := gitCommand(source, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("resolve Workspace Source HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func assertClean(source string) error {
	status, err := gitCommand(source, "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("inspect Workspace Source: %w", err)
	}
	if len(status) != 0 {
		return ErrDirtyWorkspaceSource
	}
	return nil
}

func (w *RepositoryWorkspace) materialize() error {
	if err := os.MkdirAll(w.Root(), 0o755); err != nil {
		return fmt.Errorf("create workspace directory: %w", err)
	}
	if err := gitCommand("", "clone", "--quiet", w.source, w.Root()).Run(); err != nil {
		return fmt.Errorf("clone Workspace Source: %w", err)
	}
	return nil
}

func gitCommand(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd
}
