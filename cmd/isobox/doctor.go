package main

import "errors"

// doctorCmd is the entry point for `isobox doctor [path]`. The full
// implementation lives in the doctor slice of this work; this stub exists
// so the help and command-dispatch tests can reference the command name
// without the binary failing to build. It returns a usage error that points
// the user to the per-command help until the implementation lands.
func doctorCmd(args []string) error {
	_ = args
	return errors.New("isobox doctor: not yet implemented; run `isobox doctor --help`")
}
