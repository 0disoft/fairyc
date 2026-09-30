package cli

import (
	"fmt"
	"io"
	"runtime/debug"

	"github.com/0disoft/fairyc/internal/sdkstdlib"
)

// Version inspection never reads the workspace or derives an identity from a mutable checkout.
func runVersion(stdout, stderr io.Writer) int {
	info, _ := debug.ReadBuildInfo()
	if _, err := io.WriteString(stdout, versionText(info)); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func versionText(info *debug.BuildInfo) string {
	compiler, revision, modified := "development", "unknown", "unknown"
	if info != nil {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			compiler = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if setting.Value != "" {
					revision = setting.Value
				}
			case "vcs.modified":
				if setting.Value == "true" || setting.Value == "false" {
					modified = setting.Value
				}
			}
		}
	}
	return fmt.Sprintf("Fairy C Stage 0\ncompiler: %s\nsdk: %s %s\nlanguage: %s\nruntime: %s\nrevision: %s\nmodified: %s\n", compiler, sdkstdlib.SDKIdentity, sdkstdlib.Version, sdkstdlib.LanguageRevision, sdkstdlib.RuntimeABI, revision, modified)
}
