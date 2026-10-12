package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

// pristineVersion captures the marker the package was initialized with, before
// any test mutates it through SetVersion. It is a package-level variable so
// the capture happens during package initialization, while version still holds
// its default value; tests restore it in cleanup so they stay order-independent.
var pristineVersion = version

func TestDefaultVersionMarkerIsDev(t *testing.T) {
	if defaultVersion != "dev" {
		t.Fatalf("defaultVersion = %q, want %q", defaultVersion, "dev")
	}
	if pristineVersion != "dev" {
		t.Fatalf("initial version marker = %q, want %q", pristineVersion, "dev")
	}
}

func TestSetVersionSetter(t *testing.T) {
	t.Cleanup(func() { version = pristineVersion })

	SetVersion("v1.2.3")
	if got := Version(); got != "v1.2.3" {
		t.Fatalf("Version() after SetVersion = %q, want %q", got, "v1.2.3")
	}

	SetVersion(defaultVersion)
	if got := Version(); got != "dev" {
		t.Fatalf("Version() after resetting = %q, want %q", got, "dev")
	}
}

func TestVersionJSONReportShape(t *testing.T) {
	t.Cleanup(func() { version = pristineVersion })
	SetVersion("v1.2.3")

	output := runVersionCommand(t, "--json")

	decoder := json.NewDecoder(bytes.NewBufferString(output))
	var report map[string]json.RawMessage
	if err := decoder.Decode(&report); err != nil {
		t.Fatalf("decode version JSON: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("version JSON contains additional output: %v", err)
	}

	// The document must have exactly the keys version, revision, dirty — no
	// more, no fewer — so machine callers can rely on the shape.
	if len(report) != 3 {
		t.Fatalf("version JSON fields = %v, want exactly version, revision, dirty", report)
	}
	assertJSONField(t, report, "version", "v1.2.3")
	assertJSONField(t, report, "revision", buildRevision())
	assertJSONField(t, report, "dirty", buildDirty())
}

func TestVersionTextReport(t *testing.T) {
	t.Cleanup(func() { version = pristineVersion })
	SetVersion("v1.2.3")

	output := runVersionCommand(t)

	want := "dforge v1.2.3"
	if revision := shortRevision(); revision != "" {
		want += " (" + revision
		if buildDirty() {
			want += "-dirty"
		}
		want += ")"
	}
	want += "\n"

	if output != want {
		t.Fatalf("version text output = %q, want %q", output, want)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"bogus"}, &stdout, &stderr); err == nil {
		t.Fatal("run(bogus) = nil error, want error")
	}
	if stdout.Len() != 0 {
		t.Fatalf("unknown command wrote to stdout: %q", stdout.String())
	}
}

func TestVersionRejectsExtraArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version", "extra"}, &stdout, &stderr); err == nil {
		t.Fatal("run(version extra) = nil error, want error")
	}
}

// runVersionCommand executes the version command with the given extra
// arguments and returns its stdout.
func runVersionCommand(t *testing.T, extraArgs ...string) string {
	t.Helper()

	args := append([]string{"version"}, extraArgs...)
	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("run(%v): %v (stderr %q)", args, err, stderr.String())
	}
	return stdout.String()
}

func assertJSONField[T comparable](t *testing.T, report map[string]json.RawMessage, name string, want T) {
	t.Helper()

	raw, ok := report[name]
	if !ok {
		t.Fatalf("version JSON is missing %q", name)
	}
	var got T
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode version JSON field %q: %v", name, err)
	}
	if got != want {
		t.Fatalf("version JSON field %q = %v, want %v", name, got, want)
	}
}
