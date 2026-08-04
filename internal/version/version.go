// Package version exposes build metadata injected via -ldflags.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// AppName is the single place the product name is defined. Changing it here
// changes the binary name reported by the CLI, the config directory and the
// keyring service name.
const AppName = "cooldeck"

// Tagline is the one-line product description used by --help and the TUI.
const Tagline = "A fast and polished terminal dashboard for monitoring and operating Coolify deployments."

// Values injected at build time, see the Makefile.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info is a resolved, printable snapshot of the build metadata.
type Info struct {
	Name     string
	Version  string
	Commit   string
	Date     string
	GoVer    string
	Platform string
}

// Current resolves build metadata, falling back to VCS stamps recorded by the
// Go toolchain when -ldflags were not supplied (e.g. `go install`).
func Current() Info {
	i := Info{
		Name:     AppName,
		Version:  Version,
		Commit:   Commit,
		Date:     Date,
		GoVer:    runtime.Version(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return i
	}
	if i.Version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		i.Version = bi.Main.Version
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if i.Commit == "none" && s.Value != "" {
				i.Commit = shortSHA(s.Value)
			}
		case "vcs.time":
			if i.Date == "unknown" && s.Value != "" {
				i.Date = s.Value
			}
		}
	}
	return i
}

// String renders the multi-line output of `cooldeck version`.
func (i Info) String() string {
	return fmt.Sprintf("%s %s\ncommit: %s\nbuilt: %s\ngo: %s\nplatform: %s",
		i.Name, i.Version, i.Commit, i.Date, i.GoVer, i.Platform)
}

// Short renders a compact identifier suitable for headers and log lines.
func (i Info) Short() string {
	return i.Name + " " + i.Version
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
