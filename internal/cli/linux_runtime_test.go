package cli

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cstdlibcontract"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

// The expected plans come from live Go selection and actual selected C files,
// rather than from the authored selector or its materialized symbol table.
func TestLinuxGNUCompanionRuntime(t *testing.T) {
	stdlib, err := parser.ParseFile(token.NewFileSet(), "../cstdlibcontract/contract.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var symbols []string
	ast.Inspect(stdlib, func(node ast.Node) bool {
		v, ok := node.(*ast.ValueSpec)
		if !ok || len(v.Names) != 1 || v.Names[0].Name != "symbolComponents" {
			return true
		}
		for _, element := range v.Values[0].(*ast.CompositeLit).Elts {
			symbol, err := strconv.Unquote(element.(*ast.KeyValueExpr).Key.(*ast.BasicLit).Value)
			if err != nil {
				t.Fatal(err)
			}
			symbols = append(symbols, symbol)
		}
		return false
	})
	if len(symbols) < 60 {
		t.Fatal("missing Linux standard-library symbol table")
	}
	host, err := parser.ParseFile(token.NewFileSet(), "../candidatehost/linux_paths.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(host, func(node ast.Node) bool {
		clause, ok := node.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, expression := range clause.List {
			literal, ok := expression.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				continue
			}
			symbol, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(symbol, "fairyc_private_linux_") {
				symbols = append(symbols, symbol)
			}
		}
		return true
	})
	sort.Strings(symbols)
	var cases [][]string
	for _, symbol := range symbols {
		cases = append(cases, []string{symbol})
	}
	cases = append(cases, symbols, []string{}, []string{"fy_private_vector_reserve_v0"},
		[]string{"fairyc_private_linux_directory_scan_next_v1", "fy_std_file_read_append_v0", "fy_private_vector_element_slot_v0"},
		[]string{"fairyc_private_linux_https_read_v1", "fairyc_private_linux_monotonic_ms_v1"},
		[]string{"fairyc_private_linux_package_directory_publish_v1", "fairyc_private_linux_package_file_info_v1", "fairyc_private_linux_https_open_v1"})
	components := []cstdlibcontract.Component{
		cstdlibcontract.ComponentLinuxConsole, cstdlibcontract.ComponentLinuxFile,
		cstdlibcontract.ComponentSystemHeap, cstdlibcontract.ComponentByteCompare,
		cstdlibcontract.ComponentBuffer, cstdlibcontract.ComponentLinuxFileBuffer,
		cstdlibcontract.ComponentArena, cstdlibcontract.ComponentTextView,
		cstdlibcontract.ComponentLinuxProcessArgs, cstdlibcontract.ComponentOwnedText,
	}
	files := []string{"linux_rooted_file.c", "linux_process.c", "linux_capture.c", "linux_test_input.c", "linux_paths.c", "linux_directory.c", "linux_retention.c", "linux_workspace.c", "linux_build_scratch.c", "linux_starter.c", "linux_gnu.c", "linux_package_directory.c", "linux_package_file.c", "linux_clock.c", "linux_https_protocol.c", "linux_https_worker.c", "linux_https_reader.c", "linux_https_parent.c", "linux_edit_file.c"}
	parts := make([]string, len(files))
	for i, file := range files {
		data, err := os.ReadFile(filepath.Join("..", "candidatehost", file))
		if err != nil {
			t.Fatal(err)
		}
		parts[i] = string(data)
	}
	var entry strings.Builder
	entry.WriteString("fn i32 main(){rejects_wrong_target_unknown_symbols_and_windows_dependencies();\n")
	for _, names := range cases {
		header := "/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\n" + strings.Join(names, "();\n")
		if len(names) > 0 {
			header += "();\n"
		}
		if _, err := nativesupport.Source(target.X8664LinuxGNU, names, nativesupport.Options{SDKStorage: true, CandidateHost: true}); err != nil {
			t.Fatal(err)
		}
		var standard []string
		vectors, arguments := false, false
		for _, name := range names {
			if strings.HasPrefix(name, "fy_private_vector_") {
				vectors = true
				standard = append(standard, "fy_std_system_heap_allocate_v0")
			} else {
				standard = append(standard, name)
			}
			arguments = arguments || name == "fy_std_process_arg_count_v0" || name == "fy_std_process_arg_v0"
		}
		selected, err := cstdlibcontract.ComponentsForSymbols(standard)
		if err != nil {
			t.Fatal(err)
		}
		var mask uint32
		for _, component := range selected {
			found := false
			for i, named := range components {
				if named == component {
					mask |= 1 << i
					found = true
				}
			}
			if !found {
				t.Fatalf("unmapped Linux component %s", component)
			}
		}
		source, err := candidatehost.LinuxSource(header)
		if err != nil {
			t.Fatal(err)
		}
		var hosts uint32
		for i, part := range parts {
			if strings.Contains(source, part) {
				hosts |= 1 << i
			}
		}
		fmt.Fprintf(&entry, "expected(b%q,%d,%d,%t,%t);\n", header, mask, hosts, vectors, arguments)
	}
	entry.WriteString("return 0;}\n")
	t.Logf("comparing %d Linux runtime plans against live Go selection", len(cases))
	runLinuxCandidateFixture(t, []string{"linux_runtime", "linux_runtime_contract"}, "linux_runtime", entry.String(), "")
}
