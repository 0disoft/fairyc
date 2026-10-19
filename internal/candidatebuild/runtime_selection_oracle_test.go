package candidatebuild

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cstdlibcontract"
)

// The opt-in fixture binds the authored selector to live Go tables and error
// behavior. Go prepares oracle data only; it is not invoked by the selector.
func TestCandidateRuntimeSelectionOracle(t *testing.T) {
	path := os.Getenv("FAIRYC_RUNTIME_ORACLE")
	if path == "" {
		t.Skip("set FAIRYC_RUNTIME_ORACLE to export comparison cases")
	}
	readKeys := func(path, variable string) []string {
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		ast.Inspect(f, func(n ast.Node) bool {
			v, ok := n.(*ast.ValueSpec)
			if !ok || len(v.Names) != 1 || v.Names[0].Name != variable {
				return true
			}
			for _, element := range v.Values[0].(*ast.CompositeLit).Elts {
				literal := element.(*ast.KeyValueExpr).Key.(*ast.BasicLit).Value
				names = append(names, strings.Trim(literal, "\""))
			}
			return false
		})
		if len(names) == 0 {
			t.Fatalf("missing table %s", variable)
		}
		sort.Strings(names)
		return names
	}
	standard := readKeys("../cstdlibcontract/contract.go", "windowsMSVCSymbolComponents")
	admitted := readKeys("../cstdlibcontract/contract.go", "symbolComponents")
	admitted = append(admitted, readKeys("../cstdlibcontract/contract.go", "windowsOnlySymbolComponents")...)
	hosts := readKeys("../candidatehost/runtime.go", "admitted")
	type input struct {
		value        string
		trap, direct bool
	}
	cases := []input{{}, {value: "fy_private_vector_"}, {value: "/* fy_private_vector_ */", trap: true}}
	for _, name := range append(append(append([]string{}, standard...), admitted...), "malloc", "fy_std_unknown_v0", "fy_std_") {
		cases = append(cases, input{value: name, direct: true}, input{value: name + "( );"}, input{value: name + "( );", trap: true})
	}
	for _, name := range hosts {
		cases = append(cases, input{value: name, direct: true})
		for _, gap := range []string{"", " ", "\t", "\r\n", "\f", "\f \n", "\v"} {
			cases = append(cases, input{value: "extern void " + name + gap + "(void);", trap: len(gap)%2 == 1})
		}
	}
	for _, prefix := range []string{"", " ", "!", "_", "0", "a", "é", "\x00"} {
		for _, gap := range []string{"", " ", "\t", "\r\n", "\f", "\v", "/*x*/"} {
			for _, name := range []string{"fy_std_buffer_drop_v0", "fy_std_unknown_v0", "fairyc_private_windows_ticks_v1", "fairyc_private_windows_unknown_v1"} {
				cases = append(cases, input{value: prefix + name + gap + "()"})
			}
		}
	}
	cases = append(cases, input{value: "fairyc_private_windows_z_v1(); fairyc_private_windows_a_v1();"},
		input{value: "fy_std_z_v0(); fy_std_a_v0();"}, input{value: "fairyc_private_windows_z_v1(); fy_std_z_v0();"},
		input{value: "fy_std_(); fairyc_private_windows_(); fy_std_buffer_drop_v0"},
		input{value: "// fy_std_unknown_v0()\n"}, input{value: "\"fy_std_unknown_v0()\""})
	// Every pair of component representatives plus deterministic larger sets
	// exercises dependency closure and duplicates without depending on map order.
	reps := []string{"fy_std_stdout_write_all_v0", "fy_std_file_open_v0", "fy_std_windows_read_dir_open_v0", "fy_std_process_arg_v0", "fy_std_system_heap_allocate_v0", "fy_std_bytes_match_at_v0", "fy_std_buffer_drop_v0", "fy_std_windows_file_read_append_v0", "fy_std_arena_drop_v0", "fy_std_text_utf8_valid_v0", "fy_std_string_copy_v0"}
	for _, a := range reps {
		for _, b := range reps {
			cases = append(cases, input{value: a + "();" + b + "();" + a + "();"})
		}
	}
	rng := rand.New(rand.NewSource(17))
	for i := 0; i < 256; i++ {
		var header strings.Builder
		for _, name := range standard {
			if rng.Intn(4) == 0 {
				header.WriteString(name + "();\n")
			}
		}
		if i%2 == 0 {
			header.WriteString("fy_private_vector_")
		}
		if i%3 == 0 {
			header.WriteString(" " + hosts[rng.Intn(len(hosts))] + "();")
		}
		cases = append(cases, input{value: header.String(), trap: i%2 == 1})
	}
	if packetPath := os.Getenv("FAIRYC_RUNTIME_PACKET"); packetPath != "" {
		data, err := os.ReadFile(packetPath)
		if err != nil {
			t.Fatal(err)
		}
		packet, err := Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, input{value: packet.Header}, input{value: packet.Header, trap: true})
	}
	var fixture bytes.Buffer
	put := func(n uint32) {
		if err := binary.Write(&fixture, binary.LittleEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	part := func(s string) { put(uint32(len(s))); fixture.WriteString(s) }
	fixture.WriteString("FYCRUNTIME1\n")
	put(uint32(len(cases)))
	for _, c := range cases {
		mode := uint32(0)
		if c.direct {
			mode = 1
		}
		put(mode)
		trap := uint32(0)
		if c.trap {
			trap = 1
		}
		put(trap)
		part(c.value)
		var want string
		if c.direct {
			components, err := cstdlibcontract.WindowsMSVCComponentsForSymbols([]string{c.value})
			if err != nil {
				kind := 103
				if strings.HasPrefix(err.Error(), "Stage 0 standard-library") {
					kind = 101
				} else if strings.HasPrefix(err.Error(), "unrecognized") {
					kind = 102
				}
				want = fmt.Sprint(kind)
			} else {
				// Pick the symbol's own component, not a dependency: all symbols
				// of multi-component closures belong to the last component.
				component := components[len(components)-1]
				order := []string{"linux-console", "linux-file", "windows-console", "windows-file", "windows-directory", "windows-process-args", "system-heap", "byte-compare", "buffer", "windows-file-buffer", "arena", "text-view", "linux-process-args", "owned-text"}
				for i, name := range order {
					if string(component) == name {
						want = fmt.Sprint(i + 1)
					}
				}
			}
		} else {
			_, err := runtimeSource(c.value, c.trap)
			if err != nil {
				want = "error:" + err.Error()
			} else {
				set := map[string]bool{}
				for _, m := range stdSymbols.FindAllStringSubmatch(c.value, -1) {
					set[m[1]] = true
				}
				vectors := strings.Contains(c.value, "fy_private_vector_")
				if vectors {
					set["fy_std_system_heap_allocate_v0"] = true
				}
				names := []string{}
				for name := range set {
					names = append(names, name)
				}
				sort.Strings(names)
				components, err := cstdlibcontract.WindowsMSVCComponentsForSymbols(names)
				if err != nil {
					t.Fatal(err)
				}
				exit := 100
				if c.trap {
					exit = 128
				}
				want = fmt.Sprintf("scalar:%d\n", exit)
				for _, component := range components {
					want += string(component) + "\n"
				}
				if vectors {
					want += "vectors\n"
				}
				host, err := candidatehost.WindowsSource(c.value)
				if err != nil {
					t.Fatal(err)
				}
				hostKind := 0
				if host != "" {
					hostKind = 1
				}
				if strings.Contains(host, "NtCreateFile") {
					hostKind = 2
				}
				want += fmt.Sprintf("host:%d\n", hostKind)
			}
		}
		if want == "" {
			t.Fatalf("empty expectation for %#v", c)
		}
		part(want)
	}
	if err := os.WriteFile(path, fixture.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("exported %d runtime selection cases (%d MSVC symbols, %d host symbols)", len(cases), len(standard), len(hosts))
}
