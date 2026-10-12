package release

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestArchiveName pins the asset-name matrix for all six platforms dforge
// publishes, plus the two refusals. The expected strings are stated before
// the implementation: they are the contract with .goreleaser.yaml's
// name_template (title-cased OS, amd64 rewritten to x86_64, .tar.gz except
// on Windows where format_overrides makes it .zip), and a mismatch here is a
// 404 against a release that exists.
func TestArchiveName(t *testing.T) {
	cases := []struct {
		name    string
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{name: "linux amd64", goos: "linux", goarch: "amd64", want: "dforge_Linux_x86_64.tar.gz"},
		{name: "linux arm64", goos: "linux", goarch: "arm64", want: "dforge_Linux_arm64.tar.gz"},
		{name: "darwin amd64", goos: "darwin", goarch: "amd64", want: "dforge_Darwin_x86_64.tar.gz"},
		{name: "darwin arm64", goos: "darwin", goarch: "arm64", want: "dforge_Darwin_arm64.tar.gz"},
		{name: "windows amd64", goos: "windows", goarch: "amd64", want: "dforge_Windows_x86_64.zip"},
		{name: "windows arm64", goos: "windows", goarch: "arm64", want: "dforge_Windows_arm64.zip"},
		{name: "unsupported os", goos: "freebsd", goarch: "amd64", wantErr: true},
		{name: "unsupported arch", goos: "linux", goarch: "386", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ArchiveName(tc.goos, tc.goarch)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ArchiveName(%s, %s) = %q, want an error", tc.goos, tc.goarch, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ArchiveName(%s, %s): %v", tc.goos, tc.goarch, err)
			}
			if got != tc.want {
				t.Fatalf("ArchiveName(%s, %s) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}

// TestLatestRelease drives the resolver against an httptest server: valid
// JSON resolves tag and asset URLs, malformed JSON fails, an API error
// status fails with an actionable message, and a payload without a tag is
// rejected. No test in this file may open a socket to the real network.
func TestLatestRelease(t *testing.T) {
	t.Run("valid json resolves tag and assets", func(t *testing.T) {
		var gotPath, gotAccept string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotAccept = r.Header.Get("Accept")
			_, _ = fmt.Fprint(w, `{"tag_name":"v1.2.3","assets":[
				{"name":"dforge_Linux_arm64.tar.gz","browser_download_url":"http://example.invalid/dforge_Linux_arm64.tar.gz"},
				{"name":"checksums.txt","browser_download_url":"http://example.invalid/checksums.txt"},
				{"name":"no-url"}
			]}`)
		}))
		defer server.Close()

		latest, err := LatestRelease(server.URL)
		if err != nil {
			t.Fatalf("LatestRelease: %v", err)
		}
		if latest.Tag != "v1.2.3" {
			t.Fatalf("Tag = %q, want %q", latest.Tag, "v1.2.3")
		}
		// The resolver must hit the exact documented endpoint, not a
		// hand-assembled cousin of it, or the API contract drifts silently.
		if want := "/repos/yepizrene-devoost/dforge/releases/latest"; gotPath != want {
			t.Fatalf("request path = %q, want %q", gotPath, want)
		}
		if gotAccept != "application/vnd.github+json" {
			t.Fatalf("Accept header = %q, want %q", gotAccept, "application/vnd.github+json")
		}
		url, ok := latest.AssetURL("dforge_Linux_arm64.tar.gz")
		if !ok || url != "http://example.invalid/dforge_Linux_arm64.tar.gz" {
			t.Fatalf("AssetURL = (%q, %t), want the published URL", url, ok)
		}
		if _, ok := latest.AssetURL("no-url"); ok {
			t.Fatal("an asset without a download URL resolved; it must be skipped")
		}
		if _, ok := latest.AssetURL("missing.txt"); ok {
			t.Fatal("an absent asset resolved; it must report false")
		}
	})

	t.Run("malformed json fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `{"tag_name": "v1.`)
		}))
		defer server.Close()

		if _, err := LatestRelease(server.URL); err == nil {
			t.Fatal("LatestRelease(malformed json) = nil error, want error")
		}
	})

	t.Run("api error status fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		_, err := LatestRelease(server.URL)
		if err == nil {
			t.Fatal("LatestRelease(HTTP 403) = nil error, want error")
		}
		// A bare status code reads like a bug; the message must carry the
		// actionable rate-limit explanation.
		if !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "rate limit") {
			t.Fatalf("error = %q, want it to name the status and the rate limit", err)
		}
	})

	t.Run("missing tag name fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, `{"assets":[]}`)
		}))
		defer server.Close()

		if _, err := LatestRelease(server.URL); err == nil {
			t.Fatal("LatestRelease(payload without tag) = nil error, want error")
		}
	})

	t.Run("base url fallback needs no network", func(t *testing.T) {
		// The blank-override fallback is what keeps a caller with no endpoint
		// from building a request against the empty string. It is asserted on
		// the pure resolver rather than through LatestRelease("") precisely so
		// the deterministic suite never opens a socket to the real API.
		cases := []struct {
			name string
			in   string
			want string
		}{
			{name: "blank falls back to the public api", in: "", want: "https://api.github.com"},
			{name: "whitespace falls back too", in: "   ", want: "https://api.github.com"},
			{name: "override wins", in: "https://example.test/api", want: "https://example.test/api"},
			{name: "trailing slash tolerated", in: "https://example.test/api/", want: "https://example.test/api"},
			{name: "surrounding space trimmed", in: " https://example.test/api ", want: "https://example.test/api"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if got := apiBaseURL(tc.in); got != tc.want {
					t.Fatalf("apiBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
				}
			})
		}
		if DefaultBaseURL != "https://api.github.com" {
			t.Fatalf("DefaultBaseURL = %q, want the public GitHub API", DefaultBaseURL)
		}
	})
}

// TestFetch pins the download helper: bodies come back whole, and a non-200
// response is an error rather than a plausible-looking partial payload.
func TestFetch(t *testing.T) {
	t.Run("ok returns the body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("payload"))
		}))
		defer server.Close()

		body, err := Fetch(server.URL, nil)
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if string(body) != "payload" {
			t.Fatalf("Fetch = %q, want %q", body, "payload")
		}
	})

	t.Run("non-200 fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		if _, err := Fetch(server.URL, nil); err == nil {
			t.Fatal("Fetch(HTTP 404) = nil error, want error")
		}
	})
}

// TestVerifyChecksum pins the trust boundary: matching bytes pass, anything
// else is a hard refusal — a corrupted transfer (mismatch), a release that
// never listed the asset (missing entry), or a manifest line that cannot be a
// digest at all (malformed entry).
func TestVerifyChecksum(t *testing.T) {
	archive := []byte("archive bytes")
	correctSum := sha256Hex(archive)

	cases := []struct {
		name      string
		checksums string
		wantSub   string
	}{
		{
			name:      "checksum pass",
			checksums: fmt.Sprintf("%s  dforge_Linux_arm64.tar.gz\n", correctSum),
		},
		{
			name:      "checksum mismatch refuses",
			checksums: fmt.Sprintf("%s  dforge_Linux_arm64.tar.gz\n", strings.Repeat("a", 64)),
			wantSub:   "checksum mismatch",
		},
		{
			name:      "missing entry refuses",
			checksums: fmt.Sprintf("%s  other_asset.tar.gz\n", correctSum),
			wantSub:   "no entry",
		},
		{
			name:      "empty manifest refuses",
			checksums: "",
			wantSub:   "no entry",
		},
		{
			name:      "malformed digest refuses",
			checksums: "not-a-digest  dforge_Linux_arm64.tar.gz\n",
			wantSub:   "not a valid SHA-256 digest",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyChecksum(archive, []byte(tc.checksums), "dforge_Linux_arm64.tar.gz")
			if tc.wantSub == "" {
				if err != nil {
					t.Fatalf("VerifyChecksum = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("VerifyChecksum = nil error, want a refusal")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantSub)
			}
		})
	}

	t.Run("case-insensitive digest passes", func(t *testing.T) {
		checksums := fmt.Sprintf("%s  dforge_Linux_arm64.tar.gz\n", strings.ToUpper(correctSum))
		if err := VerifyChecksum(archive, []byte(checksums), "dforge_Linux_arm64.tar.gz"); err != nil {
			t.Fatalf("VerifyChecksum(uppercase digest) = %v, want nil: hex case is not corruption", err)
		}
	})
}

// TestExtractBinary pins both archive formats: the executable is found at the
// root and in a nested directory, non-executable entries are ignored, an
// archive without the binary is refused with the installer's wording, and
// the format is routed by goos rather than sniffed.
func TestExtractBinary(t *testing.T) {
	const payload = "the new dforge binary"

	tarGz := buildTarGz(t, []tarEntry{
		{name: "LICENSE", body: []byte("license text")},
		{name: "dforge", body: []byte(payload)},
		{name: "README.md", body: []byte("readme")},
	})
	nestedTarGz := buildTarGz(t, []tarEntry{
		{name: "dforge-v1.1.0/dforge", body: []byte(payload)},
	})
	noBinaryTarGz := buildTarGz(t, []tarEntry{
		{name: "LICENSE", body: []byte("license text")},
	})
	zipArchive := buildZip(t, []zipEntry{
		{name: "README.md", body: []byte("readme")},
		{name: "dforge.exe", body: []byte(payload)},
	})
	noBinaryZip := buildZip(t, []zipEntry{
		{name: "README.md", body: []byte("readme")},
	})

	t.Run("tar.gz binary at root", func(t *testing.T) {
		got, err := ExtractBinary(tarGz, "linux")
		if err != nil {
			t.Fatalf("ExtractBinary(tar.gz): %v", err)
		}
		if string(got) != payload {
			t.Fatalf("extracted = %q, want %q", got, payload)
		}
	})

	t.Run("tar.gz nested binary", func(t *testing.T) {
		got, err := ExtractBinary(nestedTarGz, "darwin")
		if err != nil {
			t.Fatalf("ExtractBinary(nested tar.gz): %v", err)
		}
		if string(got) != payload {
			t.Fatalf("extracted = %q, want %q", got, payload)
		}
	})

	t.Run("zip binary", func(t *testing.T) {
		got, err := ExtractBinary(zipArchive, "windows")
		if err != nil {
			t.Fatalf("ExtractBinary(zip): %v", err)
		}
		if string(got) != payload {
			t.Fatalf("extracted = %q, want %q", got, payload)
		}
	})

	t.Run("tar.gz without binary refuses", func(t *testing.T) {
		_, err := ExtractBinary(noBinaryTarGz, "linux")
		if err == nil {
			t.Fatal("ExtractBinary(tar.gz without binary) = nil error, want error")
		}
		if !strings.Contains(err.Error(), `"dforge" binary`) {
			t.Fatalf("error = %q, want the installer's missing-binary wording", err)
		}
	})

	t.Run("zip without binary refuses", func(t *testing.T) {
		if _, err := ExtractBinary(noBinaryZip, "windows"); err == nil {
			t.Fatal("ExtractBinary(zip without binary) = nil error, want error")
		}
	})

	t.Run("format is routed by goos, not sniffed", func(t *testing.T) {
		// A tar.gz payload asked of the windows path must fail as a zip
		// read, proving the routing decision is explicit.
		if _, err := ExtractBinary(tarGz, "windows"); err == nil {
			t.Fatal("ExtractBinary(tar.gz, windows) = nil error, want a zip-read error")
		}
	})

	t.Run("unsupported platform refuses", func(t *testing.T) {
		if _, err := ExtractBinary(tarGz, "freebsd"); err == nil {
			t.Fatal("ExtractBinary(freebsd) = nil error, want error")
		}
	})

	t.Run("corrupt archive refuses", func(t *testing.T) {
		if _, err := ExtractBinary([]byte("not an archive"), "linux"); err == nil {
			t.Fatal("ExtractBinary(corrupt bytes) = nil error, want error")
		}
	})
}

// tarEntry is one file entry of a test tar.gz archive.
type tarEntry struct {
	name string
	body []byte
}

// zipEntry is one file entry of a test zip archive.
type zipEntry struct {
	name string
	body []byte
}

// buildTarGz assembles a tar.gz archive in memory, GoReleaser-style: files
// at the archive root, canonical regular-file type flags.
func buildTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o755,
			Size:     int64(len(entry.body)),
			Typeflag: tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header %q: %v", entry.name, err)
		}
		if _, err := tarWriter.Write(entry.body); err != nil {
			t.Fatalf("write tar body %q: %v", entry.name, err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buffer.Bytes()
}

// buildZip assembles a zip archive in memory.
func buildZip(t *testing.T, entries []zipEntry) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	for _, entry := range entries {
		writer, err := zipWriter.Create(entry.name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", entry.name, err)
		}
		if _, err := writer.Write(entry.body); err != nil {
			t.Fatalf("write zip entry %q: %v", entry.name, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buffer.Bytes()
}

// sha256Hex returns the lower-case hex SHA-256 of data, the shape
// checksums.txt carries.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
