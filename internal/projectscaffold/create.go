package projectscaffold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CreateOptions controls one creation attempt. Nil fields use Background and
// the ordinary exclusive writer/root close. Hooks must retain host semantics.
// Cancellation is cooperative between filesystem operations; rollback still runs.
type CreateOptions struct {
	Context      context.Context
	WriteFile    func(*os.Root, string, []byte) error
	CloseProject func(*os.Root) error
}

func creationPolicy(options []CreateOptions) (CreateOptions, error) {
	if len(options) > 1 {
		return CreateOptions{}, fmt.Errorf("project creation accepts at most one options value")
	}
	var policy CreateOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.WriteFile == nil {
		policy.WriteFile = writeExclusive
	}
	if policy.CloseProject == nil {
		policy.CloseProject = (*os.Root).Close
	}
	return policy, nil
}

// CreateWithOptions preserves Create's output and no-overwrite policy, with
// caller cancellation and per-call I/O controls for deterministic fault tests.
func CreateWithOptions(parentPath string, spec Spec, options ...CreateOptions) (err error) {
	policy, err := creationPolicy(options)
	if err != nil {
		return err
	}
	ctx := policy.Context
	if err := ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	if err := Validate(spec); err != nil {
		return err
	}
	files := scaffoldFiles
	directories := []string{"src", "tests"}
	if spec.Template == "win32" {
		files, directories, err = win32Files()
		if err != nil {
			return &Error{Reason: ReasonFilesystem, Path: "embedded win32 template", Cause: err}
		}
	}
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		return &Error{Reason: ReasonParentUnavailable, Cause: err}
	}
	defer parent.Close()
	// On Windows, Mkdir can succeed for a dangling directory symlink.
	if _, err := parent.Lstat(spec.Name); err == nil {
		return classifyCreateError(spec.Name, fs.ErrExist, true)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return classifyCreateError(spec.Name, err, true)
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	if err := parent.Mkdir(spec.Name, 0o755); err != nil {
		return classifyCreateError(spec.Name, err, true)
	}
	owned, err := parent.Lstat(spec.Name)
	if err != nil || !owned.IsDir() || owned.Mode()&os.ModeSymlink != 0 {
		return &Error{Reason: ReasonRollback, Path: spec.Name, Cause: errors.Join(err, fmt.Errorf("cannot establish created project identity"))}
	}
	defer func() {
		if err != nil {
			err = rollback(parent, spec.Name, owned, err)
		}
	}()
	if err := ctx.Err(); err != nil {
		return errors.Join(err, context.Cause(ctx))
	}
	project, err := parent.OpenRoot(spec.Name)
	if err != nil {
		return classifyCreateError(spec.Name, err, false)
	}
	defer func() {
		if closeErr := policy.CloseProject(project); closeErr != nil {
			err = errors.Join(err, classifyCreateError(spec.Name, closeErr, false))
		}
		err = errors.Join(err, ctx.Err(), context.Cause(ctx))
	}()
	opened, err := project.Stat(".")
	if err != nil || !os.SameFile(owned, opened) {
		return classifyCreateError(spec.Name, errors.Join(err, fmt.Errorf("created project identity changed while opening")), false)
	}
	for _, directory := range directories {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, context.Cause(ctx))
		}
		if err := project.MkdirAll(filepath.FromSlash(directory), 0o755); err != nil {
			return classifyCreateError(filepath.ToSlash(filepath.Join(spec.Name, directory)), err, false)
		}
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, context.Cause(ctx))
		}
		if err := policy.WriteFile(project, file.path, file.content(spec)); err != nil {
			return classifyCreateError(filepath.ToSlash(filepath.Join(spec.Name, file.path)), err, false)
		}
	}
	return nil
}
