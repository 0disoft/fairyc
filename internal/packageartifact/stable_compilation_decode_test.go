package packageartifact

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func stableCompilationDecodeFixture(tb testing.TB, functions int) StableCompilationArtifact {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main;\n")
	for index := range functions {
		fmt.Fprintf(&text, "fn i32 f%d() { return %d; }\n", index, index)
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text.String())), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		tb.Fatalf("syntax: %v", err)
	}
	checked, err := compiler.Check([]semantic.File{{Package: 1, Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		tb.Fatalf("compiler: %v, verified=%v", err, checked.Verified)
	}
	identity := packagegraph.Identity{PackageUID: "decode.fixture", SourceIdentity: "source", VersionIdentity: "1"}
	instance, err := packagegraph.ComputeInstanceID(identity)
	if err != nil {
		tb.Fatal(err)
	}
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.CoordinateOf(identity))
	if err != nil {
		tb.Fatal(err)
	}
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: coordinate}
	artifacts, err := ExtractAll(checked.MIR, map[hir.PackageID]packagegraph.InstanceID{1: instance}, coordinates)
	if err != nil || len(artifacts) != 1 {
		tb.Fatalf("extract: %v, artifacts=%d", err, len(artifacts))
	}
	bodies, err := contractstatemir.DescribeStableHIRStateBodies(hir.Package{}, []hir.PackageID{1}, coordinates)
	if err != nil {
		tb.Fatal(err)
	}
	artifact, err := DescribeStableCompilationArtifact(artifacts[0], bodies[1], stablePackageArtifactLimit)
	if err != nil {
		tb.Fatal(err)
	}
	return artifact
}

func TestStableCompilationDecodeStillValidatesNestedPayloads(t *testing.T) {
	base := stableCompilationDecodeFixture(t, 2)
	for _, mode := range []string{"MIR owner", "MIR functions", "state body hash", "state body owner"} {
		t.Run(mode, func(t *testing.T) {
			artifact := base
			switch mode {
			case "MIR owner":
				artifact.MIR.Coordinate = packagegraph.CoordinateID{}
			case "MIR functions":
				artifact.MIR.Functions = nil
			case "state body hash":
				artifact.StateBody.Hash[0] ^= 0xff
			case "state body owner":
				artifact.StateBody.Package = packagegraph.CoordinateID{}
			}
			// Marshal directly: the public encoder must refuse these invalid payloads.
			encoded, err := json.Marshal(stableCompilationEnvelope{Schema: StableCompilationSchema, Version: StableCompilationVersion, Compilation: artifact})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := EncodeStableCompilationArtifact(artifact, stablePackageArtifactLimit); err == nil {
				t.Fatal("encoder accepted an invalid nested payload")
			}
			if _, err := DecodeStableCompilationArtifact(encoded, stablePackageArtifactLimit); err == nil || !strings.Contains(err.Error(), "verify decoded") {
				t.Fatalf("decoder validation error = %v", err)
			}
		})
	}
}

func BenchmarkDecodeStableCompilation(b *testing.B) {
	for _, count := range []int{1, 64, 256} {
		b.Run(fmt.Sprintf("functions_%d", count), func(b *testing.B) {
			artifact := stableCompilationDecodeFixture(b, count)
			encoded, err := EncodeStableCompilationArtifact(artifact, stablePackageArtifactLimit)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				decoded, err := DecodeStableCompilationArtifact(encoded, stablePackageArtifactLimit)
				if err != nil || len(decoded.MIR.Functions) != count {
					b.Fatalf("decode: %v, functions=%d", err, len(decoded.MIR.Functions))
				}
			}
		})
	}
}
