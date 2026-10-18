package cli

import (
	"errors"
	"io"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// pipelineFailure handles propagated failures by ownership, never message text.
// Cache misses and optional cache-store failures are handled before this boundary.
func pipelineFailure(stderr io.Writer, err error) int {
	var compiled *compiler.Error
	var driver *cdriver.Error
	var loaded *workspaceloader.LoadError
	var cached *packagecache.Error
	switch {
	case errors.As(err, &compiled):
		return internalError(stderr)
	case errors.As(err, &driver):
		return buildFailure(stderr, err)
	case errors.As(err, &loaded):
		if code, handled := stableWorkspaceLoadFailure(stderr, err); handled {
			return code
		}
		return packageFailure(stderr, err)
	case errors.As(err, &cached):
		return packageFailure(stderr, err)
	default:
		return internalError(stderr)
	}
}
