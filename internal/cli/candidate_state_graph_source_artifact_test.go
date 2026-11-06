package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

const stateGraphSourceArtifactSource = `module transaction;
@contract_uid("example.source-graph") state {cell u32 count;}
@effects(storage_write) fn void transaction(){storage::write("count",7);storage::commit();}`

const stateGraphSourceArtifactProbe = `module main;
import module_state_graph_source_artifact as bound;
import module_state_graph_source as source;
import module_state_graph_program as program;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::io as io;
@effects(console) fn void failure(u32 code){discard io::write_usize(code as usize);}
fn void clobber(ref mut bytes::Buffer input){let changed=bytes::view_mut(input);changed[changed.len-1]=0;}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] block=[0;4096];while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}if(bytes::len(ref input)>9437184-n){return 2;}let ref u8[] raw=ref block;if(!bytes::append(ref mut input,raw[..n]).is_ok){return 3;}}
 var usize setup=0;var c=wire::Cursor{.offset=0,.code=0};let raw=bytes::view(ref input);let mode=wire::number_from(raw,ref mut c,ref mut setup);let length=wire::count_from(raw,ref mut c,8388608,1,ref mut setup);let size=wire::count_from(raw,ref mut c,65536,1,ref mut setup);
 if(c.code!=0 || length>raw.len-c.offset || size!=raw.len-c.offset-length){return 4;}let saved=raw[c.offset..c.offset+length];let actual=raw[c.offset+length..];
 if(mode==2){var usize work=0;switch(bound::restore(saved,actual,ref mut work)){case .ok(p):{if(!source::matches(bound::current(ref p),actual) || program::helper_count(bound::graph(ref p))!=0){return 5;}failure(0);}case .err(code):{failure(code);}};return 0;}
 if(mode>1 || length!=0){return 6;}
 let current=switch(source::produce(actual,ref mut setup)){case .ok(p):{p}case .err(code):{failure(code);return 0;}};
 var usize encoded_work=0;let encoded=switch(bound::encode(ref current,ref mut encoded_work)){case .ok(p):{p}case .err(code):{failure(code);return 0;}};
 if(mode==1){return switch(io::write_bytes(wire::content(ref encoded))){case .ok(n):{if(n==wire::content(ref encoded).len){0}else{7}}case .err(_):{8}};}
 var usize restored_work=0;let restored=switch(bound::restore(wire::content(ref encoded),actual,ref mut restored_work)){case .ok(p):{p}case .err(code):{failure(code);return 9;}};
 if(!source::matches(bound::current(ref restored),actual) || program::helper_count(bound::graph(ref restored))!=0){return 10;}
 var usize again_work=0;let again=switch(bound::encode(bound::current(ref restored),ref mut again_work)){case .ok(p):{p}case .err(_):{return 11;}};if(!bytes::equal(wire::content(ref encoded),wire::content(ref again))){return 12;}
 var usize exact=16777216-encoded_work;switch(bound::encode(ref current,ref mut exact)){case .ok(_):{}case .err(_):{return 13;}};if(exact!=16777216){return 14;}
 var usize short=16777217-encoded_work;switch(bound::encode(ref current,ref mut short)){case .ok(_):{return 15;}case .err(code):{if(code!=1002 || short>16777216){return 16;}}};
 exact=16777216-restored_work;switch(bound::restore(wire::content(ref encoded),actual,ref mut exact)){case .ok(_):{}case .err(_):{return 17;}};if(exact!=16777216){return 18;}
 short=16777217-restored_work;switch(bound::restore(wire::content(ref encoded),actual,ref mut short)){case .ok(_):{return 19;}case .err(code):{if(code!=1002 || short>16777216){return 20;}}};
 var usize exhausted=16777217;switch(bound::restore(wire::content(ref encoded),actual,ref mut exhausted)){case .ok(_):{return 21;}case .err(code):{if(code!=1002 || exhausted!=16777217){return 22;}}};
 exhausted=16777217;switch(bound::encode(ref current,ref mut exhausted)){case .ok(_):{return 23;}case .err(code):{if(code!=1002 || exhausted!=16777217){return 24;}}};
 // Both owners copied the caller's input; mutating that buffer cannot alter them.
 clobber(ref mut input);
 if(!bytes::equal(source::source(ref current),source::source(bound::current(ref restored))) || source::source(bound::current(ref restored))[source::source(bound::current(ref restored)).len-1]==0){return 25;}
 failure(0);return 0;
}`

func stateGraphSourceArtifactRequest(mode uint64, saved []byte, actual string) []byte {
	var b bytes.Buffer
	for _, n := range []uint64{mode, uint64(len(saved)), uint64(len(actual))} {
		_ = binary.Write(&b, binary.BigEndian, n)
	}
	b.Write(saved)
	b.WriteString(actual)
	return b.Bytes()
}

func stateGraphSourceArtifactParts(t *testing.T, raw []byte) (string, []byte) {
	t.Helper()
	if len(raw) < 24 || string(raw[:8]) != "FYSGSRC0" {
		t.Fatalf("expected source envelope, got %q", raw)
	}
	n := binary.BigEndian.Uint64(raw[8:16])
	if n > uint64(len(raw)-24) {
		t.Fatal("invalid source envelope length")
	}
	at := 16 + int(n)
	size := binary.BigEndian.Uint64(raw[at : at+8])
	if size != uint64(len(raw)-at-8) {
		t.Fatal("invalid graph envelope length")
	}
	return string(raw[16:at]), bytes.Clone(raw[at+8:])
}

func stateGraphSourceArtifactEnvelope(source string, graph []byte) []byte {
	var b bytes.Buffer
	b.WriteString("FYSGSRC0")
	_ = binary.Write(&b, binary.BigEndian, uint64(len(source)))
	b.WriteString(source)
	_ = binary.Write(&b, binary.BigEndian, uint64(len(graph)))
	b.Write(graph)
	return b.Bytes()
}

func TestCandidateStateGraphSourceArtifact(t *testing.T) {
	candidateAtlasProbeSize(t, "module_state_graph_source_artifact", stateGraphSourceArtifactProbe, 512<<10, func(t *testing.T, image string) {
		encode := func(source string) []byte {
			t.Helper()
			code, output := candidateAtlasProcess(t, image, stateGraphSourceArtifactRequest(1, nil, source))
			if code != 0 || !strings.HasPrefix(output, "FYSGSRC0") {
				t.Fatalf("encode exit=%d output=%q", code, output)
			}
			return []byte(output)
		}
		base := encode(stateGraphSourceArtifactSource)
		stored, graph := stateGraphSourceArtifactParts(t, base)
		if stored != stateGraphSourceArtifactSource || !bytes.Equal(base, stateGraphSourceArtifactEnvelope(stored, graph)) {
			t.Fatal("source envelope does not preserve exact source/graph bytes")
		}
		assert := func(t *testing.T, mode uint64, packet []byte, source, want string) {
			t.Helper()
			code, output := candidateAtlasProcess(t, image, stateGraphSourceArtifactRequest(mode, packet, source))
			if code != 0 || output != want {
				t.Fatalf("exit=%d output=%q want=%s", code, output, want)
			}
		}
		for _, trial := range []struct{ name, source string }{
			{"original", stateGraphSourceArtifactSource},
			{"comment", stateGraphSourceArtifactSource + "\n// retained source\n"},
			{"reject", strings.Replace(stateGraphSourceArtifactSource, "storage::commit()", "storage::reject()", 1)},
			{"trap", strings.Replace(stateGraphSourceArtifactSource, "storage::commit()", `storage::trap("source-refusal")`, 1)},
		} {
			t.Run("roundtrip/"+trial.name, func(t *testing.T) { assert(t, 0, nil, trial.source, "0") })
		}
		for _, trial := range []struct{ name, before, after string }{
			{"literal", `"count",7`, `"count",8`},
			{"module", "module transaction", "module transactioo"},
			{"uid", "example.source-graph", "example.source-graqh"},
			{"field", "count", "total"},
			{"effect", "@effects(storage_write)", "@effects(storage_read,storage_write)"},
			{"exit", "storage::commit()", "storage::reject()"},
		} {
			changed := strings.ReplaceAll(stateGraphSourceArtifactSource, trial.before, trial.after)
			other := encode(changed)
			_, otherGraph := stateGraphSourceArtifactParts(t, other)
			t.Run("stale-source/"+trial.name, func(t *testing.T) { assert(t, 2, base, changed, "1001") })
			t.Run("paired-roundtrip/"+trial.name, func(t *testing.T) { assert(t, 2, other, changed, "0") })
			// A valid new source cannot bless an independently valid old graph.
			t.Run("valid-graph-splice/"+trial.name, func(t *testing.T) {
				if bytes.Equal(graph, otherGraph) {
					// A same-width module-only edit can leave detached graph bytes
					// equal; source binding is still exact in stale-source above.
					if trial.name != "module" {
						t.Fatalf("fixture failed to change the detached graph")
					}
					assert(t, 2, stateGraphSourceArtifactEnvelope(stored, otherGraph), stored, "0")
					return
				}
				assert(t, 2, stateGraphSourceArtifactEnvelope(stored, otherGraph), stored, "1001")
			})
		}
		for _, trial := range []struct {
			name string
			want string
			edit func([]byte) []byte
		}{
			{"marker", "1001", func(b []byte) []byte { copy(b[:8], "FYSTATE1"); return b }},
			{"bare-graph", "1001", func([]byte) []byte { return graph }},
			{"empty", "1001", func([]byte) []byte { return nil }},
			{"truncated-header", "1001", func(b []byte) []byte { return b[:15] }},
			{"truncated-source", "1001", func(b []byte) []byte { return b[:16+len(stored)-1] }},
			{"truncated-graph", "1001", func(b []byte) []byte { return b[:len(b)-1] }},
			{"trailing", "1001", func(b []byte) []byte { return append(b, 0) }},
			{"source-too-large", "1002", func(b []byte) []byte { binary.BigEndian.PutUint64(b[8:], 65537); return b }},
			{"graph-too-large", "1002", func(b []byte) []byte { binary.BigEndian.PutUint64(b[16+len(stored):], 8388609); return b }},
			{"graph-marker", "1001", func(b []byte) []byte { b[24+len(stored)] ^= 1; return b }},
			{"graph-bitflip", "1001", func(b []byte) []byte { b[len(b)-1] ^= 1; return b }},
		} {
			t.Run("corrupt/"+trial.name, func(t *testing.T) { assert(t, 2, trial.edit(bytes.Clone(base)), stored, trial.want) })
		}
	}, "module_state_graph_source_artifact")
}

func TestCandidateStateGraphSourceArtifactEnvelope(t *testing.T) {
	for _, source := range []string{"", stateGraphSourceArtifactSource, "\x00\xff"} {
		t.Run(fmt.Sprint(len(source)), func(t *testing.T) {
			graph := []byte("FYSGRPH0\x00\xff")
			stored, restored := stateGraphSourceArtifactParts(t, stateGraphSourceArtifactEnvelope(source, graph))
			if stored != source || !bytes.Equal(restored, graph) {
				t.Fatal("test request envelope lost exact bytes")
			}
		})
	}
}

func TestCandidateStateGraphSourceArtifactOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"private-constructor", "E0103", "fn bound::Restored forge(){return bound::Restored{.current=0,.graph=0};}"},
		{"immutable-current", "E1008", "fn void change(ref bound::Restored proof){let ref mut source::Produced p=ref mut bound::current(proof);}"},
		{"immutable-graph", "E1008", "fn void change(ref bound::Restored proof){let ref mut program::Verified p=ref mut bound::graph(proof);}"},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_source_artifact")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_source_artifact as bound;import module_state_graph_source as source;import module_state_graph_program as program;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "source-bound graph owner", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), trial.code) {
				t.Fatalf("owner escape accepted or failed outside caller: %d %s", code, &diagnostics)
			}
		})
	}
}
