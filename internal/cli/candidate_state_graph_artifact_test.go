package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func stateGraphArtifactTrials(t *testing.T) []stateGraphHelperTrial {
	t.Helper()
	var out []stateGraphHelperTrial
	for _, trial := range stateGraphHelperTrials(t) {
		if trial.want {
			out = append(out, trial)
		}
	}
	for _, trial := range stateGraphBorrowTrials(t) {
		if trial.want && trial.body == 0 {
			out = append(out, stateGraphHelperTrial{"borrow/" + trial.name, trial.program, true, ""})
		}
	}
	out = append(out, stateGraphHelperTrial{"typed-join", stateGraphIRBase(t), true, ""})
	return out
}

func stateGraphArtifactRequest(mode, work uint64, payload []byte) []byte {
	out := make([]byte, 16, len(payload)+16)
	binary.BigEndian.PutUint64(out, mode)
	binary.BigEndian.PutUint64(out[8:], work)
	return append(out, payload...)
}

// Independent writer: production code never reads this format builder. Input
// graph packets enumerate all 19 op and 16 block/terminator scalar facts.
func stateGraphArtifactExpected(layout []byte, p stateGraphHelperPacket) []byte {
	var out bytes.Buffer
	out.WriteString("FYSGRPH0")
	number := func(n uint64) { _ = binary.Write(&out, binary.BigEndian, n) }
	number(uint64(len(layout)))
	out.Write(layout)
	number(uint64(len(p.graphs) - 1))
	number(uint64(len(p.data)))
	for _, span := range p.names {
		number(span[0])
		number(span[1])
	}
	for _, g := range p.graphs {
		number(uint64(len(g) - 8))
		out.Write(g[8:])
	}
	out.Write(p.data)
	return out.Bytes()
}

func stateGraphArtifactLayout(t *testing.T, raw []byte, p contractstatemir.Program) []byte {
	t.Helper()
	if len(raw) < 16 || string(raw[:8]) != "FYSGRPH0" {
		t.Fatalf("bad artifact header: %q", raw)
	}
	n := binary.BigEndian.Uint64(raw[8:])
	if n > uint64(len(raw)-16) {
		t.Fatal("truncated layout")
	}
	layout := raw[16 : 16+n]
	if len(layout) < 160 || string(layout[:8]) != "FYSTATE1" || binary.BigEndian.Uint64(layout[8:]) != 1 {
		t.Fatal("layout format lost")
	}
	// Go's independently authored schema hashes must survive the transport.
	hash := p.Layout.Hash
	for i := 0; i < 8; i++ {
		if binary.BigEndian.Uint64(layout[16+(9+i)*8:]) != uint64(binary.BigEndian.Uint32(hash[i*4:])) {
			t.Fatalf("layout hash word %d drift", i)
		}
	}
	fields := p.Layout.Fields
	if binary.BigEndian.Uint64(layout[152:]) != uint64(len(fields)) {
		t.Fatal("layout field count drift")
	}
	for i, f := range fields {
		at := 160 + i*120
		for j := 0; j < 8; j++ {
			if binary.BigEndian.Uint64(layout[at+(7+j)*8:]) != uint64(binary.BigEndian.Uint32(f.Prefix[j*4:])) {
				t.Fatalf("prefix %d/%d drift", i, j)
			}
		}
	}
	return layout
}

func stateGraphArtifactProbe() string {
	p := stateGraphHelperProbe()
	p = p[:strings.Index(p, "@effects(console) fn i32 main()")]
	return strings.Replace(p, "import module_state_graph_program as program;", "import module_state_graph_artifact as artifact;import module_state_artifact as layoutwire;import module_state_graph_program as program;", 1) + stateGraphArtifactMain
}

const stateGraphArtifactMain = `
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>8388640-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 var usize work=0;var c=wire::Cursor{.offset=0,.code=0};let mode=number(bytes::view(ref input),ref mut c,ref mut work);let initial=number(bytes::view(ref input),ref mut c,ref mut work);if(c.code!=0){return 3;}
 if(mode!=1){
  work=initial;let proof=switch(artifact::decode(bytes::view(ref input)[16..],ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
  if(mode==4){discard io::write_usize(work);return 0;}
  if(mode==2){discard io::write("0");return 0;}
  // Destruct the entire caller-owned wire buffer before using the decoded
  // graph, helpers, names and independent verified layout again.
  input=bytes::empty();work=0;let result=switch(artifact::encode(ref proof,ref mut work)){case .ok(p):{p}case .err(_):{return 40;}};
  discard io::write_bytes(wire::content(ref result));return 0;
 }
 work=0;let g=switch(read_program(bytes::view(ref input)[16..],ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
 var plan=switch(layout()){case .ok(p):{p}case .err(_):{return 4;}};work=0;
 let proof=switch(program::verify(g,ref plan,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};
 plan=state::empty();input=bytes::empty();if(state::count(program::layouts(ref proof))!=1){return 5;}
 work=0;let payload=switch(artifact::encode(ref proof,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};let cost=work;
 work=0;switch(layoutwire::decode(wire::content(ref payload),ref mut work)){case .ok(_):{return 20;}case .err(code):{if(code!=1001){return 21;}}};
 work=0;switch(layoutwire::decode_graph(wire::content(ref payload),ref mut work)){case .ok(_):{return 22;}case .err(code):{if(code!=1001){return 23;}}};
 work=16777216-cost;let exact=switch(artifact::encode(ref proof,ref mut work)){case .ok(p):{p}case .err(_):{return 6;}};if(work!=16777216 || !bytes::equal(wire::content(ref payload),wire::content(ref exact))){return 7;}
 work=16777216-cost+1;switch(artifact::encode(ref proof,ref mut work)){case .ok(_):{return 8;}case .err(code):{if(code!=1002 || work>16777216){return 9;}}};
 work=16777217;switch(artifact::encode(ref proof,ref mut work)){case .ok(_):{return 10;}case .err(code):{if(code!=1002 || work!=16777217){return 11;}}};
 work=0;let restored=switch(artifact::decode(wire::content(ref payload),ref mut work)){case .ok(p):{p}case .err(_):{return 12;}};let decode_cost=work;
 work=16777216-decode_cost;let exact_decoded=switch(artifact::decode(wire::content(ref payload),ref mut work)){case .ok(p):{p}case .err(_):{return 13;}};if(work!=16777216){return 14;}
 work=16777216-decode_cost+1;switch(artifact::decode(wire::content(ref payload),ref mut work)){case .ok(_):{return 15;}case .err(code):{if(code!=1002 || work>16777216){return 16;}}};
 work=0;let again=switch(artifact::encode(ref restored,ref mut work)){case .ok(p):{p}case .err(_):{return 17;}};
 work=0;let exact_again=switch(artifact::encode(ref exact_decoded,ref mut work)){case .ok(p):{p}case .err(_):{return 18;}};
 if(!bytes::equal(wire::content(ref payload),wire::content(ref again)) || !bytes::equal(wire::content(ref payload),wire::content(ref exact_again))){return 19;}
 discard io::write_bytes(wire::content(ref payload));return 0;
}`

func TestCandidateStateGraphArtifactOracle(t *testing.T) {
	for _, trial := range stateGraphArtifactTrials(t) {
		t.Run(trial.name, func(t *testing.T) { stateGraphHelperCheckOracle(t, trial) })
	}
}

func TestCandidateStateGraphArtifact(t *testing.T) {
	trials := stateGraphArtifactTrials(t)
	for _, trial := range trials {
		stateGraphHelperCheckOracle(t, trial)
	}
	candidateAtlasProbeSize(t, "module_state_graph_artifact", stateGraphArtifactProbe(), 512<<10, func(t *testing.T, image string) {
		var base, layout []byte
		for _, trial := range trials {
			t.Run("roundtrip/"+trial.name, func(t *testing.T) {
				packet := stateGraphHelperEncode(trial.p)
				code, out := candidateAtlasProcess(t, image, stateGraphArtifactRequest(1, 0, packet.bytes()))
				if code != 0 {
					t.Fatalf("roundtrip exit=%d output=%q", code, out)
				}
				raw := []byte(out)
				gotLayout := stateGraphArtifactLayout(t, raw, trial.p)
				if !bytes.Equal(raw, stateGraphArtifactExpected(gotLayout, packet)) {
					t.Fatal("independent graph/header/block/literal/name/helper facts changed")
				}
				if trial.name == "leaf" {
					base = bytes.Clone(raw)
					layout = bytes.Clone(gotLayout)
				}
				code, out = candidateAtlasProcess(t, image, stateGraphArtifactRequest(3, 0, raw))
				if code != 0 || !bytes.Equal(raw, []byte(out)) {
					t.Fatalf("decoded owner retained caller storage: exit=%d", code)
				}
			})
		}
		if len(base) == 0 {
			t.Fatal("missing base artifact")
		}
		stateGraphArtifactCorruption(t, image, base, layout)
	}, "module_state_graph_program", "module_state_declarations", "candidate_artifact_wire")
}

func stateGraphArtifactCorruption(t *testing.T, image string, base, layout []byte) {
	t.Helper()
	check := func(name string, raw []byte, want string) {
		t.Run(name, func(t *testing.T) {
			code, out := candidateAtlasProcess(t, image, stateGraphArtifactRequest(2, 0, raw))
			if code != 0 || out != want {
				t.Fatalf("exit=%d output=%q want=%s", code, out, want)
			}
		})
	}
	// Each rewrite retains all frame lengths: a failure here must come from
	// checked contents or semantic re-verification, not a truncated envelope.
	field := func(name string, at int, value uint64, want string) {
		raw := bytes.Clone(base)
		binary.BigEndian.PutUint64(raw[at:], value)
		check(name, raw, want)
	}
	afterLayout := 16 + len(layout)
	nameSpan := afterLayout + 16
	rootFrame := nameSpan + 16
	rootBody := rootFrame + 8
	rootOp := rootBody + 72
	rootBlock := rootOp + 152
	helperFrame := rootBody + int(binary.BigEndian.Uint64(base[rootFrame:]))
	helperBody := helperFrame + 8
	helperOp := helperBody + 72
	helperBlock := helperOp + 152
	for _, test := range []struct {
		name string
		at   int
		n    uint64
		code string
	}{
		{"layout-frame-limit", 8, 8388609, "1002"},
		{"layout-hash-forged", 16 + 16 + 9*8, 0, "1001"},
		{"helper-count-limit", afterLayout, 33, "1002"},
		{"name-byte-limit", afterLayout + 8, 1048577, "1002"},
		{"name-gap", nameSpan, 1, "1001"},
		{"name-span-outside", nameSpan + 8, 999, "1001"},
		{"graph-frame-limit", rootFrame, 8388609, "1002"},
		{"root-effects-missing", rootBody, 0, "1001"},
		{"root-effects-excess", rootBody, 3, "1001"},
		{"root-effects-u32-overflow", rootBody, 1 << 32, "1001"},
		{"zero-capability", rootBody + 8, 0, "1001"},
		{"layout-index-outside", rootBody + 16, 1, "1001"},
		{"entry-outside", rootBody + 24, 64, "1001"},
		{"parameter-count-limit", rootBody + 32, 524289, "1002"},
		{"argument-count-limit", rootBody + 40, 1048577, "1002"},
		{"operation-count-limit", rootBody + 48, 1025, "1002"},
		{"block-count-limit", rootBody + 56, 65, "1002"},
		{"literal-byte-limit", rootBody + 64, 1048577, "1002"},
		{"section-sum-before-allocation", rootBody + 32, 1, "1001"},
		{"callee-zero", rootOp + 14*8, 0, "1001"},
		{"callee-outside", rootOp + 14*8, 2, "1001"},
		{"call-carries-borrow", rootOp + 15*8, 1, "1001"},
		{"call-carries-literal-codec", rootOp + 8*8, 6, "1001"},
		{"call-cost-zero", rootOp + 13*8, 0, "1001"},
		{"operation-u32-overflow", rootOp, 1<<32 | 11, "1001"},
		{"block-span-outside", rootBlock + 4*8, 2, "1001"},
		{"block-parameters-outside", rootBlock + 2*8, 1, "1001"},
		{"root-return-forbidden", rootBlock + 5*8, 6, "1001"},
		{"terminator-u32-overflow", rootBlock + 5*8, 1<<32 | 3, "1001"},
		{"terminator-argument-outside", rootBlock + 8*8, 1, "1001"},
		{"trap-category-outside", rootBlock + 14*8, 1, "1001"},
		{"helper-effects-missing", helperBody, 0, "1001"},
		{"helper-capability-mismatch", helperBody + 8, 2, "1001"},
		{"helper-layout-mismatch", helperBody + 16, 1, "1001"},
		{"helper-carries-callee", helperOp + 14*8, 1, "1001"},
		{"helper-literal-span-outside", helperOp + 10*8, 5, "1001"},
		{"helper-literal-codec-forged", helperOp + 8*8, 3, "1001"},
		{"helper-commit-forbidden", helperBlock + 5*8, 3, "1001"},
	} {
		field(test.name, test.at, test.n, test.code)
	}
	for _, marker := range []string{"FYSTATE0", "FYSTATE1", "FYPROG00", "FYSGRPH1"} {
		raw := bytes.Clone(base)
		copy(raw, marker)
		check("foreign-marker/"+marker, raw, "1001")
	}
	// Correct marker with a different artifact family's body is still refused.
	spoof := bytes.Clone(layout)
	copy(spoof, "FYSGRPH0")
	check("layout-body-with-graph-marker", spoof, "1001")
	for _, n := range []int{0, 7, 8, 15, rootBody + 71, helperFrame, len(base) - 1} {
		check(fmt.Sprintf("truncated/%d", n), base[:n], "1001")
	}
	check("trailing-byte", append(bytes.Clone(base), 0), "1001")
	exact := make([]byte, 8<<20)
	copy(exact, base)
	check("8MiB-input-trailing-data", exact, "1001")
	over := make([]byte, (8<<20)+1)
	copy(over, base)
	check("8MiB-plus-one-input", over, "1002")
	// The wire writer is independent of program::verify. Complete structurally
	// valid bad helper/borrow programs must be rechecked on decode.
	for _, trial := range stateGraphHelperTrials(t) {
		if trial.want || trial.name == "layout-drift" || strings.HasSuffix(trial.name, "-effect-duplicate") || strings.HasSuffix(trial.name, "-effect-unsorted") {
			continue
		}
		stateGraphHelperCheckOracle(t, trial)
		want := "1001"
		if trial.name == "helpers-33" || trial.name == "combined-operations-1025" {
			want = "1002"
		}
		check("semantic/"+trial.name, stateGraphArtifactExpected(layout, stateGraphHelperEncode(trial.p)), want)
	}
	// A graph may retain otherwise unused literal bytes. This exercises the
	// actual 1MiB data boundary without increasing product work/byte limits.
	packet := stateGraphHelperEncode(stateGraphHelperProgram(t, "leaf"))
	graph := packet.graphs[0]
	binary.BigEndian.PutUint64(graph[72:], 1<<20)
	packet.graphs[0] = append(graph, make([]byte, 1<<20)...)
	large := stateGraphArtifactExpected(layout, packet)
	check("1MiB-literal-data", large, "0")
	// Every phase's exact/one-short budget is additionally exercised in native
	// mode 1 for every roundtrip, rather than only testing an empty-ledger case.
	for _, test := range []struct {
		name string
		work uint64
	}{{"exhausted", 16777216}, {"already-over", 16777217}, {"usize-max", ^uint64(0)}} {
		t.Run("work/"+test.name, func(t *testing.T) {
			code, out := candidateAtlasProcess(t, image, stateGraphArtifactRequest(2, test.work, base))
			if code != 0 || out != "1002" {
				t.Fatalf("exit=%d output=%q", code, out)
			}
		})
	}
	code, out := candidateAtlasProcess(t, image, stateGraphArtifactRequest(4, 0, base))
	cost, err := strconv.ParseUint(out, 10, 64)
	if code != 0 || err != nil || cost == 0 || cost > 16777216 {
		t.Fatalf("decode cost exit=%d output=%q", code, out)
	}
	for _, test := range []struct {
		name string
		work uint64
		want string
	}{{"exact", 16777216 - cost, "0"}, {"one-short", 16777216 - cost + 1, "1002"}} {
		t.Run("work/"+test.name, func(t *testing.T) {
			code, out := candidateAtlasProcess(t, image, stateGraphArtifactRequest(2, test.work, base))
			if code != 0 || out != test.want {
				t.Fatalf("exit=%d output=%q want=%s", code, out, test.want)
			}
		})
	}
}

func TestCandidateStateGraphArtifactOwner(t *testing.T) {
	for _, test := range []struct{ name, body, code string }{
		{"no-artifact-proof-constructor", "fn program::Verified forge(){return program::Verified{.program=0,.layouts=0};}", "E0103"},
		{"no-mutable-layout", "fn void mutate(ref program::Verified p){let ref mut state::Plan x=ref mut program::layouts(p);}", "E1008"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_artifact")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_artifact as artifact;import module_state_graph_program as program;import module_state_layout as state;"+test.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed detached artifact", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), test.code) {
				t.Fatalf("owner escape accepted or unrelated failure: %d %s", code, &diagnostics)
			}
		})
	}
}
