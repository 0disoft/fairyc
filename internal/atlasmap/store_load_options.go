package atlasmap

import (
	"context"
	"fmt"
	"os"
)

// MapLoadOptions override dependencies for one load. Nil fields use the opened
// store root and a background context. Callback paths are root-relative.
// Callbacks must honor the supplied context; host filesystem syscalls themselves
// cannot be interrupted. ReadGeneration must report verified corruption with
// ErrStoredMapCorrupt; other failures never authorize deletion.
type MapLoadOptions struct {
	Context        context.Context
	ReadGeneration func(context.Context, string, string, string) (Partition, error)
	VisitDirectory func(context.Context, string, func(os.DirEntry) error) error
	Remove         func(context.Context, string) error
}

func (store *MapStore) loadOptions(options []MapLoadOptions) (MapLoadOptions, error) {
	if len(options) > 1 {
		return MapLoadOptions{}, fmt.Errorf("at most one map load options value is supported")
	}
	var selected MapLoadOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.Context == nil {
		selected.Context = context.Background()
	}
	if err := selected.Context.Err(); err != nil {
		return selected, err
	}
	if selected.ReadGeneration == nil {
		selected.ReadGeneration = func(ctx context.Context, path, name, directory string) (Partition, error) {
			if err := ctx.Err(); err != nil {
				return Partition{}, err
			}
			return store.readGeneration(path, name, directory)
		}
	}
	if selected.VisitDirectory == nil {
		selected.VisitDirectory = func(ctx context.Context, name string, visit func(os.DirEntry) error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return visitMapDirectory(store.root, name, visit)
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
