// Package projectscaffold owns the deterministic, no-overwrite Fairy C starter project.
package projectscaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

const (
	DefaultVersion  = "0.1.0"
	DefaultLanguage = "fairyc-language/5"
)

// FailureReason is a stable internal category for one rejected or incomplete scaffold attempt.
type FailureReason string

const (
	ReasonInvalidName       FailureReason = "invalid-name"
	ReasonInvalidUID        FailureReason = "invalid-uid"
	ReasonParentUnavailable FailureReason = "parent-unavailable"
	ReasonTargetExists      FailureReason = "target-exists"
	ReasonPermission        FailureReason = "permission-denied"
	ReasonFilesystem        FailureReason = "filesystem-failure"
	ReasonRollback          FailureReason = "rollback-failure"
)

// Error keeps host paths and raw operating-system diagnostics out of stable CLI output while
// retaining the original cause for internal inspection and tests.
type Error struct {
	Reason FailureReason
	Path   string
	Cause  error
}

func (e *Error) Error() string {
	switch e.Reason {
	case ReasonInvalidName:
		return "package name must use portable lowercase kebab-case"
	case ReasonInvalidUID:
		return "package UID must use at least two lowercase dot-separated segments"
	case ReasonParentUnavailable:
		return "project parent directory is unavailable"
	case ReasonTargetExists:
		return fmt.Sprintf("project path %q already exists", e.Path)
	case ReasonPermission:
		return fmt.Sprintf("permission denied while creating %q", e.Path)
	case ReasonRollback:
		return fmt.Sprintf("project creation failed and %q could not be rolled back", e.Path)
	default:
		return fmt.Sprintf("could not create %q", e.Path)
	}
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// Spec is the complete caller-owned identity of one starter project.
type Spec struct {
	Name string
	UID  string
}

// Validate rejects values that cannot be represented by manifest v1 or a single portable project
// directory. It deliberately does not synthesize a stable UID from a movable directory name.
func Validate(spec Spec) error {
	if !projectmanifest.ValidPackageName(spec.Name) || !projectmanifest.ValidPortablePathSegment(spec.Name) {
		return &Error{Reason: ReasonInvalidName}
	}
	if !projectmanifest.ValidPackageUID(spec.UID) {
		return &Error{Reason: ReasonInvalidUID}
	}
	return nil
}

type scaffoldFile struct {
	path    string
	content func(Spec) []byte
}

var scaffoldFiles = []scaffoldFile{
	{path: ".gitignore", content: func(Spec) []byte { return []byte("/build/\n") }},
	{path: "README.md", content: readmeBytes},
	{path: projectmanifest.FileName, content: manifestBytes},
	{path: "src/main.fy", content: func(Spec) []byte { return []byte(mainSource) }},
	{path: "tests/main.fy", content: func(Spec) []byte { return []byte(testSource) }},
}

const mainSource = `module main;

pub fn i32 answer() {
    return 42;
}

fn i32 main() {
    return answer() - 42;
}
`

const testSource = `module tests::main;

import main as app;

@test
fn void answer_is_42() {
    assert(app::answer() == 42);
}
`

func readmeBytes(spec Spec) []byte {
	return []byte(fmt.Sprintf(`# %s

A Fairy C starter package.

## Check and test

Run these commands manually from this directory. The first check validates the workspace and
creates `+"`fairy.lock`"+`.

`+"```text"+`
fairyc check
fairyc test --locked
`+"```"+`

## Run on Windows

With x64 MSVC and a Windows SDK installed:

`+"```text"+`
fairyc run --locked
fairyc build --release --locked
.\build\%s.exe
`+"```"+`

`+"`fairyc run`"+` forwards program output and returns `+"`main`"+`'s `+"`i32`"+` as the process status.
The generated starter returns zero after checking that `+"`answer()`"+` is 42.

## Write to standard output

Replace `+"`src/main.fy`"+` with:

`+"```fairyc"+`
module main;

import std::io as io;

@effects(console)
fn i32 main() {
    let ref str message = "Fairy C says hi\n";
    return switch (io::write(message)) {
        case .ok(written) : {
            if (written == message.len) {
                0
            } else {
                1
            }
        }
        case .err(error) : {
            if (error.status != 0 && error.written <= message.len) {
                2
            } else {
                3
            }
        }
    };
}
`+"```"+`

Then refresh the lock and run the program directly:

`+"```text"+`
fairyc check
fairyc run --release --locked
`+"```"+`
`, spec.Name, spec.Name))
}

// Create creates exactly one new child directory below parentPath. Existing targets are never
// opened or modified. Any ordinary failure after directory creation attempts a contained rollback.
func Create(parentPath string, spec Spec) error {
	if err := Validate(spec); err != nil {
		return err
	}
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		return &Error{Reason: ReasonParentUnavailable, Cause: err}
	}
	defer parent.Close()

	if err := parent.Mkdir(spec.Name, 0o755); err != nil {
		return classifyCreateError(spec.Name, err, true)
	}
	project, err := parent.OpenRoot(spec.Name)
	if err != nil {
		return rollback(parent, spec.Name, classifyCreateError(spec.Name, err, false))
	}
	projectOpen := true
	closeProject := func() error {
		if !projectOpen {
			return nil
		}
		projectOpen = false
		return project.Close()
	}
	defer closeProject()

	for _, directory := range []string{"src", "tests"} {
		if err := project.Mkdir(directory, 0o755); err != nil {
			closeErr := closeProject()
			return rollback(parent, spec.Name, errors.Join(classifyCreateError(filepath.ToSlash(filepath.Join(spec.Name, directory)), err, false), closeErr))
		}
	}
	for _, file := range scaffoldFiles {
		if err := writeExclusive(project, file.path, file.content(spec)); err != nil {
			closeErr := closeProject()
			return rollback(parent, spec.Name, errors.Join(classifyCreateError(filepath.ToSlash(filepath.Join(spec.Name, file.path)), err, false), closeErr))
		}
	}
	if err := closeProject(); err != nil {
		return rollback(parent, spec.Name, classifyCreateError(spec.Name, err, false))
	}
	return nil
}

func manifestBytes(spec Spec) []byte {
	return []byte(fmt.Sprintf(`schema = %q

[package]
uid = %q
name = %q
version = %q
language = %q
source = "src"
`, projectmanifest.SchemaVersion, spec.UID, spec.Name, DefaultVersion, DefaultLanguage))
}

func writeExclusive(root *os.Root, name string, content []byte) error {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := io.Copy(file, bytes.NewReader(content))
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

func classifyCreateError(path string, err error, target bool) error {
	reason := ReasonFilesystem
	switch {
	case target && errors.Is(err, fs.ErrExist):
		reason = ReasonTargetExists
	case errors.Is(err, fs.ErrPermission):
		reason = ReasonPermission
	}
	return &Error{Reason: reason, Path: path, Cause: err}
}

func rollback(parent *os.Root, name string, cause error) error {
	if err := parent.RemoveAll(name); err != nil {
		return &Error{Reason: ReasonRollback, Path: name, Cause: errors.Join(cause, err)}
	}
	return cause
}
