package cli

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"testing"
)

// This probe exercises framing only. It cannot manufacture the Verified proof
// or sealed Input required by response admission. Large views are grown at
// runtime, never emitted as large compile-time Fairy literals.
const stateGraphResponseWireProbe = `module main;
import module_state_graph_response_wire as wire;
import module_state_graph_response_common as common;
import std::bytes as bytes;import std::io as io;
fn usize next(ref u8[] raw,ref mut usize at){assert(*at<=raw.len && raw.len-*at>=8);let n=wire::number(raw,*at,8);*at+=8;return n;}
@effects(console) fn void field(usize n){discard io::write_usize(n);discard io::write(",");}
@effects(console) fn void head(common::Header h){discard io::write("H,");field(h.status);field(h.exit);field(h.trap);field(h.cost);field(h.operations);field(h.events);field(h.writes);field(h.write_bytes);field(h.payload);}
@effects(console) fn void observation(common::Event e){discard io::write("E,");field(e.operation);field(e.site);field(e.key_start);field(e.key_end);field(e.value_start);field(e.value_end);field(e.found as usize);}
@effects(console) fn void stored(common::Write w){discard io::write("W,");var usize i=0;while(i<8){field(w.key[i] as usize);i+=1;}field(if(w.deleted){1}else{0});field(w.start);field(w.end);}
@effects(console) fn void failure(u32 code){discard io::write("R,");field(code as usize);}
@effects(console) fn i32 main(){
 var input=bytes::empty();var u8[4096] chunk=[0;4096];
 while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 2;}};if(n==0){break;}let ref u8[] part=ref chunk;if(bytes::len(ref input)>1048576-n || !bytes::append(ref mut input,part[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);var usize at=0;let count=next(raw,ref mut at);var usize index=0;
 while(index<count){
  let mode=next(raw,ref mut at);let structure=next(raw,ref mut at);let byte_work=next(raw,ref mut at);var usize cursor=next(raw,ref mut at);let expand=next(raw,ref mut at);
  let size=next(raw,ref mut at);let request_size=next(raw,ref mut at);let digest_size=next(raw,ref mut at);
  assert(at<=raw.len && size<=raw.len-at);var output=switch(bytes::copy(raw[at..at+size])){case .ok(p):{p}case .err(_):{return 3;}};at+=size;
  assert(at<=raw.len && request_size<=raw.len-at);let request=raw[at..at+request_size];at+=request_size;
  assert(at<=raw.len && digest_size<=raw.len-at);let digest=raw[at..at+digest_size];at+=digest_size;
  if(expand!=0){assert(expand<=67108865);if(!bytes::resize_zeroed(ref mut output,expand).is_ok){return 3;}}
  let data=bytes::view(ref output);var work=common::Work{.structure,.bytes=byte_work};
  if(mode==0){switch(wire::header(data,request,digest,ref mut work)){case .ok(h):{head(h);}case .err(code):{failure(code);}};}
  else if(mode==1){switch(wire::event(data,ref mut cursor,ref mut work)){case .ok(e):{observation(e);}case .err(code):{failure(code);}};}
  else if(mode==2){switch(wire::write(data,ref mut cursor,ref mut work)){case .ok(w):{stored(w);}case .err(code):{failure(code);}};}
  else if(mode==3){
   switch(wire::header(data,request,digest,ref mut work)){
    case .err(code):{failure(code);}
    case .ok(h):{
     head(h);var bool good=true;var usize i=0;
     while(i<h.events && good){switch(wire::event(data,ref mut cursor,ref mut work)){case .ok(e):{observation(e);}case .err(code):{failure(code);good=false;}};i+=1;}
     i=0;while(i<h.writes && good){switch(wire::write(data,ref mut cursor,ref mut work)){case .ok(w):{stored(w);}case .err(code):{failure(code);good=false;}};i+=1;}
    }
   };
  }else if(mode==4){
   discard io::write("P,");field(wire::number(data,0,8));field(wire::number(data,0,4));field(wire::number(data,0,1));field(wire::number(data,8,0));let k=wire::key(data[8..40]);var usize i=0;while(i<8){field(k[i] as usize);i+=1;}
  }else{return 4;}
  discard io::write("M,");field(work.structure);field(work.bytes);field(cursor);discard io::write("\n");index+=1;
 }return if(at==raw.len){0}else{5};
}`

type stateGraphResponseWireCase struct {
	name                           string
	mode, structure, bytes, cursor uint64
	expand                         uint64
	output, request, digest        []byte
	want                           string
}

func stateGraphResponseWireHeader(request []byte) ([]byte, []byte) {
	digest := sha256.Sum256(request)
	output := make([]byte, 186)
	copy(output, "FYGROUT1")
	copy(output[8:112], request[8:112])
	copy(output[112:144], digest[:])
	return output, digest[:]
}

func stateGraphResponseWireHeadText(output []byte) string {
	return fmt.Sprintf("H,%d,%d,%d,%d,%d,%d,%d,%d,%d,", output[144], output[145],
		binary.BigEndian.Uint32(output[146:150]), binary.BigEndian.Uint64(output[150:158]),
		binary.BigEndian.Uint64(output[158:166]), binary.BigEndian.Uint32(output[166:170]),
		binary.BigEndian.Uint32(output[170:174]), binary.BigEndian.Uint64(output[174:182]),
		binary.BigEndian.Uint32(output[182:186]))
}

func stateGraphResponseWireCases(t *testing.T) []stateGraphResponseWireCase {
	t.Helper()
	const structureLimit, byteLimit = 16777216, 1073741824
	request := make([]byte, 195) // Digest includes bytes beyond the fixed request header.
	copy(request, "FYGRIN01")
	for i := 8; i < len(request); i++ {
		request[i] = byte(i*37 + 11)
	}
	base, digest := stateGraphResponseWireHeader(request)
	var cases []stateGraphResponseWireCase
	add := func(name string, mode uint64, output []byte, cursor uint64, text string, charged uint64) {
		cases = append(cases, stateGraphResponseWireCase{name: name, mode: mode, output: output, request: request, digest: digest, cursor: cursor,
			want: fmt.Sprintf("%sM,1,%d,%d,", text, charged, cursor)})
	}
	header := func(name string, output []byte, valid bool) {
		text := "R,1001,"
		if valid {
			text = stateGraphResponseWireHeadText(output)
		}
		add(name, 0, output, 0, text, 330)
	}
	mutate := func(at int, value byte) []byte {
		out := append([]byte(nil), base...)
		out[at] = value
		return out
	}
	header("empty-success", base, true)
	for n := 0; n < 186; n++ {
		add(fmt.Sprintf("header-truncated-%d", n), 0, base[:n], 0, "R,1001,", 0)
	}
	for n := 0; n < 152; n++ {
		add(fmt.Sprintf("request-truncated-%d", n), 0, base, 0, "R,1001,", 0)
		cases[len(cases)-1].request = request[:n]
	}
	for _, n := range []int{0, 31, 33} {
		add(fmt.Sprintf("digest-width-%d", n), 0, base, 0, "R,1001,", 0)
		cases[len(cases)-1].digest = make([]byte, n)
	}
	for i := 0; i < 144; i++ {
		header(fmt.Sprintf("forged-output-identity-%d", i), mutate(i, base[i]^1), false)
	}
	for _, i := range []int{0, 7, 8, 39, 40, 71, 72, 103, 104, 111} {
		header(fmt.Sprintf("forged-request-identity-%d", i), base, false)
		changed := append([]byte(nil), request...)
		changed[i] ^= 1
		cases[len(cases)-1].request = changed
	}
	for _, i := range []int{112, 151, 152, len(request) - 1} {
		header(fmt.Sprintf("actual-digest-covers-request-%d", i), base, false)
		changed := append([]byte(nil), request...)
		changed[i] ^= 1
		hash := sha256.Sum256(changed)
		cases[len(cases)-1].request, cases[len(cases)-1].digest = changed, hash[:]
	}
	minimumRequest := request[:152]
	minimumOutput, minimumDigest := stateGraphResponseWireHeader(minimumRequest)
	header("minimum-request", minimumOutput, true)
	cases[len(cases)-1].request, cases[len(cases)-1].digest = minimumRequest, minimumDigest
	for status := byte(0); status <= 11; status++ {
		header(fmt.Sprintf("status-%d", status), mutate(144, status), true)
	}
	for exit := byte(1); exit <= 2; exit++ {
		header(fmt.Sprintf("success-exit-%d", exit), mutate(145, exit), true)
	}
	for _, tc := range []struct {
		name  string
		at    int
		value byte
	}{
		{"status-range", 144, 12}, {"exit-range", 145, 3}, {"payload-absent", 185, 1},
		{"event-minimum", 169, 1}, {"write-minimum", 173, 1},
	} {
		header(tc.name, mutate(tc.at, tc.value), false)
	}
	for _, at := range []int{150, 158, 174} {
		out := append([]byte(nil), base...)
		binary.BigEndian.PutUint64(out[at:at+8], math.MaxUint64)
		header(fmt.Sprintf("u64-max-%d", at), out, true)
	}
	for _, status := range []byte{0, 1, 11} {
		for _, at := range []int{145, 149, 173, 181} {
			out := mutate(144, status)
			out[at] = 1
			if status == 0 {
				out[145] = 1 // Reject exits cannot claim writes/write_bytes.
			}
			header(fmt.Sprintf("non-success-%d-field-%d", status, at), out, status == 0 && (at == 145 || at == 149))
		}
	}
	for _, counts := range [][2]uint32{{1, 1}, {math.MaxUint32, 0}, {0, math.MaxUint32}, {math.MaxUint32, math.MaxUint32}} {
		out := append(append([]byte(nil), base...), make([]byte, 58)...)
		binary.BigEndian.PutUint32(out[166:170], counts[0])
		binary.BigEndian.PutUint32(out[170:174], counts[1])
		binary.BigEndian.PutUint32(out[182:186], 58)
		header(fmt.Sprintf("counts-%d-%d", counts[0], counts[1]), out, counts == [2]uint32{1, 1})
		if counts == [2]uint32{1, 1} {
			out = append([]byte(nil), out[:len(out)-1]...)
			binary.BigEndian.PutUint32(out[182:186], 57)
			header("combined-count-minimum-one-short", out, false)
		}
	}
	header("trailing-byte", append(append([]byte(nil), base...), 0), false)
	for _, size := range []uint32{67108864, 67108865} {
		out := append([]byte(nil), base...)
		binary.BigEndian.PutUint32(out[182:186], size-186)
		header(fmt.Sprintf("response-cap-%d", size), out, size == 67108864)
		c := &cases[len(cases)-1]
		c.expand = uint64(size)
		if size > 67108864 {
			c.want = "R,1001,M,1,0,0,"
		}
	}

	event := make([]byte, 26)
	binary.BigEndian.PutUint64(event[0:8], math.MaxUint64)
	binary.BigEndian.PutUint32(event[8:12], math.MaxUint32)
	binary.BigEndian.PutUint32(event[12:16], 3)
	binary.BigEndian.PutUint32(event[16:20], 2)
	event[20] = 1
	copy(event[21:], []byte{7, 8, 9, 10, 11})
	eventText := "E,18446744073709551615,4294967295,21,24,24,26,1,"
	add("event-u64-max-offsets", 1, event, 26, eventText, 21)
	cases[len(cases)-1].cursor = 0
	for n := 0; n < 26; n++ {
		charged := uint64(21)
		if n < 21 {
			charged = 0
		}
		add(fmt.Sprintf("event-truncated-%d", n), 1, event[:n], 0, "R,1001,", charged)
	}
	for _, at := range []int{12, 16} {
		out := append([]byte(nil), event...)
		binary.BigEndian.PutUint32(out[at:at+4], math.MaxUint32)
		add(fmt.Sprintf("event-forged-length-%d", at), 1, out, 0, "R,1001,", 21)
	}
	zeroEvent := make([]byte, 21)
	add("empty-event", 1, zeroEvent, 21, "E,0,0,21,21,21,21,0,", 21)
	cases[len(cases)-1].cursor = 0
	add("event-nonzero-cursor", 1, append([]byte{99, 98}, event...), 28, "E,18446744073709551615,4294967295,23,26,26,28,1,", 21)
	cases[len(cases)-1].cursor = 2

	write := make([]byte, 40)
	var keyText strings.Builder
	for i := 0; i < 32; i++ {
		write[i] = byte(i*7 + 3)
	}
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&keyText, "%d,", binary.BigEndian.Uint32(write[i*4:i*4+4]))
	}
	binary.BigEndian.PutUint32(write[33:37], 3)
	copy(write[37:], []byte{20, 30, 40})
	writeText := "W," + keyText.String() + "0,37,40,"
	add("write-key-and-offsets", 2, write, 40, writeText, 37)
	cases[len(cases)-1].cursor = 0
	for n := 0; n < 40; n++ {
		charged := uint64(37)
		if n < 37 {
			charged = 0
		}
		add(fmt.Sprintf("write-truncated-%d", n), 2, write[:n], 0, "R,1001,", charged)
	}
	for _, flag := range []byte{1, 2, 255} {
		out := append([]byte(nil), write[:37]...)
		out[32] = flag
		binary.BigEndian.PutUint32(out[33:37], 0)
		text, cursor := "R,1001,", uint64(0)
		if flag == 1 {
			text, cursor = "W,"+keyText.String()+"1,37,37,", 37
		}
		add(fmt.Sprintf("deleted-flag-%d", flag), 2, out, cursor, text, 37)
		cases[len(cases)-1].cursor = 0
	}
	for _, n := range []uint32{4, math.MaxUint32} {
		out := append([]byte(nil), write...)
		binary.BigEndian.PutUint32(out[33:37], n)
		add(fmt.Sprintf("write-forged-length-%d", n), 2, out, 0, "R,1001,", 37)
	}
	add("write-nonzero-cursor", 2, append([]byte{99}, write...), 41, "W,"+keyText.String()+"0,38,41,", 37)
	cases[len(cases)-1].cursor = 1
	for _, mode := range []uint64{1, 2} {
		for _, cursor := range []uint64{40, 41, math.MaxUint64} {
			add(fmt.Sprintf("cursor-%d-%d", mode, cursor), mode, write, cursor, "R,1001,", 0)
		}
	}

	// Charge only bytes actually inspected by the parser: payloads are borrowed
	// offsets, not scanned/copied here. All three calls share the same ledgers.
	combined := append(append(append([]byte(nil), base...), event...), write...)
	binary.BigEndian.PutUint32(combined[166:170], 1)
	binary.BigEndian.PutUint32(combined[170:174], 1)
	binary.BigEndian.PutUint32(combined[182:186], uint32(len(combined)-186))
	chainHead := stateGraphResponseWireHeadText(combined)
	chainEvent := "E,18446744073709551615,4294967295,207,210,210,212,1,"
	chainWrite := "W," + keyText.String() + "0,249,252,"
	chainText := chainHead + chainEvent + chainWrite
	cases = append(cases, stateGraphResponseWireCase{name: "shared-ledgers", mode: 3, cursor: 186, output: combined, request: request, digest: digest, want: chainText + "M,3,388,252,"})
	for _, mode := range []uint64{0, 1, 2, 3} {
		out, cursor, text, cost, calls, end := base, uint64(0), stateGraphResponseWireHeadText(base), uint64(330), uint64(1), uint64(0)
		if mode == 1 {
			out, text, cost, end = event, eventText, 21, 26
		} else if mode == 2 {
			out, text, cost, end = write, writeText, 37, 40
		} else if mode == 3 {
			out, cursor, text, cost, calls, end = combined, 186, chainText, 388, 3, 252
		}
		for _, meter := range []string{"exact", "structure-short", "bytes-short", "structure-overdrawn", "bytes-overdrawn"} {
			c := stateGraphResponseWireCase{name: fmt.Sprintf("meter-%d-%s", mode, meter), mode: mode, output: out, request: request, digest: digest, cursor: cursor,
				structure: structureLimit - calls, bytes: byteLimit - cost}
			switch meter {
			case "exact":
				c.want = fmt.Sprintf("%sM,%d,%d,%d,", text, structureLimit, byteLimit, end)
			case "structure-short":
				c.structure++
				if mode == 3 {
					c.want = fmt.Sprintf("%s%sR,1002,M,%d,%d,212,", chainHead, chainEvent, structureLimit, byteLimit-37)
				} else {
					c.want = fmt.Sprintf("R,1002,M,%d,%d,%d,", c.structure, c.bytes, cursor)
				}
			case "bytes-short":
				c.bytes++
				if mode == 3 {
					c.want = fmt.Sprintf("%s%sR,1002,M,%d,%d,212,", chainHead, chainEvent, structureLimit, byteLimit-36)
				} else {
					c.want = fmt.Sprintf("R,1002,M,%d,%d,%d,", structureLimit, c.bytes, cursor)
				}
			case "structure-overdrawn":
				c.structure = math.MaxUint64
				c.want = fmt.Sprintf("R,1002,M,%d,%d,%d,", c.structure, c.bytes, cursor)
			case "bytes-overdrawn":
				c.bytes = math.MaxUint64
				c.want = fmt.Sprintf("R,1002,M,%d,%d,%d,", c.structure, c.bytes, cursor)
			}
			cases = append(cases, c)
		}
	}
	// A malformed final frame keeps charges from the two preceding successes.
	badChain := append([]byte(nil), combined...)
	badChain[244] = 2 // write starts at 212; deletion flag is at +32.
	cases = append(cases, stateGraphResponseWireCase{name: "shared-malformed-final-frame", mode: 3, cursor: 186, output: badChain, request: request, digest: digest,
		want: chainHead + chainEvent + "R,1001,M,3,388,212,"})
	primitive := append([]byte{255, 255, 255, 255, 255, 255, 255, 255}, write[:32]...)
	cases = append(cases, stateGraphResponseWireCase{name: "number-and-key", mode: 4, output: primitive,
		want: "P,18446744073709551615,4294967295,255,0," + keyText.String() + "M,0,0,0,"})
	return cases
}

func TestCandidateStateGraphResponseWire(t *testing.T) {
	cases := stateGraphResponseWireCases(t)
	packet := binary.BigEndian.AppendUint64(nil, uint64(len(cases)))
	for _, c := range cases {
		for _, n := range []uint64{c.mode, c.structure, c.bytes, c.cursor, c.expand, uint64(len(c.output)), uint64(len(c.request)), uint64(len(c.digest))} {
			packet = binary.BigEndian.AppendUint64(packet, n)
		}
		packet = append(packet, c.output...)
		packet = append(packet, c.request...)
		packet = append(packet, c.digest...)
	}
	if len(packet) > 1<<20 {
		t.Fatalf("bounded native probe packet is too large: %d", len(packet))
	}
	t.Logf("%d bounded framing/meter cases per Windows profile; %d input bytes", len(cases), len(packet))
	candidateAtlasProbeSize(t, "module_state_graph_response_wire", stateGraphResponseWireProbe, 4<<10, func(t *testing.T, image string) {
		code, output := candidateAtlasProcess(t, image, packet)
		if code != 0 {
			t.Fatalf("wire probe exit=%d output=%q", code, output)
		}
		lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
		if len(lines) != len(cases) {
			t.Fatalf("wire probe returned %d records, want %d", len(lines), len(cases))
		}
		for i, c := range cases {
			if lines[i] != c.want {
				t.Errorf("%s: got %q want %q", c.name, lines[i], c.want)
			}
		}
	})
}
