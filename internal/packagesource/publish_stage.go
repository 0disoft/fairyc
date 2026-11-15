package packagesource

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/durablefile"
)

// Optional per-call operations use host defaults. Context remains the existing
// caller argument (nil means Background). Overrides retain host ownership and
// rename semantics; cooperative checks cannot interrupt a blocked host call.
type sourcePublishOptions struct {
	Random    func([]byte) (int, error)
	Write     func(*os.File, []byte) (int, error)
	Sync      func(*os.File) error
	Close     func(*os.File) error
	Rename    func(*os.Root, string, string) error
	RemoveAll func(*os.Root, string) error
}

func sourcePublicationPolicy(options []sourcePublishOptions) (sourcePublishOptions, error) {
	if len(options) > 1 {
		return sourcePublishOptions{}, fmt.Errorf("at most one package publication options value is supported")
	}
	var policy sourcePublishOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Random == nil {
		policy.Random = rand.Read
	}
	if policy.Write == nil {
		policy.Write = (*os.File).Write
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
	if policy.RemoveAll == nil {
		policy.RemoveAll = (*os.Root).RemoveAll
	}
	return policy, nil
}

func writeSourceStageFile(ctx context.Context, file *os.File, data []byte, policy sourcePublishOptions) (err error) {
	// A failed injected close must not leak the host descriptor.
	defer file.Close()
	defer func() { err = errors.Join(err, policy.Close(file), ctx.Err(), context.Cause(ctx)) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	n, writeErr := policy.Write(file, data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	if err := errors.Join(writeErr, ctx.Err()); err != nil {
		return err
	}
	return policy.Sync(file)
}

func checkSourceStage(root *os.Root, name string, identity os.FileInfo) error {
	current, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if identity == nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(identity, current) {
		return fmt.Errorf("package staging path replaced; preserved: %w", durablefile.ErrIdentityChanged)
	}
	return nil
}

func removeSourceStage(root *os.Root, name string, identity os.FileInfo, remove func(*os.Root, string) error) error {
	if err := checkSourceStage(root, name, identity); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return remove(root, name)
}
