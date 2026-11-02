package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

// Capture exactly the fresh packet handed to the original C-only companion.
// The proxy does not parse Fairy, produce C, replay cached packets or alter CLI
// product builds. The additional observer image below is a test-only consumer.
func installGenericCLIPacketCapture(t *testing.T, image string) {
	t.Helper()
	dir := filepath.Dir(image)
	proxy := filepath.Join(dir, "fairyc-candidate-build.exe")
	real := filepath.Join(t.TempDir(), "fairyc-candidate-build.exe")
	if err := os.Rename(proxy, real); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(proxy); err != nil && !os.IsNotExist(err) {
			t.Error(err)
		}
		if err := os.Rename(real, proxy); err != nil {
			t.Error(err)
		}
	})
	source := filepath.Join(t.TempDir(), "main.go")
	writeTestSource(t, filepath.Dir(source), "main.go", `package main
import("fmt";"os";"os/exec")
func main(){
 args:=os.Args[1:]
 for i:=0;i+1<len(args);i++{if args[i]=="--packet-input"{
  raw,e:=os.ReadFile(args[i+1]);if e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)}
  path:=os.Getenv("FAIRYC_GENERIC_CLI_PACKET");if path==""{fmt.Fprintln(os.Stderr,"missing test packet capture path");os.Exit(1)}
  if e=os.WriteFile(path,raw,0600);e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)}
 }}
 cmd:=exec.Command(`+strconv.Quote(real)+`,args...)
 cmd.Stdin=os.Stdin;cmd.Stdout=os.Stdout;cmd.Stderr=os.Stderr
 if e:=cmd.Run();e!=nil{if x,ok:=e.(*exec.ExitError);ok{os.Exit(x.ExitCode())};fmt.Fprintln(os.Stderr,e);os.Exit(1)}
}
`)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-o", filepath.Join(dir, "fairyc-candidate-build.exe"), source)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("capture proxy: %v %s", err, out)
	}
}

func observeGenericCLIPacket(t *testing.T, path string, second bool, code int32) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := candidatebuild.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	var observer strings.Builder
	observer.WriteString("#include \"fy_project.h\"\n#include <limits.h>\n#include <string.h>\nint fairyc_generic_observe(void){\n")
	for _, v := range genericCLIValues {
		pattern := `(?m)^(?:extern )?` + regexp.QuoteMeta(v.c) + ` (fy_fn_[0-9]+)\(` + regexp.QuoteMeta(v.c) + ` [^,()]+,\s*` + regexp.QuoteMeta(v.c) + ` [^,()]+,\s*int32_t [^,()]+\);`
		found := regexp.MustCompile(pattern).FindAllStringSubmatch(packet.Header, -1)
		if len(found) != 1 {
			t.Fatalf("expected one private %s three-argument observer entry, got %d; header=%s", v.fairy, len(found), packet.Header)
		}
		want := v.first
		if second {
			want = v.second
		}
		fmt.Fprintf(&observer, "{%s actual=%s((%s)(%s),(%s)(%s),0);%s expected=(%s)(%s);if(memcmp(&actual,&expected,sizeof(expected))!=0)return 1;}\n", v.c, found[0][1], v.c, v.first, v.c, v.second, v.c, v.c, want)
	}
	observer.WriteString("return 0;}\n")
	packet.Header += "\nint fairyc_generic_observe(void);\n"
	entry := regexp.MustCompile(`\bint main\([^)]*\)\s*\{`)
	changed := 0
	for i, unit := range packet.Units {
		changed += len(entry.FindAllStringIndex(unit, -1))
		packet.Units[i] = entry.ReplaceAllString(unit, "${0}if(fairyc_generic_observe()!=0)return 99;")
	}
	if changed != 1 {
		t.Fatalf("expected one private C main for test-only observation, got %d", changed)
	}
	rt, err := cruntime.ScalarSource(128)
	if err != nil {
		t.Fatal(err)
	}
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := cdriver.BuildWindowsMSVCTargetParts(tc, cdriver.LowProfile, packet.Header, packet.Literals, packet.Units, rt+"\n"+observer.String())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
	if err != nil || result.ExitCode != code || result.Stdout != "" || result.Stderr != "" {
		t.Fatalf("seven-scalar packet observation: %+v %v", result, err)
	}
	t.Logf("seven current packet scalar values observed; second=%t exit=%d", second, code)
}
