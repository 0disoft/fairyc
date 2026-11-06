package cdriver

import "fmt"

const windowsProjectHeaderBytes = 2 << 20
const windowsCompilerHostHeaderBytes = 8 << 20

// Compiler images have a distinct trusted-source entry; stack reservation does
// not implicitly widen source admission. No I/O occurs before this validation.
func validateWindowsProjectInputs(header, literals string, units []string, runtimeSource string, compilerHost bool) error {
	headerLimit := windowsProjectHeaderBytes
	if compilerHost {
		headerLimit = windowsCompilerHostHeaderBytes
	}
	if len(header) == 0 || len(header) > headerLimit || len(literals) == 0 || len(literals) > 2<<20 || len(units) == 0 || len(units) > 4096 || len(runtimeSource) == 0 || len(runtimeSource) > 8<<20 {
		return fmt.Errorf("invalid bounded project C inputs")
	}
	total := len(header) + len(literals) + len(runtimeSource)
	for _, unit := range units {
		if len(unit) == 0 || len(unit) > 2<<20 {
			return fmt.Errorf("invalid bounded project C body")
		}
		total += len(unit)
		if total > 96<<20 {
			return fmt.Errorf("project C transport exceeds 96 MiB")
		}
	}
	return nil
}
