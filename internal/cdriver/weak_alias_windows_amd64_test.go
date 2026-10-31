//go:build windows && amd64

package cdriver

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func compileWeakAliasC(t *testing.T, tc WindowsHostToolchain, profile Profile, name, text string) WindowsHostObject {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, name+".c")
	obj := filepath.Join(dir, name+".obj")
	if err := os.WriteFile(src, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	opt := "/Od"
	if profile == OptimizedProfile {
		opt = "/O2"
	}
	if _, err := runWindowsHostCommand(tc.Path, tc.Environment, dir, "/nologo", "/std:c11", "/TC", "/W4", "/WX", "/MT", opt, "/c", src, "/Fo:"+obj); err != nil {
		t.Fatal(err)
	}
	raw, err := readArtifact(obj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pecoff.InspectObject(raw); err != nil {
		t.Fatal(err)
	}
	return WindowsHostObject{Name: name + ".obj", Bytes: raw}
}

// wchar.h supplies this real CRT data alternative without a handwritten pragma.
func TestWindowsActualCRTWeakDataAlternative(t *testing.T) {
	tc, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	probe := compileWeakAliasC(t, tc, LowProfile, "header_inventory", "#include <wchar.h>\nint main(void){return 42;}\n")
	inventory, err := pecoff.InspectWeakAliases(probe.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, alias := range inventory.Directives {
		if alias.Name == "_Avx2WmemEnabled" && alias.Fallback == "_Avx2WmemEnabledWeakValue" {
			found = true
		}
	}
	if !found {
		t.Skip("selected SDK wchar.h does not offer the measured CRT data alternative")
	}
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, override := range []bool{false, true} {
				t.Run(fmt.Sprintf("override-%t", override), func(t *testing.T) {
					want := uint32(0)
					provider := "int weak_unused_provider(void){return 19;}\n"
					selected, owner := "_Avx2WmemEnabledWeakValue", "caller.obj"
					if override {
						want = 7
						provider = "int _Avx2WmemEnabled=7;\nint weak_unused_provider(void){return 19;}\n"
						selected, owner = "_Avx2WmemEnabled", "provider.obj"
					}
					caller := fmt.Sprintf("#include <wchar.h>\nint main(void){return _Avx2WmemEnabled==%d?42:1;}\n", want)
					objects := []WindowsHostObject{compileWeakAliasC(t, tc, profile, "caller", caller), compileWeakAliasC(t, tc, profile, "provider", provider)}
					inputs := []pecoff.WeakAliasObject{{Name: objects[0].Name, Bytes: objects[0].Bytes}, {Name: objects[1].Name, Bytes: objects[1].Bytes}}
					a, err := buildWindowsMSVCWithOptions(tc, profile, C11ConsumerMode, nil, WindowsMSVCBuildOptions{CaptureMap: true, PrebuiltObjects: objects})
					if err != nil {
						t.Fatal(err)
					}
					proof, err := pecoff.InspectDirectWeakDataAliasLink(inputs, "caller.obj", "main", "_Avx2WmemEnabled", a.Bytes, a.LinkMap)
					if err != nil {
						t.Fatal(err)
					}
					if proof.Plan.Selected != selected || proof.Plan.Owner != owner || proof.TargetDataValue != want {
						t.Fatalf("CRT data alternative %+v", proof)
					}
					run, err := ExecuteWindowsMSVCTarget(a)
					if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
						t.Fatalf("CRT data execution %+v %v", run, err)
					}
					t.Logf("actual header alias -> %s from %s value %d; relocation %#x displacement %d", proof.Plan.Selected, proof.Plan.Owner, proof.TargetDataValue, proof.Kind, proof.Actual)
				})
			}
		})
	}
}

func TestWindowsActualCRTWeakAlternatives(t *testing.T) {
	tc, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	libTool := filepath.Join(filepath.Dir(tc.Path), "lib.exe")
	archive := filepath.Clean(filepath.Join(filepath.Dir(tc.Path), "..", "..", "..", "lib", "x64", "libcmt.lib"))
	listing, err := runWindowsHostCommandWithLimit(libTool, tc.Environment, t.TempDir(), 1024*1024, "/nologo", "/list", archive)
	if err != nil {
		t.Fatal(err)
	}
	member := ""
	for _, line := range strings.Split(listing, "\n") {
		if strings.EqualFold(filepath.Base(strings.TrimSpace(line)), "initializers.obj") {
			if member != "" {
				t.Fatal("ambiguous CRT member")
			}
			member = strings.TrimSpace(line)
		}
	}
	if member == "" {
		t.Fatal("CRT initializers member missing")
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "initializers.obj")
	if _, err := runWindowsHostCommand(libTool, tc.Environment, dir, "/nologo", archive, "/extract:"+member, "/out:"+output); err != nil {
		t.Fatal(err)
	}
	raw, err := readArtifact(output)
	if err != nil {
		t.Fatal(err)
	}
	directives, err := pecoff.InspectCOFFAlternateDirectives(raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("libcmt.lib(initializers.obj) metadata sha256=%x alternate directives=%d", sha256.Sum256(raw), len(directives))
	for _, root := range []string{"__acrt_initialize", "__vcrt_initialize"} {
		found := false
		for _, alias := range directives {
			if alias.Name == root && alias.Fallback == "__scrt_stub_for_acrt_initialize" {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing actual CRT alternative %s", root)
		}
	}
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			a, err := BuildWindowsMSVCProbeWithLibraryLog(tc, profile, "int main(void){return 42;}\n")
			if err != nil {
				t.Fatal(err)
			}
			loaded := false
			for _, line := range strings.Split(string(a.LinkerVerbose), "\n") {
				lower := strings.ToLower(strings.TrimSpace(line))
				if strings.HasPrefix(lower, "loaded ") && strings.Contains(lower, "libcmt.lib(") && strings.HasSuffix(lower, "initializers.obj)") {
					loaded = true
				}
			}
			if !loaded {
				t.Fatal("actual CRT member absent from link evidence")
			}
			for _, root := range []string{"__acrt_initialize", "__vcrt_initialize"} {
				matches := 0
				for _, line := range strings.Split(string(a.LinkMap), "\n") {
					fields := strings.Fields(line)
					if len(fields) >= 4 && fields[1] == root {
						matches++
						owner := fields[len(fields)-1]
						want := "libucrt:initialization.obj"
						if root == "__vcrt_initialize" {
							want = "libvcruntime:initialization.obj"
						}
						if !strings.EqualFold(owner, want) {
							t.Fatalf("CRT alternative did not select its strong runtime owner: %s", line)
						}
						t.Logf("actual CRT %s strong owner %s", root, owner)
					}
				}
				if matches != 1 {
					t.Fatalf("ambiguous/missing CRT root %s", root)
				}
			}
			run, err := ExecuteWindowsMSVCTarget(a)
			if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
				t.Fatalf("CRT startup %+v %v", run, err)
			}
		})
	}
}

func TestWindowsDirectWeakAliasSelection(t *testing.T) {
	tc, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MSVC %s", tc.Version)
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			for _, form := range []string{"coff", "directive"} {
				t.Run(form, func(t *testing.T) {
					for _, scenario := range []struct {
						name, strong string
						cycle        bool
						wantA, wantB int
					}{
						{"fallback", "", false, 17, 17}, {"root-override", "fy_wa", false, 42, 17}, {"middle-override", "fy_wb", false, 42, 42},
						{"cycle-root-cut", "fy_wa", true, 42, 42}, {"cycle-middle-cut", "fy_wb", true, 42, 42}, {"unresolved-cycle", "", true, 0, 0},
					} {
						t.Run(scenario.name, func(t *testing.T) {
							provider := "int fy_fa(void){return 17;}\n"
							if scenario.strong != "" {
								provider += fmt.Sprintf("int %s(void){return 42;}\n", scenario.strong)
							}
							caller := fmt.Sprintf("int fy_ca(void);int fy_cb(void);int main(void){return fy_ca()==%d && fy_cb()==%d?42:1;}\n", scenario.wantA, scenario.wantB)
							var objects []WindowsHostObject
							rootObject, first, second := "weak.obj", "fy_ca", "fy_cb"
							if form == "coff" {
								raw, err := pecoff.BuildAMD64WeakChainProbeObject(scenario.cycle)
								if err != nil {
									t.Fatal(err)
								}
								objects = append(objects, WindowsHostObject{Name: "weak.obj", Bytes: raw})
								provider = "int weak_unused_provider(void){return 19;}\n"
								if scenario.strong != "" {
									provider += fmt.Sprintf("int %s(void){return 42;}\n", scenario.strong)
								}
							} else {
								terminal := "fy_fa"
								if scenario.cycle {
									terminal = "fy_wa"
								}
								caller = fmt.Sprintf("#pragma comment(linker,\"/alternatename:fy_wa=fy_wb\")\n#pragma comment(linker,\"/alternatename:fy_wb=%s\")\nint fy_wa(void);int fy_wb(void);int main(void){return fy_wa()==%d && fy_wb()==%d?42:1;}\n", terminal, scenario.wantA, scenario.wantB)
								rootObject = "caller.obj"
								first = "main"
								second = "main"
							}
							objects = append(objects, compileWeakAliasC(t, tc, profile, "caller", caller), compileWeakAliasC(t, tc, profile, "provider", provider))
							var inputs []pecoff.WeakAliasObject
							for _, obj := range objects {
								inputs = append(inputs, pecoff.WeakAliasObject{Name: obj.Name, Bytes: obj.Bytes})
							}
							plan, err := pecoff.PlanDirectWeakAlias(inputs, rootObject, "fy_wa")
							if scenario.name == "unresolved-cycle" {
								if err == nil || !strings.Contains(err.Error(), "cycle") {
									t.Fatalf("unresolved cycle: %+v %v", plan, err)
								}
								// Independently observe the actual linker's failure, bounded so
								// malformed cycles cannot stall this Windows test process.
								dir := t.TempDir()
								args := []string{"/NOLOGO", "/SUBSYSTEM:CONSOLE", "/OUT:" + filepath.Join(dir, "cycle.exe")}
								for _, obj := range objects {
									path := filepath.Join(dir, obj.Name)
									if err := os.WriteFile(path, obj.Bytes, 0600); err != nil {
										t.Fatal(err)
									}
									args = append(args, path)
								}
								output, linkErr := runWindowsHostCommandWithLimitAndTimeout(filepath.Join(filepath.Dir(tc.Path), "link.exe"), tc.Environment, dir, 64*1024, 5*time.Second, args...)
								if linkErr == nil {
									t.Fatal("LINK accepted unresolved weak cycle")
								}
								if !strings.Contains(output, "LNK2019") || !strings.Contains(output, "LNK1120") || !strings.Contains(output, "fy_wa") || !strings.Contains(output, "fy_wb") {
									t.Fatalf("unclassified cycle failure: %v\n%s", linkErr, output)
								}
								t.Logf("cycle oracle rejects before link; independent LINK failure: %s", strings.TrimSpace(output))
								return
							}
							if err != nil {
								t.Fatal(err)
							}
							for _, reverse := range []bool{false, true} {
								t.Run(fmt.Sprintf("reverse-%t", reverse), func(t *testing.T) {
									ordered := append([]WindowsHostObject(nil), objects...)
									if reverse {
										for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
											ordered[i], ordered[j] = ordered[j], ordered[i]
										}
									}
									a, err := buildWindowsMSVCWithOptions(tc, profile, C11ConsumerMode, nil, WindowsMSVCBuildOptions{CaptureMap: true, PrebuiltObjects: ordered})
									if err != nil {
										t.Fatal(err)
									}
									for _, entry := range []struct{ name, root string }{{first, "fy_wa"}, {second, "fy_wb"}} {
										proof, err := pecoff.InspectDirectWeakAliasLink(inputs, rootObject, entry.name, entry.root, a.Bytes, a.LinkMap)
										if err != nil {
											t.Fatal(err)
										}
										if proof.Expected != proof.Actual {
											t.Fatal("unproven weak displacement")
										}
										pe, err := pecoff.InspectExecutable(a.Bytes)
										if err != nil {
											t.Fatal(err)
										}
										mutated := bytes.Clone(a.Bytes)
										changed := false
										for _, section := range pe.Sections {
											if proof.SiteRVA >= section.VirtualAddress && uint64(proof.SiteRVA-section.VirtualAddress)+4 <= uint64(section.RawSize) {
												at := uint64(section.RawOffset) + uint64(proof.SiteRVA-section.VirtualAddress)
												mutated[at] ^= 1
												changed = true
												break
											}
										}
										if !changed {
											t.Fatal("missing linked weak site")
										}
										if _, err := pecoff.InspectDirectWeakAliasLink(inputs, rootObject, entry.name, entry.root, mutated, a.LinkMap); err == nil {
											t.Fatal("accepted changed weak displacement")
										}
										wrong := bytes.ReplaceAll(a.LinkMap, []byte(proof.Plan.Owner), []byte("wrong.obj"))
										if _, err := pecoff.InspectDirectWeakAliasLink(inputs, rootObject, entry.name, entry.root, a.Bytes, wrong); err == nil {
											t.Fatal("accepted wrong map owner")
										}
										t.Logf("%s -> %s from %s; path %v; displacement %d", entry.root, proof.Plan.Selected, proof.Plan.Owner, proof.Plan.Path, proof.Actual)
									}
									if _, err := pecoff.InspectAMD64Unwind(a.Bytes); err != nil {
										t.Fatal(err)
									}
									run, err := ExecuteWindowsMSVCTarget(a)
									if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
										t.Fatalf("weak native outcome %+v %v", run, err)
									}
								})
							}
						})
					}
				})
			}
		})
	}
}
