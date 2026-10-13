//go:build ignore

// Go workspace/Flow/interfaces/specializations/body hashes supply key templates.
// Only the exact executing compiler image field varies during Fairy verification.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func write(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
func manifest(uid, extra string) string {
	return "schema='fairyc.manifest/v1'\n[package]\nuid='" + uid + "'\nname='complete-keys'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n" + extra
}
func dep(name, path, features string) string {
	return "[dependencies." + name + "]\npath='" + path + "'\nversion='1.0.0'\nfeatures=" + features + "\n"
}
func field(out *bytes.Buffer, value []byte) {
	binary.Write(out, binary.BigEndian, uint64(len(value)))
	out.Write(value)
}
func genericBytes(p packagecache.GenericInputs) ([]byte, error) {
	base, err := packagecache.Encode(p.Base)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	field(&out, []byte("fairyc.generic-package-cache-key/v0"))
	binary.Write(&out, binary.BigEndian, uint64(len(base)))
	out.Write(base)
	out.Write(p.CurrentInterface[:])
	binary.Write(&out, binary.BigEndian, uint64(len(p.Dependencies)))
	for _, d := range p.Dependencies {
		out.Write(d.Coordinate[:])
		out.Write(d.Interface[:])
	}
	binary.Write(&out, binary.BigEndian, uint64(len(p.Specializations)))
	for _, id := range p.Specializations {
		field(&out, []byte(id))
	}
	key, err := packagecache.ComputeGeneric(p)
	if err != nil {
		return nil, err
	}
	if sha256.Sum256(out.Bytes()) != [32]byte(key) {
		return nil, fmt.Errorf("generic preimage disagrees with production ComputeGeneric")
	}
	return out.Bytes(), nil
}
func run(root string) error {
	spec, ok := target.Lookup(target.X8664WindowsMSVC)
	if !ok {
		return fmt.Errorf("missing target")
	}
	sdk, err := sdkstdlib.Load()
	if err != nil {
		return err
	}
	standard, err := packagegraph.ComputeInstanceID(packagegraph.Identity{PackageUID: sdkstdlib.PackageUID, SourceIdentity: sdk.SourceIdentity, VersionIdentity: sdkstdlib.Version, Features: []string{}})
	if err != nil {
		return err
	}
	build := buildidentity.Encoding + ":" + strings.Repeat("1", 64)
	if _, err := buildidentity.Parse(build); err != nil {
		return err
	}
	names := []string{"Aliases", "PrivateEdit", "Reordered", "Vec", "State", "Plain"}
	var observations []map[string]any
	for _, name := range names {
		a, b := dep("a", "left", "['fast']"), dep("r", "right", "[]")
		if name == "Reordered" {
			a, b = b, a
		}
		main := "module main;import left as a;import left as b;import right as r;fn i32 constant<T>(){return 42;}fn i32 main(){discard constant<u16>();discard constant<u64>();return b::id<i32>(42);}\n"
		if name == "Vec" {
			main = "module main;import left as a;import left as b;import right as r;import std::vec as v;fn i32 constant<T>(){return 42;}fn i32 main(){discard constant<u16>();discard constant<u64>();var v::Vec<i32> items=v::empty<i32>();discard v::push<i32>(ref mut items,a::id<i32>(1));return b::id<i32>(42);}\n"
		}
		if name == "State" {
			main = "module main;import left as a;import right as r;@contract_uid(\"example.keys\")state{cell u64 total;map bytes20 u64 balances;}fn i32 main(){return a::id<i32>(42);}\n"
		}
		if name == "Plain" {
			main = "module main;import left as a;import right as r;fn i32 main(){return 42;}\n"
		}
		body := "return x;"
		if name == "PrivateEdit" {
			body = "let T result=x;return result;"
		}
		inputs := map[string]string{"fairy.toml": manifest("example.keys.root", "[workspace]\nmembers=['left','right']\n"+a+b+dep("alias-left", "left", "['fast']")), "src/main.fy": main, "left/fairy.toml": manifest("example.keys.left", ""), "left/src/left.fy": "module left;pub fn T id<T>(T x){return copy<T>(x);}fn T copy<T>(T x){" + body + "}\n", "right/fairy.toml": manifest("example.keys.right", ""), "right/src/right.fy": "module right;pub fn i32 value(){return 7;}\n"}
		dir := filepath.Join(root, name)
		for path, data := range inputs {
			if err := write(filepath.Join(dir, filepath.FromSlash(path)), []byte(data)); err != nil {
				return err
			}
		}
		loaded, err := workspaceloader.Load(dir)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		metadata, err := loaded.SemanticPackages()
		if err != nil {
			return err
		}
		input, err := loaded.SemanticInput()
		if err != nil {
			return err
		}
		coords, err := loaded.PackageCoordinates()
		if err != nil {
			return err
		}
		var owners []hir.PackageID
		sdkOwners, err := loaded.SDKStorageOwners()
		if err != nil {
			return err
		}
		byID := map[hir.PackageID]workspaceloader.SemanticPackage{}
		for _, p := range metadata {
			owners = append(owners, p.ID)
			byID[p.ID] = p
		}
		var analyzed compiler.GenericAnalysis
		if len(sdkOwners) > 0 {
			analyzed, err = compiler.AnalyzeSDKStoragePackages(input, coords, genericplan.DefaultPlanLimits(), sdkOwners)
		} else {
			analyzed, err = compiler.AnalyzeGenericPackages(input, coords, genericplan.DefaultPlanLimits())
		}
		if err != nil {
			return err
		}
		checked := analyzed.Analysis.Semantic
		if checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone || len(checked.Diagnostics) > 0 || len(checked.Unsupported) > 0 {
			return fmt.Errorf("%s incomplete: %v %v", name, checked.Diagnostics, checked.Unsupported)
		}
		bodies, err := contractstatemir.ComputeHIRStateBodyHashes(checked.HIR, owners, coords)
		if err != nil {
			return err
		}
		var out bytes.Buffer
		out.WriteString("FYCKEYS1")
		binary.Write(&out, binary.BigEndian, uint64(len(metadata)))
		var rows []map[string]any
		for _, p := range metadata {
			ordinary := packagecache.Inputs{CompilerBuild: build, LanguageRevision: sdkstdlib.LanguageRevision, MIRVersion: "fairyc.candidate-owned-body-ir/0", TargetSchema: spec.SchemaVersion, ABIProfile: strings.Join([]string{"fairyc.target-abi-profile/0", string(spec.ID), spec.DataModel, spec.CallingConvention, spec.AggregateLayout, spec.ReturnClassifier, spec.ObjectFormat, spec.UnwindModel}, ";"), ToolchainFingerprint: "not-applicable:verified-candidate-owned-body-ir", Backend: "verified-candidate-owned-body-ir", RuntimeABI: sdkstdlib.RuntimeABI, StandardLibrary: standard, Package: p.Instance, Source: p.Source, CurrentInterface: analyzed.OrdinaryInterfaces[p.ID], CurrentBody: bodies[p.ID], Features: slices.Clone(p.Features)}
			for _, id := range p.Dependencies {
				ordinary.Dependencies = append(ordinary.Dependencies, packagecache.Dependency{Coordinate: byID[id].Coordinate, Interface: analyzed.OrdinaryInterfaces[id]})
			}
			ordinary.Dependencies, err = packagecache.CanonicalizeDependencies(ordinary.Dependencies)
			if err != nil {
				return err
			}
			g := packagecache.GenericInputs{Base: ordinary, CurrentInterface: analyzed.Interfaces[p.ID], Specializations: slices.Clone(analyzed.Specializations[p.ID])}
			for _, d := range ordinary.Dependencies {
				for _, id := range p.Dependencies {
					if byID[id].Coordinate == d.Coordinate {
						g.Dependencies = append(g.Dependencies, packagecache.GenericDependency{Coordinate: d.Coordinate, Interface: analyzed.Interfaces[id]})
					}
				}
			}
			baseBytes, err := packagecache.Encode(ordinary)
			if err != nil {
				return err
			}
			baseKey, err := packagecache.Compute(ordinary)
			if err != nil {
				return err
			}
			if sha256.Sum256(baseBytes) != [32]byte(baseKey) {
				return fmt.Errorf("base preimage disagrees")
			}
			sourceBytes, err := genericBytes(g)
			if err != nil {
				return err
			}
			field(&out, []byte(p.Instance.String()))
			field(&out, []byte(p.Coordinate.String()))
			field(&out, baseBytes)
			field(&out, sourceBytes)
			rows = append(rows, map[string]any{"instance": p.Instance.String(), "coordinate": p.Coordinate.String(), "ordinaryBytes": len(baseBytes), "genericBytes": len(sourceBytes), "ordinary": baseKey.String(), "ordinaryInterface": analyzed.OrdinaryInterfaces[p.ID].String(), "sourceInterface": analyzed.Interfaces[p.ID].String(), "body": bodies[p.ID].String(), "specializations": g.Specializations, "features": p.Features, "dependencies": ordinary.Dependencies})
		}
		if out.Len() > 65536 {
			return fmt.Errorf("fixture exceeds manifest bound")
		}
		if err := write(filepath.Join(dir, "keys.bin"), out.Bytes()); err != nil {
			return err
		}
		observations = append(observations, map[string]any{"case": name, "packages": rows})
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "fixture": root, "compilerTemplate": build, "cases": observations, "standardLibrary": standard.String(), "genericEnvelopeVerifiedAgainstComputeGeneric": true})
}
func main() {
	root := flag.String("output", ".cache/complete-keys-20261014/fixture", "fixture directory")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
