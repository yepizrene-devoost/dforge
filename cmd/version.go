// Package cmd holds the dforge command surface: `version` and `update`. The
// version marker is injected from main via -ldflags "-X main.version=..."; the
// commit the binary was built from is read from Go's own VCS stamp (buildvcs),
// so no revision is ever injected by hand.
package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
)

// defaultVersion is the marker a development build carries: no release tag was
// injected, so the binary must never claim to be a released version.
const defaultVersion = "dev"

// version is the channel/version marker, overridden at build time via
// -ldflags "-X main.version=$(VERSION)". Development builds keep "dev".
var version = defaultVersion

// SetVersion stores the marker injected by main and makes it the value the
// version command reports.
func SetVersion(v string) {
	version = v
}

// Version returns the marker this binary reports.
func Version() string {
	return version
}

// Execute dispatches os.Args on the command surface and exits non-zero when a
// command fails. It mirrors gojira's SetVersion/Execute wiring: main owns the
// ldflags target, cmd owns the behavior.
func Execute() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

// run is the testable body of Execute: it writes to the given writers instead
// of the process streams so unit tests stay deterministic.
func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stdout)
		return nil
	}

	switch args[0] {
	case "version":
		return runVersion(args[1:], stdout, stderr)
	case "update":
		return runUpdate(args[1:], stdout, stderr)
	case "--version":
		_, err := fmt.Fprintln(stdout, "dforge "+version)
		return err
	case "help", "--help", "-h":
		usage(stdout)
		return nil
	default:
		_, _ = fmt.Fprintf(stderr, "dforge: unknown command %q\n", args[0])
		usage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// runVersion implements `dforge version [--json]`.
func runVersion(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dforge version", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "Print the version as a single JSON document")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "dforge version: unexpected argument %q\n", flags.Arg(0))
		return errors.New("unexpected argument")
	}

	if *jsonOutput {
		return writeVersionJSON(stdout)
	}
	return writeVersionText(stdout)
}

// writeVersionText prints `dforge <marker>` plus the abbreviated commit when
// the binary carries a VCS stamp, suffixed "-dirty" for an uncommitted tree.
// A binary built without a stamp (for example from a release archive) shows
// the marker alone: an absent second token is honest, an empty one is not.
func writeVersionText(out io.Writer) error {
	line := "dforge " + version
	if revision := shortRevision(); revision != "" {
		line += " (" + revision
		if buildDirty() {
			line += "-dirty"
		}
		line += ")"
	}
	_, err := fmt.Fprintln(out, line)
	return err
}

// writeVersionJSON prints the machine-readable report of `dforge version
// --json`: one document with exactly the keys version, revision and dirty.
// Revision is the full commit hash (empty when unstamped), never a sentence.
func writeVersionJSON(out io.Writer) error {
	report := versionReport{
		Version:  version,
		Revision: buildRevision(),
		Dirty:    buildDirty(),
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(b))
	return err
}

// versionReport is the machine-readable output of `dforge version --json`.
// Field order is the document order encoding/json produces.
type versionReport struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
	Dirty    bool   `json:"dirty"`
}

func usage(w io.Writer) {
	_, _ = fmt.Fprint(w, `dforge bootstraps AI-agent harnesses and engineering policy.

Usage:
  dforge <command> [flags]

Commands:
  version     Show the version marker and the commit this binary was built from
  update      Update dforge to the latest published release
  help        Show this help

Flags:
  --version   Print the version marker
`)
}

// buildInfo returns the embedded build/VCS info, or nil when absent (for
// example, a build without a git stamp).
func buildInfo() *debug.BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	return info
}

func settingValue(settings []debug.BuildSetting, key string) string {
	for _, s := range settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

// buildRevision returns the full 40-char commit hash, or "" when the binary
// carries no VCS stamp.
func buildRevision() string {
	info := buildInfo()
	if info == nil {
		return ""
	}
	return settingValue(info.Settings, "vcs.revision")
}

// shortRevision returns the 7-character abbreviation `git log --oneline`
// prints, so a user can copy the token straight back into a Git command. A
// stamp shorter than the abbreviation is returned whole.
func shortRevision() string {
	revision := buildRevision()
	if len(revision) > 7 {
		return revision[:7]
	}
	return revision
}

// buildDirty reports whether the tree had uncommitted changes at build time.
func buildDirty() bool {
	info := buildInfo()
	if info == nil {
		return false
	}
	return settingValue(info.Settings, "vcs.modified") == "true"
}
