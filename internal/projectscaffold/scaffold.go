// Package projectscaffold owns the deterministic, no-overwrite Fairy C starter project.
package projectscaffold

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/0disoft/fairyc/internal/projectmanifest"
)

const (
	DefaultVersion  = "0.1.0"
	DefaultLanguage = "fairyc-language/20"
)

// FailureReason is a stable internal category for one rejected or incomplete scaffold attempt.
type FailureReason string

const (
	ReasonInvalidName       FailureReason = "invalid-name"
	ReasonInvalidUID        FailureReason = "invalid-uid"
	ReasonInvalidTemplate   FailureReason = "invalid-template"
	ReasonWin32Identity     FailureReason = "win32-identity-too-long"
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
	case ReasonInvalidTemplate:
		return "template must be console or win32"
	case ReasonWin32Identity:
		return "win32 template requires a name of at most 127 ASCII bytes and UID of at most 255 ASCII bytes"
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
	Name     string
	UID      string
	Template string
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
	if spec.Template != "" && spec.Template != "console" && spec.Template != "win32" {
		return &Error{Reason: ReasonInvalidTemplate}
	}
	if spec.Template == "win32" && (len(spec.Name) > 127 || len(spec.UID) > 255) {
		return &Error{Reason: ReasonWin32Identity}
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
    return 77;
}

fn i32 main() {
    return answer() - 77;
}
`

const testSource = `module tests::main;

import main as app;

@test
fn void answer_is_77() {
    assert(app::answer() == 77);
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

The default test command uses the fast semantic interpreter.

## Run on Windows

With x64 MSVC and a Windows SDK installed:

`+"```text"+`
fairyc test --native --locked
fairyc run --locked
fairyc build --release --locked
.\build\%s.exe
`+"```"+`

`+"`fairyc test --native`"+` compiles every test into a separate temporary Windows image.
`+"`fairyc run`"+` forwards program output and returns `+"`main`"+`'s `+"`i32`"+` as the process status.
The generated starter returns zero after checking that `+"`answer()`"+` is 77.

## Run on Linux

With Linux GNU amd64, glibc and GCC or Clang on PATH:

`+"```sh"+`
fairyc test --native --locked
fairyc run --locked
fairyc build --release --locked
`+"```"+`

Run the executable under `+"`build/`"+` with this package's name and no extension.
Linux process status preserves the low eight bits of `+"`main`"+`'s return.
Use default semantic tests for fast checks or `+"`test --native`"+` for isolated ELF execution.

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
	return CreateWithOptions(parentPath, spec)
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

func rollback(parent *os.Root, name string, owned os.FileInfo, cause error) error {
	current, err := parent.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return cause
	}
	if err != nil {
		return &Error{Reason: ReasonRollback, Path: name, Cause: errors.Join(cause, err)}
	}
	// Never recursively delete a path already observed to belong to someone
	// else. This recheck is not atomic with RemoveAll against hostile writers.
	if owned == nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(owned, current) {
		return &Error{Reason: ReasonRollback, Path: name, Cause: errors.Join(cause, fmt.Errorf("project rollback target identity changed"))}
	}
	if err := parent.RemoveAll(name); err != nil {
		return &Error{Reason: ReasonRollback, Path: name, Cause: errors.Join(cause, err)}
	}
	return cause
}
