// dforge bootstraps AI-agent harnesses and engineering policy.
//
// This stage is the binary skeleton only: the version marker is injected at
// build time via ldflags (-X main.version=$(VERSION)) and development builds
// keep "dev". The installed revision is self-reported from the binary's own
// VCS stamp (buildvcs), so the tag comes from the linker while the commit
// identity comes from the Go toolchain.
package main

import (
	"github.com/yepizrene-devoost/dforge/cmd"
)

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
