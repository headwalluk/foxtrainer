// Package buildinfo reports the foxtrainer version stamped into, or recorded by Go in, the binary.
package buildinfo

import (
	"runtime/debug"
	"strings"
)

// developmentVersion is reported when neither the linker nor the Go toolchain recorded a version.
const developmentVersion = "0.1.0-dev"

// stampedVersion is set by the Makefile with -ldflags -X.
var stampedVersion = ""

// Version returns the release version: the stamped one, else the module version from `go install`, else a dev marker.
func Version() string {
	version := developmentVersion

	if stampedVersion != "" {
		version = stampedVersion
	} else if info, found := debug.ReadBuildInfo(); found && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = strings.TrimPrefix(info.Main.Version, "v")
	}

	return version
}
