package sourceedit

import (
	"context"
	"fmt"
	"os"

	"github.com/0disoft/fairyc/internal/durablefile"
)

// ApplyOptions supplies per-operation I/O and cancellation without changing
// Apply's call shape. Nil fields retain Background and the host operations.
// Overrides must preserve file ownership and successful-rename semantics.
type ApplyOptions struct {
	Context         context.Context
	CreateTemporary func(*os.Root, string) (*os.File, string, error)
	Write           func(*os.File, []byte) (int, error)
	Chmod           func(*os.File, os.FileMode) error
	Sync            func(*os.File) error
	Close           func(*os.File) error
	Rename          func(*os.Root, string, string) error
}

func editApplyPolicy(options []ApplyOptions) (ApplyOptions, error) {
	if len(options) > 1 {
		return ApplyOptions{}, fmt.Errorf("at most one edit apply options value is supported")
	}
	var policy ApplyOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.CreateTemporary == nil {
		policy.CreateTemporary = durablefile.CreateTemporary
	}
	if policy.Write == nil {
		policy.Write = (*os.File).Write
	}
	if policy.Chmod == nil {
		policy.Chmod = (*os.File).Chmod
	}
	if policy.Sync == nil {
		policy.Sync = (*os.File).Sync
	}
	if policy.Close == nil {
		policy.Close = (*os.File).Close
	}
	if policy.Rename == nil {
		policy.Rename = (*os.Root).Rename
	}
	return policy, nil
}
