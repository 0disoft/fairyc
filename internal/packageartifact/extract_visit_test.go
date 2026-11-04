package packageartifact

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestVisitAllMatchesOwnedExtraction(t *testing.T) {
	program, identities, coordinates := extractionContextFixture(8, 2)
	typed := typedProjectionArtifact(t, 1)
	program.StructTypes = append(program.StructTypes, typed.Context.StructTypes[0])
	for id, coordinate := range typed.PackageCoordinates {
		coordinates[id] = coordinate
		identities[id] = typed.PackageIdentities[id]
	}
	want, err := ExtractAll(program, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	var got []Artifact
	err = VisitAllContext(context.Background(), program, identities, coordinates, func(artifact Artifact) error {
		got = append(got, artifact)
		return nil
	})
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("streamed package order, bytes or bindings changed: %v", err)
	}
	got[0].Context.StructTypes[0].Fields[0].Name = "changed"
	if program.StructTypes[0].Fields[0].Name == "changed" || got[1].Context.StructTypes[0].Fields[0].Name == "changed" {
		t.Fatal("delivered context aliases source or another package")
	}
	got[0].Functions[0].Body.Blocks[0].Terminator.ReturnValue.I32 = 99
	if program.Functions[0].Blocks[0].Terminator.ReturnValue.I32 != 42 {
		t.Fatal("delivered body aliases source")
	}
}

func TestVisitAllStopsBeforeExtractingTheNextPackage(t *testing.T) {
	for _, cancelRequest := range []bool{false, true} {
		program, identities, coordinates := extractionContextFixture(32, 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		failure := errors.New("publication failed")
		verifications, visits := 0, 0
		verify := func(program mir.Program) []mir.VerificationError {
			verifications++
			return mir.Verify(program)
		}
		err := visitAll(ctx, program, identities, coordinates, verify, func(Artifact) error {
			visits++
			if cancelRequest {
				cancel()
				return nil
			}
			return failure
		})
		if cancelRequest {
			failure = context.Canceled
		}
		if !errors.Is(err, failure) || visits != 1 || verifications != 2 {
			t.Fatalf("eager extraction after abort: visits=%d verifications=%d error=%v", visits, verifications, err)
		}
	}
}

func TestVisitAllDeliversBeforeLaterExtractionFailure(t *testing.T) {
	program, identities, coordinates := extractionContextFixture(3, 1)
	delete(identities, hir.PackageID(101))
	visits := 0
	err := VisitAllContext(context.Background(), program, identities, coordinates, func(Artifact) error { visits++; return nil })
	if err == nil || visits != 1 {
		t.Fatalf("not a demand-driven extraction: visits=%d error=%v", visits, err)
	}
}

func TestVisitAllPreconditions(t *testing.T) {
	program, identities, coordinates := extractionContextFixture(1, 1)
	if err := VisitAllContext(context.Background(), program, identities, coordinates, nil); err == nil {
		t.Fatal("nil visitor accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := VisitAllContext(ctx, program, identities, coordinates, func(Artifact) error { t.Fatal("canceled visitor called"); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
