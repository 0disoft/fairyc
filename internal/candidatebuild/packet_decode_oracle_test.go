package candidatebuild

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// The opt-in binary ledger is consumed by the authored Fairy reader tests.
// Expected admission always comes from the actual Go Decode implementation.
func TestPacketDecodeOracle(t *testing.T) {
	header := "#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n#endif\n"
	literals := "/* fairyc.candidate-project-literals-c/v1 */\n#include \"fy_project.h\"\n"
	unit := "/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\nextern int32_t fy_fn_1(void){return 0;}\nint32_t fy_candidate_entry(void){return fy_fn_1();}\n"
	encode := func(h, l string, units ...string) []byte {
		var out bytes.Buffer
		out.WriteString("FYCPARTS1\n")
		fmt.Fprintf(&out, "%d\n%s%d\n%s%d\n", len(h), h, len(l), l, len(units))
		for _, u := range units {
			fmt.Fprintf(&out, "%d\n%s", len(u), u)
		}
		return out.Bytes()
	}
	type record struct {
		Name string `json:"name"`
		Data []byte `json:"-"`
		Want bool   `json:"accepted"`
	}
	var cases []record
	add := func(name string, data []byte, want bool) { cases = append(cases, record{name, data, want}) }
	valid := encode(header, literals, unit)
	add("minimal", valid, true)
	for i := range valid {
		add(fmt.Sprintf("truncated-%d", i), valid[:i], false)
	}
	add("trailing", append(bytes.Clone(valid), 'x'), false)
	add("magic", bytes.Replace(valid, []byte("FYCPARTS1"), []byte("FYCPARTS2"), 1), false)
	for _, n := range []string{"", "00", "01", "-1", "+1", "1\r", "999999999", "2097153"} {
		add("number-"+n, append([]byte("FYCPARTS1\n"+n+"\n"), valid[10:]...), false)
	}
	add("empty-header", encode("", literals, unit), false)
	add("header-prefix", encode("x"+header, literals, unit), false)
	add("header-suffix", encode(header+"x", literals, unit), false)
	add("literal-marker", encode(header, "x"+literals, unit), false)
	add("unit-marker", encode(header, literals, "x"+unit), false)
	add("empty-units", encode(header, literals), false)
	add("too-many-units", encode(header, literals, strings.Split(strings.Repeat("x,", 4096)+"x", ",")...), false)
	add("part-boundary", encode(header, literals, unit+strings.Repeat(" ", 2097152-len(unit))), true)
	add("part-too-large", encode(header, literals, unit+strings.Repeat(" ", 2097153-len(unit))), false)
	add("header-static-literal", encode(strings.TrimSuffix(header, "#endif\n")+"static const uint8_t fy_unit_literal_\n#endif\n", literals, unit), false)
	add("unit-local-literal", encode(header, literals, unit+"const uint8_t fy_unit_literal_"), false)
	add("no-entry", encode(header, literals, strings.ReplaceAll(unit, "int32_t fy_candidate_entry(void){", "int32_t other(void){")), false)
	add("duplicate-entry", encode(header, literals, unit+"int32_t fy_candidate_entry(void){"), false)
	add("no-frame", encode(header, literals, strings.ReplaceAll(unit, "extern int32_t", "static int32_t")), false)
	for _, f := range []struct {
		text string
		want bool
	}{
		{"0", false}, {"16384", true}, {"16385", false}, {"0001", true}, {strings.Repeat("9", 50), false},
	} {
		add("frame-"+f.text, encode(header, literals, strings.ReplaceAll(unit, "fy_fn_1", "fy_fn_"+f.text)), f.want)
	}
	add("duplicate-frame", encode(header, literals, unit, strings.ReplaceAll(unit, "int32_t fy_candidate_entry(void){", "int32_t other(void){")), false)
	add("ascending-frames", encode(header, literals, unit, strings.ReplaceAll(strings.ReplaceAll(unit, "fy_fn_1", "fy_fn_2"), "int32_t fy_candidate_entry(void){", "int32_t other(void){")), true)
	add("greedy-frame", encode(header, literals, unit+"extern int32_t fy_fn_0(void){return 0;} extern int32_t fy_fn_2(void){return 0;}\n"), true)
	digest := func(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
	emptyID, zeroID, oneID := digest(nil), digest([]byte{0}), digest([]byte{1})
	withDeclarations := func(s string) string { return strings.TrimSuffix(header, "#endif\n") + s + "#endif\n" }
	declaration := func(id string) string { return "extern const uint8_t fy_unit_literal_" + id + "[];\n" }
	definition := func(id, values string) string {
		return "const uint8_t fy_unit_literal_" + id + "[]={" + values + "};\n"
	}
	for _, lit := range []struct {
		name, id, values string
		want             bool
	}{
		{"one", oneID, "1", true}, {"zero", zeroID, "0", true}, {"empty", emptyID, "0", true},
		{"leading-zero-byte", oneID, "0001", true}, {"byte-too-large", oneID, "256", false},
		{"byte-overflow", oneID, strings.Repeat("9", 40), false}, {"empty-byte", oneID, ",1", false},
		{"trailing-comma", oneID, "1,", false}, {"double-comma", oneID, "1,,1", false},
		{"bad-byte", oneID, "a", false}, {"digest-mismatch", oneID, "2", false},
	} {
		add(lit.name, encode(withDeclarations(declaration(lit.id)), literals+definition(lit.id, lit.values), unit), lit.want)
	}
	add("missing-definition", encode(withDeclarations(declaration(oneID)), literals, unit), false)
	add("missing-declaration", encode(header, literals+definition(oneID, "1"), unit), false)
	add("duplicate-literals", encode(withDeclarations(declaration(oneID)+declaration(oneID)), literals+definition(oneID, "1")+definition(oneID, "1"), unit), false)
	add("ascending-literals", encode(withDeclarations(declaration(oneID)+declaration(emptyID)), literals+definition(oneID, "1")+definition(emptyID, "0"), unit), true)
	add("unordered-literals", encode(withDeclarations(declaration(emptyID)+declaration(oneID)), literals+definition(emptyID, "0")+definition(oneID, "1"), unit), false)
	add("different-identities", encode(withDeclarations(declaration(emptyID)), literals+definition(oneID, "1"), unit), false)
	add("unanchored-definition", encode(withDeclarations(declaration(oneID)), literals+" "+definition(oneID, "1"), unit), false)
	add("unrecognized-C-record", encode(header, literals+definition(oneID, "a"), unit), true)
	// Exercise both the real bootstrap-sized packet and the fresh-file reader.
	filePacket := valid
	if path := os.Getenv("FAIRYC_PACKET_DECODE_FULL_PACKET"); path != "" {
		var err error
		filePacket, err = os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		add("full-compiler-packet", filePacket, true)
	}
	var ledger bytes.Buffer
	ledger.WriteString("FYCPACKETTEST1\n")
	write := func(n int) {
		if err := binary.Write(&ledger, binary.LittleEndian, uint32(n)); err != nil {
			t.Fatal(err)
		}
	}
	write(len(cases))
	for _, c := range cases {
		p, err := Decode(c.Data)
		if (err == nil) != c.Want {
			t.Fatalf("%s: expected accepted=%t, got %v", c.Name, c.Want, err)
		}
		if err == nil {
			ledger.WriteByte(1)
		} else {
			ledger.WriteByte(0)
		}
		write(len(c.Data))
		write(len(p.Header))
		write(len(p.Literals))
		write(len(p.Units))
		ledger.Write(c.Data)
	}
	if path := os.Getenv("FAIRYC_PACKET_DECODE_FIXTURE"); path != "" {
		if err := os.WriteFile(path, ledger.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".packet", filePacket, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".invalid", []byte("not a C packet"), 0600); err != nil {
			t.Fatal(err)
		}
		data, err := json.MarshalIndent(cases, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path+".json", append(data, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d packet cases matched Go admission", len(cases))
}
