package cbackendnative

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func resolveToolchains() ([]cdriver.Toolchain, error) {
	toolchains := make([]cdriver.Toolchain, 0, 2)
	for _, name := range []string{"clang", "gcc"} {
		toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, name)
		if err != nil {
			return nil, fmt.Errorf("native C differential requires %s: %w", name, err)
		}
		toolchains = append(toolchains, toolchain)
	}
	return toolchains, nil
}
