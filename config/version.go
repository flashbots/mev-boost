package config

import "runtime/debug"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		Version = versionFromBuildInfo(Version, info)
	}
}

func versionFromBuildInfo(version string, info *debug.BuildInfo) string {
	if version != "dev" || info == nil || info.Main.Path != "github.com/flashbots/mev-boost" ||
		info.Main.Version == "" || info.Main.Version == "(devel)" {
		return version
	}

	return info.Main.Version
}
