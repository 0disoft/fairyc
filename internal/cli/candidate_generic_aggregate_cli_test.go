package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const aggregateCLITests = `module tests::aggregates;import maker as m;
@test fn void values(){let box=m::make<i32>(42);assert(m::score(box)==42);let pair=m::Pair<i32,u8>{.left=2,.right=3};assert(pair.left==2);assert(pair.right==3);
var cell=m::Cell<i32>{.value=40};let copy=cell;cell.value+=2;assert(copy.value==40);assert(cell.value==42);
let m::Cell<i32>[2] rows=[cell,copy];assert(rows[0].value==42);assert(rows[1].value==40);
let Option<m::Holder<i32>> nested=.some(m::make<i32>(5));let empty=m::empty();}
`

func aggregateCLIWorkspace(t *testing.T) (string, string) {
	t.Helper()
	source := strings.Replace(aggregatePlanRoot, "module root;", "module main;", 1)
	source = strings.Replace(source, "return box.payload.value;", "return m::score(box);", 1)
	maker := strings.Replace(aggregatePlanMaker, ".value=value", ".value=h::finish<T>(value)", 1)
	maker += "pub fn i32 score(Holder<i32> value){return value.payload.value;}"
	maker += "pub resource Empty<T>{cleanup{}}pub fn Empty<bool> empty(){return Empty<bool>{};}"
	root := candidateAggregateCacheWorkspace(t, candidateAggregateLayoutFixture{root: source, maker: maker})
	if err := os.Rename(filepath.Join(root, "src", "root.fy"), filepath.Join(root, "src", "main.fy")); err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "tests/aggregates.fy", aggregateCLITests)
	if err := os.Mkdir(filepath.Join(root, "cache"), 0700); err != nil {
		t.Fatal(err)
	}
	return root, source
}

func aggregateCLICache(t *testing.T, root string) map[string][]byte {
	t.Helper()
	return candidateAggregateCacheFiles(t, filepath.Join(root, "cache"))
}

func aggregateCLIOracle(t *testing.T, root string) (string, string, string) {
	t.Helper()
	workspace, _, err := loadWorkspaceWithLock(root, false)
	if err != nil {
		t.Fatal(err)
	}
	check := func(workspace workspaceloader.Workspace) compiler.Result {
		t.Helper()
		input, err := workspace.SemanticInput()
		if err != nil {
			t.Fatal(err)
		}
		coordinates, err := workspace.PackageCoordinates()
		if err != nil {
			t.Fatal(err)
		}
		result, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
			t.Fatalf("private Go aggregate oracle: %v %+v %+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
		}
		return result
	}
	production := check(workspace)
	input, err := workspace.SemanticInput()
	if err != nil {
		t.Fatal(err)
	}
	wantCheck := fmt.Sprintf("check ok: %d instance(s), %d source file(s), %d MIR function(s), lock verified\n", len(workspace.Graph.Nodes), len(input.Files), len(production.MIR.Functions))
	workspace, err = workspaceloader.WithPackageTests(workspace, ".")
	if err != nil {
		t.Fatal(err)
	}
	var diagnostic, listing, report bytes.Buffer
	tests, _, code := prepareWorkspaceTests(workspace, ".", "", false, &diagnostic)
	if code != 0 {
		t.Fatalf("Go aggregate tests: %d %s", code, &diagnostic)
	}
	checked := check(workspace)
	outcomes, code := runInterpreterWorkspaceTests(checked.MIR, tests, interpreter.Options{Context: t.Context()}, &diagnostic)
	if code != 0 {
		t.Fatalf("Go aggregate interpreter: %d %s", code, &diagnostic)
	}
	options := stableTestOptions{json: true}
	if writeStableTestListJSON(tests, "verified", options, &listing, &diagnostic) != 0 || writeStableTestReportJSON(outcomes, "verified", options, &report, &diagnostic) != 0 {
		t.Fatal("Go aggregate reports", &diagnostic)
	}
	return wantCheck, listing.String(), report.String()
}

func TestCandidateGenericAggregateCLI(t *testing.T) {
	for _, profile := range copiedClosureCLIProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			root, source := aggregateCLIWorkspace(t)
			wantCheck, wantList, wantTests := aggregateCLIOracle(t, root)
			image := copiedClosureCLIHostWithLimit(t, profile, 2<<20)
			// The sidecar only consumes Fairy's generated C. No compiler or
			// package resolver can be substituted through PATH at execution.
			t.Setenv("PATH", "")
			command := copiedClosureCommand{t: t, image: image, root: root}
			if out, stderr := command.run(0, "check", "--locked"); out != wantCheck || stderr != "" {
				t.Fatalf("aggregate check differs from Go: %q %q, want %q", out, stderr, wantCheck)
			}
			command.cache(0, "cache: cold stored", "check", "--locked")
			before := aggregateCLICache(t, root)
			command.cache(0, "cache: hit", "check", "--locked")
			if !reflect.DeepEqual(before, aggregateCLICache(t, root)) {
				t.Fatal("warm aggregate check changed cached generations")
			}
			for _, flags := range [][]string{nil, {"--release"}} {
				args := append([]string{"build", "--locked"}, flags...)
				command.cache(0, "cache: hit", args...)
				ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
				out, err := exec.CommandContext(ctx, filepath.Join(root, "build", "root.exe")).CombinedOutput()
				cancel()
				if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 42 || len(out) != 0 {
					t.Fatalf("published aggregate program: %v %s", err, out)
				}
				args[0] = "run"
				command.cache(42, "cache: hit", args...)
			}
			command.run(42, "run", "--locked")
			if out, stderr := command.run(0, "test", "--list", "--json", "--locked"); out != wantList || stderr != "" {
				t.Fatalf("aggregate test list differs from Go: %s %s, want %s", out, stderr, wantList)
			}
			if out, stderr := command.run(0, "test", "--json", "--locked"); out != wantTests || stderr != "" {
				t.Fatalf("aggregate interpreter differs from Go: %s %s, want %s", out, stderr, wantTests)
			}
			for _, flags := range [][]string{{}, {"--native"}, {"--native", "--release"}} {
				args := append([]string{"test", "--json", "--locked", "--cache-root", "cache", "--cache-status"}, flags...)
				out, _ := command.run(0, args...)
				if !strings.Contains(out, `"passed":1`) {
					t.Fatal("aggregate cached test did not execute", out)
				}
				command.cache(0, "cache: units hit", append([]string{"test", "--json", "--locked"}, flags...)...)
			}
			writeTestSource(t, root, "tests/aggregates.fy", strings.Replace(aggregateCLITests, "score(box)==42", "score(box)==43", 1))
			command.run(5, "test", "--json", "--locked", "--cache-root", "cache", "--cache-status")
			if out := command.cache(5, "cache: units hit", "test", "--json", "--locked"); !strings.Contains(out, `"failed":1`) {
				t.Fatal("cached aggregate test failure was not rerun", out)
			}
			writeTestSource(t, root, "tests/aggregates.fy", aggregateCLITests)
			command.run(0, "test", "--json", "--locked", "--cache-root", "cache")
			writeTestSource(t, root, "libs/helper/src/helper.fy", strings.Replace(aggregatePlanHelper, "return value;", "return value+1;", 1))
			before = aggregateCLICache(t, root)
			command.cache(7, "cache: skipped", "check", "--locked")
			if !reflect.DeepEqual(before, aggregateCLICache(t, root)) {
				t.Fatal("stale lock changed aggregate cache")
			}
			_, status := command.run(43, "run", "--cache-root", "cache", "--cache-status")
			if strings.HasPrefix(status, "cache: hit\n") || !strings.HasPrefix(status, "cache:") {
				t.Fatal("aggregate provider edit reused stale Complete", status)
			}
			command.cache(43, "cache: hit", "run", "--locked")
			for name := range aggregateCLICache(t, root) {
				if err := os.WriteFile(filepath.Join(root, "cache", name), []byte("corrupt"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			command.cache(0, "cache: cold stored", "check", "--locked")
			command.cache(43, "cache: hit", "run", "--locked")
			before = aggregateCLICache(t, root)
			writeTestSource(t, root, "src/main.fy", strings.Replace(source, ".left=2", ".missing=2", 1))
			command.cache(3, "cache: cold uncached", "check")
			if !reflect.DeepEqual(before, aggregateCLICache(t, root)) {
				t.Fatal("failed aggregate Complete published cache bytes")
			}
			writeTestSource(t, root, "src/main.fy", source)
			command.cache(0, "cache: hit", "check")
			_, status = command.run(43, "run", "--locked", "--cache-root", "missing", "--cache-status")
			if !strings.HasPrefix(status, "cache: cold uncached\n") {
				t.Fatal("aggregate cold fallback status", status)
			}
			if _, err := os.Stat(filepath.Join(root, "missing")); !os.IsNotExist(err) {
				t.Fatal("cold fallback created a cache directory", err)
			}
			t.Logf("%d independent aggregate CLI processes", command.calls)
		})
	}
}

func TestCandidateAggregateCommandOrdinarySDK(t *testing.T) {
	main := `module main;import project_check_cli as check;@effects(filesystem,environment,clock,console)fn i32 main(){return check::arguments();}`
	candidateAtlasProbeSize(t, "project_check_cli", main, 512<<10, func(t *testing.T, image string) {
		root := writeWorkspaceSessionFixture(t, `module main;import std::vec as v;fn i32 main(){var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);return *v::at<i32>(ref values,0);}`)
		code, want, diagnostic := runForTest([]string{"check"}, root)
		if code != 0 || diagnostic != "" {
			t.Fatalf("Go SDK control: %d %s", code, diagnostic)
		}
		code, want, diagnostic = runForTest([]string{"check", "--locked"}, root)
		if code != 0 || diagnostic != "" {
			t.Fatalf("Go locked SDK control: %d %s", code, diagnostic)
		}
		lock, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
		if err != nil {
			t.Fatal(err)
		}
		code, out, stderr := fetchCLIProcess(t, image, root, []string{"check", "--locked"}, "")
		current, err := os.ReadFile(filepath.Join(root, "fairy.lock"))
		if code != 0 || stderr != "" || out != want || err != nil || !bytes.Equal(lock, current) {
			t.Fatalf("ordinary authenticated SDK lane: %d %s %s (%v), want %s", code, out, stderr, err, want)
		}
	})
}
