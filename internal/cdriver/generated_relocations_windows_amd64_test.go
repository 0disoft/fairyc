//go:build windows && amd64

package cdriver_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestGeneratedRelocationCoverage(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MSVC %s", tc.Version)
	for _, sample := range []struct {
		name   string
		exit   int32
		stdout string
		args   []string
	}{{"std_bytes_quickstart", 0, "", nil}, {"std_alloc_quickstart", 42, "", nil}, {"std_text_from_buffer_quickstart", 42, "", nil}, {"std_vec_quickstart", 0, "42", nil}, {"process_args_quickstart", 42, "", []string{"", "Fairy", "--locked", `a b"c\d\`, "한글λ"}}} {
		t.Run(sample.name, func(t *testing.T) {
			workspace, err := workspaceloader.Load(filepath.Join("..", "..", "examples", sample.name))
			if err != nil {
				t.Fatal(err)
			}
			checked, err := workspacecompiler.Check(workspace)
			if err != nil || !checked.Verified {
				t.Fatalf("source check: %v verified=%v", err, checked.Verified)
			}
			id, err := workspace.RootPackageID()
			if err != nil {
				t.Fatal(err)
			}
			unit, err := cbackend.EmitStorageTransfers(checked.MIR, hir.QualifiedModulePath(id, "main")+"::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			support, err := nativesupport.Source(target.X8664WindowsMSVC, unit.ForeignSymbols, nativesupport.Options{SDKStorage: true})
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					a, err := cdriver.BuildWindowsHostWithLibraryLog(tc, profile, unit.Source, support)
					if err != nil {
						t.Fatal(err)
					}
					run, err := cdriver.ExecuteWindowsMSVCTargetWithInputAndArgs(a, strings.NewReader(""), sample.args)
					if err != nil || run.ExitCode != sample.exit || run.Stdout != sample.stdout || run.Stderr != "" {
						t.Fatalf("execution: %+v %v", run, err)
					}
					if len(a.Objects) != 2 {
						t.Fatal("missing production objects")
					}
					recordGeneratedDefaultLibraries(t, a)
					for _, obj := range a.Objects {
						inventory, err := pecoff.InspectWeakAliases(obj.Bytes)
						if err != nil {
							t.Fatal(err)
						}
						maxDirectives := 0
						if sample.name == "process_args_quickstart" && obj.Name == "runtime.obj" {
							maxDirectives = 1
						}
						// Older SDK headers may omit this alternative entirely.
						if len(inventory.WeakExternals) != 0 || len(inventory.Directives) > maxDirectives {
							t.Fatalf("new project weak use needs review: %s %+v", obj.Name, inventory)
						}
						for _, alias := range inventory.Directives {
							if alias.Name != "_Avx2WmemEnabled" || alias.Fallback != "_Avx2WmemEnabledWeakValue" {
								t.Fatalf("unmeasured runtime alias %+v", alias)
							}
							r, err := pecoff.InspectObject(obj.Bytes)
							if err != nil {
								t.Fatal(err)
							}
							refs := 0
							for _, rel := range r.RelocationRecords {
								if rel.Symbol == alias.Name {
									refs++
								}
							}
							if refs != 0 {
								t.Fatalf("new live SDK alias references need review: %s count %d", alias.Name, refs)
							}
							var linked []string
							for _, line := range strings.Split(string(a.LinkMap), "\n") {
								fields := strings.Fields(line)
								if len(fields) >= 4 && (fields[1] == alias.Name || fields[1] == alias.Fallback) {
									linked = append(linked, strings.TrimSpace(line))
								}
							}
							t.Logf("SDK alias %s: object references=%d map=%v", alias.Name, refs, linked)
						}
						t.Logf("%s project weak externals=0 alternate directives=%v", obj.Name, inventory.Directives)
					}
					coverage, err := pecoff.InspectGeneratedRelocations(a.Objects[0].Bytes, a.Objects[1].Bytes, a.Bytes, a.LinkMap, a.LinkerVerbose)
					if err != nil {
						t.Fatal(err)
					}
					if coverage.Linked == 0 || coverage.Observed != coverage.Linked+coverage.Discarded {
						t.Fatal("incomplete relocation accounting")
					}
					t.Logf("sources unit=%x runtime=%x; objects unit=%x runtime=%x; image=%x", sha256.Sum256([]byte(unit.Source)), sha256.Sum256([]byte(support)), sha256.Sum256(a.Objects[0].Bytes), sha256.Sum256(a.Objects[1].Bytes), sha256.Sum256(a.Bytes))
					for _, family := range coverage.Families {
						t.Logf("%s type %#04x: observed %d linked %d discarded %d", family.Object, family.Type, family.Observed, family.Linked, family.Discarded)
					}
					checkGeneratedRelocationMutations(t, a, coverage)

				})
			}
		})
	}
}

func checkGeneratedRelocationMutations(t *testing.T, a cdriver.WindowsHostArtifact, coverage pecoff.GeneratedRelocationCoverage) {
	t.Helper()
	inspect := func(unit, runtime, image, linkMap, log []byte) error {
		_, err := pecoff.InspectGeneratedRelocations(unit, runtime, image, linkMap, log)
		return err
	}
	first, second := a.Objects[0].Bytes, a.Objects[1].Bytes
	if inspect(first, second, a.Bytes, nil, a.LinkerVerbose) == nil {
		t.Fatal("accepted missing map")
	}
	wrongOwner := bytes.ReplaceAll(a.LinkMap, []byte("unit.obj"), []byte("other.obj"))
	if inspect(first, second, a.Bytes, wrongOwner, a.LinkerVerbose) == nil {
		t.Fatal("accepted wrong source ownership")
	}
	if coverage.Discarded > 0 && inspect(first, second, a.Bytes, a.LinkMap, []byte("no discard evidence")) == nil {
		t.Fatal("accepted absent discard evidence")
	}
	pe, err := pecoff.InspectExecutable(a.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, value := range coverage.Values {
		key := fmt.Sprintf("%s:%d", value.Object, value.Type)
		if seen[key] {
			continue
		}
		seen[key] = true
		index := 0
		if value.Object == "runtime.obj" {
			index = 1
		}
		object, err := pecoff.InspectObject(a.Objects[index].Bytes)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, rel := range object.RelocationRecords {
			if rel.Type != value.Type || rel.VirtualAddress != value.ObjectOffset || rel.SectionIndex != value.SectionIndex || rel.Symbol != value.Target {
				continue
			}
			mutated := bytes.Clone(a.Objects[index].Bytes)
			at := uint64(object.Sections[rel.SectionIndex-1].RawOffset) + uint64(rel.VirtualAddress)
			mutated[at] ^= 1
			unit, runtime := first, second
			if index == 0 {
				unit = mutated
			} else {
				runtime = mutated
			}
			if inspect(unit, runtime, a.Bytes, a.LinkMap, a.LinkerVerbose) == nil {
				t.Fatalf("accepted %s object addend mutation", key)
			}
			found = true
			break
		}
		if !found {
			t.Fatalf("missing source field for %s", key)
		}
		mutated := bytes.Clone(a.Bytes)
		found = false
		for _, section := range pe.Sections {
			if value.SiteRVA >= section.VirtualAddress && uint64(value.SiteRVA-section.VirtualAddress) < uint64(section.RawSize) {
				at := uint64(section.RawOffset) + uint64(value.SiteRVA-section.VirtualAddress)
				mutated[at] ^= 1
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing linked site for %s", key)
		}
		if inspect(first, second, mutated, a.LinkMap, a.LinkerVerbose) == nil {
			t.Fatalf("accepted %s linked field mutation", key)
		}
	}
}
