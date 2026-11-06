package config

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionFromBuildInfo(t *testing.T) {
	tests := []struct {
		name    string
		version string
		info    *debug.BuildInfo
		want    string
	}{
		{
			name:    "tagged module installation",
			version: "dev",
			info:    &debug.BuildInfo{Main: debug.Module{Path: "github.com/flashbots/mev-boost", Version: "v1.11.0"}},
			want:    "v1.11.0",
		},
		{
			name:    "pseudo-version module installation",
			version: "dev",
			info:    &debug.BuildInfo{Main: debug.Module{Path: "github.com/flashbots/mev-boost", Version: "v1.11.1-0.20261106000000-203bb9659eea"}},
			want:    "v1.11.1-0.20261106000000-203bb9659eea",
		},
		{
			name:    "linker override takes precedence",
			version: "v1.11-custom",
			info:    &debug.BuildInfo{Main: debug.Module{Path: "github.com/flashbots/mev-boost", Version: "v1.11.0"}},
			want:    "v1.11-custom",
		},
		{
			name:    "local source build",
			version: "dev",
			info:    &debug.BuildInfo{Main: debug.Module{Path: "github.com/flashbots/mev-boost", Version: "(devel)"}},
			want:    "dev",
		},
		{
			name:    "empty module version",
			version: "dev",
			info:    &debug.BuildInfo{Main: debug.Module{Path: "github.com/flashbots/mev-boost"}},
			want:    "dev",
		},
		{
			name:    "unrelated main module",
			version: "dev",
			info:    &debug.BuildInfo{Main: debug.Module{Path: "example.com/another-app", Version: "v2.0.0"}},
			want:    "dev",
		},
		{
			name:    "missing build info",
			version: "dev",
			want:    "dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, versionFromBuildInfo(tt.version, tt.info))
		})
	}
}
