package abpcrud

import (
	"fmt"
	"io"
	"runtime/debug"
)

var (
	Version   = "dev"
	Commit    = "local"
	BuildDate = "unknown"
)

func EffectiveVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}
	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return Version
	}
	return info.Main.Version
}

func PrintVersion(w io.Writer) {
	fmt.Fprintf(w, "abp-cli %s\n", EffectiveVersion())
	fmt.Fprintf(w, "commit: %s\n", Commit)
	fmt.Fprintf(w, "built: %s\n", BuildDate)
}
