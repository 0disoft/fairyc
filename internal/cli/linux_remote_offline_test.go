package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/packagesource"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Windows runs the actual Fairy offline owner over an independent Linux ABI
// namespace model. Go fetch operates on real Windows files to supply expected
// outcomes and complete final trees. This does not stand in for Linux syscalls.
func TestLinuxRemoteOfflinePolicy(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows policy oracle; actual Linux filesystem replay stays separate")
	}
	fixtures := linuxOfflineFixtures(t)
	fixtures = append(fixtures, linuxOfflineFaults(t, fixtures[1])...)
	var cases strings.Builder
	cases.WriteString("static void offline_fixture(uint32_t index){switch(index){\n")
	for index, fixture := range fixtures {
		prefix := "oracle/"
		if fixture.fault != 0 {
			prefix = "expectation/"
		}
		t.Run(prefix+fixture.name, func(t *testing.T) {
			if fixture.fault == 0 {
				root := t.TempDir()
				for _, name := range linuxOfflineNames(fixture.initial) {
					path := filepath.Join(root, filepath.FromSlash(name))
					if fixture.initial[name] == nil {
						if err := os.MkdirAll(path, 0700); err != nil {
							t.Fatal(err)
						}
					} else {
						if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, fixture.initial[name], 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				requests := 0
				fetcher := packagesource.Fetcher{Client: &http.Client{Transport: remoteOfflineTransport(func(*http.Request) (*http.Response, error) {
					requests++
					return nil, fmt.Errorf("offline transport called")
				})}}
				result, err := fetcher.Fetch(t.Context(), root, true)
				if requests != 0 {
					t.Fatalf("offline made %d requests", requests)
				}
				fixture.ok = err == nil
				if fixture.ok {
					fixture.output = fmt.Sprintf("ok %d %d %d %d\n", result.Packages, result.CacheHits, result.Installed, result.Reused)
				}
				fixture.final = linuxOfflineTree(t, root)
			}
			fmt.Fprintf(&cases, "case %d:\n", index)
			for _, name := range linuxOfflineNames(fixture.initial) {
				linuxOfflineSeed(&cases, "offline_seed", name, fixture.initial[name], fixture.kinds[name])
			}
			fmt.Fprintf(&cases, "offline_fault=%d;offline_alter();\n", fixture.fault)
			cases.WriteString("break;\n")
		})
	}
	cases.WriteString("default:assert(0);}}\nstatic void offline_expected(uint32_t index){switch(index){\n")
	for index, fixture := range fixtures {
		fmt.Fprintf(&cases, "case %d:\n", index)
		for _, name := range linuxOfflineNames(fixture.final) {
			linuxOfflineSeed(&cases, "offline_expect", name, fixture.final[name], fixture.kinds[name])
		}
		fmt.Fprintf(&cases, "offline_expect_count(%d);break;\n", len(fixture.final))
	}
	cases.WriteString("default:assert(0);}}\n")
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_remote_offline")
	writeTestSource(t, root, "src/main.fy", linuxOfflineEntry)
	build, support := linuxOfflineBuild(t, root, target.X8664WindowsMSVC)
	// Enlarge only model storage/ledgers. The original filesystem model's fault
	// behavior and native ABI stay shared with its independent focused controls.
	model := strings.NewReplacer("uint8_t data[256]", "uint8_t data[65536]", "size+count<=256", "size+count<=65536",
		"remote_handles[256]", "remote_handles[4096]", "remote_handle_count<256", "remote_handle_count<4096",
		"remote_cursors[64]", "remote_cursors[2048]", "remote_cursor_count<64", "remote_cursor_count<2048").Replace(linuxRemoteFileSystemStub)
	for _, name := range []string{"create", "publish", "erase"} {
		model = strings.Replace(model, "uint32_t fairyc_private_linux_package_directory_"+name+"_v1(", "static uint32_t offline_base_"+name+"(", 1)
	}
	support += "\n" + model + linuxOfflineModel + linuxOfflineFaultModel + cases.String() + linuxOfflineModelEntry
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "offline-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for index, fixture := range fixtures {
				t.Run(fixture.name, func(t *testing.T) {
					cmd := exec.CommandContext(t.Context(), path)
					cmd.Env = append(os.Environ(), "PATH=", fmt.Sprintf("FAIRYC_OFFLINE_CASE=%d", index))
					output, err := cmd.CombinedOutput()
					if fixture.output != "" {
						if fixture.fault != 0 {
							e, ok := err.(*exec.ExitError)
							if !ok || e.ExitCode() != 7 {
								t.Fatalf("fault outcome: %v %q", err, output)
							}
						} else if err != nil {
							t.Fatalf("successful outcome: %v %q", err, output)
						}
						if string(output) != fixture.output {
							t.Fatalf("outcome: %v %q want %q", err, output, fixture.output)
						}
					} else {
						e, ok := err.(*exec.ExitError)
						var stage, code, cleanup uint32
						if !ok || e.ExitCode() != 7 {
							t.Fatalf("failure outcome: %v %q", err, output)
						}
						if _, err := fmt.Sscanf(string(output), "error %d %d %d\n", &stage, &code, &cleanup); err != nil || stage == 0 || code == 0 || cleanup != 0 {
							t.Fatalf("failure receipt: %v %q", err, output)
						}
					}
				})
			}
		})
	}
}

type linuxOfflineFixture struct {
	name           string
	initial, final map[string][]byte // nil denotes directory, []byte{} empty file
	ok             bool
	output         string
	fault          uint32
	kinds          map[string]uint32
}

func linuxOfflineFixtures(t *testing.T) []*linuxOfflineFixture {
	t.Helper()
	var fixtures []*linuxOfflineFixture
	add := func(name string, archive []byte, alter func(map[string][]byte, string)) {
		t.Helper()
		sum := sha256.Sum256(archive)
		digest := hex.EncodeToString(sum[:])
		plan, err := json.Marshal(packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: "https://packages.invalid/math.zip", SHA256: digest}}})
		if err != nil {
			t.Fatal(err)
		}
		cache := "build/package-sources/" + digest + ".zip"
		tree := map[string][]byte{"fairy.toml": []byte("schema='fairyc.manifest/v1'\n[package]\nuid='example.app'\nname='app'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n[workspace]\nmembers=['vendor/math']\n[dependencies.math]\npath='vendor/math'\nversion='1.0.0'\nfeatures=[]\n"), "fairy.sources.json": plan, "fairy.lock": []byte("unchanged lock\n"), "build": nil, "build/package-sources": nil, cache: bytes.Clone(archive)}
		if alter != nil {
			alter(tree, cache)
		}
		fixtures = append(fixtures, &linuxOfflineFixture{name: name, initial: tree})
	}
	library := func(tree map[string][]byte) {
		tree["vendor"] = nil
		tree["vendor/math"] = nil
		tree["vendor/math/src"] = nil
		tree["vendor/math/fairy.toml"] = []byte(remoteLibraryManifest)
		tree["vendor/math/src/math.fy"] = []byte("module math;pub fn i32 answer(){return 42;}\n")
		tree["vendor/math/.review"] = []byte{0, 255, 1, 2}
	}
	stored, deflated := remoteArchiveValid(t, zip.Store), remoteArchiveValid(t, zip.Deflate)
	add("stored", stored, nil)
	add("deflated", deflated, nil)
	add("zip64", remoteArchiveZIP64(t, deflated), nil)
	add("exact-reuse", deflated, func(tree map[string][]byte, _ string) { library(tree) })
	add("extra-empty-directory", deflated, func(tree map[string][]byte, _ string) { library(tree); tree["vendor/math/empty"] = nil })
	add("cache-miss", deflated, func(tree map[string][]byte, cache string) { delete(tree, cache) })
	add("cache-directory-miss", deflated, func(tree map[string][]byte, cache string) { delete(tree, cache); delete(tree, "build/package-sources") })
	add("corrupt-cache", deflated, func(tree map[string][]byte, cache string) { tree[cache] = []byte("corrupt") })
	add("corrupt-warm-cache", deflated, func(tree map[string][]byte, cache string) { library(tree); tree[cache] = []byte("corrupt") })
	add("cache-case-alias", deflated, func(tree map[string][]byte, cache string) {
		tree[strings.TrimSuffix(cache, ".zip")+".ZIP"] = tree[cache]
		delete(tree, cache)
	})
	add("member-case-alias", deflated, func(tree map[string][]byte, _ string) { tree["vendor"] = nil; tree["vendor/Math"] = nil })
	add("changed-existing", deflated, func(tree map[string][]byte, _ string) {
		library(tree)
		tree["vendor/math/src/math.fy"] = []byte("changed")
	})
	add("extra-file", deflated, func(tree map[string][]byte, _ string) { library(tree); tree["vendor/math/extra"] = []byte("preserved") })
	add("incomplete-existing", deflated, func(tree map[string][]byte, _ string) {
		tree["vendor"] = nil
		tree["vendor/math"] = nil
		tree["vendor/math/fairy.toml"] = []byte(remoteLibraryManifest)
	})
	add("destination-file", deflated, func(tree map[string][]byte, _ string) {
		tree["vendor"] = nil
		tree["vendor/math"] = []byte("preserved")
	})
	add("lock-held", deflated, func(tree map[string][]byte, _ string) {
		tree[".fairyc-fetch.lock"] = nil
		tree[".fairyc-fetch.lock/keep"] = []byte("foreign lock")
	})
	// Go uses its own random stage names; this authored digest-stage control is
	// independent and checked separately below rather than falsely called parity.
	add("invalid-plan", deflated, func(tree map[string][]byte, _ string) { tree["fairy.sources.json"] = []byte("{}") })
	add("invalid-manifest", deflated, func(tree map[string][]byte, _ string) { tree["fairy.toml"] = []byte("broken") })
	add("unbound-member", deflated, func(tree map[string][]byte, _ string) {
		tree["fairy.toml"] = bytes.ReplaceAll(tree["fairy.toml"], []byte("members=['vendor/math']"), []byte("members=[]"))
	})
	for _, name := range []string{"../escape", ".git/config", "src/con"} {
		add("bad-archive/"+name, remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(remoteLibraryManifest)}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;")}, remoteArchiveEntry{name: name, data: []byte("bad")}), nil)
	}
	add("identity", remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(strings.Replace(remoteLibraryManifest, "example.math", "example.other", 1))}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;")}), nil)
	second := remoteArchiveBytes(t, zip.Deflate, remoteArchiveEntry{name: "fairy.toml", data: []byte(strings.Replace(remoteLibraryManifest, "example.math", "example.answer", 1))}, remoteArchiveEntry{name: "src/math.fy", data: []byte("module math;")})
	for _, partial := range []bool{false, true} {
		name := "two-packages"
		if partial {
			name = "partial-success"
		}
		add(name, deflated, func(tree map[string][]byte, cache string) {
			tree["fairy.toml"] = bytes.ReplaceAll(tree["fairy.toml"], []byte("members=['vendor/math']"), []byte("members=['vendor/answer','vendor/math']"))
			var plan packagesource.Plan
			if err := json.Unmarshal(tree["fairy.sources.json"], &plan); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(second)
			digest := hex.EncodeToString(sum[:])
			plan.Packages = append(plan.Packages, packagesource.Pin{Path: "vendor/answer", PackageUID: "example.answer", Version: "1.0.0", URL: "https://packages.invalid/answer.zip", SHA256: digest})
			data, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			tree["fairy.sources.json"] = data
			tree["build/package-sources/"+digest+".zip"] = second
			if partial {
				tree[cache] = []byte("corrupt later package")
			}
		})
	}
	return fixtures
}

func linuxOfflineNames(tree map[string][]byte) []string {
	names := make([]string, 0, len(tree))
	for name := range tree {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
func linuxOfflineTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	tree := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		if entry.IsDir() {
			tree[name] = nil
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(data) == 0 {
			data = []byte{}
		}
		tree[name] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}
func linuxOfflineSeed(out *strings.Builder, function, name string, data []byte, kind uint32) {
	if kind == 0 {
		kind = 32768
		if data == nil {
			kind = 16384
		}
	}
	fmt.Fprintf(out, "%s(%q,%d,%du,", function, name, len(data), kind)
	if data == nil {
		out.WriteString("NULL);\n")
		return
	}
	out.WriteString("(const uint8_t[]){")
	if len(data) == 0 {
		out.WriteString("0")
	}
	for index, value := range data {
		if index > 0 {
			out.WriteByte(',')
		}
		fmt.Fprintf(out, "%d", value)
	}
	out.WriteString("});\n")
}
func linuxOfflineBuild(t *testing.T, root string, selected target.ID) (cbackend.BuildUnit, string) {
	t.Helper()
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "Linux offline policy", &diagnostics)
	if code != ExitSuccess {
		t.Fatalf("lower: %d %s", code, &diagnostics)
	}
	var build cbackend.BuildUnit
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.EmitBuildContext(t.Context(), lowered.program, lowered.entryName, profile, true, 128<<10)
		if err != nil {
			t.Fatal(err)
		}
		for _, symbol := range unit.Unit.ForeignSymbols {
			if strings.Contains(strings.ToLower(symbol), "http") || strings.HasPrefix(symbol, "fairyc_private_windows_") {
				t.Fatalf("offline host/transport leak: %s", symbol)
			}
		}
		if _, err := candidatehost.LinuxSource(strings.Join(unit.Unit.ForeignSymbols, "(\n") + "("); err != nil {
			t.Fatal(err)
		}
		if profile.Target == selected {
			build = unit
		}
	}
	if build.Header == "" {
		t.Fatal("missing partitioned selected-target emission")
	}
	for index, part := range build.Sources {
		build.Sources[index] = strings.Replace(part, "#include \"fy_bootstrap.h\"", "#include \"fy_project.h\"", 1)
	}
	var symbols []string
	for _, symbol := range build.Unit.ForeignSymbols {
		if !strings.HasPrefix(symbol, "fairyc_private_linux_") && !strings.HasPrefix(symbol, "offline_probe_") {
			symbols = append(symbols, symbol)
		}
	}
	support, err := nativesupport.Source(selected, symbols, nativesupport.Options{SDKStorage: true})
	if err != nil {
		t.Fatal(err)
	}
	if selected == target.X8664LinuxGNU {
		host, err := candidatehost.LinuxSource(strings.Join(build.Unit.ForeignSymbols, "(\n") + "(")
		if err != nil {
			t.Fatal(err)
		}
		support = host + "\n" + support
	}
	return build, support
}

const linuxOfflineEntry = `module main;
import linux_remote_offline as offline;import std::io as io;
@link_name("offline_probe_setup") unsafe extern C fn void setup();
@link_name("offline_probe_finish") unsafe extern C fn void finish();
@effects(filesystem,clock,console) fn i32 main(){
    unsafe{setup();}let result=offline::restore(b"/work");unsafe{finish();}
    switch(result){case .ok(p):{discard io::write_bytes(b"ok ");discard io::write_usize(p.packages);discard io::write_bytes(b" ");discard io::write_usize(p.cache_hits);discard io::write_bytes(b" ");discard io::write_usize(p.installed);discard io::write_bytes(b" ");discard io::write_usize(p.reused);discard io::write_bytes(b"\n");return 0;}
    case .err(p):{discard io::write_bytes(b"error ");discard io::write_u32(p.stage);discard io::write_bytes(b" ");discard io::write_u32(p.code);discard io::write_bytes(b" ");discard io::write_u32(p.cleanup_code);discard io::write_bytes(b"\n");return 7;}};return 7;
}`

const linuxOfflineModel = `
static uint32_t offline_index,offline_fault,offline_calls;
static int offline_path(const char *name,int create){
    int parent=0;const char *start=name;const char *end;
    while((end=strchr(start,'/'))!=NULL){char leaf[256];size_t length=(size_t)(end-start);assert(length>0 && length<sizeof leaf);memcpy(leaf,start,length);leaf[length]=0;
        int next=remote_find(parent,leaf);if(next<0 && create){next=remote_add(parent,leaf,16384);}assert(next>=0 && remote_nodes[next].kind==16384);parent=next;start=end+1;}
    int node=remote_find(parent,start);if(node<0 && create){node=remote_add(parent,start,32768);}assert(node>=0);return node;
}
static void offline_seed(const char *name,size_t size,uint32_t kind,const uint8_t *data){int node=offline_path(name,1);assert(size<=sizeof remote_nodes[node].data);remote_nodes[node].kind=kind;remote_nodes[node].size=size;if(size){memcpy(remote_nodes[node].data,data,size);}}
static void offline_expect(const char *name,size_t size,uint32_t kind,const uint8_t *data){int node=offline_path(name,0);assert(remote_nodes[node].kind==kind && remote_nodes[node].size==size);if(size){assert(memcmp(remote_nodes[node].data,data,size)==0);}}
static void offline_expect_count(size_t wanted){size_t count=0;for(size_t i=1;i<remote_node_count;++i){if(remote_nodes[i].active && remote_nodes[i].parent>=0){++count;}}assert(count==wanted && remote_live==0);}
uint32_t fairyc_private_linux_monotonic_ms_v1(size_t *output){
    ++offline_calls;*output=1000;
    if(offline_fault==9 || (offline_fault==10 && offline_calls>1)){*output=0;return 5;}
    if((offline_fault==11 && offline_calls>2) || (offline_fault==12 && remote_publications>=3)){*output=121000;}return 0;
}
`
const linuxOfflineModelEntry = `
void offline_probe_setup(void){const char *value=getenv("FAIRYC_OFFLINE_CASE");assert(value!=NULL);offline_index=(uint32_t)strtoul(value,NULL,10);offline_calls=0;remote_fs_probe_mode(0);offline_fixture(offline_index);}
void offline_probe_finish(void){offline_expected(offline_index);}
`
