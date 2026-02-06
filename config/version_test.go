package config

import (
	"runtime/debug"
	"testing"
)

func TestVersionFromBuildInfo(t *testing.T) {
	bi := &debug.BuildInfo{Main: debug.Module{Version: "v1.11.0"}}
	if got := versionFromBuildInfo("dev", bi); got != "v1.11.0" {
		t.Fatalf("expected v1.11.0, got %q", got)
	}

	biDevel := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}
	if got := versionFromBuildInfo("dev", biDevel); got != "dev" {
		t.Fatalf("expected dev for (devel), got %q", got)
	}

	if got := versionFromBuildInfo("v9.9.9", bi); got != "v9.9.9" {
		t.Fatalf("expected to keep non-dev version, got %q", got)
	}
}
