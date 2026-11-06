package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Independent header writer: no candidate encoder, wire helper or request
// implementation supplies offsets, ordering, lengths, or the expected digest.
func stateRequestOracle(snapshot contractstate.PersistentSnapshot, limits [4]uint64) []byte {
	out := make([]byte, 152)
	copy(out, "FYGRIN01")
	copy(out[8:40], snapshot.LayoutHash[:])
	copy(out[40:72], bytes.Repeat([]byte{0x42}, 32))
	copy(out[72:104], bytes.Repeat([]byte{0xa5}, 32))
	binary.BigEndian.PutUint64(out[104:112], snapshot.Version)
	for i, n := range limits {
		binary.BigEndian.PutUint64(out[112+i*8:120+i*8], n)
	}
	binary.BigEndian.PutUint32(out[144:148], uint32(len(snapshot.Entries)))
	for _, entry := range snapshot.Entries {
		out = append(out, entry.Key[:]...)
		out = binary.BigEndian.AppendUint32(out, uint32(len(entry.Value)))
		out = append(out, entry.Value...)
	}
	binary.BigEndian.PutUint32(out[148:152], uint32(len(out)-152))
	return out
}

func stateRequestProbe() string {
	probe := stateGraphHelperProbe()
	probe = probe[:strings.Index(probe, "@effects(console) fn i32 main()")]
	probe = strings.Replace(probe, "import module_state_graph_program as program;", "import module_state_graph_request as request;import module_state_graph_transport_plan as plan;import std::process as process;import module_state_graph_program as program;", 1)
	return probe + stateRequestMain
}

const stateRequestMain = `
fn usize n64(ref u8[] raw,usize at){var usize out=0;var usize i=0;while(i<8){out=out*256+(raw[at+i] as usize);i+=1;}return out;}
fn usize n32(ref u8[] raw,usize at){var usize out=0;var usize i=0;while(i<4){out=out*256+(raw[at+i] as usize);i+=1;}return out;}
fn u32[8] key32(ref u8[] raw,usize at){var u32[8] out=[0;8];var usize i=0;while(i<8){let offset=at+i*4;out[i]=(raw[offset] as u32)<<24|(raw[offset+1] as u32)<<16|(raw[offset+2] as u32)<<8|(raw[offset+3] as u32);i+=1;}return out;}
fn bytes::Buffer owned(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn Result<request::Snapshot,u32> snapshot(ref u8[] raw,usize mode){
 let count=n32(raw,144);var rows=v::empty<request::Entry>();assert(v::reserve<request::Entry>(ref mut rows,count).is_ok);var usize at=152;var usize i=0;
 while(i<count){let key=key32(raw,at);let size=n32(raw,at+32);at+=36;let start=at;at+=size;
  let row=request::Entry{.key,.start=if(mode==1){at+1}else{start},.end=if(mode==2){raw.len+1}else{at}};
  assert(v::push<request::Entry>(ref mut rows,row).is_ok);i+=1;}
 assert(at==raw.len);var layout=key32(raw,8);if(mode==3){layout[0]^=1;}
 return .ok(request::snapshot_of(layout,n64(raw,104),owned(raw),rows));
}
fn bool failed(ref plan::Plan plan,ref u8[] l,ref u8[] b,ref u8[] p,ref request::Snapshot s,request::Limits limits,ref mut request::Work w){return switch(request::encode(plan,l,b,p,s,limits,w)){case .ok(_):{false}case .err(c):{c==1002}};}
@effects(console) fn i32 main(){
 // This test reader deliberately bypasses the old 2MiB graph-script reader.
 var input=bytes::empty();assert(bytes::reserve(ref mut input,20971520).is_ok);var u8[4096] block=[0;4096];
 while(true){let n=switch(io::read_stdin(ref mut block)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] raw=ref block;if(bytes::len(ref input)>20971520-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 var u8[32] arg=[0;32];let arglen=switch(process::arg(1,ref mut arg)){case .ok(n):{n}case .err(_):{0}};let ref u8[] args=ref arg;let usize mode=if(arglen==0){0}else{(args[3]-48) as usize};
 let result={let raw=bytes::view(ref input);let size=n64(raw,0);var usize setup=0;
  let g=switch(read_program(raw[8..8+size],ref mut setup)){case .ok(p):{p}case .err(c):{discard io::write_u32(c);return 0;}};
  let layouts=switch(layout()){case .ok(p):{p}case .err(_):{return 3;}};setup=0;
  let proof=switch(program::verify(g,ref layouts,ref mut setup)){case .ok(p):{p}case .err(c):{discard io::write_u32(c);return 0;}};setup=0;
  let owner=switch(plan::build(ref proof,ref mut setup)){case .ok(p):{p}case .err(c):{discard io::write_u32(c);return 0;}};
  let data=raw[8+size..];var snap=switch(snapshot(data,mode)){case .ok(p):{p}case .err(_):{return 4;}};
  let limits=request::Limits{.writes=n64(data,112),.bytes=n64(data,120),.value=n64(data,128),.key=n64(data,136)};
  let l=data[8..if(mode==4){39}else{40}];let b=if(mode==5){b"\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"}else{data[40..72]};let p=data[72..104];
  var work=request::empty_work();let packet=switch(request::encode(ref owner,l,b,p,ref snap,limits,ref mut work)){case .ok(p):{p}case .err(c):{discard io::write_u32(c);return 0;}};
  if(mode==0){let cost=work;
   work=request::Work{.structure=16777216-cost.structure,.bytes=1073741824-cost.bytes};
   let exact=switch(request::encode(ref owner,l,b,p,ref snap,limits,ref mut work)){case .ok(p):{p}case .err(_):{return 10;}};
   assert(work.structure==16777216 && work.bytes==1073741824 && bytes::equal(request::content(ref exact),request::content(ref packet)) && bytes::equal(request::digest(ref exact),request::digest(ref packet)));
   work=request::Work{.structure=16777217-cost.structure,.bytes=0};assert(failed(ref owner,l,b,p,ref snap,limits,ref mut work) && work.structure<=16777216);
   work=request::Work{.structure=0,.bytes=1073741825-cost.bytes};assert(failed(ref owner,l,b,p,ref snap,limits,ref mut work) && work.bytes<=1073741824);
   work=request::Work{.structure=16777217,.bytes=0};assert(failed(ref owner,l,b,p,ref snap,limits,ref mut work) && work.structure==16777217);
   work=request::Work{.structure=0,.bytes=1073741825};assert(failed(ref owner,l,b,p,ref snap,limits,ref mut work) && work.bytes==1073741825);
  }
  // Drop every snapshot allocation while the detached packet stays alive.
  snap=request::snapshot_of([0;8],0,bytes::empty(),v::empty<request::Entry>());packet
 };
 // Mutate then release the original hashes, spans, and value bytes as well.
 {let changed=bytes::view_mut(ref mut input);changed[0]=255;changed[changed.len-1]^=255;}input=bytes::empty();
 discard io::write_bytes(request::digest(ref result));discard io::write_bytes(request::content(ref result));return 0;
}`

func TestCandidateStateGraphRequest(t *testing.T) {
	p := stateGraphHelperProgram(t, "empty")
	graph := stateGraphHelperEncode(p).bytes()
	limits := [4]uint64{1, 1, 1, 1} // Reads may exceed all write limits.
	base := contractstate.PersistentSnapshot{LayoutHash: p.Layout.Hash, Version: ^uint64(0)}
	sortEntries := func(s *contractstate.PersistentSnapshot) {
		slices.SortFunc(s.Entries, func(a, b contractstate.Entry) int { return bytes.Compare(a.Key[:], b.Key[:]) })
	}
	full := base
	values := [][]byte{{1}, {0, 0, 0, 255}, {255, 255, 255, 255, 255, 255, 255, 255}, bytes.Repeat([]byte{0x80}, 20), bytes.Repeat([]byte{0xfe}, 32), {0, 255, 128}, []byte("한글\x00🙂")}
	for i, raw := range values {
		full.Entries = append(full.Entries, contractstate.Entry{Key: p.Layout.Fields[i].Prefix, Value: raw})
	}
	sortEntries(&full)
	candidateAtlasProbeSize(t, "module_state_graph_request", stateRequestProbe(), 512<<10, func(t *testing.T, image string) {
		run := func(t *testing.T, s contractstate.PersistentSnapshot, lim [4]uint64, mode int, reject string) {
			t.Helper()
			want := stateRequestOracle(s, lim)
			input := binary.BigEndian.AppendUint64(nil, uint64(len(graph)))
			input = append(input, graph...)
			input = append(input, want...)
			// Four-byte ASCII mode, with the final digit read by the probe.
			arg := fmt.Sprintf("000%c", byte('0'+mode))

			code, output := candidateAtlasProcess(t, image, input, arg)
			if reject != "" {
				if code != 0 || output != reject {
					t.Fatalf("exit=%d output=%q want=%s", code, output, reject)
				}
				return
			}
			digest := sha256.Sum256(want)
			expected := append(digest[:], want...)
			if code != 0 || !bytes.Equal([]byte(output), expected) {
				t.Fatalf("exit=%d bytes=%d want=%d (full bytes and SHA256 differ)", code, len(output), len(expected))
			}
		}
		t.Run("empty", func(t *testing.T) { run(t, base, limits, 0, "") })
		t.Run("all-seven-codecs", func(t *testing.T) { run(t, full, limits, 0, "") })
		t.Run("64-bit-limits", func(t *testing.T) {
			run(t, full, [4]uint64{1<<63 - 1, ^uint64(0), 1<<63 - 1, 1<<63 - 1}, 0, "")
		})
		for i := range 4 {
			t.Run(fmt.Sprintf("zero-limit-%d", i), func(t *testing.T) { lim := limits; lim[i] = 0; run(t, base, lim, 0, "1001") })
			if i != 1 {
				t.Run(fmt.Sprintf("signed-overflow-limit-%d", i), func(t *testing.T) { lim := limits; lim[i] = 1 << 63; run(t, base, lim, 0, "1001") })
			}
		}
		for mode := 1; mode <= 5; mode++ {
			t.Run(fmt.Sprintf("invalid-shape-%d", mode), func(t *testing.T) { run(t, full, limits, mode, "1001") })
		}
		for i, name := range []string{"flag", "count", "total", "owner", "root", "label"} {
			t.Run("invalid-codec-"+name, func(t *testing.T) {
				s := base
				field := []int{0, 1, 2, 3, 4, 6}[i]
				s.Entries = []contractstate.Entry{{Key: p.Layout.Fields[field].Prefix, Value: []byte{2}}}
				if name == "label" {
					s.Entries[0].Value = []byte{0xed, 0xa0, 0x80}
				}
				run(t, s, limits, 0, "1001")
			})
		}
		t.Run("empty-present-bytes-text", func(t *testing.T) {
			s := base
			s.Entries = []contractstate.Entry{{Key: p.Layout.Fields[5].Prefix}, {Key: p.Layout.Fields[6].Prefix}}
			sortEntries(&s)
			run(t, s, limits, 0, "")
		})
		t.Run("duplicate", func(t *testing.T) {
			s := base
			s.Entries = []contractstate.Entry{full.Entries[0], full.Entries[0]}
			run(t, s, limits, 0, "1001")
		})
		t.Run("unsorted", func(t *testing.T) {
			s := base
			s.Entries = slices.Clone(full.Entries)
			slices.Reverse(s.Entries)
			run(t, s, limits, 0, "1001")
		})
		t.Run("opaque-unknown", func(t *testing.T) {
			s := base
			s.Entries = []contractstate.Entry{{Key: contractstate.Key{}, Value: []byte{255, 0, 254}}, {Key: contractstate.Key{0: 128}, Value: nil}, {Key: contractstate.Key{31: 1}, Value: []byte{0xc0, 0x80}}}
			sortEntries(&s)
			run(t, s, limits, 0, "")
		})
		t.Run("malformed-utf8", func(t *testing.T) {
			for _, value := range [][]byte{{0xc0, 0x80}, {0x80}, {0xe2, 0x82}, {0xf4, 0x90, 0x80, 0x80}, {0xff}} {
				s := base
				s.Entries = []contractstate.Entry{{Key: p.Layout.Fields[6].Prefix, Value: value}}
				run(t, s, limits, 0, "1001")
			}
		})
		t.Run("utf8-byte-budget", func(t *testing.T) {
			s := base
			s.Entries = []contractstate.Entry{{Key: p.Layout.Fields[6].Prefix, Value: bytes.Repeat([]byte("한글"), 4096)}}
			run(t, s, limits, 0, "")
		})
		t.Run("multiple-entry-payload-overflow", func(t *testing.T) {
			s := base
			s.Entries = []contractstate.Entry{{Value: bytes.Repeat([]byte{'a'}, (16<<20)-36)}, {Key: contractstate.Key{31: 1}}}
			run(t, s, limits, 6, "1002")
		})
		for _, n := range []int{65535, 65536, 65537} {
			t.Run(fmt.Sprintf("entry-count-%d", n), func(t *testing.T) {
				s := base
				s.Entries = make([]contractstate.Entry, n)
				for i := range s.Entries {
					binary.BigEndian.PutUint64(s.Entries[i].Key[24:], uint64(i))
				}
				reject := ""
				if n > 65536 {
					reject = "1002"
				}
				run(t, s, limits, 6, reject)
			})
		}
		for _, field := range []int{5, 6} {
			for _, delta := range []int{-1, 0, 1} {
				t.Run(fmt.Sprintf("16MiB-field-%d-delta-%d", field, delta), func(t *testing.T) {
					s := base
					s.Entries = []contractstate.Entry{{Key: p.Layout.Fields[field].Prefix, Value: bytes.Repeat([]byte{'a'}, (16<<20)-36+delta)}}
					reject := ""
					if delta > 0 {
						reject = "1002"
					}
					run(t, s, limits, 6, reject)
				})
			}
		}
	}, "module_state_graph_program", "module_state_declarations", "candidate_artifact_wire")
}

func TestCandidateStateGraphRequestSealedPacket(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"constructor", "E0103", "fn request::Packet forge(){return request::Packet{.data=bytes::empty(),.hash=bytes::empty()};}"},
		{"private-content", "E0103", "fn void change(ref mut request::Packet p){p.data=bytes::empty();}"},
		{"immutable-content", "E1008", "fn void change(ref request::Packet p){let ref mut u8[] data=ref mut request::content(p);}"},
		{"immutable-digest", "E1008", "fn void change(ref request::Packet p){let ref mut u8[] data=ref mut request::digest(p);}"},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_request")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_request as request;import std::bytes as bytes;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostics bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed request", &diagnostics)
			if code == ExitSuccess || !strings.Contains(diagnostics.String(), "main.fy:") || !strings.Contains(diagnostics.String(), trial.code) {
				t.Fatalf("sealed request escaped: %d %s", code, &diagnostics)
			}
		})
	}
}
