package packageartifact

import (
	"errors"

	"github.com/0disoft/fairyc/internal/mir"
)

// ErrClosureValuesUnsupported keeps the legacy package formats closed until
// stable lifted-target identities and source-owned restoration are implemented.
// Whole-program closure MIR has its own exact codec; it is not a package cache.
var ErrClosureValuesUnsupported = errors.New("copied closure values require a new package artifact format")

func rejectLegacyClosureValues(program mir.Program) error {
	if mir.HasCopiedClosureValues(program) {
		return ErrClosureValuesUnsupported
	}
	return nil
}
