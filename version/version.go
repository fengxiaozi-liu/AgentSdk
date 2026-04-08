package version

import "runtime/debug"

// Build-time parameters set via -ldflags.
var Version = "unknown"

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	mainVersion := info.Main.Version
	if mainVersion == "" || mainVersion == "(devel)" {
		return
	}
	Version = mainVersion
}
