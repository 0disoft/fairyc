package durablefile

import (
	"context"
	"fmt"
	"os"
)

// PublicationOptions overrides dependencies for one operation. Nil fields use
// background cancellation and the opened-root host operations. Callbacks must
// honor ctx; blocked host syscalls cannot be preempted and completed mutations
// are not rolled back. Compare must distinguish a verified mismatch from I/O
// failure: an error never authorizes removal of the destination.
type PublicationOptions struct {
	Context context.Context
	Move    func(context.Context, *os.Root, string, string) error
	Flush   func(context.Context, *os.Root, string, string) error
	Compare func(context.Context, *os.Root, string, []byte) (bool, error)
	Remove  func(context.Context, *os.Root, string) error
}

func publicationOptions(options []PublicationOptions) (PublicationOptions, error) {
	if len(options) > 1 {
		return PublicationOptions{}, fmt.Errorf("at most one publication options value is supported")
	}
	var selected PublicationOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.Context == nil {
		selected.Context = context.Background()
	}
	if err := selected.Context.Err(); err != nil {
		return selected, err
	}
	if selected.Move == nil {
		selected.Move = publicationActionWithContext(rename)
	}
	if selected.Flush == nil {
		selected.Flush = publicationActionWithContext(syncPublished)
	}
	if selected.Compare == nil {
		selected.Compare = compareFile
	}
	if selected.Remove == nil {
		selected.Remove = func(ctx context.Context, root *os.Root, name string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return root.Remove(name)
		}
	}
	return selected, nil
}

func publicationActionWithContext(action publicationAction) func(context.Context, *os.Root, string, string) error {
	return func(ctx context.Context, root *os.Root, from, to string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return action(root, from, to)
	}
}
