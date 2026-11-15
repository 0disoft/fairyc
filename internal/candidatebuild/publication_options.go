package candidatebuild

import (
	"context"
	"errors"
	"fmt"
	"os"
)

const maxPublishedImageBytes = 64 << 20

// PublicationOptions replaces dependencies for one publication. Nil fields use
// Background, crypto/rand.Read and the host filesystem. Callbacks must retain
// the corresponding host operation's semantics. Cancellation is cooperative
// between calls; a blocked filesystem or random-source call is not preempted.
type PublicationOptions struct {
	Context context.Context
	Random  func([]byte) (int, error)
	Write   func(*os.File, []byte) error
	Sync    func(*os.File) error
	Close   func(*os.File) error
	Rename  func(*os.Root, string, string) error
}

// PublishWithOptions preserves Publish's required inputs and result. Cancellation
// after a successful Rename is reported without removing the committed image.
func PublishWithOptions(path string, image []byte, options ...PublicationOptions) error {
	hooks, err := publicationPolicy(options)
	if err != nil {
		return err
	}
	return publish(path, image, hooks)
}

func publicationPolicy(options []PublicationOptions) (publishHooks, error) {
	if len(options) > 1 {
		return publishHooks{}, fmt.Errorf("candidate publication accepts at most one options value")
	}
	var policy PublicationOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	return publishHooks{context: policy.Context, random: policy.Random, write: policy.Write, sync: policy.Sync, close: policy.Close, rename: policy.Rename}, nil
}

func (hooks publishHooks) check(err error) error {
	if hooks.context != nil && hooks.context.Err() != nil {
		return errors.Join(err, hooks.context.Err(), context.Cause(hooks.context))
	}
	return err
}
