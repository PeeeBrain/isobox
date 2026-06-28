package doctorproject

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"isobox/internal/doctor"
	"isobox/internal/projectpolicy"
)

// Checks runs read-only project Doctor Checks for target. If git is unavailable
// or target is outside a Git worktree, project checks are skipped.
func Checks(target string, gitAvailable, bwrapAvailable bool) (string, []doctor.Check) {
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", nil
		}
		target = cwd
	}
	if !gitAvailable {
		return "", []doctor.Check{doctor.Warning("project-git", "project checks skipped because git is not on PATH", "Git-based project discovery cannot run without git", "install git and rerun `isobox doctor`")}
	}
	root, ok := gitRoot(target)
	if !ok {
		return "", nil
	}
	config := filepath.Join(root, ".isobox", "config.yaml")
	if _, err := os.Stat(config); err != nil {
		if os.IsNotExist(err) {
			return root, []doctor.Check{doctor.Warning("project-policy", "project policy is missing", "`isobox tool` cannot run in this repository without .isobox/config.yaml", "run `isobox init` at the repository root")}
		}
		return root, []doctor.Check{doctor.Error("project-policy", "project policy cannot be inspected", "`isobox tool` cannot verify this repository's policy", "fix permissions for "+config)}
	}

	data, err := os.ReadFile(config)
	var checks []doctor.Check
	var policy projectpolicy.ProjectPolicy
	parsed := false
	if err != nil {
		checks = append(checks, doctor.Error("project-policy", "project policy cannot be read", "`isobox tool` cannot verify this repository's policy", "fix permissions for "+config))
	} else if p, err := projectpolicy.Parse(data); err != nil {
		checks = append(checks, doctor.Error("project-policy", "project policy is malformed", "`isobox tool` cannot parse .isobox/config.yaml", "repair the YAML or rerun `isobox init` after backing up local changes"))
	} else {
		policy = p
		parsed = true
		checks = append(checks, doctor.OK("project-policy", "project policy is parseable", config))
		checks = append(checks, compatibility(policy)...)
		if policy.ToolCall.Enabled && policy.RuntimeBackend == projectpolicy.RuntimeBackendBubblewrap && !bwrapAvailable {
			checks = append(checks, doctor.Error("project-bwrap", "bubblewrap (bwrap) is required by project policy but is not on PATH", "`isobox tool` cannot create a Tool-Call Sandbox for this repository", "install bubblewrap (bwrap) and ensure it is reachable on PATH"))
		}
	}
	checks = append(checks, filesystemChecks(root)...)
	if parsed {
		_ = policy
	}
	return root, checks
}

func gitRoot(dir string) (string, bool) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	root := strings.TrimSpace(string(out))
	return root, root != ""
}

func compatibility(p projectpolicy.ProjectPolicy) []doctor.Check {
	var out []doctor.Check
	add := func(field, got, want, fix string) {
		out = append(out, doctor.Error("project-policy-"+strings.ReplaceAll(field, ".", "-"), fmt.Sprintf("project policy %s=%q is unsupported", field, got), "`isobox tool` will reject this project policy during preflight", fix))
	}
	if p.RuntimeBackend != projectpolicy.RuntimeBackendBubblewrap {
		add("runtime_backend", p.RuntimeBackend, projectpolicy.RuntimeBackendBubblewrap, "set runtime_backend: bubblewrap in .isobox/config.yaml")
	}
	if p.DevelopmentEnv.PathMode != projectpolicy.PathModeBackendDefault {
		add("development_environment.path_mode", p.DevelopmentEnv.PathMode, projectpolicy.PathModeBackendDefault, "set development_environment.path_mode: backend_default in .isobox/config.yaml")
	}
	if p.WorkspaceSource.Kind != projectpolicy.WorkspaceSourceProjectRoot {
		add("workspace_source.kind", p.WorkspaceSource.Kind, projectpolicy.WorkspaceSourceProjectRoot, "set workspace_source.kind: project_root in .isobox/config.yaml")
	}
	if p.Credentials.Default != projectpolicy.CredentialsDefaultDeny {
		add("credentials.default", p.Credentials.Default, projectpolicy.CredentialsDefaultDeny, "set credentials.default: deny in .isobox/config.yaml")
	}
	if p.Promotion.Mode != projectpolicy.PromotionModeManual {
		add("promotion.mode", p.Promotion.Mode, projectpolicy.PromotionModeManual, "set promotion.mode: manual in .isobox/config.yaml")
	}
	if p.Network.Default != projectpolicy.NetworkDefaultDeny && p.Network.Default != projectpolicy.NetworkDefaultInherited {
		add("network.default", p.Network.Default, "deny|inherited", "set network.default: deny or network.default: inherited in .isobox/config.yaml")
	}
	if len(p.Network.Allow) > 0 {
		out = append(out, doctor.Error("project-policy-network-allow", "project policy network.allow entries are unsupported", "`isobox tool` will reject host/domain allowlists in the first Tool-Call milestone", "remove network.allow entries and use network.default: deny or inherited"))
	}
	return out
}

func filesystemChecks(root string) []doctor.Check {
	var out []doctor.Check
	if gitignoreContains(filepath.Join(root, ".gitignore"), ".isobox/tasks/") {
		out = append(out, doctor.OK("project-gitignore", ".isobox/tasks/ is ignored", filepath.Join(root, ".gitignore")))
	} else {
		out = append(out, doctor.Warning("project-gitignore", ".isobox/tasks/ is not ignored by .gitignore", "Task Records may be accidentally committed", "add .isobox/tasks/ to the repository .gitignore"))
	}
	if dirty(root) {
		out = append(out, doctor.Warning("project-dirty", "trusted repository has uncommitted tracked or untracked changes", "`isobox tool` preflight will reject the repository until it is clean", "commit, stash, or remove the changes before running `isobox tool`"))
	}
	tasks := filepath.Join(root, ".isobox", "tasks")
	iso := filepath.Join(root, ".isobox")
	if info, err := os.Stat(tasks); err == nil {
		if !info.IsDir() {
			out = append(out, doctor.Error("project-task-store", ".isobox/tasks exists but is not a directory", "Task Records cannot be stored", "remove the file and let isobox create the task directory when needed"))
		} else if writable(tasks) {
			out = append(out, doctor.OK("project-task-store", "task store is writable", tasks))
		} else {
			out = append(out, doctor.Error("project-task-store", "task store is not writable", "Task Records cannot be stored", "fix permissions for "+tasks))
		}
	} else if os.IsNotExist(err) {
		if writable(iso) {
			out = append(out, doctor.OK("project-task-store", "task store parent is writable", iso))
		} else {
			out = append(out, doctor.Error("project-task-store", "task store parent is not writable", "Task Records cannot be created", "fix permissions for "+iso))
		}
	}
	return out
}
func gitignoreContains(path, needle string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), needle)
}
func dirty(root string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
}
func writable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0200 != 0
}
