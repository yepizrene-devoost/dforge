package cmd

// Tests for `dforge update`. The style mirrors version_test.go: runUpdate is
// driven directly with explicit writers, the process-wide version marker is
// set and restored through the pristineVersion capture, and every network
// call lands on an httptest server — no test may reach the real GitHub API.

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dforge/internal/release"
)

// withVersion sets the package version marker for one test and restores the
// pristine value afterwards, so tests stay order-independent — the same
// cleanup contract version_test.go uses.
func withVersion(t *testing.T, marker string) {
	t.Helper()
	t.Cleanup(func() { version = pristineVersion })
	SetVersion(marker)
}

// withUpdateTarget points updateTarget at a test-owned file for one test and
// restores the real resolver afterwards. The real one returns the running
// test binary, and replacing that would end the suite mid-run.
func withUpdateTarget(t *testing.T, path string) {
	t.Helper()
	original := updateTarget
	updateTarget = func() (string, error) { return path, nil }
	t.Cleanup(func() { updateTarget = original })
}

// stubRenameFailures fails the first max staged renames — the Windows running-executable shape — and counts renameBinary calls.
func stubRenameFailures(t *testing.T, max int) *int {
	t.Helper()
	original, calls, tries := renameBinary, 0, 0
	renameBinary = func(oldpath, newpath string) error {
		calls++
		if strings.HasPrefix(filepath.Base(oldpath), ".dforge.tmp.") {
			if tries++; tries <= max {
				return errors.New("simulated ERROR_ACCESS_DENIED: the executable is running")
			}
		}
		return os.Rename(oldpath, newpath)
	}
	t.Cleanup(func() { renameBinary = original })
	return &calls
}

// newFakeBinary creates the file an update is expected to replace, carrying
// the given mode so mode preservation is observable.
func newFakeBinary(t *testing.T, mode os.FileMode) string {
	t.Helper()
	dir := t.TempDir()
	target := filepath.Join(dir, "dforge")
	if err := os.WriteFile(target, []byte("old binary"), 0o644); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	if err := os.Chmod(target, mode); err != nil {
		t.Fatalf("chmod fake binary: %v", err)
	}
	return target
}

// trapServer fails the test on any request it receives. It stands behind the
// API-URL override in provenance tests, proving the gate refuses before a
// single byte would be fetched.
func trapServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the provenance gate let a request through: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	return server
}

// serveRelease starts an httptest server that answers the latest-release
// endpoint for tag and, when archive is non-nil, serves both release assets
// (the platform archive and checksums.txt) at URLs the resolver can follow.
// A nil archive models a release whose payload omits the platform asset.
func serveRelease(t *testing.T, tag, archiveName string, archive, checksums []byte) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/yepizrene-devoost/dforge/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		if archive == nil {
			_, _ = fmt.Fprintf(w, `{"tag_name":%q,"assets":[]}`, tag)
			return
		}
		_, _ = fmt.Fprintf(w, `{"tag_name":%q,"assets":[
			{"name":%q,"browser_download_url":%q},
			{"name":"checksums.txt","browser_download_url":%q}
		]}`, tag, archiveName, server.URL+"/asset/"+archiveName, server.URL+"/asset/checksums.txt")
	})
	if archive != nil {
		mux.HandleFunc("/asset/"+archiveName, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(archive)
		})
		mux.HandleFunc("/asset/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(checksums)
		})
	}
	return server
}

// buildHostArchive builds a release archive for the host platform in memory:
// a tar.gz containing "dforge" on Linux and macOS, a zip containing
// "dforge.exe" on Windows — the exact shapes GoReleaser publishes.
func buildHostArchive(t *testing.T, payload string) []byte {
	t.Helper()

	if runtime.GOOS == "windows" {
		var buffer bytes.Buffer
		writer := zip.NewWriter(&buffer)
		entry, err := writer.Create("dforge.exe")
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(payload)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close zip writer: %v", err)
		}
		return buffer.Bytes()
	}

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{Name: "dforge", Mode: 0o755, Size: int64(len(payload)), Typeflag: tar.TypeReg}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tarWriter.Write([]byte(payload)); err != nil {
		t.Fatalf("write tar body: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buffer.Bytes()
}

// sha256HexOf returns the lower-case hex SHA-256 of data, the shape
// checksums.txt carries.
func sha256HexOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestRunUpdateRefusesWithoutProvenance pins the provenance gate: a dev
// build and a snapshot build are refused with a reason that names their
// marker, before any network call — the trap server proves no request is
// even attempted.
func TestRunUpdateRefusesWithoutProvenance(t *testing.T) {
	cases := []struct {
		name    string
		marker  string
		wantSub string
	}{
		{name: "dev marker", marker: "dev", wantSub: `"dev"`},
		{name: "snapshot marker", marker: "snapshot-abc123", wantSub: "snapshot-abc123"},
		{name: "empty marker", marker: "", wantSub: "no version marker"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withVersion(t, tc.marker)
			t.Setenv(updateAPIURLEnv, trapServer(t).URL)

			var stdout, stderr bytes.Buffer
			err := runUpdate(nil, &stdout, &stderr)
			if err == nil {
				t.Fatal("runUpdate = nil error, want a provenance refusal")
			}
			if stdout.Len() != 0 {
				t.Fatalf("refused update wrote to stdout: %q", stdout.String())
			}
			// The refusal must name the marker kind, not emit a generic
			// "not a release": only the exact reason tells the user which
			// build path produced the binary they are holding.
			if !strings.Contains(stderr.String(), "no release provenance") {
				t.Fatalf("stderr = %q, want the provenance reason", stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.wantSub) {
				t.Fatalf("stderr = %q, want it to contain %q", stderr.String(), tc.wantSub)
			}
		})
	}
}

// TestRunUpdatePassesGateForReleaseMarker pins the pass-through half of the
// gate: a release marker (v1.0.0) reaches the resolver, and a latest release
// equal to the current version reports "already up to date" with a nil error
// — exit 0 — without downloading anything.
func TestRunUpdatePassesGateForReleaseMarker(t *testing.T) {
	withVersion(t, "v1.0.0")

	archiveName, err := release.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("ArchiveName: %v", err)
	}
	// A nil archive models a payload with no downloadable assets: if the
	// command tried to download, the test would fail on the missing asset,
	// which is exactly what "already up to date downloads nothing" means.
	server := serveRelease(t, "v1.0.0", archiveName, nil, nil)
	t.Setenv(updateAPIURLEnv, server.URL)

	var stdout, stderr bytes.Buffer
	if err := runUpdate(nil, &stdout, &stderr); err != nil {
		t.Fatalf("runUpdate(v1.0.0 with latest v1.0.0): %v (stderr %q)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "already up to date") {
		t.Fatalf("stdout = %q, want the up-to-date report", stdout.String())
	}
	if !strings.Contains(stdout.String(), "v1.0.0") {
		t.Fatalf("stdout = %q, want it to name the version", stdout.String())
	}
}

// TestRunUpdateInstallsLatestRelease drives the whole flow end to end against
// an httptest server: gate passes, latest resolves, the archive verifies
// against checksums.txt, the binary is extracted, and the fake target is
// replaced with its mode preserved and an old → new summary on stdout.
func TestRunUpdateInstallsLatestRelease(t *testing.T) {
	withVersion(t, "v1.0.0")

	const payload = "the new dforge binary"
	target := newFakeBinary(t, 0o755)
	withUpdateTarget(t, target)

	archive := buildHostArchive(t, payload)
	archiveName, err := release.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("ArchiveName: %v", err)
	}
	checksums := fmt.Sprintf("%s  %s\n", sha256HexOf(archive), archiveName)
	server := serveRelease(t, "v1.1.0", archiveName, archive, []byte(checksums))
	t.Setenv(updateAPIURLEnv, server.URL)

	var stdout, stderr bytes.Buffer
	if err := runUpdate(nil, &stdout, &stderr); err != nil {
		t.Fatalf("runUpdate: %v (stderr %q)", err, stderr.String())
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read updated binary: %v", err)
	}
	if string(got) != payload {
		t.Fatalf("target content = %q, want %q", got, payload)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("stat updated binary: %v", err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Fatalf("target mode = %v, want the original 0755 preserved", info.Mode().Perm())
		}
	}
	if !strings.Contains(stdout.String(), "v1.0.0 → v1.1.0") {
		t.Fatalf("stdout = %q, want the old → new version summary", stdout.String())
	}
	if !strings.Contains(stdout.String(), "binary: "+target) {
		t.Fatalf("stdout = %q, want it to name the replaced binary", stdout.String())
	}

	// The stage must not outlive the swap: the directory holds the target
	// and nothing else.
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil {
		t.Fatalf("read target directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "dforge" {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("target directory holds %v, want only the replaced binary", names)
	}
}

// TestRunUpdateChecksumMismatchLeavesBinaryUntouched pins the trust
// boundary end to end: a downloaded archive whose SHA-256 does not match the
// published entry is refused, and the current binary keeps its old bytes.
func TestRunUpdateChecksumMismatchLeavesBinaryUntouched(t *testing.T) {
	withVersion(t, "v1.0.0")

	target := newFakeBinary(t, 0o755)
	withUpdateTarget(t, target)

	archive := buildHostArchive(t, "the new dforge binary")
	archiveName, err := release.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("ArchiveName: %v", err)
	}
	checksums := fmt.Sprintf("%s  %s\n", strings.Repeat("a", 64), archiveName)
	server := serveRelease(t, "v1.1.0", archiveName, archive, []byte(checksums))
	t.Setenv(updateAPIURLEnv, server.URL)

	var stdout, stderr bytes.Buffer
	err = runUpdate(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("runUpdate with a mismatched checksum = nil error, want refusal")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %q, want the checksum-mismatch refusal", err)
	}

	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read binary after refusal: %v", readErr)
	}
	if string(got) != "old binary" {
		t.Fatalf("target content = %q, want the original untouched", got)
	}
}

// TestRunUpdateMissingPlatformAssetRefuses pins the incomplete-release
// refusal: a latest release that does not publish the host platform's
// archive fails before anything is written.
func TestRunUpdateMissingPlatformAssetRefuses(t *testing.T) {
	withVersion(t, "v1.0.0")

	target := newFakeBinary(t, 0o755)
	withUpdateTarget(t, target)

	archiveName, err := release.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("ArchiveName: %v", err)
	}
	server := serveRelease(t, "v1.1.0", archiveName, nil, nil)
	t.Setenv(updateAPIURLEnv, server.URL)

	var stdout, stderr bytes.Buffer
	err = runUpdate(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("runUpdate with a missing platform asset = nil error, want error")
	}
	if !strings.Contains(err.Error(), archiveName) {
		t.Fatalf("error = %q, want it to name the missing asset %q", err, archiveName)
	}
}

// TestRunUpdateRejectsExtraArguments mirrors the version command's argument
// discipline: `dforge update` takes no positional arguments.
func TestRunUpdateRejectsExtraArguments(t *testing.T) {
	withVersion(t, "v1.0.0")

	var stdout, stderr bytes.Buffer
	if err := runUpdate([]string{"extra"}, &stdout, &stderr); err == nil {
		t.Fatal("runUpdate(extra) = nil error, want error")
	}
}

// TestReplaceBinary pins the replacement flow against a temp directory: the
// new bytes land, the original file mode is preserved, and no staged file is
// left behind. Failure shapes (a directory, a missing target) error without
// touching anything.
func TestReplaceBinary(t *testing.T) {
	cases := []struct {
		name string
		mode os.FileMode
	}{
		{name: "preserves 0755", mode: 0o755},
		{name: "preserves 0700", mode: 0o700},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := newFakeBinary(t, tc.mode)

			if err := replaceBinary([]byte("new binary"), target); err != nil {
				t.Fatalf("replaceBinary: %v", err)
			}

			got, err := os.ReadFile(target)
			if err != nil {
				t.Fatalf("read target: %v", err)
			}
			if string(got) != "new binary" {
				t.Fatalf("target content = %q, want %q", got, "new binary")
			}
			if runtime.GOOS != "windows" {
				info, err := os.Stat(target)
				if err != nil {
					t.Fatalf("stat target: %v", err)
				}
				if info.Mode().Perm() != tc.mode {
					t.Fatalf("target mode = %v, want %v preserved", info.Mode().Perm(), tc.mode)
				}
			}

			entries, err := os.ReadDir(filepath.Dir(target))
			if err != nil {
				t.Fatalf("read target directory: %v", err)
			}
			if len(entries) != 1 || entries[0].Name() != "dforge" {
				t.Fatalf("target directory holds %d entries, want only the replaced binary", len(entries))
			}
		})
	}

	t.Run("directory target refuses", func(t *testing.T) {
		dir := t.TempDir()
		if err := replaceBinary([]byte("new"), dir); err == nil {
			t.Fatal("replaceBinary(directory) = nil error, want error")
		}
	})

	t.Run("missing target refuses", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "dforge")
		if err := replaceBinary([]byte("new"), missing); err == nil {
			t.Fatal("replaceBinary(missing) = nil error, want error")
		}
	})
}

// TestReplaceBinaryRenameAsideDance pins the R4 correction: the rename-aside
// dance after a failed direct rename, restore after a failed retry, plain path.
func TestReplaceBinaryRenameAsideDance(t *testing.T) {
	t.Run("aside dance lands the new binary", func(t *testing.T) {
		target := newFakeBinary(t, 0o755)
		// No stale <target>.old exists: its ENOENT removal must be tolerated, not surfaced.
		stubRenameFailures(t, 1)
		if err := replaceBinary([]byte("new binary"), target); err != nil {
			t.Fatalf("replaceBinary: %v", err)
		}
		got, err := os.ReadFile(target)
		aside, asideErr := os.ReadFile(target + ".old")
		if err != nil || string(got) != "new binary" || asideErr != nil || string(aside) != "old binary" {
			t.Fatalf("target = %q (%v), aside = %q (%v); want the new binary in place and the old one aside", got, err, aside, asideErr)
		}
		if info, statErr := os.Stat(target); runtime.GOOS != "windows" && (statErr != nil || info.Mode().Perm() != 0o755) {
			t.Fatalf("target stat = %v, err = %v; want mode 0755 preserved", info, statErr)
		}
	})

	t.Run("failed retry restores the aside", func(t *testing.T) {
		target := newFakeBinary(t, 0o755)
		stubRenameFailures(t, 2) // the retry fails too; only the restore may rename
		replaceErr := replaceBinary([]byte("new binary"), target)
		got, err := os.ReadFile(target)
		_, asideErr := os.Stat(target + ".old")
		if replaceErr == nil || err != nil || string(got) != "old binary" || !os.IsNotExist(asideErr) {
			t.Fatalf("replaceErr = %v, target = %q (%v), aside = %v; want failure with the original restored", replaceErr, got, err, asideErr)
		}
	})

	t.Run("normal path renames once with no aside", func(t *testing.T) {
		target := newFakeBinary(t, 0o755)
		renames := stubRenameFailures(t, 0)
		replaceErr := replaceBinary([]byte("new binary"), target)
		if _, asideErr := os.Stat(target + ".old"); replaceErr != nil || *renames != 1 || !os.IsNotExist(asideErr) {
			t.Fatalf("err = %v, renames = %d, aside = %v; want one plain rename and no aside", replaceErr, *renames, asideErr)
		}
	})
}

// TestStageBinary pins the atomicity precondition: the stage lives in the
// target's own directory (a rename across filesystems would degrade to a
// copy), carries the requested mode, and holds exactly the new bytes.
func TestStageBinary(t *testing.T) {
	dir := t.TempDir()

	stagePath, err := stageBinary(dir, []byte("payload"), 0o755)
	if err != nil {
		t.Fatalf("stageBinary: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(stagePath) })

	if filepath.Dir(stagePath) != dir {
		t.Fatalf("stage path = %q, want it inside %q so the rename is same-filesystem atomic", stagePath, dir)
	}
	got, err := os.ReadFile(stagePath)
	if err != nil {
		t.Fatalf("read stage: %v", err)
	}
	if string(got) != "payload" {
		t.Fatalf("stage content = %q, want %q", got, "payload")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(stagePath)
		if err != nil {
			t.Fatalf("stat stage: %v", err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Fatalf("stage mode = %v, want 0755", info.Mode().Perm())
		}
	}
}

// TestUpdateBaseURL pins the endpoint override: a non-blank DFORGE_UPDATE_API_URL
// wins (trimmed), and a blank one falls back to the public API so an empty
// variable can never turn every lookup into a request to "".
func TestUpdateBaseURL(t *testing.T) {
	t.Run("override wins", func(t *testing.T) {
		t.Setenv(updateAPIURLEnv, " https://example.test/api ")
		if got := updateBaseURL(); got != "https://example.test/api" {
			t.Fatalf("updateBaseURL() = %q, want the trimmed override", got)
		}
	})

	t.Run("blank falls back", func(t *testing.T) {
		t.Setenv(updateAPIURLEnv, "   ")
		if got := updateBaseURL(); got != release.DefaultBaseURL {
			t.Fatalf("updateBaseURL() = %q, want %q", got, release.DefaultBaseURL)
		}
	})
}
