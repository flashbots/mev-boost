package config

import (
	"runtime/debug"
)

func versionFromBuildInfo(current string, info *debug.BuildInfo) string {
	if current != "dev" || info == nil {
		return current
	}
	// For `go install module@version`, this is typically a semver tag like
	// "v1.11.0". For local builds, it is usually "(devel)".
	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return current
	}
	return info.Main.Version
}

func init() {
	if Version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		Version = versionFromBuildInfo(Version, info)
	}
}
