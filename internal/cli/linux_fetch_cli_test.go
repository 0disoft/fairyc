package cli

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/packagesource"
	"github.com/0disoft/fairyc/internal/target"
)

type linuxFetchCase struct {
	name, out, diagnostic, broken string
	args                          []string
	code                          int
	noIO                          bool
	fixture                       *linuxOnlineFixture
}

func TestLinuxFetchCommand(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows command/namespace/HTTPS boundary oracle; native Linux replay stays batched")
	}
	cases := linuxFetchCases(t)
	var fixtures []*linuxOnlineFixture
	for _, c := range cases {
		fixtures = append(fixtures, c.fixture)
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "linux_fetch_cli")
	writeTestSource(t, root, "src/main.fy", linuxFetchEntry)
	build, support := linuxRemoteBuild(t, root, target.X8664WindowsMSVC, true)
	model := linuxOnlineModelSupport(fixtures)
	needle := `size==6 && strcmp((const char *)path,"/work")==0`
	if strings.Count(model, needle) != 1 {
		t.Fatal("model workspace boundary changed")
	}
	model = strings.Replace(model, needle, `size==2 && strcmp((const char *)path,".")==0`, 1)
	model = strings.Replace(model, `void online_probe_finish(void){online_expected(offline_index);}`, `void online_probe_finish(void){online_expected(offline_index);const char *no_io=getenv("FAIRYC_FETCH_NO_IO");if(no_io!=NULL && strcmp(no_io,"1")==0){assert(offline_calls==0 && remote_handle_count==1 && remote_cursor_count==0 && online_opens==0);}}`, 1)
	support += "\n" + model
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
			path := filepath.Join(root, "fetch-"+profile.Name+".exe")
			if err := os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			for index, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					t.Setenv("FAIRYC_ONLINE_CASE", strconv.Itoa(index))
					t.Setenv("FAIRYC_FETCH_NO_IO", map[bool]string{true: "1", false: "0"}[c.noIO])
					code, out, diagnostic := fetchCLIProcess(t, path, t.TempDir(), c.args, c.broken)
					if code != c.code || out != c.out || diagnostic != c.diagnostic {
						t.Fatalf("CLI: (%d,%q,%q) want (%d,%q,%q)", code, out, diagnostic, c.code, c.out, c.diagnostic)
					}
				})
			}
		})
	}
}

func linuxFetchCases(t *testing.T) []linuxFetchCase {
	t.Helper()
	linuxHelp := linuxFetchHelp(t)
	_, goHelp, _ := runForTest([]string{"--help"}, t.TempDir())
	var cases []linuxFetchCase
	empty := func() *linuxOnlineFixture {
		return &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: map[string][]byte{}, final: map[string][]byte{}}}
	}
	for index, args := range [][]string{
		{"experimental"}, {"experimental", "--help"}, {"experimental", "help"},
		{"experimental", "fetch-packages", "--help"}, {"experimental", "fetch-packages", "-h"}, {"experimental", "fetch-packages", "help"},
		{"experimental", "unknown\ncommand"}, {"experimental", "unknown", "--help"},
		{"experimental", "fetch-packages", "--bad"}, {"experimental", "fetch-packages", "--offline", "--offline"},
		{"experimental", "fetch-packages", "--offline", "extra"}, {"experimental", "fetch-packages", "--"}, {"experimental", "fetch-packages", "--offline=true"},
		{"experimental", strings.Repeat("x", 4095)}, {"experimental", strings.Repeat("x", 4096)},
	} {
		code, out, diagnostic := runForTest(args, t.TempDir())
		cases = append(cases, linuxFetchCase{name: fmt.Sprintf("syntax-%02d", index), args: args, code: code,
			out: strings.ReplaceAll(out, goHelp, linuxHelp), diagnostic: strings.ReplaceAll(diagnostic, goHelp, linuxHelp), noIO: true, fixture: empty()})
	}
	for _, c := range []struct{ name, arg, message string }{
		{"command-over-limit", strings.Repeat("x", 4097), "argument exceeds the supported byte limit"},
		{"option-over-limit", strings.Repeat("x", 4097), "experimental fetch-packages accepts only optional --offline"},
	} {
		args := []string{"experimental", c.arg}
		if c.name == "option-over-limit" {
			args = []string{"experimental", "fetch-packages", c.arg}
		}
		cases = append(cases, linuxFetchCase{name: c.name, args: args, code: 2, diagnostic: "fairyc: " + c.message + "\n\n" + linuxHelp, noIO: true, fixture: empty()})
	}
	for _, args := range [][]string{{"experimental", "fetch-packages"}, {"experimental", "fetch-packages", "--offline"}} {
		cases = append(cases, linuxFetchCase{name: fmt.Sprintf("missing-plan-%d", len(args)), args: args, code: 7, diagnostic: "package fetch failed: cannot read source plan or root manifest\n", fixture: empty()})
	}
	base := linuxOnlineCold(t, remoteArchiveValid(t, zip.Deflate), "https://packages.invalid/math.zip")
	base.name = "download/deflated"
	linuxFetchReference(t, base, false)
	for _, offline := range []bool{false, true} {
		f := &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: linuxOnlineClone(base.final)}, replies: base.replies}
		linuxFetchReference(t, f, offline)
		args := []string{"experimental", "fetch-packages"}
		if offline {
			args = append(args, "--offline")
		}
		cases = append(cases, linuxFetchCase{name: fmt.Sprintf("cached-reuse-offline-%t", offline), args: args, fixture: f, out: linuxFetchSuccess(t, f)})
	}
	cases = append(cases, linuxFetchCase{name: "cold-online", args: []string{"experimental", "fetch-packages"}, fixture: base, out: linuxFetchSuccess(t, base)})
	miss := &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: linuxOnlineClone(base.initial), final: linuxOnlineClone(base.initial)}}
	cases = append(cases, linuxFetchCase{name: "offline-miss-zero-network", args: []string{"experimental", "fetch-packages", "--offline"}, fixture: miss, code: 7, diagnostic: "package fetch failed: offline source cache miss\n"})
	fresh := &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: linuxOnlineClone(base.initial)}, replies: base.replies}
	for name, data := range base.final {
		if strings.HasSuffix(name, ".zip") {
			fresh.initial[name] = bytes.Clone(data)
		}
	}
	linuxFetchReference(t, fresh, true)
	cases = append(cases, linuxFetchCase{name: "offline-fresh-install", args: []string{"experimental", "fetch-packages", "--offline"}, fixture: fresh, out: linuxFetchSuccess(t, fresh)})
	badPlan := &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: linuxOnlineClone(base.initial)}}
	badPlan.initial[packagesource.FileName] = []byte("{}")
	badPlan.final = linuxOnlineClone(badPlan.initial)
	cases = append(cases, linuxFetchCase{name: "invalid-plan", args: []string{"experimental", "fetch-packages"}, fixture: badPlan, code: 7, diagnostic: "package fetch failed: invalid source plan, manifest or workspace binding\n"})
	corrupt := &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: linuxOnlineClone(base.final)}}
	for name := range corrupt.initial {
		if strings.HasSuffix(name, ".zip") {
			corrupt.initial[name] = []byte("corrupt")
		}
	}
	corrupt.final = linuxOnlineClone(corrupt.initial)
	cases = append(cases, linuxFetchCase{name: "corrupt-warm-cache-zero-network", args: []string{"experimental", "fetch-packages"}, fixture: corrupt, code: 7, diagnostic: "package fetch failed: source archive SHA-256 mismatch\n"})
	for _, size := range []int{21, 22, 23} {
		badZip := linuxOnlineCold(t, bytes.Repeat([]byte{'x'}, size), "https://packages.invalid/math.zip")
		badZip.final, badZip.calls, badZip.closes = linuxOnlineClone(badZip.initial), 1, 1
		message := "package fetch failed: invalid source ZIP or package identity\n"
		if size < 22 {
			message = "package fetch failed: source ZIP file, byte or work limit exceeded\n"
		}
		cases = append(cases, linuxFetchCase{name: fmt.Sprintf("invalid-ZIP-bytes-%d-before-cache-publication", size), args: []string{"experimental", "fetch-packages"}, fixture: badZip, code: 7, diagnostic: message})
	}
	conflict := &linuxOnlineFixture{linuxOfflineFixture: &linuxOfflineFixture{initial: linuxOnlineClone(base.initial)}, replies: base.replies, calls: 1, closes: 1}
	conflict.initial["vendor"], conflict.initial["vendor/math"] = nil, []byte("keep existing file")
	conflict.final = linuxOnlineClone(conflict.initial)
	for name, data := range base.final {
		if strings.HasSuffix(name, ".zip") {
			conflict.final[name] = bytes.Clone(data)
		}
	}
	cases = append(cases, linuxFetchCase{name: "existing-package-conflict-retains-cache", args: []string{"experimental", "fetch-packages"}, fixture: conflict, code: 7, diagnostic: "package fetch failed: existing package conflicts with pinned archive\n"})
	for _, f := range linuxOnlineFaults(t, []*linuxOnlineFixture{base}) {
		c := linuxFetchCase{name: f.name, args: []string{"experimental", "fetch-packages"}, fixture: f}
		if f.ok {
			c.out = linuxFetchSuccess(t, f)
		} else {
			c.code = 7
			c.diagnostic = linuxFetchDiagnostic(t, f.output)
		}
		cases = append(cases, c)
	}
	for _, c := range []linuxFetchCase{
		{name: "help-stdout-error", args: []string{"experimental", "--help"}, code: 4, diagnostic: "fairyc: write output failed\n", broken: "stdout", noIO: true, fixture: empty()},
		{name: "syntax-stderr-error", args: []string{"experimental", "fetch-packages", "--bad"}, code: 4, broken: "stderr", noIO: true, fixture: empty()},
		{name: "failure-stderr-error", args: []string{"experimental", "fetch-packages", "--offline"}, code: 4, broken: "stderr", fixture: empty()},
		{name: "completed-fetch-stdout-error", args: []string{"experimental", "fetch-packages"}, code: 4, diagnostic: "fairyc: write output failed\n", broken: "stdout", fixture: base},
	} {
		cases = append(cases, c)
	}
	return cases
}

func linuxFetchReference(t *testing.T, f *linuxOnlineFixture, offline bool) {
	t.Helper()
	root := linuxOfflineMaterialize(t, f.initial)
	fetcher := packagesource.Fetcher{Client: &http.Client{Transport: remoteOfflineTransport(func(request *http.Request) (*http.Response, error) {
		f.calls++
		data, ok := f.replies[request.URL.String()]
		if !ok {
			t.Fatalf("unexpected Go request %s", request.URL)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data)), Request: request}, nil
	})}}
	result, err := fetcher.Fetch(t.Context(), root, offline)
	if err != nil {
		t.Fatal(err)
	}
	f.ok, f.closes = true, f.calls
	f.final = linuxOfflineTree(t, root)
	f.output = fmt.Sprintf("ok %d %d %d %d %d\n", result.Packages, result.Downloaded, result.CacheHits, result.Installed, result.Reused)
}

func linuxFetchSuccess(t *testing.T, f *linuxOnlineFixture) string {
	t.Helper()
	var packages, downloaded, hits, installed, reused uint32
	if _, err := fmt.Sscanf(f.output, "ok %d %d %d %d %d\n", &packages, &downloaded, &hits, &installed, &reused); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("packages fetched: %d package(s), %d downloaded, %d cache hit(s), %d installed, %d reused\n", packages, downloaded, hits, installed, reused)
}

func linuxFetchDiagnostic(t *testing.T, receipt string) string {
	t.Helper()
	var stage, code, native, cleanup uint32
	if _, err := fmt.Sscanf(receipt, "error %d %d %d %d\n", &stage, &code, &native, &cleanup); err != nil {
		t.Fatal(err)
	}
	message := map[uint32]string{3: "cannot acquire exclusive package fetch lock", 5: "source archive SHA-256 mismatch", 7: "cannot publish or reuse source package", 8: "package fetch lock cleanup failed", 9: "package fetch deadline exceeded", 10: "source HTTPS request failed or timed out", 11: "cannot publish immutable source archive cache"}[stage]
	if stage == 10 && code == 4 {
		message = "source HTTPS response status or encoding rejected"
	}
	if stage == 10 && code == 5 {
		message = "source download byte limit exceeded"
	}
	if message == "" {
		t.Fatal("missing independent error expectation", receipt)
	}
	if cleanup != 0 {
		message += "; cleanup failed; inspect leftover fetch lock or staging directory"
	}
	return "package fetch failed: " + message + "\n"
}

func linuxFetchHelp(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", "linux_cli_data.fy"))
	if err != nil {
		t.Fatal(err)
	}
	start, end := strings.Index(string(source), "return "), strings.LastIndex(string(source), "; }")
	if start < 0 || end <= start+7 {
		t.Fatal("Linux help source boundary changed")
	}
	text := strings.TrimSpace(string(source)[start+7 : end])
	value, err := strconv.Unquote(text)
	if err != nil {
		t.Fatalf("Linux help contract: %v %q", err, value)
	}
	for _, line := range []string{"usage: lexer-candidate-linux --help | --version | targets\n", "new <package-name> --uid <package-uid>\n", "check [--locked]", "build [--release]", "run [--release]", "test --native", "experimental fetch-packages [--offline]\n", "experimental atlas-impact ", "experimental atlas-mcp --allow-impact ", "experimental atlas-symbols list ", "experimental atlas-calls list ", "--c-workspace <root> [--locked]\n", "No arguments: private binary stdin protocol.\n"} {
		if !strings.Contains(value, line) {
			t.Fatalf("Linux help missing %q", line)
		}
	}
	return value
}

const linuxFetchEntry = `module main;
import linux_fetch_cli as fetch;
@link_name("online_probe_setup") unsafe extern C fn void setup();
@link_name("online_probe_finish") unsafe extern C fn void finish();
@effects(filesystem,console,network,clock) fn i32 main(){unsafe{setup();}let code=fetch::arguments();unsafe{finish();}return code;}
`
