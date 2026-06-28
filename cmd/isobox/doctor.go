package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"isobox/internal/doctor"
	"isobox/internal/doctorenv"
	"isobox/internal/doctorproject"
)

// doctorUsageError is returned when `isobox doctor` receives an invalid
// argument shape (more than one path, a non-existent path, a non-
// directory path). The message is short and points the user at the
// per-command help so they have a next step.
const doctorUsage = "usage: isobox doctor [path]"

// doctorCmd is the entry point for `isobox doctor [path]`. It runs the
// read-only Doctor Checks that establish the 0.1.1 doctor skeleton:
//
//   - the version metadata check is always run
//   - the target path is validated (when supplied) but no checks are
//     performed against it in this slice; the project section is still
//     rendered so the grouping is visible to users from day one
//
// doctor never mutates host or project state, so the call is safe to
// run before any other isobox command. The exit code is 1 when any
// Doctor Finding has severity error; warnings never break the exit
// code so doctor remains useful as a pre-flight smoke test.
func doctorCmd(args []string) error {
	target, err := resolveDoctorTarget(args)
	if err != nil {
		return err
	}

	lookup := doctorenv.NewHostLookup()
	checks := doctorenv.GlobalChecks(doctorenv.CheckInputs{
		Version: version,
		Commit:  commit,
		Lookup:  lookup,
	})
	_, gitErr := exec.LookPath("git")
	_, bwrapErr := exec.LookPath("bwrap")
	projectRoot, projectChecks := doctorproject.Checks(target, gitErr == nil, bwrapErr == nil)
	checks = append(checks, projectChecks...)

	report := doctor.NewReport(version, commit, projectRoot, checks)
	fmt.Print(report.Format())

	if report.ExitCode() == 0 {
		return nil
	}
	return commandExitError{code: report.ExitCode()}
}

// resolveDoctorTarget validates the [path] argument and returns the
// absolute target path, or the empty string when no path was supplied.
// The path must be an existing directory; any other shape (missing,
// regular file, more than one argument) returns a usage-shaped error
// that exits 1 with a clear next step.
func resolveDoctorTarget(args []string) (string, error) {
	switch len(args) {
	case 0:
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("isobox doctor: determine current directory: %w", err)
		}
		return cwd, nil
	case 1:
		path := args[0]
		info, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("isobox doctor: path %s does not exist; %s", path, doctorUsage)
			}
			return "", fmt.Errorf("isobox doctor: inspect %s: %w", path, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("isobox doctor: %s is not a directory; %s", path, doctorUsage)
		}
		return path, nil
	default:
		return "", fmt.Errorf("isobox doctor: %s", doctorUsage)
	}
}
