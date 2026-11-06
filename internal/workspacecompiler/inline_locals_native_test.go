package workspacecompiler

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
)

func inlineLocalsNativeUnit(t *testing.T, posix bool) (cbackend.Unit, string) {
	t.Helper()
	root := writeCacheWorkspace(t, `module answer;@inline pub fn i32 value(i32 a,i32 b){let i32 first=a*3;let i32 total=first+b;let i32 unused=12/3;return total/2;}`)
	writeFile(t, filepath.Join(root, "src", "main.fy"), `module main;import answer as a;
 unsafe extern C fn i32 inline_produce(i32 index);
 unsafe extern C fn i32 inline_finish(i32 result);
 fn i32 main(){let i32 result=a::value(unsafe{inline_produce(1)},unsafe{inline_produce(2)});return unsafe{inline_finish(result)};}`)
	targetID := target.X8664WindowsMSVC
	profile := cbackend.Stage0WindowsMSVCProfile()
	if posix {
		targetID = target.X8664LinuxGNU
		profile = cbackend.Stage0Profile()
	}
	session, err := NewSession(Configuration{Target: targetID, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	workspace := loadCacheWorkspace(t, root)
	cold, err := session.Check(workspace)
	if err != nil || !cold.Compiler.Verified {
		t.Fatalf("cold: %v %+v", err, cold.Cache)
	}
	warm, err := session.Check(workspace)
	if err != nil || warm.Cache.Outcome != OutcomeAllHit || warm.Cache.Lowered {
		t.Fatalf("warm: %v %+v", err, warm.Cache)
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	unit, err := cbackend.Emit(warm.Compiler.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", profile)
	if err != nil || !strings.Contains(unit.Source, "static inline int32_t") {
		t.Fatal("restored cross-package inline body missing", err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	return unit, scalar + `
static int32_t inline_position=0;
int32_t inline_produce(int32_t index){if(index!=++inline_position)return -1000;return index==1?20:24;}
int32_t inline_finish(int32_t result){return inline_position==2&&result==42?42:1;}
`
}

func TestCrossPackageInlineLocalsWindows(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows amd64 MSVC")
	}
	unit, support := inlineLocalsNativeUnit(t, false)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []cdriver.ConsumerMode{cdriver.C11ConsumerMode, cdriver.C17ConsumerMode} {
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(string(mode)+"/"+profile.Name, func(t *testing.T) {
				a, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(tc, profile, mode, unit.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				if len(a.Objects) != 2 {
					t.Fatal("separate generated caller and observer objects missing")
				}
				if _, err := pecoff.InspectExecutable(a.Bytes); err != nil {
					t.Fatal(err)
				}
				if _, err := pecoff.InspectAMD64Unwind(a.Bytes); err != nil {
					t.Fatal(err)
				}
				out, err := cdriver.ExecuteWindowsMSVCABIProbe(a)
				if err != nil || out.ExitCode != 42 || out.Stdout != "" || out.Stderr != "" {
					t.Fatalf("inline crossing: %+v %v", out, err)
				}
				t.Logf("MSVC %s: warm cross-package inline body, checked arithmetic and once-only ordered arguments; exit42 empty streams", tc.Version)
			})
		}
	}
}
