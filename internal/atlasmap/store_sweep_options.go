package atlasmap

import (
	"context"
	"fmt"
	"os"
	"time"
)

// MapSweepOptions supplies per-operation dependencies. Nil fields use the host
// clock, opened-root I/O and background context. Callbacks use root-relative
// paths and must honor ctx. Cancellation is cooperative: a blocked syscall or
// mutex wait is not interruptible and completed removals are not rolled back.
type MapSweepOptions struct {
	Context        context.Context
	Now            func() time.Time
	VisitDirectory func(context.Context, string, func(os.DirEntry) error) error
	Info           func(context.Context, os.DirEntry) (os.FileInfo, error)
	Remove         func(context.Context, string) error
}

func (store *MapStore) sweepOptions(options []MapSweepOptions) (MapSweepOptions, error) {
	if len(options) > 1 {
		return MapSweepOptions{}, fmt.Errorf("at most one map sweep options value is supported")
	}
	var selected MapSweepOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.Context == nil {
		selected.Context = context.Background()
	}
	if err := selected.Context.Err(); err != nil {
		return selected, err
	}
	if selected.Now == nil {
		selected.Now = time.Now
	}
	if selected.VisitDirectory == nil {
		selected.VisitDirectory = func(ctx context.Context, name string, visit func(os.DirEntry) error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return visitMapDirectory(store.root, name, visit)
		}
	}
	if selected.Info == nil {
		selected.Info = func(ctx context.Context, entry os.DirEntry) (os.FileInfo, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return entry.Info()
		}
	}
	if selected.Remove == nil {
		selected.Remove = func(ctx context.Context, name string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return store.root.Remove(name)
		}
	}
	return selected, nil
}
