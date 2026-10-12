package release

import "testing"

// The expected values in this table are stated before the implementation: the
// table is the contract of the provenance gate, not a transcript of it.
//
// Decisions pinned here:
//
//   - "0.1.0" without the v prefix is release-shaped (true). dforge tags are
//     v-prefixed, so this repository's release flow never injects that exact
//     string, but the predicate classifies markers the way dflow does — it
//     does not validate tag shape. Shape validation belongs to the release
//     checks, not to this gate.
//   - "DEV" is release-shaped (true): markers are matched exactly and
//     case-sensitively, again mirroring dflow. The build only ever injects
//     lowercase "dev" (the Makefile falls back to `echo dev`), so uppercase
//     DEV is not a marker this repository produces and reserving it would
//     reserve strings the toolchain never emits.
func TestHasReleaseProvenance(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    bool
	}{
		{name: "empty marker", version: "", want: false},
		{name: "dev marker", version: "dev", want: false},
		{name: "snapshot marker", version: "snapshot-abc123", want: false},
		{name: "bare snapshot prefix", version: "snapshot-", want: false},
		{name: "tagged release", version: "v0.1.0", want: true},
		{name: "prerelease tag", version: "v1.2.3-rc.1", want: true},
		{name: "release without v prefix", version: "0.1.0", want: true},
		{name: "whitespace only", version: "   ", want: false},
		{name: "uppercase dev", version: "DEV", want: true},
		{name: "whitespace around dev", version: "  dev  ", want: false},
		{name: "whitespace around release", version: " v1.0.0 ", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasReleaseProvenance(tc.version); got != tc.want {
				t.Fatalf("HasReleaseProvenance(%q) = %t, want %t", tc.version, got, tc.want)
			}
		})
	}
}

// TestClassifyMarker pins the marker kind behind the gate so the self-update
// stage can name the refusal ("development build" versus "snapshot build")
// instead of emitting a generic "not a release".
func TestClassifyMarker(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    MarkerKind
	}{
		{name: "empty", version: "", want: MarkerNone},
		{name: "whitespace only", version: " \t\n", want: MarkerNone},
		{name: "dev", version: "dev", want: MarkerDev},
		{name: "padded dev", version: "  dev  ", want: MarkerDev},
		{name: "snapshot", version: "snapshot-abc123", want: MarkerSnapshot},
		{name: "bare snapshot prefix", version: "snapshot-", want: MarkerSnapshot},
		{name: "release tag", version: "v1.0.0", want: MarkerRelease},
		{name: "uppercase dev", version: "DEV", want: MarkerRelease},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyMarker(tc.version); got != tc.want {
				t.Fatalf("ClassifyMarker(%q) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}
