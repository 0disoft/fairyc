package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"testing"
)

const stateLocalBase64Probe = `module main;
import module_state_local_base64 as codec;
import std::bytes as bytes;import std::io as io;
fn u8 octet(usize value){let d=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (d[(value/16)%16]<<4)|d[value%16];}
fn usize number(ref u8[] raw,usize at){var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[at+i] as usize);i+=1;}return n;}
fn void put(ref mut u8[] raw,usize at,usize n){var usize rest=n;var usize i=0;while(i<8){raw[at+7-i]=octet(rest%256);rest/=256;i+=1;}}
@effects(console) fn i32 main(){
    var input=bytes::empty();var u8[4096] chunk=[0;4096];
    while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>67108864-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
    var usize offset=0;var usize work=0;
    while(offset<bytes::len(ref input)){
        if(bytes::len(ref input)-offset<32){return 3;}
        let mode=number(bytes::view(ref input),offset);let maximum=number(bytes::view(ref input),offset+8);
        let start=number(bytes::view(ref input),offset+16);let size=number(bytes::view(ref input),offset+24);offset+=32;
        if(size>bytes::len(ref input)-offset){return 4;}if(mode<2){work=start;}
        var u32 code=0;
        var output=switch(if(mode%2==0){codec::encode(bytes::view(ref input)[offset..offset+size],ref mut work)}else{codec::decode(bytes::view(ref input)[offset..offset+size],maximum,ref mut work)}){
            case .ok(value):{value}case .err(c):{code=c;bytes::empty()}
        };
        // Mutate the borrowed source after return: successful results own their bytes.
        {let raw=bytes::view_mut(ref mut input);var usize i=offset;while(i<offset+size){raw[i]=0;i+=1;}}
        var u8[24] header=[0;24];put(ref mut header,0,code as usize);put(ref mut header,8,work);put(ref mut header,16,bytes::len(ref output));
        if(!io::write_bytes(ref header).is_ok || !io::write_bytes(bytes::view(ref output)).is_ok){return 5;}
        offset+=size;
    }
    return 0;
}`

func TestCandidateStateLocalBase64(t *testing.T) {
	const budget uint64 = 1 << 30
	const maximum = 16 << 20
	type fixture struct {
		name                         string
		mode, max, start, code, work uint64
		input, want                  []byte
	}
	var small []fixture
	valid := func(name string, raw []byte) {
		encoded := []byte(base64.StdEncoding.EncodeToString(raw))
		decoded, err := base64.StdEncoding.Strict().DecodeString(string(encoded))
		if err != nil || !bytes.Equal(raw, decoded) {
			t.Fatalf("Go oracle: %v", err)
		}
		encCost := uint64(len(raw) + 2*len(encoded))
		decCost := uint64(2*len(encoded) + 2*len(raw))
		if len(encoded) != 0 {
			decCost += 4
		}
		small = append(small,
			fixture{name: name + "/encode", max: maximum, start: 17, work: 17 + encCost, input: raw, want: encoded},
			fixture{name: name + "/decode", mode: 1, max: maximum, start: 29, work: 29 + decCost, input: encoded, want: raw})
	}
	for _, s := range []string{"", "f", "fo", "foo", "foob", "fooba", "foobar", "\x00\xff\x80\x00\xfe"} {
		valid(fmt.Sprintf("RFC-or-binary-%x", s), []byte(s))
	}
	for n := 0; n <= 129; n++ {
		raw := make([]byte, n)
		for i := range raw {
			raw[i] = byte(i*73 + n*17)
		}
		valid(fmt.Sprintf("mod3-%d", n), raw)
	}
	for _, s := range []string{"A", "AA", "AAA", "AAAAA", "=AAA", "A=AA", "AA=A", "====", "A===", "Zg=", "Zg===", "Zg==AAAA", "Zm8=AAAA", "Zm=v", "Zg==\n", "\r\nZg==", "Z g==", "Zg\t==", "Zm9v\n", "Zm9v!===", "Zm9v-___", "Zm9v\x00", "Zm9v\xff", "Zh==", "Zv==", "Zm9=", "Zm+=", "/x==", "//9="} {
		// Go Strict intentionally permits CR/LF; this local format forbids them too.
		_, err := base64.StdEncoding.Strict().DecodeString(s)
		if err == nil && !bytes.ContainsAny([]byte(s), "\r\n") {
			t.Fatalf("invalid oracle fixture %q", s)
		}
		small = append(small, fixture{name: fmt.Sprintf("malformed-%q", s), mode: 1, max: maximum, start: 41, code: 1001, work: 41 + uint64(len(s)) + 4, input: []byte(s)})
	}
	small = append(small,
		fixture{name: "padding-adjusted-one-byte", mode: 1, max: 1, input: []byte("/w=="), want: []byte{255}, work: 14},
		fixture{name: "padding-adjusted-two-bytes", mode: 1, max: 2, input: []byte("//8="), want: []byte{255, 255}, work: 16},
		fixture{name: "decoded-over-maximum", mode: 1, max: 1, input: []byte("//8="), code: 1002, work: 8},
		fixture{name: "decoded-unpadded-over-maximum", mode: 1, max: 2, input: []byte("////"), code: 1002, work: 8})
	for _, max := range []uint64{0, maximum + 1, ^uint64(0)} {
		small = append(small, fixture{name: fmt.Sprintf("invalid-maximum-%d", max), mode: 1, max: max, start: 11, code: 1002, work: 11})
	}
	for _, mode := range []uint64{0, 1} {
		for _, start := range []uint64{budget, budget + 1, ^uint64(0)} {
			code := uint64(0)
			if start > budget {
				code = 1002
			}
			small = append(small, fixture{name: fmt.Sprintf("empty-mode-%d-budget-%d", mode, start), mode: mode, max: maximum, start: start, code: code, work: start})
		}
	}
	small = append(small,
		fixture{name: "validation-exact-malformed", mode: 1, max: maximum, start: budget - 8, input: []byte("Zh=="), code: 1001, work: budget},
		fixture{name: "validation-one-short-malformed", mode: 1, max: maximum, start: budget - 7, input: []byte("Zh=="), code: 1002, work: budget - 7},
		fixture{name: "overdrawn-nonempty-encode", start: budget + 1, input: []byte{0}, code: 1002, work: budget + 1},
		fixture{name: "overdrawn-nonempty-decode", mode: 1, max: maximum, start: budget + 1, input: []byte("AA=="), code: 1002, work: budget + 1})
	for _, raw := range [][]byte{{255}, {0, 255}, {0, 255, 128}} {
		encoded := []byte(base64.StdEncoding.EncodeToString(raw))
		for _, mode := range []uint64{0, 1} {
			input, want := raw, encoded
			cost := uint64(len(raw) + 2*len(encoded))
			if mode == 1 {
				input, want = encoded, raw
				cost = uint64(2*len(encoded) + 2*len(raw) + 4)
			}
			small = append(small, fixture{name: fmt.Sprintf("exact-mode-%d-mod-%d", mode, len(raw)), mode: mode, max: maximum, start: budget - cost, work: budget, input: input, want: want})
			remaining := budget - cost + 1
			if mode == 1 {
				remaining += uint64(len(encoded) + 4)
			}
			small = append(small, fixture{name: fmt.Sprintf("one-short-mode-%d-mod-%d", mode, len(raw)), mode: mode, max: maximum, start: budget - cost + 1, code: 1002, work: remaining, input: input})
		}
	}
	// Failed validation retains scan work; subsequent calls share that same ledger.
	small = append(small,
		fixture{name: "shared-malformed", mode: 1, max: maximum, start: budget - 8, input: []byte("Zh=="), code: 1001, work: budget},
		fixture{name: "shared-after-error", mode: 2, max: maximum, input: []byte{1}, code: 1002, work: budget},
		fixture{name: "shared-empty-after-error", mode: 3, max: maximum, work: budget},
		fixture{name: "shared-encode", input: []byte{0}, want: []byte("AA=="), work: 9},
		fixture{name: "shared-decode", mode: 3, max: maximum, input: []byte("AA=="), want: []byte{0}, work: 23})
	check := func(t *testing.T, image string, cases []fixture) {
		t.Helper()
		var input []byte
		for _, c := range cases {
			for _, n := range []uint64{c.mode, c.max, c.start, uint64(len(c.input))} {
				input = binary.BigEndian.AppendUint64(input, n)
			}
			input = append(input, c.input...)
		}
		code, output := candidateAtlasProcess(t, image, input)
		if code != 0 {
			t.Fatalf("probe exit=%d output bytes=%d", code, len(output))
		}
		remaining := []byte(output)
		for _, c := range cases {
			if len(remaining) < 24 {
				t.Fatalf("%s: missing result header", c.name)
			}
			code, work, size := binary.BigEndian.Uint64(remaining), binary.BigEndian.Uint64(remaining[8:]), binary.BigEndian.Uint64(remaining[16:])
			remaining = remaining[24:]
			if code != c.code || work != c.work || size != uint64(len(c.want)) {
				t.Fatalf("%s: code/work/size=%d/%d/%d want=%d/%d/%d", c.name, code, work, size, c.code, c.work, len(c.want))
			}
			if size > uint64(len(remaining)) || !bytes.Equal(remaining[:size], c.want) {
				t.Fatalf("%s: bytes differ from independent Go base64 oracle", c.name)
			}
			remaining = remaining[size:]
		}
		if len(remaining) != 0 {
			t.Fatalf("extra output: %d bytes", len(remaining))
		}
		t.Logf("verified %d cases against Go base64.StdEncoding; exact bytes, ledger, source ownership", len(cases))
	}
	candidateAtlasProbeSize(t, "module_state_local_base64", stateLocalBase64Probe, 32<<10, func(t *testing.T, image string) {
		t.Run("small-batch", func(t *testing.T) { check(t, image, small) })
		for _, n := range []int{65536, 65537, 65538, maximum} {
			t.Run(fmt.Sprintf("large-%d", n), func(t *testing.T) {
				raw := make([]byte, n)
				for i := range raw {
					raw[i] = byte(i*131 + i/251)
				}
				encoded := []byte(base64.StdEncoding.EncodeToString(raw))
				check(t, image, []fixture{{name: "encode", input: raw, want: encoded, work: uint64(n + 2*len(encoded))}})
				check(t, image, []fixture{{name: "decode", mode: 1, max: uint64(n), input: encoded, want: raw, work: uint64(2*len(encoded) + 2*n + 4)}})
			})
		}
		t.Run("hard-limits", func(t *testing.T) {
			check(t, image, []fixture{{name: "raw-over-16MiB", input: make([]byte, maximum+1), code: 1002}})
			check(t, image, []fixture{{name: "encoded-over-22369624", mode: 1, max: maximum, input: make([]byte, 22369625), code: 1002}})
			// Same legal encoded length as a 16MiB value, but padding represents 16MiB+1/2.
			encoded := bytes.Repeat([]byte{'A'}, 22369624)
			encoded[len(encoded)-1] = '='
			check(t, image, []fixture{{name: "decoded-16MiB-plus-one", mode: 1, max: maximum, input: encoded, code: 1002, work: 22369628}})
			encoded[len(encoded)-1] = 'A'
			check(t, image, []fixture{{name: "decoded-16MiB-plus-two", mode: 1, max: maximum, input: encoded, code: 1002, work: 22369628}})
		})
	})
}
