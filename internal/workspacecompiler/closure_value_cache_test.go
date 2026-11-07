package workspacecompiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/inlinebody"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// These are real source packages; no lifted metadata is manufactured for admission.
// Distinct, noncommutative bodies distinguish the target and written capture order.
const closureValueCacheProvider = `module answer;
pub struct Holder{fn i32(i32) action;}
pub fn fn i32(i32) make(i32 seed){
 var i32 a=seed;let i32 b=3;let i32 c=2;let i32 d=4;
 let fn i32(i32) action=fn [i32 a,i32 b,i32 c,i32 d] i32(i32 x){let reduced=a-b*x;return reduced+c-d;};
 a=999;return action;
}
pub fn fn i32(i32) other(){let fn i32(i32) action=fn [] i32(i32 x){return x-1;};return action;}
pub fn fn i32(i32) trapping(){let fn i32(i32) action=fn [] i32(i32 x){let unused=1/0;return x;};return action;}
pub fn Holder boxed(i32 seed){let action=make(seed);return Holder{.action=action};}
pub fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
`

const closureValueCacheConsumer = `module main;import answer as a;
fn i32 main(){
 let original=a::make(50);let copy=original;let holder=a::boxed(50);let stored=holder;let extracted=stored.action;
 let different=a::other();let unused=a::trapping();
 assert(a::apply(different,43)==42);assert(a::apply(copy,2)==a::apply(extracted,2));
 var i32 count=0;let result=extracted({count=count+1;2});assert(count==1);return result;
}`

type closureValueCacheCase struct {
	name, consumer, trap string
	want                 int32
}

func closureValueCacheCases() []closureValueCacheCase {
	return []closureValueCacheCase{
		{"factory-struct-snapshot-targets-unused-trap-argument-once", closureValueCacheConsumer, "", 42},
		{"invoked-unused-local-traps", `module main;import answer as a;fn i32 main(){let action=a::trapping();return a::apply(action,42);}`, "division-by-zero", 0},
		{"argument-traps-before-body", `module main;import answer as a;fn i32 main(){let action=a::make(-2147483648);return a::apply(action,1/0);}`, "division-by-zero", 0},
	}
}

func closureValueCacheWorkspace(t *testing.T, tc closureValueCacheCase) string {
	t.Helper()
	root := writeCacheWorkspace(t, closureValueCacheProvider)
	writeFile(t, filepath.Join(root, "src", "main.fy"), tc.consumer)
	t.Logf("provider-sha256=%x consumer-sha256=%x", sha256.Sum256([]byte(closureValueCacheProvider)), sha256.Sum256([]byte(tc.consumer)))
	return root
}

func closureValueCacheFresh(t *testing.T, workspace workspaceloader.Workspace) compiler.Result {
	t.Helper()
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := compiler.CheckPackages(input)
	if err != nil || !fresh.Verified || len(fresh.Semantic.Diagnostics) != 0 || len(fresh.Semantic.Unsupported) != 0 {
		t.Fatalf("compiler.CheckPackages source admission: error=%v verified=%v diagnostics=%+v unsupported=%+v", err, fresh.Verified, fresh.Semantic.Diagnostics, fresh.Semantic.Unsupported)
	}
	return fresh
}

func closureValueCacheEntry(t *testing.T, workspace workspaceloader.Workspace) string {
	t.Helper()
	id, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	return hir.QualifiedModulePath(id, "main") + "::main"
}

func closureValueCacheOutcome(t *testing.T, program mir.Program, entry string, tc closureValueCacheCase) interpreter.Outcome {
	t.Helper()
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("closure MIR verification: %+v", failures)
	}
	out, err := interpreter.Execute(program, entry)
	if err != nil || tc.trap == "" && (out.Kind != interpreter.OutcomeReturn || out.Value != tc.want) || tc.trap != "" && (out.Kind != interpreter.OutcomeTrap || out.TrapCategory != tc.trap) || len(out.Stdout) != 0 || len(out.Stderr) != 0 {
		t.Fatalf("%s execution: %+v error=%v want=%d trap=%s", tc.name, out, err, tc.want, tc.trap)
	}
	return out
}

func closureValueCacheCompare(t *testing.T, workspace workspaceloader.Workspace, restored compiler.Result, tc closureValueCacheCase) {
	t.Helper()
	entry := closureValueCacheEntry(t, workspace)
	fresh := closureValueCacheFresh(t, workspace)
	want := closureValueCacheOutcome(t, fresh.MIR, entry, tc)
	got := closureValueCacheOutcome(t, restored.MIR, entry, tc)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restored execution differs from fresh source: got=%+v fresh=%+v", got, want)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(restored.MIR, entry, profile)
		if err != nil || !strings.Contains(unit.Source, "fy_closure_i32") {
			t.Fatalf("restored closure emission: %v", err)
		}
	}
}

func TestClosureValueCacheSourceAdmission(t *testing.T) {
	for _, tc := range closureValueCacheCases() {
		t.Run(tc.name, func(t *testing.T) {
			workspace := loadCacheWorkspace(t, closureValueCacheWorkspace(t, tc))
			fresh := closureValueCacheFresh(t, workspace)
			closureValueCacheCompare(t, workspace, fresh, tc)
			targets := 0
			for _, fn := range fresh.MIR.Functions {
				if fn.ClosureTarget {
					targets++
				}
			}
			if targets != 3 {
				t.Fatalf("real source must produce three distinct lifted targets, got %d", targets)
			}
		})
	}
}

func closureValueCacheCheck(t *testing.T, session *Session, root string, tc closureValueCacheCase, hits int, allHit bool) (workspaceloader.Workspace, Result) {
	t.Helper()
	workspace := loadCacheWorkspace(t, root)
	lower := session.lower
	if allHit {
		// Reuse the existing fail-on-lowering guard, not a report flag alone.
		forbidInlineCallsLowering(t, session)
		defer func() { session.lower = lower }()
	}
	r, err := session.Check(workspace)
	if err != nil || !r.Compiler.Verified || r.Cache.Packages != 2 || r.Cache.Hits != hits || (r.Cache.Outcome == OutcomeAllHit) != allHit || r.Cache.Lowered == allHit || !allHit && r.Cache.Stored != 2 {
		t.Fatalf("closure value Session.Check: error=%v cache=%+v diagnostics=%+v unsupported=%+v", err, r.Cache, r.Compiler.Semantic.Diagnostics, r.Compiler.Semantic.Unsupported)
	}
	closureValueCacheCompare(t, workspace, r.Compiler, tc)
	t.Logf("%s cache=%+v", tc.name, r.Cache)
	return workspace, r
}

func TestClosureValueCacheRestore(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		for _, tc := range closureValueCacheCases() {
			t.Run(storage+"/"+tc.name, func(t *testing.T) {
				root := closureValueCacheWorkspace(t, tc)
				// Prove ordinary source admission independently of artifact support.
				closureValueCacheFresh(t, loadCacheWorkspace(t, root))
				config := inlineCallsConfiguration(t, storage, t.TempDir())
				session := newInlineCallsSession(t, config)
				closureValueCacheCheck(t, session, root, tc, 0, false)
				closureValueCacheCheck(t, session, root, tc, 2, true)
				if storage == StorageDisk {
					if err := session.Close(); err != nil {
						t.Fatal(err)
					}
					session = newInlineCallsSession(t, config)
					closureValueCacheCheck(t, session, root, tc, 2, true)
				}
			})
		}
	}
}

func TestClosureValueCacheProviderBodyInvalidation(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			tc := closureValueCacheCases()[0]
			root := closureValueCacheWorkspace(t, tc)
			session := newInlineCallsSession(t, inlineCallsConfiguration(t, storage, t.TempDir()))
			closureValueCacheCheck(t, session, root, tc, 0, false)
			closureValueCacheCheck(t, session, root, tc, 2, true)
			// Only a private lifted body changes; every written public signature stays.
			changed := strings.Replace(closureValueCacheProvider, "return reduced+c-d;", "return reduced+c-d+1;", 1)
			if changed == closureValueCacheProvider {
				t.Fatal("body edit did not change fixture")
			}
			writeFile(t, filepath.Join(root, "libs", "answer", "src", "answer.fy"), changed)
			tc.want = 43
			// Consumer may be a raw hit with a stable signature, but the program
			// must lower/repair and can never execute the old provider helper.
			workspace := loadCacheWorkspace(t, root)
			r, err := session.Check(workspace)
			if err != nil || !r.Compiler.Verified || !r.Cache.Lowered || r.Cache.Outcome == OutcomeAllHit || r.Cache.Hits > 1 || r.Cache.Stored != 2 {
				t.Fatalf("provider body invalidation: %v %+v", err, r.Cache)
			}
			closureValueCacheCompare(t, workspace, r.Compiler, tc)
			closureValueCacheCheck(t, session, root, tc, 2, true)
		})
	}
}

func closureValueCacheNative(t *testing.T, workspace workspaceloader.Workspace, r Result, tc closureValueCacheCase) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 native only; Linux validation is batched separately")
	}
	toolchain, err := cdriver.ResolveNativeToolchain(cbackend.Stage0WindowsMSVCProfile().Target, "")
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(r.Compiler.MIR, closureValueCacheEntry(t, workspace), cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.WindowsRunSource(cdriver.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: profile, UnitSource: unit.Source, RuntimeSource: support, TrapControl: true})
			if err != nil {
				t.Fatal(err)
			}
			out, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{RejectOutput: true})
			if err != nil || out.TrapCategory != tc.trap || tc.trap == "" && out.ExitCode != tc.want {
				t.Fatalf("restored native closure: %v %+v", err, out)
			}
			t.Logf("%s storage=%s C-sha256=%x image-sha256=%x outcome=%+v", profile.Name, r.Cache.Storage, sha256.Sum256([]byte(unit.Source)), sha256.Sum256(artifact.Bytes), out)
		})
	}
}

func TestClosureValueCacheNativeRestore(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 native only; Linux validation is batched separately")
	}
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			tc := closureValueCacheCases()[0]
			root := closureValueCacheWorkspace(t, tc)
			cacheRoot := t.TempDir()
			config := inlineCallsConfiguration(t, storage, cacheRoot)
			session := newInlineCallsSession(t, config)
			closureValueCacheCheck(t, session, root, tc, 0, false)
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newInlineCallsSession(t, config)
			}
			workspace, restored := closureValueCacheCheck(t, session, root, tc, 2, true)
			closureValueCacheNative(t, workspace, restored, tc)
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestClosureValueCacheChild$", "-test.count=1", "-test.v")
				command.Env = append(os.Environ(), "FAIRYC_CLOSURE_VALUE_CHILD="+root, "FAIRYC_CLOSURE_VALUE_STORE="+cacheRoot, "FAIRYC_CLOSURE_VALUE_NATIVE=1")
				output, err := command.CombinedOutput()
				if err != nil || !strings.Contains(string(output), "closure-value-new-process-disk-ok") {
					t.Fatalf("new process: %v\n%s", err, output)
				}
				t.Logf("new process: %s", output)
			}
		})
	}
}

func TestClosureValueCacheChild(t *testing.T) {
	root := os.Getenv("FAIRYC_CLOSURE_VALUE_CHILD")
	if root == "" {
		t.Skip("new-process disk consumer only")
	}
	tc := closureValueCacheCases()[0]
	if name := os.Getenv("FAIRYC_CLOSURE_VALUE_CASE"); name != "" {
		found := false
		for _, candidate := range closureValueCacheCases() {
			if candidate.name == name {
				tc, found = candidate, true
			}
		}
		if !found {
			t.Fatalf("unknown child fixture %q", name)
		}
	}
	session := newInlineCallsSession(t, inlineCallsConfiguration(t, StorageDisk, os.Getenv("FAIRYC_CLOSURE_VALUE_STORE")))
	workspace, restored := closureValueCacheCheck(t, session, root, tc, 2, true)
	if os.Getenv("FAIRYC_CLOSURE_VALUE_NATIVE") == "1" {
		closureValueCacheNative(t, workspace, restored, tc)
	}
	fmt.Println("closure-value-new-process-disk-ok")
}

func TestClosureValueCacheNewProcess(t *testing.T) {
	for _, tc := range closureValueCacheCases() {
		t.Run(tc.name, func(t *testing.T) {
			root := closureValueCacheWorkspace(t, tc)
			cacheRoot := t.TempDir()
			session := newInlineCallsSession(t, inlineCallsConfiguration(t, StorageDisk, cacheRoot))
			closureValueCacheCheck(t, session, root, tc, 0, false)
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestClosureValueCacheChild$", "-test.count=1", "-test.v")
			command.Env = append(os.Environ(), "FAIRYC_CLOSURE_VALUE_CHILD="+root, "FAIRYC_CLOSURE_VALUE_STORE="+cacheRoot, "FAIRYC_CLOSURE_VALUE_CASE="+tc.name, "FAIRYC_CLOSURE_VALUE_NATIVE=0")
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "closure-value-new-process-disk-ok") {
				t.Fatalf("new-process %s: %v\n%s", tc.name, err, output)
			}
			t.Logf("new-process %s: %s", tc.name, output)
		})
	}
}

// The injected products remain valid MIR. Cache/source validation must bind
// target choice, ordered snapshot operands and helper body to the source rather
// than accepting a structurally safe but semantically different executable.
func forgeClosureValueCacheProduct(t *testing.T, program *mir.Program, kind string) {
	t.Helper()
	changed := false
	for fi := range program.Functions {
		fn := &program.Functions[fi]
		if kind == "body" && fn.ClosureTarget && fn.ClosureCaptures == 4 {
			for bi := range fn.Blocks {
				term := &fn.Blocks[bi].Terminator
				if term.Kind == mir.TerminatorReturn {
					term.ReturnValue = mir.Operand{Kind: mir.OperandI32, I32: 99}
					changed = true
				}
			}
		}
		for bi := range fn.Blocks {
			if kind == "arithmetic" && fn.ClosureTarget && fn.ClosureCaptures == 4 && fn.Blocks[bi].Terminator.Operator == "-" {
				fn.Blocks[bi].Terminator.Operator = "+"
			}
			for oi := range fn.Blocks[bi].Operations {
				op := &fn.Blocks[bi].Operations[oi]
				if kind == "arithmetic" && fn.ClosureTarget && fn.ClosureCaptures == 4 && (op.Kind == mir.OperationBinary || op.Kind == mir.OperationArithmeticCheck) && op.Operator == "-" {
					op.Operator = "+"
					changed = true
				}
				if op.Kind != mir.OperationClosureCreate {
					continue
				}
				if kind == "captures" && len(op.Operands) == 4 {
					op.Operands[0], op.Operands[1] = op.Operands[1], op.Operands[0]
					changed = true
				}
				if kind == "target" && strings.HasSuffix(fn.Name, "::other") {
					for _, candidate := range program.Functions {
						if candidate.ClosureTarget && candidate.ClosureCaptures == 0 && candidate.Symbol != op.Symbol {
							op.Symbol = candidate.Symbol
							changed = true
							break
						}
					}
				}
			}
		}
	}
	if !changed {
		t.Fatalf("no closure %s metadata to forge", kind)
	}
	if failures := mir.Verify(*program); len(failures) != 0 {
		t.Fatalf("source-inconsistent %s fixture must remain structurally valid: %+v", kind, failures)
	}
}

func TestClosureValueCacheNoBadPublication(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		for _, kind := range []string{"body", "arithmetic", "target", "captures"} {
			t.Run(storage+"/"+kind, func(t *testing.T) {
				tc := closureValueCacheCases()[0]
				root := closureValueCacheWorkspace(t, tc)
				config := inlineCallsConfiguration(t, storage, t.TempDir())
				session := newInlineCallsSession(t, config)
				lower := session.lower
				session.lower = func(ctx context.Context, analysis compiler.Analysis) (compiler.Result, error) {
					compiled, err := lower(ctx, analysis)
					if err == nil && compiled.Verified {
						forgeClosureValueCacheProduct(t, &compiled.MIR, kind)
					}
					return compiled, err
				}
				bad, err := session.Check(loadCacheWorkspace(t, root))
				entries := session.Stats().Store.Entries
				if storage == StorageDisk {
					stats, statsErr := session.PersistentStats()
					if statsErr != nil {
						t.Fatal(statsErr)
					}
					entries = stats.Store.Entries
				}
				if err == nil || bad.Compiler.Verified || len(bad.Compiler.MIR.Functions) != 0 || entries != 0 {
					t.Fatalf("forged %s executable/publication escaped: error=%v verified=%v functions=%d retained=%d cache=%+v", kind, err, bad.Compiler.Verified, len(bad.Compiler.MIR.Functions), entries, bad.Cache)
				}
				t.Logf("refused forged %s before executable/publication: %v", kind, err)
				session.lower = lower
				if storage == StorageDisk {
					if err := session.Close(); err != nil {
						t.Fatal(err)
					}
					session = newInlineCallsSession(t, config)
				}
				closureValueCacheCheck(t, session, root, tc, 0, false)
				closureValueCacheCheck(t, session, root, tc, 2, true)
			})
		}
	}
}

// Tamper with actual retained cache products, not session.restore or lower.
// The codec accepts the new, well-typed product and all key/interface/source
// identities remain intact, so a checksum or ordinary MIR gate cannot repair it.
func tamperClosureValueCache(t *testing.T, session *Session, config Configuration, workspace workspaceloader.Workspace, cold Result, kind string) string {
	t.Helper()
	forged := mir.Clone(cold.Compiler.MIR)
	forgeClosureValueCacheProduct(t, &forged, kind)
	out, err := interpreter.Execute(forged, closureValueCacheEntry(t, workspace))
	if err != nil || out.Kind == interpreter.OutcomeReturn && out.Value == 42 {
		t.Fatalf("forgery must change real execution before injecting it: %+v %v", out, err)
	}
	t.Logf("well-typed %s forged execution=%+v", kind, out)
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	identities, coordinates, err := packageIdentityMaps(metadata)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := packageartifact.ExtractAll(forged, identities, coordinates)
	if err != nil {
		t.Fatal("structurally valid forgery must pass artifact extraction", err)
	}
	var provider packageartifact.Artifact
	for _, artifact := range artifacts {
		for _, record := range artifact.Functions {
			if record.Body.ClosureTarget && record.Body.ClosureCaptures == 4 {
				provider = artifact
			}
		}
	}
	if len(provider.Functions) == 0 {
		t.Fatal("provider artifact missing")
	}
	if config.Persistent == nil {
		interfaces, err := packageInterfaces(cold.Compiler.Semantic.HIR, metadata, coordinates)
		if err != nil {
			t.Fatal(err)
		}
		owners := functionOwners(cold.Compiler.Semantic.HIR)
		bodies, err := contractstatemir.ComputeHIRStateBodyHashes(cold.Compiler.Semantic.HIR, owners, coordinates)
		if err != nil {
			t.Fatal(err)
		}
		inline, err := inlinebody.HIR(cold.Compiler.Semantic.HIR, coordinates)
		if err != nil {
			t.Fatal(err)
		}
		keys, err := session.packageKeys(metadata, interfaces, bodies, owners, inline)
		if err != nil {
			t.Fatal(err)
		}
		if err := session.cache.Put(keys[provider.PackageID], interfaces[provider.PackageID], provider); err != nil {
			t.Fatal("well-typed forged process artifact was not accepted", err)
		}
		return ""
	}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	stable, err := packageartifact.DescribeStableArtifact(provider, config.MaxArtifactBytes)
	if err != nil {
		t.Fatal("well-typed forged stable projection", err)
	}
	var records []string
	if err := filepath.WalkDir(config.Persistent.Root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Ext(path) == ".json" {
			records = append(records, path)
		}
		return err
	}); err != nil || len(records) != 2 {
		t.Fatalf("expected actual two-record disk seed: %v %v", records, err)
	}
	for _, path := range records {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// Reuse the existing cache fixture's exact outer wire shape.
		var outer compositeOptionDiskEnvelope
		if err := json.Unmarshal(raw, &outer); err != nil {
			t.Fatal(err)
		}
		canonical, err := json.Marshal(outer)
		if err != nil || !bytes.Equal(raw, canonical) || len(outer.Record.Artifacts) != 1 {
			t.Fatal("disk seed is not the expected canonical envelope", err)
		}
		compilation, err := packageartifact.DecodeStableCompilationArtifact(outer.Record.Artifacts[0].Data, config.MaxArtifactBytes)
		if err != nil {
			t.Fatal(err)
		}
		if compilation.MIR.Package != provider.Package {
			continue
		}
		compilation.MIR = stable
		outer.Record.Artifacts[0].Data, err = packageartifact.EncodeStableCompilationArtifact(compilation, config.MaxArtifactBytes)
		if err != nil {
			t.Fatal("well-typed forged compilation must pass codec before injection", err)
		}
		encoded, err := json.Marshal(outer)
		if err != nil || bytes.Equal(encoded, raw) {
			t.Fatal("disk forgery did not change the seed", err)
		}
		digest := sha256.Sum256(encoded)
		forgedPath := filepath.Join(filepath.Dir(path), hex.EncodeToString(digest[:])+".json")
		if err := os.WriteFile(forgedPath, encoded, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		t.Logf("same source key=%s/interface=%s; forged valid generation sha256=%x", outer.Record.Key, outer.Record.PublicInterface, digest)
		return forgedPath
	}
	t.Fatal("provider disk record missing")
	return ""
}

func TestClosureValueCacheForgedRestoreRepair(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		for _, kind := range []string{"arithmetic", "captures", "target", "body"} {
			t.Run(storage+"/"+kind, func(t *testing.T) {
				tc := closureValueCacheCases()[0]
				root := closureValueCacheWorkspace(t, tc)
				config := inlineCallsConfiguration(t, storage, t.TempDir())
				session := newInlineCallsSession(t, config)
				workspace, cold := closureValueCacheCheck(t, session, root, tc, 0, false)
				forgedPath := tamperClosureValueCache(t, session, config, workspace, cold, kind)
				if storage == StorageDisk {
					session = newInlineCallsSession(t, config)
				}
				lower, calls := session.lower, 0
				session.lower = func(ctx context.Context, analysis compiler.Analysis) (compiler.Result, error) {
					calls++
					return lower(ctx, analysis)
				}
				repaired, err := session.Check(workspace)
				if err != nil || !repaired.Compiler.Verified || repaired.Cache.Outcome != OutcomeFallback || !repaired.Cache.ReassemblyFallback || !repaired.Cache.Lowered || repaired.Cache.Hits != 2 || repaired.Cache.Stored != 2 || calls != 1 {
					t.Fatalf("well-typed forged %s was not repaired from source: error=%v calls=%d cache=%+v", kind, err, calls, repaired.Cache)
				}
				closureValueCacheCompare(t, workspace, repaired.Compiler, tc)
				t.Logf("source repaired %s with exactly %d lower hook call: %+v", kind, calls, repaired.Cache)
				if forgedPath != "" {
					if _, err := os.Stat(forgedPath); !os.IsNotExist(err) {
						t.Fatalf("forged generation not quarantined: %v", err)
					}
					if err := session.Close(); err != nil {
						t.Fatal(err)
					}
					session = newInlineCallsSession(t, config)
				}
				closureValueCacheCheck(t, session, root, tc, 2, true)
			})
		}
	}
}

func TestClosureValueCachePrivateFormats(t *testing.T) {
	workspace := loadCacheWorkspace(t, closureValueCacheWorkspace(t, closureValueCacheCases()[0]))
	fresh := closureValueCacheFresh(t, workspace)
	metadata, err := workspace.SemanticPackages()
	if err != nil {
		t.Fatal(err)
	}
	identities, coordinates, err := packageIdentityMaps(metadata)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := packageartifact.ExtractAll(fresh.MIR, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := packageartifact.NewProcessCodec(8 << 20)
	if err != nil {
		t.Fatal(err)
	}
	// Each fixture has a closure field in its projected context. Inspect exact
	// wire schemas and reject downgrades rather than accepting a legacy fallback.
	check := func(data []byte, encodeErr error, schema, version, legacySchema, legacyVersion string, decode func([]byte) error) {
		t.Helper()
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		var header struct{ Schema, Version string }
		if err := json.Unmarshal(data, &header); err != nil || header.Schema != schema || header.Version != version {
			t.Fatalf("exact closure format: error=%v header=%+v want=%s/%s", err, header, schema, version)
		}
		if err := decode(data); err != nil {
			t.Fatal("new exact format did not decode", err)
		}
		downgraded := bytes.Replace(data, []byte(schema), []byte(legacySchema), 1)
		downgraded = bytes.Replace(downgraded, []byte(version), []byte(legacyVersion), 1)
		if err := decode(downgraded); err == nil {
			t.Fatal("closure content accepted under a legacy envelope")
		}
		t.Logf("exact closure schema=%s version=%s legacy downgrade refused", schema, version)
	}
	targets := 0
	for _, artifact := range artifacts {
		data, err := codec.Encode(artifact)
		check(data, err, "fairyc.package-mir-closure-process/v0", "fairyc-package-mir-closure-process/1", "fairyc.package-mir-process/v3", "fairyc-package-mir-process/3", func(data []byte) error { _, err := codec.Decode(data); return err })
		stable, err := packageartifact.DescribeStableArtifact(artifact, 8<<20)
		if err != nil {
			t.Fatal(err)
		}
		data, err = packageartifact.EncodeStableArtifact(stable, 8<<20)
		check(data, err, "fairyc.stable-package-closure-mir/v0", "fairyc-stable-package-closure-mir/1", "fairyc.stable-package-mir/v1", "fairyc-stable-package-mir/5", func(data []byte) error { _, err := packageartifact.DecodeStableArtifact(data, 8<<20); return err })
		data, err = packageartifact.EncodeContextDescriptor(stable.Context, 8<<20)
		check(data, err, "fairyc.package-type-closure-context/v0", "fairyc-package-type-closure-context/1", "fairyc.package-type-context/v2", "fairyc-package-type-context/2", func(data []byte) error { _, err := packageartifact.DecodeContextDescriptor(data, 8<<20); return err })
		for _, body := range stable.Functions {
			if !body.Function.ClosureTarget {
				continue
			}
			targets++
			data, err = packageartifact.EncodeStableBody(body, 8<<20)
			check(data, err, "fairyc.package-function-closure-body/v0", "fairyc-package-function-closure-body/1", "fairyc.package-function-body/v4", "fairyc-package-function-body/4", func(data []byte) error { _, err := packageartifact.DecodeStableBody(data, 8<<20); return err })
		}
	}
	if targets != 3 {
		t.Fatalf("format gate did not exercise all real helper targets: %d", targets)
	}
	// A fixed, feature-free legacy vector stays byte-for-byte unchanged.
	legacy, err := packageartifact.EncodeContextDescriptor(packageartifact.ContextDescriptor{}, 8<<20)
	const want = `{"schema":"fairyc.package-type-context/v2","version":"fairyc-package-type-context/2","context":{"structs":null,"enums":null,"resources":null,"foreignFunctions":null}}`
	if err != nil || string(legacy) != want {
		t.Fatalf("feature-free legacy bytes changed: %v %s", err, legacy)
	}
}

func TestClosureValueCacheUnusedCarrier(t *testing.T) {
	for _, storage := range []string{StorageMemory, StorageDisk} {
		t.Run(storage, func(t *testing.T) {
			// A dormant Fn field does not select any function for reconstruction.
			// Projection may omit that declaration; it must not invent a source
			// mismatch or force a perfectly good scalar package back to lowering.
			root := writeCacheWorkspace(t, `module answer;pub struct Dormant{fn i32(i32) action;}pub fn i32 value(){return 42;}`)
			config := inlineCallsConfiguration(t, storage, t.TempDir())
			session := newInlineCallsSession(t, config)
			// This existing helper checks scalar execution/cache reports without
			// demanding a closure C representation for an unused declaration.
			checkInlineCallsCache(t, session, root, 42, 0, false)
			checkInlineCallsCache(t, session, root, 42, 2, true)
			if storage == StorageDisk {
				if err := session.Close(); err != nil {
					t.Fatal(err)
				}
				session = newInlineCallsSession(t, config)
				checkInlineCallsCache(t, session, root, 42, 2, true)
			}
		})
	}
}

func TestClosureValueCacheSourceProofStructure(t *testing.T) {
	workspace := loadCacheWorkspace(t, closureValueCacheWorkspace(t, closureValueCacheCases()[0]))
	fresh := closureValueCacheFresh(t, workspace)
	if err := verifyClosureValueSource(t.Context(), fresh.Semantic.HIR, fresh.MIR); err != nil {
		t.Fatal("unchanged source proof control", err)
	}
	t.Run("canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if err := verifyClosureValueSource(ctx, fresh.Semantic.HIR, fresh.MIR); !errors.Is(err, context.Canceled) {
			t.Fatalf("source proof ignored cancellation: %v", err)
		}
	})
	t.Run("carrier-field", func(t *testing.T) {
		forged := mir.Clone(fresh.MIR)
		changed := false
		for si := range forged.StructTypes {
			for fi := range forged.StructTypes[si].Fields {
				field := &forged.StructTypes[si].Fields[fi]
				if field.Type == hir.TypeClosureI32 {
					field.Name = "forged_action"
					changed = true
				}
			}
		}
		if !changed {
			t.Fatal("no actual carrier declaration to forge")
		}
		if err := verifyClosureValueSource(t.Context(), fresh.Semantic.HIR, forged); err == nil || !strings.Contains(err.Error(), "carrier field") {
			t.Fatalf("source-owned carrier field was not refused: %v", err)
		}
	})
	t.Run("duplicate-qualified-name", func(t *testing.T) {
		forged := mir.Clone(fresh.MIR)
		duplicate := forged.Functions[0]
		for _, fn := range forged.Functions {
			duplicate.Symbol = max(duplicate.Symbol, fn.Symbol)
		}
		duplicate.Symbol++
		forged.Functions = append(forged.Functions, duplicate)
		if err := verifyClosureValueSource(t.Context(), fresh.Semantic.HIR, forged); err == nil || !strings.Contains(err.Error(), "ambiguous function name") {
			t.Fatalf("duplicate qualified function name was not refused: %v", err)
		}
	})
}
