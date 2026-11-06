package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
	"github.com/0disoft/fairyc/internal/target"
)

type linuxOnlineFixture struct {
	*linuxOfflineFixture
	replies             map[string][]byte
	mode, calls, closes uint32
}

// Actual Fairy online orchestration, hashing, ZIP validation, immutable cache
// and publication run over independent filesystem/HTTPS ABI models. Go fetch
// supplies reference outcomes and complete real-filesystem final namespaces.
func TestLinuxRemoteOnlinePolicy(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows policy oracle; Linux syscall/TLS acceptance remains batched")
	}
	fixtures := linuxOnlineFixtures(t)
	var cases strings.Builder
	cases.WriteString("static void online_fixture(uint32_t index){switch(index){\n")
	for _, f := range fixtures {
		passed := t.Run(f.name, func(t *testing.T) {
			root := linuxOfflineMaterialize(t, f.initial)
			calls := uint32(0)
			referenceReplies := make(map[string][]byte)
			for address, payload := range f.replies {
				parsed, err := url.Parse(address)
				if err != nil {
					t.Fatal(err)
				}
				referenceReplies[parsed.String()] = payload
			}
			fetcher := packagesource.Fetcher{Client: &http.Client{Transport: remoteOfflineTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				payload, ok := referenceReplies[request.URL.String()]
				if !ok {
					t.Fatalf("unexpected Go request %s", request.URL)
				}
				if request.Method != "GET" || request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" {
					t.Fatal("unexpected request method or ambient credentials")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(payload)), ContentLength: int64(len(payload)), Request: request}, nil
			})}}
			result, err := fetcher.Fetch(t.Context(), root, false)
			f.ok, f.calls, f.closes = err == nil, calls, calls
			if f.ok {
				f.output = fmt.Sprintf("ok %d %d %d %d %d\n", result.Packages, result.Downloaded, result.CacheHits, result.Installed, result.Reused)
			}
			f.final = linuxOfflineTree(t, root)
		})
		if !passed {
			t.FailNow()
		}
	}
	fixtures = append(fixtures, linuxOnlineFaults(t, fixtures)...)
	for index, f := range fixtures {
		fmt.Fprintf(&cases, "case %d:\n", index)
		for _, name := range linuxOfflineNames(f.initial) {
			linuxOfflineSeed(&cases, "offline_seed", name, f.initial[name], f.kinds[name])
		}
		var urls []string
		for url := range f.replies {
			urls = append(urls, url)
		}
		sort.Strings(urls)
		for _, url := range urls {
			var reply, address strings.Builder
			linuxOfflineSeed(&reply, "online_reply", url, f.replies[url], 0)
			address.WriteByte('"')
			for _, value := range []byte(url) {
				fmt.Fprintf(&address, "\\%03o", value)
			}
			address.WriteByte('"')
			cases.WriteString(strings.Replace(reply.String(), fmt.Sprintf("%q", url), address.String(), 1))
		}
		fmt.Fprintf(&cases, "offline_fault=%d;online_mode=%d;offline_alter();break;\n", f.fault, f.mode)
	}
	cases.WriteString("default:assert(0);}}\nstatic void online_expected(uint32_t index){switch(index){\n")
	for index, f := range fixtures {
		fmt.Fprintf(&cases, "case %d:\n", index)
		for _, name := range linuxOfflineNames(f.final) {
			linuxOfflineSeed(&cases, "offline_expect", name, f.final[name], f.kinds[name])
		}
		fmt.Fprintf(&cases, "offline_expect_count(%d);assert(online_live==0 && online_opens==%d && online_closes==%d);", len(f.final), f.calls, f.closes)
		if f.mode == 11 || f.mode == 12 || f.mode == 13 || f.mode == 14 || f.mode == 15 {
			fmt.Fprintf(&cases, "assert(remote_anonymous==%d && online_cache_publications==%d);", map[uint32]int{11: 1, 12: 1, 13: 1, 14: 4, 15: 1}[f.mode], map[uint32]int{11: 0, 12: 0, 13: 1, 14: 1, 15: 1}[f.mode])
		}
		cases.WriteString("break;\n")
	}
	cases.WriteString("default:assert(0);}}\n")
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_remote_online")
	writeTestSource(t, root, "src/main.fy", linuxOnlineEntry)
	build, support := linuxRemoteBuild(t, root, target.X8664WindowsMSVC, true)
	model := strings.NewReplacer("uint8_t data[256]", "uint8_t data[65536]", "size+count<=256", "size+count<=65536",
		"remote_handles[256]", "remote_handles[4096]", "remote_handle_count<256", "remote_handle_count<4096",
		"remote_cursors[64]", "remote_cursors[2048]", "remote_cursor_count<64", "remote_cursor_count<2048").Replace(linuxRemoteFileSystemStub)
	for _, name := range []string{"create", "publish", "erase"} {
		model = strings.Replace(model, "uint32_t fairyc_private_linux_package_directory_"+name+"_v1(", "static uint32_t offline_base_"+name+"(", 1)
	}
	model = strings.Replace(model, "uint32_t fairyc_private_linux_root_file_publish_new_v1(", "static uint32_t online_base_file_publish(", 1)
	base := strings.Split(linuxOfflineModel, "uint32_t fairyc_private_linux_monotonic_ms_v1(")[0]
	support += "\n" + model + base + linuxOfflineFaultModel + linuxOnlineModel + cases.String() + linuxOnlineModelEntry
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
			path := filepath.Join(root, "online-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for index, f := range fixtures {
				t.Run(f.name, func(t *testing.T) {
					cmd := exec.CommandContext(t.Context(), path)
					cmd.Env = append(os.Environ(), "PATH=", fmt.Sprintf("FAIRYC_ONLINE_CASE=%d", index))
					out, err := cmd.CombinedOutput()
					if f.ok {
						if err != nil || string(out) != f.output {
							t.Fatalf("success: %v %q want %q", err, out, f.output)
						}
						return
					}
					if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 7 {
						t.Fatalf("failure: %v %q", err, out)
					}
					if f.output != "" {
						if string(out) != f.output {
							t.Fatalf("receipt %q want %q", out, f.output)
						}
					} else {
						var stage, code, native, cleanup uint32
						if _, err := fmt.Sscanf(string(out), "error %d %d %d %d\n", &stage, &code, &native, &cleanup); err != nil || stage == 0 || code == 0 || cleanup != 0 {
							t.Fatalf("failure receipt: %v %q", err, out)
						}
					}
				})
			}
		})
	}
}

func linuxOnlineFixtures(t *testing.T) []*linuxOnlineFixture {
	t.Helper()
	base := linuxOfflineFixtures(t)
	corpus := make(map[string][]byte)
	for _, f := range base {
		for name, data := range f.initial {
			if strings.HasSuffix(name, ".zip") {
				sum := sha256.Sum256(data)
				corpus[hex.EncodeToString(sum[:])] = data
			}
		}
	}
	var result []*linuxOnlineFixture
	for _, f := range base {
		initial := linuxOnlineClone(f.initial)
		variants := []*linuxOfflineFixture{{name: "cache/" + f.name, initial: initial}}
		cold := linuxOnlineClone(initial)
		removed := false
		for name, data := range cold {
			sum := sha256.Sum256(data)
			if name == "build/package-sources/"+hex.EncodeToString(sum[:])+".zip" {
				delete(cold, name)
				removed = true
			}
		}
		if removed {
			variants = append(variants, &linuxOfflineFixture{name: "download/" + f.name, initial: cold})
		}
		for _, variant := range variants {
			replies := make(map[string][]byte)
			var plan packagesource.Plan
			if json.Unmarshal(initial["fairy.sources.json"], &plan) == nil {
				for _, pin := range plan.Packages {
					if data, ok := corpus[pin.SHA256]; ok {
						replies[pin.URL] = data
					}
				}
			}
			result = append(result, &linuxOnlineFixture{linuxOfflineFixture: variant, replies: replies})
		}
	}
	// Inputs for byte-exact URL preservation and fresh cache-parent creation.
	for _, url := range []string{"https://packages.invalid/a%2Fb.zip", "https://packages.invalid/한글.zip"} {
		f := linuxOnlineCold(t, remoteArchiveValid(t, zip.Deflate), url)
		f.name = "download/url/" + url
		result = append(result, f)
	}
	f := linuxOnlineCold(t, remoteArchiveValid(t, zip.Deflate), "https://packages.invalid/math.zip")
	f.name = "download/create-cache-parents"
	delete(f.initial, "build/package-sources")
	delete(f.initial, "build")
	return append(result, f)
}

func linuxOnlineClone(tree map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(tree))
	for name, data := range tree {
		result[name] = bytes.Clone(data)
	}
	return result
}

func linuxOnlineCold(t *testing.T, payload []byte, url string) *linuxOnlineFixture {
	t.Helper()
	base := linuxOfflineFixtures(t)[1]
	tree := linuxOnlineClone(base.initial)
	for name := range tree {
		if strings.HasSuffix(name, ".zip") {
			delete(tree, name)
		}
	}
	sum := sha256.Sum256(payload)
	data, err := json.Marshal(packagesource.Plan{Schema: packagesource.Schema, Packages: []packagesource.Pin{{Path: "vendor/math", PackageUID: "example.math", Version: "1.0.0", URL: url, SHA256: hex.EncodeToString(sum[:])}}})
	if err != nil {
		t.Fatal(err)
	}
	tree["fairy.sources.json"] = data
	return &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: tree}, replies: map[string][]byte{url: payload}}
}

const linuxOnlineEntry = `module main;
import linux_remote_online as online;import std::io as io;
@link_name("online_probe_setup") unsafe extern C fn void setup();
@link_name("online_probe_finish") unsafe extern C fn void finish();
@effects(filesystem,network,clock,console) fn i32 main(){
    assert(online::archive_limit(0)==16777216 && online::archive_limit(50331647)==16777216 && online::archive_limit(50331648)==16777216);
    assert(online::archive_limit(50331649)==16777215 && online::archive_limit(67108863)==1 && online::archive_limit(67108864)==0 && online::archive_limit(18446744073709551615)==0);
    assert(online::request_limit(0)==0 && online::request_limit(1)==1 && online::request_limit(29999)==29999 && online::request_limit(30000)==30000 && online::request_limit(30001)==30000);
    unsafe{setup();}let result=online::fetch(b"/work");unsafe{finish();}
    switch(result){case .ok(p):{discard io::write_bytes(b"ok ");discard io::write_usize(p.packages);discard io::write_bytes(b" ");discard io::write_usize(p.downloaded);discard io::write_bytes(b" ");discard io::write_usize(p.cache_hits);discard io::write_bytes(b" ");discard io::write_usize(p.installed);discard io::write_bytes(b" ");discard io::write_usize(p.reused);discard io::write_bytes(b"\n");return 0;}
    case .err(p):{discard io::write_bytes(b"error ");discard io::write_u32(p.stage);discard io::write_bytes(b" ");discard io::write_u32(p.code);discard io::write_bytes(b" ");discard io::write_u32(p.native_code);discard io::write_bytes(b" ");discard io::write_u32(p.cleanup_code);discard io::write_bytes(b"\n");return 7;}};return 7;
}`
