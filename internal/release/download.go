package release

// This file is the network and artifact half of the self-update flow:
// resolving the newest published release from the GitHub API, naming the
// platform's release assets exactly as GoReleaser publishes them, fetching
// them, and verifying and extracting what was fetched — all in memory. It is
// deliberately stdlib-only (net/http, archive/tar, archive/zip, compress/gzip,
// crypto/sha256) so the updater adds no dependency to a binary that ships
// none. The flow mirrors dflow's self-update package; the asset names are
// this repository's own (title-cased OS, x86_64 arch, checksums.txt).

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the public GitHub REST API root the updater queries
	// unless a caller injects another one. Tests point this at an httptest
	// server; the production path uses the same host the installer documents.
	DefaultBaseURL = "https://api.github.com"

	// releaseRepository is the GitHub slug dforge publishes its releases from.
	// It is a constant rather than a parameter because the updater only ever
	// updates dforge: letting a caller point it at another repository would
	// turn a bug into a way to download an arbitrary archive.
	releaseRepository = "yepizrene-devoost/dforge"

	// ChecksumsAssetName is the name of the checksum manifest every release
	// carries, fixed by .goreleaser.yaml (`checksum: name_template:
	// checksums.txt`) and parsed identically by scripts/install.sh.
	ChecksumsAssetName = "checksums.txt"

	// BinaryBaseName is the file name GoReleaser gives the dforge executable
	// on every platform except Windows, where the Go toolchain appends ".exe"
	// (the name scripts/install.ps1 looks for).
	BinaryBaseName = "dforge"

	// apiTimeout bounds one release lookup. Without a timeout a stalled
	// connection would hang `dforge update` forever, and a user who reached
	// for an update wants a prompt answer far more than a complete one.
	apiTimeout = 30 * time.Second

	// downloadTimeout bounds one asset download. A release archive is a few
	// megabytes; a minute of silence means the connection is dead, not slow.
	downloadTimeout = 2 * time.Minute
)

// Release is the subset of a GitHub release the updater needs: its tag and
// its downloadable assets, addressed by the exact names GoReleaser publishes.
type Release struct {
	// Tag is the release tag as GitHub reports it, e.g. "v0.1.0". It is the
	// string the build-time version marker was injected from, so the caller
	// can compare it against the marker byte for byte.
	Tag string

	// assets maps an asset name to its browser_download_url. A map rather
	// than a slice because every consumer looks an asset up by name.
	assets map[string]string
}

// AssetURL returns the download URL of the named release asset and whether
// the release actually carries it. The second return is load-bearing: a
// release missing its platform archive is a real (and common) failure, and
// callers must distinguish "no such asset" from "asset with an empty URL".
func (r *Release) AssetURL(name string) (string, bool) {
	url, ok := r.assets[name]
	return url, ok
}

// releaseResponse is the parsed payload of the GitHub "latest release"
// endpoint. Unknown fields are ignored because GitHub grows its payload over
// time and a new field must never break an already-shipped binary.
type releaseResponse struct {
	TagName string          `json:"tag_name"`
	Assets  []assetResponse `json:"assets"`
}

// assetResponse is one entry of a release's asset list.
type assetResponse struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// apiBaseURL resolves the endpoint a lookup runs against: the injected base
// URL when it carries anything after trimming (tolerating a trailing slash),
// otherwise the public API. It is its own function so the blank-value
// fallback is testable without opening a socket to the real GitHub — a
// deterministic suite never resolves an empty override into a live request.
func apiBaseURL(base string) string {
	if trimmed := strings.TrimSpace(base); trimmed != "" {
		return strings.TrimRight(trimmed, "/")
	}
	return DefaultBaseURL
}

// LatestRelease asks GitHub for the newest published dforge release.
//
// baseURL is injected so tests can drive the resolver against an httptest
// server without touching the real network; a blank value falls back to
// DefaultBaseURL so a caller with no endpoint in mind cannot build a request
// against the empty string. The endpoint is unauthenticated on purpose: the
// releases are public and the installer already documents this policy; the
// trade-off is the per-IP rate limit, which releaseStatusError translates
// into a message a user can act on instead of a bare status code.
func LatestRelease(baseURL string) (*Release, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", apiBaseURL(baseURL), releaseRepository)

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not build the GitHub API request: %w", err)
	}
	// Pin the media type instead of relying on the account default, so the
	// payload shape stays what this parser was written against.
	request.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: apiTimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not reach the GitHub API at %s: %w", endpoint, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, releaseStatusError(response.StatusCode, endpoint)
	}

	var payload releaseResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("could not decode the GitHub release response from %s: %w", endpoint, err)
	}

	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		// An empty tag is the one malformed shape worth rejecting loudly:
		// without it there is no version to compare and no asset path to
		// derive, so pretending the response was usable would only push the
		// failure somewhere less obvious.
		return nil, fmt.Errorf("the GitHub release response did not include a tag name")
	}

	assets := make(map[string]string, len(payload.Assets))
	for _, asset := range payload.Assets {
		// An asset without a name or a download URL is not addressable; skip
		// it rather than register a key that resolves to an empty URL.
		if asset.Name == "" || asset.BrowserDownloadURL == "" {
			continue
		}
		assets[asset.Name] = asset.BrowserDownloadURL
	}

	return &Release{Tag: tag, assets: assets}, nil
}

// releaseStatusError turns a non-200 GitHub response into a message aimed at
// the user rather than the developer: each branch names the likely cause and
// what to do about it, because "HTTP 403" on its own reads like a bug.
func releaseStatusError(status int, endpoint string) error {
	switch {
	case status == http.StatusNotFound:
		return fmt.Errorf("the GitHub API returned 404 Not Found for %s: no release is published for %s yet", endpoint, releaseRepository)
	case status == http.StatusForbidden || status == http.StatusTooManyRequests:
		return fmt.Errorf("the GitHub API returned HTTP %d for %s: the unauthenticated rate limit (60 requests per hour per IP) was probably exceeded, so wait a few minutes and try again", status, endpoint)
	case status >= http.StatusInternalServerError:
		return fmt.Errorf("the GitHub API returned HTTP %d for %s: GitHub is having trouble on its side, so try again later", status, endpoint)
	default:
		return fmt.Errorf("the GitHub API returned HTTP %d for %s", status, endpoint)
	}
}

// ArchiveName returns the release archive file name GoReleaser publishes for
// the given platform, matching .goreleaser.yaml's name_template:
//
//	dforge_Linux_x86_64.tar.gz    dforge_Linux_arm64.tar.gz
//	dforge_Darwin_x86_64.tar.gz   dforge_Darwin_arm64.tar.gz
//	dforge_Windows_x86_64.zip     dforge_Windows_arm64.zip
//
// goos and goarch are parameters rather than a direct runtime.GOOS read so
// the whole matrix is testable from any host; production callers pass the
// runtime values. These strings are not cosmetic: a mismatch produces a 404
// on a release that exists, which is the least diagnosable failure an updater
// can have, so the translation lives in one pinned place.
func ArchiveName(goos, goarch string) (string, error) {
	osToken, archToken, extension, err := archivePlatform(goos, goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s_%s%s", BinaryBaseName, osToken, archToken, extension), nil
}

// archivePlatform resolves the OS token, architecture token and file
// extension for a platform. The errors distinguish an unknown OS from an
// unknown architecture so a user on an unsupported machine learns which half
// is unsupported, not merely that something is.
//
// The mapping is a switch rather than a package-level map on purpose: package
// maps are mutable global state, and this package commits to none.
func archivePlatform(goos, goarch string) (osToken, archToken, extension string, err error) {
	switch goos {
	case "linux":
		osToken = "Linux"
	case "darwin":
		osToken = "Darwin"
	case "windows":
		osToken = "Windows"
		extension = ".zip"
	default:
		return "", "", "", fmt.Errorf("unsupported platform %s/%s: dforge publishes Linux, macOS and Windows builds", goos, goarch)
	}

	switch goarch {
	case "amd64":
		archToken = "x86_64"
	case "arm64":
		archToken = "arm64"
	default:
		return "", "", "", fmt.Errorf("unsupported platform %s/%s: dforge publishes x86_64 and arm64 builds", goos, goarch)
	}

	if extension == "" {
		extension = ".tar.gz"
	}
	return osToken, archToken, extension, nil
}

// Fetch downloads the resource at url into memory.
//
// client is injectable so tests can point at an httptest server; a nil client
// falls back to one carrying downloadTimeout, which keeps a caller from
// accidentally constructing a client that can hang forever. A non-200
// response is an error naming the URL and the status, so a failed download can
// never be mistaken for a complete one.
//
// There is deliberately no size limit: the URL is only ever a release asset
// named by the release metadata, and the bytes are discarded unless their
// published SHA-256 checksum matches (VerifyChecksum). A byte cap would guard
// against a threat the checksum step already rejects, at the cost of
// rejecting a legitimately large archive.
func Fetch(url string, client *http.Client) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: downloadTimeout}
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not build the download request for %s: %w", url, err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not download %s: %w", url, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not download %s: the server returned HTTP %d %s", url, response.StatusCode, http.StatusText(response.StatusCode))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the download from %s: %w", url, err)
	}
	return body, nil
}

// sha256HexLen is the length of a hex-encoded SHA-256 digest, named so the
// "is this digest even well-formed?" check reads as intent, not arithmetic.
const sha256HexLen = sha256.Size * 2

// VerifyChecksum checks a downloaded archive against the checksum manifest
// published alongside it, mirroring the verification scripts/install.sh
// performs before it installs anything: same file (checksums.txt), same
// parsing (whitespace-split fields, exact name match), same refusal wording.
//
// Trust boundary: the checksums file is downloaded from the same release as
// the archive, so this check proves the archive matches what its publisher
// published. It reliably detects truncated, corrupted, or mismatched
// transfers, but it cannot detect a compromised release itself — matching
// bytes from a bad origin verify cleanly. Because this release flow publishes
// no signatures, this checksum match is the strongest integrity guarantee
// available to both installers.
//
// The comparison is case-insensitive because a digest is hex and some
// checksum tools emit upper case; a case difference is not a corrupted
// download. Every failure is fatal to the update by design: an unverifiable
// archive is worse than no update, so the caller must never install one.
func VerifyChecksum(archive, checksums []byte, assetName string) error {
	expected, found := findChecksum(string(checksums), assetName)
	if !found {
		return fmt.Errorf("the checksums file has no entry for %s: the release looks incomplete, so I will not trust the download", assetName)
	}
	if !isSHA256Digest(expected) {
		return fmt.Errorf("the checksum entry for %s is not a valid SHA-256 digest: %q", assetName, expected)
	}

	sum := sha256.Sum256(archive)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(expected, actual) {
		// The expected/actual pair is spelled out because that is the only
		// way a user can tell a truncated download from a swapped archive.
		return fmt.Errorf("checksum mismatch for %s:\n  expected: %s\n  actual:   %s\nrefusing to trust a download that does not match the published checksum", assetName, expected, actual)
	}
	return nil
}

// findChecksum locates the digest for one asset in a checksums file.
//
// The format is the standard sha256sum one — `<digest>  <filename>` — which
// is also what scripts/install.sh parses with awk. Splitting on any run of
// whitespace accepts both the two-space form and older single-space output,
// while a line that does not yield exactly two fields carries no usable entry
// and is skipped: the caller then gets the explicit "no entry" error rather
// than a comparison against a half-parsed digest.
func findChecksum(contents, assetName string) (string, bool) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == assetName {
			return fields[0], true
		}
	}
	return "", false
}

// isSHA256Digest reports whether value looks like a hex-encoded SHA-256
// digest. A published file can carry a corrupted line, and "expected" text
// that cannot be a digest should fail as malformed, not as a confusing
// mismatch.
func isSHA256Digest(value string) bool {
	if len(value) != sha256HexLen {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// ExtractBinary returns the dforge executable's bytes from a downloaded
// release archive, in memory.
//
// The archive format is chosen from goos rather than by sniffing the bytes
// (".tar.gz" for Linux and macOS, ".zip" for Windows, matching
// .goreleaser.yaml's format_overrides), so a corrupted download is reported
// as a corrupted download, not as an unrecognized archive.
//
// Only the executable is taken; the LICENSE, README and CHANGELOG entries the
// archive also carries are ignored. The executable is matched by its base
// name so a release that nests it in a directory still works, even though
// GoReleaser currently places it at the archive root. Because the bytes stay
// in memory until replaceBinary stages them next to the target, no archive
// entry name ever touches the filesystem, so there is no extraction path to
// guard — the entry's name is used for matching only.
func ExtractBinary(archive []byte, goos string) ([]byte, error) {
	switch goos {
	case "windows":
		return extractZip(archive, goos)
	case "linux", "darwin":
		return extractTarGz(archive, goos)
	default:
		return nil, fmt.Errorf("cannot extract a dforge binary for unsupported platform %q: dforge publishes Linux, macOS and Windows builds", goos)
	}
}

// extractTarGz walks a gzip-compressed tar archive and returns the first
// regular-file entry that is the dforge executable.
func extractTarGz(archive []byte, goos string) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("could not read the download as a gzip archive: %w", err)
	}
	defer func() { _ = gzipReader.Close() }()

	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not read the download as a tar archive: %w", err)
		}

		// Directories, symlinks and every other special entry are skipped, and
		// so is the pre-POSIX regular-file marker (archive/tar's deprecated
		// TypeRegA, byte 0x00): GoReleaser always emits the canonical
		// TypeReg, and accepting only that keeps the binary check strict
		// instead of guessing what a legacy archive meant.
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if !isBinaryEntry(header.Name, goos) {
			continue
		}
		body, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("could not read the %q entry from the archive: %w", header.Name, err)
		}
		return body, nil
	}

	return nil, missingBinaryError()
}

// extractZip is the Windows counterpart of extractTarGz.
func extractZip(archive []byte, goos string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("could not read the download as a zip archive: %w", err)
	}

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || !isBinaryEntry(entry.Name, goos) {
			continue
		}
		body, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("could not read the %q entry from the archive: %w", entry.Name, err)
		}
		data, readErr := io.ReadAll(body)
		_ = body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("could not read the %q entry from the archive: %w", entry.Name, readErr)
		}
		return data, nil
	}

	return nil, missingBinaryError()
}

// isBinaryEntry reports whether an archive entry is the dforge executable.
// Archive entry names are always slash-separated, so path.Base (not
// filepath.Base) is the right splitter on every host. Windows releases carry
// "dforge.exe"; see scripts/install.ps1.
func isBinaryEntry(name, goos string) bool {
	base := path.Base(name)
	if base == BinaryBaseName {
		return true
	}
	return goos == "windows" && base == BinaryBaseName+".exe"
}

// missingBinaryError is the refusal for an archive that carries no dforge
// executable. It mirrors the installer's wording because a user hitting it
// may have already run that script and knows its message.
func missingBinaryError() error {
	return fmt.Errorf("the archive did not contain a %q binary: the release looks incomplete", BinaryBaseName)
}
