package cmd

// This file is stage 8 of the releasing layer: `dforge update`. The flow
// mirrors dflow's self-update command (provenance gate → resolve latest
// release → compare → download → verify SHA-256 → extract → atomic replace)
// but uses this repository's stdlib flag dispatcher and its own asset names.
// The command body is runUpdate, written in the same testable shape as
// runVersion: it takes args and writers instead of touching the process
// streams, so every outcome is assertable without a network or a terminal.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yepizrene-devoost/dforge/internal/release"
)

// updateAPIURLEnv names the environment variable that overrides the GitHub
// API base URL the updater queries. It exists so tests can point the command
// at an httptest server and never touch the real network, and so a user
// behind a mirror can redirect the lookup. The default is the public GitHub
// API the installer already documents.
const updateAPIURLEnv = "DFORGE_UPDATE_API_URL"

// updateTarget resolves the binary an update replaces. It is a variable
// rather than a direct call so tests drive the whole flow against a temp file
// instead of the path of the running test binary — replacing the test
// process's own executable would end the suite mid-run.
var updateTarget = defaultUpdateTarget

// renameBinary is a variable, not a direct call, so tests can simulate the
// Windows locked-executable rename failure without building on Windows.
var renameBinary = os.Rename

// defaultUpdateTarget returns the path of the running dforge binary with
// symlinks resolved, so a binary invoked through a symlinked entry is updated
// at its real location rather than replacing the link itself.
//
// os.Executable reports the path the process was started from, which on some
// platforms is a symlink (and on macOS may traverse the /var to /private/var
// link). When EvalSymlinks cannot resolve it, the raw path is the best answer
// available, so the update proceeds there instead of aborting over a path
// quirk.
func defaultUpdateTarget() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not locate the running dforge binary: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return executable, nil
	}
	return resolved, nil
}

// runUpdate implements `dforge update`. The order of the steps is the point:
//
//  1. The provenance gate runs FIRST, before any network call: a binary
//     without release provenance ("dev" or "snapshot-*") has no published
//     version to update from, so it is refused with the reason named, not
//     compared against a release it never came from.
//  2. The latest release is resolved from the GitHub API.
//  3. If the latest tag is the version this binary already reports, the
//     command reports "already up to date" and succeeds without downloading.
//  4. Otherwise the platform archive and checksums.txt are downloaded, the
//     archive's SHA-256 is verified against the published entry (a hard
//     failure on mismatch or missing entry), and the new binary is extracted
//     in memory.
//  5. The running binary is replaced atomically: staged as a temp file in its
//     own directory, chmod'ed to the original mode, renamed over it.
//  6. An old → new summary is printed.
//
// A failure at any step leaves the current binary untouched.
func runUpdate(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dforge update", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "dforge update: unexpected argument %q\n", flags.Arg(0))
		return errors.New("unexpected argument")
	}

	// Step 1: provenance gate. Nothing is fetched until the marker says this
	// binary came from a published release.
	if !release.HasReleaseProvenance(version) {
		reason := provenanceRefusal(version)
		_, _ = fmt.Fprintf(stderr, "dforge update: refusing to update: %s\n", reason)
		return errors.New("this binary has no release provenance")
	}

	// Step 2: resolve the newest published release.
	latest, err := release.LatestRelease(updateBaseURL())
	if err != nil {
		return err
	}

	// Step 3: same version, nothing to do. The comparison is exact equality
	// because both sides are the same tag: the marker was injected from
	// {{.Tag}} at link time and latest.Tag is what GitHub reports for that
	// same release.
	if latest.Tag == version {
		_, err := fmt.Fprintf(stdout, "dforge %s is already up to date\n", version)
		return err
	}

	// Step 4: download, verify, extract — all in memory, nothing on disk yet.
	newBinary, err := fetchVerifiedBinary(latest)
	if err != nil {
		return err
	}

	// Step 5: atomic replacement of the running binary.
	target, err := updateTarget()
	if err != nil {
		return err
	}
	if err := replaceBinary(newBinary, target); err != nil {
		return err
	}

	// Step 6: the summary names both versions and the file that now carries
	// the new one, so a user can verify the claim with `dforge version`.
	if _, err := fmt.Fprintf(stdout, "dforge updated: %s → %s\n", version, latest.Tag); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "binary: %s\n", target)
	return err
}

// updateBaseURL returns the GitHub API base URL to query, honouring the
// updateAPIURLEnv override and falling back to the public API. Blank and
// whitespace-only overrides are ignored so an empty environment variable
// cannot silently turn every lookup into a request to the empty string.
func updateBaseURL() string {
	if override := strings.TrimSpace(os.Getenv(updateAPIURLEnv)); override != "" {
		return override
	}
	return release.DefaultBaseURL
}

// provenanceRefusal names the exact reason the provenance gate refuses. It
// uses release.ClassifyMarker so the message distinguishes a development
// build from a snapshot build from a binary with no marker at all — a user
// told only "not a release" cannot tell which of their three build paths
// produced the binary they are holding.
func provenanceRefusal(marker string) string {
	switch release.ClassifyMarker(marker) {
	case release.MarkerDev:
		return fmt.Sprintf("this binary carries the %q version marker: it was built from a development tree, has no release provenance, and therefore has no published version to update from. Install a released build instead (scripts/install.sh).", release.DevMarker)
	case release.MarkerSnapshot:
		return fmt.Sprintf("this binary is a snapshot build (%q): a snapshot is a development cut that was never published as a release, so it has no release provenance and cannot be updated. Install a released build instead (scripts/install.sh).", strings.TrimSpace(marker))
	default:
		return "this binary carries no version marker, so it has no release provenance and no version to compare against the latest release."
	}
}

// fetchVerifiedBinary downloads the host platform's release archive and the
// release's checksums.txt, verifies the archive's SHA-256 against the
// published entry, and returns the extracted binary bytes.
//
// Every failure is fatal by design: a missing asset means the release looks
// incomplete, and an unverifiable or unextractable archive must never reach
// replaceBinary. The archive name comes from release.ArchiveName so this
// command and the installer cannot drift apart on the asset spelling.
func fetchVerifiedBinary(latest *release.Release) ([]byte, error) {
	archiveName, err := release.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	archiveURL, ok := latest.AssetURL(archiveName)
	if !ok {
		return nil, fmt.Errorf("the %s release does not publish the %s asset: the release looks incomplete", latest.Tag, archiveName)
	}
	checksumsURL, ok := latest.AssetURL(release.ChecksumsAssetName)
	if !ok {
		return nil, fmt.Errorf("the %s release does not publish the %s asset: the release looks incomplete", latest.Tag, release.ChecksumsAssetName)
	}

	archive, err := release.Fetch(archiveURL, nil)
	if err != nil {
		return nil, err
	}
	checksums, err := release.Fetch(checksumsURL, nil)
	if err != nil {
		return nil, err
	}
	if err := release.VerifyChecksum(archive, checksums, archiveName); err != nil {
		return nil, err
	}
	return release.ExtractBinary(archive, runtime.GOOS)
}

// replaceBinary atomically replaces the file at target with newBinary.
//
// The sequence is the one scripts/install.sh uses, one level up: stat the
// current binary for its mode, stage the new bytes as a temporary file in
// the target's own directory (same filesystem, so the final rename is atomic
// and a reader sees either the whole old binary or the whole new one, never a
// half-written file), chmod the stage to the original binary's mode, then
// os.Rename it over the target. The stage is removed on every path that does
// not end in the rename, so a failure leaves the running binary untouched.
func replaceBinary(newBinary []byte, target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("could not inspect the current dforge binary at %s: %w", target, err)
	}
	if info.IsDir() {
		return fmt.Errorf("cannot update dforge: %s is a directory, not a binary", target)
	}

	stagePath, err := stageBinary(filepath.Dir(target), newBinary, info.Mode())
	if err != nil {
		return err
	}
	// After a successful rename this remove is a harmless ENOENT, which is
	// why it is not worth branching on.
	defer func() { _ = os.Remove(stagePath) }()

	if err := renameBinary(stagePath, target); err != nil {
		// Windows locks a running executable: the rename fails with
		// ERROR_ACCESS_DENIED, but renaming it aside is allowed. Move it
		// to <target>.old, retry, restore on a failed retry, and remove a
		// stale aside best-effort — never leave the machine without a binary.
		aside := target + ".old"
		_ = os.Remove(aside)
		if asideErr := renameBinary(target, aside); asideErr == nil {
			if err = renameBinary(stagePath, target); err == nil {
				return nil
			}
			if restoreErr := renameBinary(aside, target); restoreErr != nil {
				err = fmt.Errorf("%w (restoring the original also failed: %v)", err, restoreErr)
			}
		}
		return fmt.Errorf("could not replace the dforge binary at %s: %w", target, err)
	}
	return nil
}

// stageBinary writes data to a new temporary file in dir, chmod'ed to mode,
// and returns the stage path. It exists as its own function so the atomicity
// precondition — the stage lives in the target's directory, never in the
// system temp dir, because a rename across filesystems degrades to a copy —
// is directly testable.
//
// The stage is created dot-prefixed so a crash can never leave a file an
// installer would mistake for an artifact.
func stageBinary(dir string, data []byte, mode os.FileMode) (string, error) {
	stage, err := os.CreateTemp(dir, ".dforge.tmp.*")
	if err != nil {
		return "", fmt.Errorf("could not stage the new dforge binary in %s: %w", dir, err)
	}
	stagePath := stage.Name()

	// Remove the stage on every path that does not end in a successful write;
	// replaceBinary owns the stage only after this function returns it.
	complete := false
	defer func() {
		if !complete {
			_ = stage.Close()
			_ = os.Remove(stagePath)
		}
	}()

	if _, err := stage.Write(data); err != nil {
		return "", fmt.Errorf("could not write the staged dforge binary at %s: %w", stagePath, err)
	}
	// Chmod before the rename so the landing binary carries the original
	// mode at the instant it appears, never for a window in between. The
	// mode comes from the binary being replaced, not from a fixed 0755, so
	// an installation with tighter permissions keeps them.
	if err := stage.Chmod(mode); err != nil {
		return "", fmt.Errorf("could not mark the staged dforge binary at %s executable: %w", stagePath, err)
	}
	if err := stage.Close(); err != nil {
		return "", fmt.Errorf("could not finish writing the staged dforge binary at %s: %w", stagePath, err)
	}

	complete = true
	return stagePath, nil
}
