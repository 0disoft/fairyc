package cli

import (
	"encoding/json"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

func runAtlasEdit(arguments []string, root string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 || arguments[0] != "replace-body" {
		return invocationError(stderr, "atlas-edit requires replace-body")
	}
	values := map[string]string{}
	apply := false
	required := []string{"--package-uid", "--symbol", "--body-file", "--expected-source-sha256"}
	flags := []flagDefinition{{Name: "--apply", Kind: flagSwitch}}
	for _, flag := range required {
		flags = append(flags, flagDefinition{Name: flag, Kind: flagScalar})
	}
	_, err := parseCommandFlags(arguments[1:], flagSchema{Command: "atlas-edit replace-body", Experimental: true, Flags: flags, Required: required}, func(flag, value string) error {
		if flag == "--apply" {
			apply = true
		} else {
			values[flag] = value
		}
		return nil
	})
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	prepared, code := prepareAtlasWorkspaceSchema(root, true, stderr, atlasmap.LogicalSchemaV3)
	if code != ExitSuccess {
		return code
	}
	pkg, ok := prepared.workspace.Package(".")
	if !ok || pkg.Manifest.Package.UID != values["--package-uid"] {
		return invocationError(stderr, "only the exact root package is editable")
	}
	p, found := selectAtlasPartition(prepared.partitions, values["--package-uid"])
	if !found {
		return invocationError(stderr, "edit package not found")
	}
	if _, err := atlasmap.NewQueryIndexV3(p, prepared.expectations[p.Header().PackageInstance]); err != nil {
		return internalError(stderr)
	}
	var key semanticsnapshot.DeclarationKey
	matches := 0
	for _, symbol := range p.Symbols() {
		if symbol.Visible == values["--symbol"] || symbol.Full == values["--symbol"] {
			key = symbol.Declaration
			matches++
		}
	}
	if matches != 1 || key.Kind != semanticsnapshot.DeclarationFunction {
		return invocationError(stderr, "edit subject must be one exact ordinary function")
	}
	opened, err := os.OpenRoot(prepared.workspace.Root)
	if err != nil {
		return packageFailure(stderr, err)
	}
	body, err := sourceedit.ReadBodyFile(prepared.workspace, opened, values["--body-file"])
	opened.Close()
	if err != nil {
		return packageFailure(stderr, err)
	}
	plan, err := sourceedit.PlanBody(prepared.workspace, key.Module, key.Name, values["--expected-source-sha256"], body)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	preview, err := json.Marshal(plan.Report)
	if err != nil || len(preview) > 8192 {
		return invocationError(stderr, "edit report exceeds output byte limit")
	}
	if apply {
		if err := plan.Apply(); err != nil {
			return packageFailure(stderr, err)
		}
	}
	encoded, err := json.Marshal(plan.Report)
	if err != nil {
		return internalError(stderr)
	}
	if len(encoded) > 8192 {
		return internalError(stderr)
	}
	if _, err := stdout.Write(encoded); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
