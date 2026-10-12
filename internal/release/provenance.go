// Package release holds the version-provenance gate of the releasing layer:
// the predicates that decide whether a build's version marker identifies a
// published release. Self-update and release-comparison paths ask this package
// before they treat a version as comparable against the latest release;
// development and snapshot builds must be refused with a clear message instead
// of silently compared.
//
// The semantics mirror dflow's HasReleaseProvenance: the check classifies
// markers, it does not validate tag shape.
package release

import "strings"

const (
	// DevMarker is the version marker a build carries when no release tag was
	// injected at link time: a plain `go build`, a `go install`, or the
	// Makefile default while HEAD is not exactly on a tag. A binary reporting
	// DevMarker was built from a development tree and has no published version
	// to compare against.
	DevMarker = "dev"

	// SnapshotPrefix is the version prefix GoReleaser stamps on snapshot
	// builds (`snapshot-<commit>` in .goreleaser.yaml). The dash is part of
	// the prefix: a snapshot is a cut of the development branch that was never
	// published as a release, so "latest release" is not its upgrade path
	// either.
	SnapshotPrefix = "snapshot-"
)

// MarkerKind classifies a version string for the provenance gate. It exists so
// a caller can name the refusal ("development build" versus "snapshot build")
// instead of emitting a generic "not a release".
type MarkerKind int

const (
	// MarkerRelease means the string carries no reserved marker: it is
	// version-shaped and may name a published release. This is the only kind
	// with release provenance.
	MarkerRelease MarkerKind = iota
	// MarkerNone means the string is empty or whitespace-only: no marker was
	// injected at all.
	MarkerNone
	// MarkerDev means the string is exactly DevMarker (surrounding whitespace
	// aside): an untagged development tree.
	MarkerDev
	// MarkerSnapshot means the string starts with SnapshotPrefix: a
	// GoReleaser snapshot build.
	MarkerSnapshot
)

// ClassifyMarker reports which reserved marker the version string carries.
// Comparison is exact and case-sensitive after trimming surrounding
// whitespace, so the reserved set stays precisely the markers the build
// emits: `dev` and `snapshot-*`.
func ClassifyMarker(version string) MarkerKind {
	marker := strings.TrimSpace(version)
	switch {
	case marker == "":
		return MarkerNone
	case marker == DevMarker:
		return MarkerDev
	case strings.HasPrefix(marker, SnapshotPrefix):
		return MarkerSnapshot
	default:
		return MarkerRelease
	}
}

// HasReleaseProvenance reports whether a version marker identifies a
// published release whose version can be compared against the latest one.
//
// It is false for an empty (or whitespace-only) marker and for the two
// markers a non-release build carries: DevMarker (the linker never injected a
// version) and SnapshotPrefix builds (a development cut that was never
// published). Only when this returns true is it meaningful for the caller to
// compare the marker against the latest release and offer an update.
func HasReleaseProvenance(version string) bool {
	return ClassifyMarker(version) == MarkerRelease
}
