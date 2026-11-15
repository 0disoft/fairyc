package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

// CreateStaging replaces random-name generation and exclusive file creation
// together. Nil hooks retain the host operations and Background context.
// Cancellation is cooperative; cleanup still runs after cancellation.
type buildPublicationOptions struct {
	Context       context.Context
	CreateStaging func(*os.Root, string) (string, *os.File, error)
	Promote       func(*os.Root, string, string, bool) error
	Remove        func(*os.Root, string) error
}

func buildPublicationPolicy(options []buildPublicationOptions) (buildPublicationOptions, error) {
	if len(options) > 1 {
		return buildPublicationOptions{}, fmt.Errorf("build publication accepts at most one options value")
	}
	var selected buildPublicationOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.Context == nil {
		selected.Context = context.Background()
	}
	if selected.CreateStaging == nil {
		selected.CreateStaging = createBuildStagingFile
	}
	if selected.Promote == nil {
		selected.Promote = func(root *os.Root, temporary, final string, replace bool) error {
			if replace {
				return root.Rename(temporary, final)
			}
			return root.Link(temporary, final)
		}
	}
	if selected.Remove == nil {
		selected.Remove = (*os.Root).Remove
	}
	return selected, nil
}

func publishBuildArtifactWithOptions(sourceRoot, relativePath string, contents []byte, replace bool, options ...buildPublicationOptions) (err error) {
	policy, err := buildPublicationPolicy(options)
	if err != nil {
		return err
	}
	ctx := policy.Context
	defer func() { err = errors.Join(err, ctx.Err(), context.Cause(ctx)) }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := validateBuildOutput(relativePath); err != nil {
		return err
	}
	if len(contents) == 0 {
		return fmt.Errorf("refusing to write an empty executable")
	}
	root, err := os.OpenRoot(sourceRoot)
	if err != nil {
		return fmt.Errorf("open source root for build output: %w", err)
	}
	defer root.Close()
	localPath := filepath.FromSlash(relativePath)
	parent := filepath.Dir(localPath)
	if err := root.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create build output directory %q: %w", path.Dir(relativePath), err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	temporaryPath, file, err := policy.CreateStaging(root, parent)
	if err != nil {
		return err
	}
	if file == nil {
		return fmt.Errorf("build staging file handle is missing")
	}
	stagingExists := true
	defer func() {
		if stagingExists {
			_ = policy.Remove(root, temporaryPath)
		}
	}()
	defer file.Close()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := bytes.NewReader(contents).WriteTo(file); err != nil {
		return fmt.Errorf("write staged build output: %w", err)
	}
	if err := file.Chmod(0o700); err != nil {
		return fmt.Errorf("set staged build output executable mode: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync staged build output: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close staged build output: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := policy.Promote(root, temporaryPath, localPath, replace); err != nil {
		if replace {
			return fmt.Errorf("replace build output %q: %w", relativePath, err)
		}
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("build output %q already exists; refusing to overwrite it", relativePath)
		}
		return fmt.Errorf("promote build output %q: %w", relativePath, err)
	}
	if replace {
		stagingExists = false
		return nil
	}
	if err := policy.Remove(root, temporaryPath); err != nil {
		// Promotion is the commit point. The final name may already belong to
		// another publisher; deleting it cannot safely roll back our own link.
		return fmt.Errorf("remove staged build output after promotion; final output retained: %w", err)
	}
	stagingExists = false
	return nil
}
