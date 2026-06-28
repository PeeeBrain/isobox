package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"isobox/internal/update"
)

// updateUsage is the help line emitted for `isobox update` argument
// errors. It is short because the rich per-command help is one flag
// away; the message exists to give the user a next step.
const updateUsage = "usage: isobox update [--check]"

// managedPathPrefixesEnvVar is the environment variable that adds
// extra managed path prefixes to the updater's refusal list. The
// default list already covers the common package-manager-managed
// locations; the env var exists for two narrow cases: (1) power
// users with unusual system-managed install locations who want the
// updater to refuse to replace those binaries, and (2) the
// integration test suite, which uses a temp directory whose path
// would not be in the default list.
//
// The variable holds a single path per line so the value can be
// edited without escaping commas.
const managedPathPrefixesEnvVar = "ISOBOX_UPDATE_MANAGED_PATH_PREFIXES"

// extraManagedPathPrefixes reads the managed-path env var and
// returns the parsed list. Empty lines are skipped. The function is
// called once per `isobox update --check` invocation.
func extraManagedPathPrefixes() []string {
	raw := os.Getenv(managedPathPrefixesEnvVar)
	if raw == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

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

// updateCmd is the entry point for `isobox update` and
// `isobox update --check`. The check mode reports the current
// version, latest stable release, selected Update Target, and any
// duplicate isobox binaries on PATH. The default mode additionally
// downloads the selected release assets, verifies and smoke-tests the
// downloaded binary, replaces the target with a backup, and rolls back
// when post-replacement validation fails.
func updateCmd(args []string) error {
	checkOnly := false
	if len(args) > 0 {
		if args[0] != "--check" {
			return fmt.Errorf("isobox update: unknown flag %q; %s", args[0], updateUsage)
		}
		if len(args) > 1 {
			return fmt.Errorf("isobox update: %s", updateUsage)
		}
		checkOnly = true
	}

	if err := update.RefuseDevVersion(version); err != nil {
		return err
	}

	lookup := update.NewHostLookup()
	target, err := update.ResolveUpdateTarget(lookup)
	if err != nil {
		return err
	}
	if err := update.CheckManagedTargetWithPrefixes(target.Path, extraManagedPathPrefixes()); err != nil {
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
		return nil
	}
	fmt.Println("a newer stable release is available")
	if checkOnly {
		return nil
	}

	prepared, err := update.PrepareRelease(latest, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	defer prepared.Cleanup()
	result, err := update.InstallPreparedRelease(prepared, target)
	if err != nil {
		return err
	}
	fmt.Printf("updated isobox to %s\n", result.InstalledVersion)
	return nil
}
