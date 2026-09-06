package compiler

import (
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestGenericDiagnosticDoesNotInventSourceOwnership(t *testing.T) {
	for _, mutation := range []string{"owner", "span", "kind", "duplicate", "unlocated", "unknown-expansion"} {
		t.Run(mutation, func(t *testing.T) {
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{7: {1}}
			input := semantic.PackageInput{Files: []semantic.File{{Package: 7, Path: "main.fy", Module: &ast.Module{Path: "main", SourceSize: 9}}}}
			source := &genericsource.SourceError{Owner: genericPackageOwner(coordinates[7]), Module: "main", Span: diagnostic.Span{StartByte: 1, EndByte: 2}, Kind: genericsource.FailureRequest, Cause: errors.New("test source rejection")}
			var err error = source
			switch mutation {
			case "owner":
				source.Owner = "wrong"
			case "span":
				source.Span.EndByte = 10
			case "kind":
				source.Kind = "unknown"
			case "duplicate":
				input.Files = append(input.Files, input.Files[0])
			case "unlocated":
				err = errors.New("internal failure")
			case "unknown-expansion":
				source.Kind = genericsource.FailureExpansion
			}
			if d, ok := genericSourceDiagnostic(err, input, coordinates); ok || d.Code != "" {
				t.Fatalf("manufactured source diagnostic: %+v", d)
			}
		})
	}
}
