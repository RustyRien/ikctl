// Package version holds build information injected via -ldflags.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Set at build time:
//
//	-X github.com/rustyrien/ikctl/internal/version.Version=v0.1.0
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

// Info is the client build information.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	Date      string `json:"date,omitempty"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// Get returns the build information, falling back to VCS data embedded by the Go toolchain.
func Get() Info {
	info := Info{Version: Version, Commit: Commit, Date: Date, GoVersion: runtime.Version(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch {
			case s.Key == "vcs.revision" && info.Commit == "":
				info.Commit = s.Value
			case s.Key == "vcs.time" && info.Date == "":
				info.Date = s.Value
			}
		}
		if info.Version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			info.Version = bi.Main.Version
		}
	}
	return info
}

// UserAgent returns the HTTP User-Agent for ikctl.
func UserAgent() string {
	return fmt.Sprintf("ikctl/%s (%s/%s)", Get().Version, runtime.GOOS, runtime.GOARCH)
}
