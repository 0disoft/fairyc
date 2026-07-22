package cli

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/lockfile"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func runWorkspaceCheck(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	locked, err := parseOptionalLocked("check-workspace", arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	workspace, lockState, err := loadWorkspaceWithLock(sourceRoot, locked)
	if err != nil {
		return packageFailure(stderr, err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		return packageFailure(stderr, err)
	}
	result, err := compiler.CheckPackages(input)
	if err != nil {
		return internalError(stderr)
	}
	if len(result.Semantic.Diagnostics) != 0 {
		if err := writeWorkspaceDiagnostics(stderr, workspace, result.Semantic.Diagnostics); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if len(result.Semantic.Unsupported) != 0 {
		if err := writeUnsupportedSemantics(stderr, result.Semantic.Unsupported); err != nil {
			return ExitInternalFailure
		}
		return ExitSourceDiagnostics
	}
	if !result.Verified {
		return internalError(stderr)
	}
	if _, err := fmt.Fprintf(stdout, "workspace check ok: %d instance(s), %d source file(s), %d MIR function(s), lock %s\n", len(workspace.Graph.Nodes), len(input.Files), len(result.MIR.Functions), lockState); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseOptionalLocked(command string, arguments []string) (bool, error) {
	switch {
	case len(arguments) == 0:
		return false, nil
	case len(arguments) == 1 && arguments[0] == "--locked":
		return true, nil
	default:
		return false, fmt.Errorf("experimental command %q accepts only optional --locked", command)
	}
}

func loadWorkspaceWithLock(sourceRoot string, locked bool) (workspaceloader.Workspace, string, error) {
	workspace, err := workspaceloader.Load(sourceRoot)
	if err != nil {
		return workspaceloader.Workspace{}, "", err
	}
	document, err := lockfile.FromWorkspace(workspace)
	if err != nil {
		return workspaceloader.Workspace{}, "", err
	}
	canonical, err := lockfile.Marshal(document)
	if err != nil {
		return workspaceloader.Workspace{}, "", err
	}
	lockPath := filepath.Join(sourceRoot, lockfile.FileName)
	lockState := "unchanged"
	if locked {
		existing, _, err := lockfile.Read(lockPath)
		if err != nil {
			return workspaceloader.Workspace{}, "", fmt.Errorf("locked workspace requires a valid %s: %w", lockfile.FileName, err)
		}
		if err := lockfile.MatchesWorkspace(existing, workspace); err != nil {
			return workspaceloader.Workspace{}, "", err
		}
		lockState = "verified"
	} else {
		_, existingBytes, readErr := lockfile.Read(lockPath)
		if readErr != nil || !bytes.Equal(existingBytes, canonical) {
			if err := lockfile.WriteAtomic(lockPath, canonical); err != nil {
				return workspaceloader.Workspace{}, "", err
			}
			lockState = "updated"
		}
	}
	return workspace, lockState, nil
}

func writeWorkspaceDiagnostics(writer io.Writer, workspace workspaceloader.Workspace, diagnostics []semantic.Diagnostic) error {
	for _, value := range diagnostics {
		display, contents, found := workspace.DiagnosticSource(value.Primary.Package, value.Primary.Source)
		if !found {
			return fmt.Errorf("resolve workspace diagnostic source")
		}
		line, column := bytePosition(contents, value.Primary.StartByte)
		if _, err := fmt.Fprintf(writer, "%s:%d:%d: %s: %s [bytes %d..%d]\n", safeDisplayPath(display), line, column, value.Code, semanticDiagnosticSummary(value.Code), value.Primary.StartByte, value.Primary.EndByte); err != nil {
			return err
		}
		for _, related := range value.Related {
			relatedDisplay, relatedContents, found := workspace.DiagnosticSource(related.Span.Package, related.Span.Source)
			if !found {
				return fmt.Errorf("resolve related workspace diagnostic source")
			}
			relatedLine, relatedColumn := bytePosition(relatedContents, related.Span.StartByte)
			if _, err := fmt.Fprintf(writer, "  %s: %s:%d:%d [bytes %d..%d]\n", related.Role, safeDisplayPath(relatedDisplay), relatedLine, relatedColumn, related.Span.StartByte, related.Span.EndByte); err != nil {
				return err
			}
		}
	}
	return nil
}

func packageFailure(stderr io.Writer, err error) int {
	detail := "package workspace failure"
	if err != nil {
		detail += ": " + err.Error()
	}
	if _, writeErr := fmt.Fprintf(stderr, "fairyc: %s\n", detail); writeErr != nil {
		return ExitInternalFailure
	}
	return ExitPackageFailure
}
