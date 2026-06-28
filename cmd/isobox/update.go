package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"isobox/internal/update"
)

// updateUsage is the help line emitted for `isobox update` argument
// errors. It is short because the rich per-command help is one flag
// away; the message exists to give the user a next step.
const updateUsage = "usage: isobox update --check"

// updateClientFactory builds the ReleaseClient used by the update
// command. The factory reads ISOBOX_UPDATE_CLIENT from the
// environment; when set, the command treats the value as an absolute
// path to an executable and runs `<path> list` to obtain a JSON
// release list. When unset, the command uses the production GitHub
// Releases HTTP client. The indirection lets integration tests
// substitute a fake client without touching the live network.
func updateClientFactory() (update.ReleaseClient, error) {
	client := os.Getenv("ISOBOX_UPDATE_CLIENT")
	if client == "" {
		return update.NewGitHubReleaseClient(), nil
	}
	return &execReleaseClient{path: client}, nil
}

// execReleaseClient is the test-only ReleaseClient that runs an
// external command to obtain the release list. The contract is
// deliberately small: the command must accept a single `list`
// subcommand and emit a JSON array of releases on stdout. Any non-
// zero exit or non-JSON output is treated as an error so test
// failures surface as a clear message rather than a silent empty
// list.
type execReleaseClient struct {
	path string
}

func (c *execReleaseClient) ListReleases() ([]update.Release, error) {
	cmd := exec.Command(c.path, "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run update client %s list: %w", c.path, err)
	}
	var releases []update.Release
	if err := json.Unmarshal(output, &releases); err != nil {
		return nil, fmt.Errorf("decode update client output: %w", err)
	}
	return releases, nil
}

// updateCmd is the entry point for `isobox update --check`. The
// command is the observability-only slice: it reports the current
// version, the latest stable release, the selected Update Target,
// and any duplicate isobox binaries on PATH, but it does not
// download, verify, or replace anything.
//
// The full update path (download, checksum verification, replace,
// rollback) is intentionally out of scope here; follow-up slices
// will add it.
func updateCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("isobox update: %s", updateUsage)
	}
	if args[0] != "--check" {
		return fmt.Errorf("isobox update: unknown flag %q; %s", args[0], updateUsage)
	}
	if len(args) > 1 {
		return fmt.Errorf("isobox update: %s", updateUsage)
	}

	if err := update.RefuseDevVersion(version); err != nil {
		return err
	}

	lookup := update.NewHostLookup()
	target, err := update.ResolveUpdateTarget(lookup)
	if err != nil {
		return err
	}
	if err := update.CheckManagedTarget(target.Path); err != nil {
		return err
	}

	client, err := updateClientFactory()
	if err != nil {
		return err
	}
	latest, err := update.SelectLatestStable(client)
	if err != nil {
		return err
	}

	status, err := update.CompareVersions(version, latest.TagName)
	if err != nil {
		return err
	}

	fmt.Printf("isobox update --check\n")
	fmt.Printf("  current: %s\n", version)
	fmt.Printf("  latest:  %s\n", latest.TagName)
	fmt.Printf("  status:  %s\n", status)
	fmt.Printf("  target:  %s\n", target.Path)
	if target.IsManaged {
		fmt.Printf("  note:    target looks package-manager-managed; self-update would be refused at the actual update step\n")
	}
	if len(target.Duplicates) > 0 {
		fmt.Printf("  warning: additional isobox binaries on PATH: %s\n", strings.Join(target.Duplicates, ", "))
	}

	if status == update.StatusUpToDate {
		fmt.Println("isobox is already up to date")
	} else {
		fmt.Println("a newer stable release is available")
	}
	return nil
}
