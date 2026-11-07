package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func TestCandidateProjectDropCap(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows cleanup drop capacity replay; Linux execution is batched")
	}
	read := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	inputs := map[string][32]byte{}
	for _, name := range []string{"module_unit_cleanup_mir_model", "module_unit_cleanup_mir_verify", "module_project_cleanup", "module_cleanup_artifact_decode"} {
		path := filepath.Join("..", "..", "examples", "lexer_candidate", "src", name+".fy")
		inputs[path] = sha256.Sum256(read(path))
		t.Logf("%s sha256=%x", name, inputs[path])
	}
	fixturePath := filepath.Join("testdata", "project_drop_cap", "main.fy")
	fixture := read(fixturePath)
	inputs[fixturePath] = sha256.Sum256(fixture)
	t.Logf("fixture sha256=%x", inputs[fixturePath])
	unchanged := func() {
		t.Helper()
		for path, want := range inputs {
			if sha256.Sum256(read(path)) != want {
				t.Fatalf("drop capacity replay input changed: %s", path)
			}
		}
	}
	root := t.TempDir()
	copyLexerExample(t, root)
	copyCandidateParityModules(t, root, "module_project_cleanup", "module_cleanup_artifact_encode", "module_cleanup_artifact_decode")
	// Fresh test-only constructors are appended to temporary copies inside their
	// owning modules. Product factory visibility and proof gates remain intact.
	for name, extra := range map[string]string{
		"module_types":              `pub fn Graph drop_cap_graph(v::Vec<Type> types){return Graph{.types,.aggregates=false};}`,
		"module_project_types":      `pub fn Plan drop_cap_plan(types::Graph graph,v::Vec<Unit> units){return Plan{.graph,.units,.maps=v::empty<usize>(),.prefix=2,.work=0};}`,
		"module_project_frames":     `pub fn Plan drop_cap_plan(v::Vec<clean::Frame> frames,v::Vec<Origin> origins,v::Vec<Range> units,v::Vec<usize> maps){return Plan{.frames,.origins,.units,.maps,.work=0};}`,
		"module_project_metadata":   `pub fn Plan drop_cap_plan(v::Vec<Choice> choices){return Plan{.layouts=v::empty<ir::Layout>(),.fields=v::empty<interfaces::Field>(),.callables=v::empty<interfaces::Callable>(),.parameters=v::empty<usize>(),.names=bytes::empty(),.ranges=v::empty<Range>(),.maps=v::empty<usize>(),.choices,.work=0};}`,
		"module_unit_body_ir_model": `pub fn Program drop_cap_wrap(clean_ir::Program cleanup_ir){return Program{.states=state::empty(),.sdk_sources=sdk_sources::empty(),.cleanup_ir,.bodies=v::empty<Body>(),.layouts=v::empty<Layout>(),.fields=v::empty<interfaces::Field>(),.callables=v::empty<interfaces::Callable>(),.parameters=v::empty<usize>(),.names=bytes::empty(),.work=0};}`,
	} {
		path := filepath.Join(root, "src", name+".fy")
		writeTestSource(t, root, "src/"+name+".fy", string(read(path))+"\n"+extra+"\n")
	}
	writeTestSource(t, root, "src/main.fy", string(fixture))
	build, support := candidateParityBuildPartsWithLimit(t, root, nil, 256<<10)
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			unchanged()
			image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(tc, profile, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "drop-cap-"+profile.Name+".exe")
			if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
				t.Fatal(err)
			}
			t.Logf("native PE sha256=%x", sha256.Sum256(image.Bytes))
			ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, path).Output()
			if err != nil {
				t.Fatalf("detached cleanup boundary replay: %v", err)
			}
			validateProjectDropCap(t, out)
			unchanged()
		})
	}
}

func validateProjectDropCap(t *testing.T, raw []byte) {
	t.Helper()
	if len(raw) < 8 || string(raw[:8]) != "FYDPCAP1" {
		t.Fatal("missing bounded drop replay ledger")
	}
	r := bytes.NewReader(raw[8:])
	number := func() uint64 {
		t.Helper()
		var n uint64
		if err := binary.Read(r, binary.BigEndian, &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if number() != 6 {
		t.Fatal("missing artifact boundary cases")
	}
	for i, count := range []uint64{16384, 16384, 23100, 32767, 32768, 32768} {
		project, aggregate := number(), number()
		if project != boolNumber(i != 0) || aggregate != boolNumber(i == 5) || number() != count {
			t.Fatal("incorrect mode or artifact boundary order")
		}
		ew, dw, size := number(), number(), number()
		want := projectDropGolden(project, aggregate, count)
		if size != uint64(len(want)) || size > uint64(r.Len()) || ew == 0 || dw == 0 || ew > 16777216 || dw > 16777216 {
			t.Fatal("incorrect artifact length or work ledger")
		}
		got := make([]byte, int(size))
		if _, err := r.Read(got); err != nil || !bytes.Equal(got, want) {
			t.Fatalf("independent golden artifact mismatch: project=%d aggregate=%d count=%d", project, aggregate, count)
		}
		t.Logf("artifact project=%d aggregate=%d drops=%d bytes=%d encodeWork=%d decodeWork=%d golden=equal", project, aggregate, count, size, ew, dw)
	}
	if number() != 6 {
		t.Fatal("missing selected project append cases")
	}
	for i, count := range []uint64{16384, 23100, 32767, 32768, 32769, 32768} {
		n, skip, code, work := number(), number(), number(), number()
		wantCode := uint64(0)
		if count == 32769 {
			wantCode = 1002
		}
		if n != count || skip != boolNumber(i == 5) || code != wantCode || work == 0 || work > 16777216 {
			t.Fatal("selected project append boundary or work mismatch")
		}
		t.Logf("selected drops=%d discardedUnit=%d code=%d work=%d remap=equal inputs=unchanged", n, skip, code, work)
	}
	if r.Len() != 0 {
		t.Fatal("unexpected trailing replay output")
	}
}

func boolNumber(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

// Independent fixed-schema golden, with no production codec or count helper.
func projectDropGolden(project, aggregate, count uint64) []byte {
	var out bytes.Buffer
	marker := "FYCLEAN0"
	if aggregate != 0 {
		marker = "FYCLEAN1"
	}
	out.WriteString(marker)
	numbers := func(values ...uint64) {
		for _, n := range values {
			_ = binary.Write(&out, binary.BigEndian, n)
		}
	}
	numbers(project, 2)
	numbers(1, 7, 0, 0, 0, 0, 3, 0, 0, 1, 0, 0)       // void, resource
	numbers(0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0) // glues
	numbers(1, 3, 1, 0, 2, 1, 0, 0, 0)                // frame count and frame
	numbers(0, 1, 1, 1, 16384, 2, 10, 20)             // fields, operand count and operand
	numbers(count)
	for i := uint64(0); i < count; i++ {
		numbers(1, i%9+1, 100+i*2, 101+i*2)
	}
	return out.Bytes()
}
