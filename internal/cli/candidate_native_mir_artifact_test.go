package cli

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func nativeMIRArtifactCases() []candidateClosureCCase {
	all := candidateClosureCCases()
	var selected []candidateClosureCCase
	for _, index := range []int{0, 2, 4, 5, 6, 8, 10, 13} {
		selected = append(selected, all[index])
	}
	return selected
}

func nativeMIRArtifactWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyLexerExample(t, root)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "native_mir_artifact.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::native_mir_artifact;", "module main;\nimport native_artifact_current_fixture as source_tests;", 1)
	main = strings.ReplaceAll(main, "@test ", "")
	bound, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "native_mir_artifact_current.fy"))
	if err != nil {
		t.Fatal(err)
	}
	fixture := strings.Replace(string(bound), "module tests::native_mir_artifact_current;", "module native_artifact_current_fixture;", 1)
	fixture = strings.ReplaceAll(fixture, "@test fn", "pub fn")
	writeTestSource(t, root, "src/native_artifact_current_fixture.fy", fixture)
	main += "\n@effects(console) fn i32 main(){if(process::arg_count()>1){return consume();}\n"
	main += "native_mir_artifact_requires_exact_family_and_canonical_feature();\n"
	for _, name := range []string{
		"source_bound_native_mir_restores_both_exact_families",
		"source_bound_native_mir_rejects_stale_source_and_module",
		"source_bound_native_mir_rejects_forged_helpers_captures_signatures_effects_and_bodies",
		"source_bound_native_mir_revalidates_source_even_when_saved_source_is_updated",
		"source_bound_native_mir_rebuilds_written_helper_body_and_rejects_invalid_source",
		"source_bound_native_mir_charges_exact_shared_work_without_reset",
		"source_bound_native_mir_rejects_noncanonical_versions_and_trailing_bytes",
	} {
		main += "assert(io::write(" + strconv.Quote("source-test: "+name+"\n") + ").is_ok);\n"
		if strings.Contains(name, "forged_helpers") {
			for i := 1; i <= 10; i++ {
				main += "assert(io::write(" + strconv.Quote(fmt.Sprintf("forgery: %d ", i)) + ").is_ok);{let code=source_tests::source_bound_native_mir_forgery_probe(" + strconv.Itoa(i) + ");assert(io::write_u32(code).is_ok && io::write(\"\\n\").is_ok);assert(code==0);}\n"
			}
		} else {
			main += "source_tests::" + name + "();\n"
		}
	}
	main += "assert(io::write(\"source-tests-passed\\n\").is_ok);\n"
	for _, tc := range nativeMIRArtifactCases() {
		main += "member(b" + strconv.Quote(tc.raw) + ");\n"
	}
	main += "return 0;}\n"
	writeTestSource(t, root, "src/main.fy", main)
	copyCandidateParityModules(t, root, "native_mir_artifact_encode", "native_mir_artifact_decode", "native_mir_artifact_current", "mir_lower", "hir_lower", "name_resolution", "type_check", "c_emit")
	return root
}

type nativeMIRSection struct{ start, count, width int }

// This parser locates mutation offsets in genuine Fairy-produced snapshots. It
// neither constructs MIR nor supplies source/helper facts to the producer.
func nativeMIRSections(t *testing.T, raw []byte) map[string]nativeMIRSection {
	t.Helper()
	at := 16
	read := func() int {
		if at < 0 || at > len(raw)-8 {
			t.Fatal("truncated producer artifact")
		}
		n := binary.BigEndian.Uint64(raw[at : at+8])
		at += 8
		if n > uint64(len(raw)) {
			t.Fatal("invalid producer artifact size")
		}
		return int(n)
	}
	for i := 0; i < 2; i++ {
		n := read()
		at += n
	}
	sections := make(map[string]nativeMIRSection)
	for _, s := range []struct {
		name  string
		width int
	}{{"shapes", 5}, {"fields", 6}, {"abi", 6}, {"sdk", 3}, {"drops", 4}, {"targets", 14}, {"signatures", 1}, {"functions", 11}, {"locals", 7}, {"blocks", 20}, {"operations", 17}, {"edges", 1}, {"closures", 2}} {
		if s.name == "closures" && raw[7] == '0' {
			break
		}
		n := read()
		sections[s.name] = nativeMIRSection{at, n, s.width * 8}
		at += n * s.width * 8
	}
	if at != len(raw) {
		t.Fatal("noncanonical producer artifact framing")
	}
	return sections
}

func checkNativeMIRMutations(t *testing.T, image string, raw []byte) int {
	t.Helper()
	sections := nativeMIRSections(t, raw)
	var mutations [][]byte
	change := func(at int, value uint64) {
		changed := bytes.Clone(raw)
		binary.BigEndian.PutUint64(changed[at:at+8], value)
		mutations = append(mutations, changed)
	}
	change(16, 1048577) // bounded owned source length, before allocation
	change(sections["shapes"].start-8, 513)
	change(sections["targets"].start, 2)         // noncanonical boolean
	change(sections["targets"].start+8*4, 1<<32) // narrowed u32 must not truncate
	changed := bytes.Clone(raw)
	changed[7] = '0'
	mutations = append(mutations, changed)
	c := sections["closures"]
	if c.count == 0 {
		t.Fatal("closure mutation fixture has no helper")
	}
	change(c.start, 0)
	change(c.start+8, 5)
	change(c.start-8, 513)
	helper := int(binary.BigEndian.Uint64(raw[c.start : c.start+8]))
	target := sections["targets"].start + (helper-1)*sections["targets"].width
	change(target+8*3, 1)  // unsafe lifted helper
	change(target+8*4, 1)  // effectful lifted helper
	change(target+8*8, 13) // wrong helper result signature
	for _, pair := range []struct {
		section     string
		kind, value int
	}{{"operations", 2, 40}, {"blocks", 4, 8}} {
		s := sections[pair.section]
		found := false
		for i := 0; i < s.count; i++ {
			at := s.start + i*s.width
			if binary.BigEndian.Uint64(raw[at+8*pair.kind:at+8*(pair.kind+1)]) != uint64(pair.value) {
				continue
			}
			found = true
			if pair.section == "operations" {
				change(at+8*6, 0)  // nonexistent construction target
				change(at+8*16, 5) // excess capture operands
			} else {
				change(at+8*5, binary.BigEndian.Uint64(raw[at+8*6:at+8*7])) // i32 as callable
			}
			break
		}
		if !found {
			t.Fatalf("missing %s closure fixture", pair.section)
		}
	}
	for i, changed := range mutations {
		if code, out := candidateAtlasProcess(t, image, changed, "restore"); code != 17 || out != "" {
			t.Fatalf("forged artifact %d: exit=%d stdout=%q", i, code, out)
		}
	}
	return len(mutations)
}

func TestCandidateNativeMIRArtifact(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows native MIR artifacts; Linux remains batched")
	}
	previous := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	root := nativeMIRArtifactWorkspace(t)
	build, support := candidateParityBuildParts(t, root)
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const observer = "#include <stdint.h>\n#include <stdlib.h>\n_Noreturn void fy_runtime_trap(uint32_t category){exit(80+(int)category);}\n"
	var first [][]byte
	for _, producer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		image, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, producer, build.Header, "#include \"fy_project.h\"\n", build.Sources, support, time.Minute, 16<<20)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "artifact-"+producer.Name+".exe")
		if err = os.WriteFile(path, image.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		code, output := candidateAtlasProcess(t, path, nil)
		if code != 0 {
			t.Fatalf("artifact producer %s: exit=%d progress=%q", producer.Name, code, output)
		}
		reader := bufio.NewReader(strings.NewReader(output))
		for i := 0; i < 18; i++ {
			line, err := reader.ReadString('\n')
			if err != nil || i < 17 && !strings.HasPrefix(line, "source-test: ") && !strings.HasPrefix(line, "forgery: ") || i == 17 && line != "source-tests-passed\n" {
				t.Fatalf("missing source-binding acceptance %q: %v", line, err)
			}
		}
		for i, tc := range nativeMIRArtifactCases() {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			n, err := strconv.Atoi(strings.TrimSuffix(line, "\n"))
			if err != nil || n < 8 || n > 8388608 {
				t.Fatalf("invalid packet size %q", line)
			}
			saved := make([]byte, n)
			if _, err = io.ReadFull(reader, saved); err != nil {
				t.Fatal(err)
			}
			if producer.Name == cdriver.LowProfile.Name {
				first = append(first, saved)
			} else if !bytes.Equal(first[i], saved) {
				t.Fatalf("profile-dependent artifact: %s", tc.name)
			}
			// A fresh process reads serialized bytes, verifies and emits C.
			code, unit := candidateAtlasProcess(t, path, saved, "restore")
			if code != 0 || unit == "" {
				t.Fatalf("fresh-process restore %s/%s: %d", producer.Name, tc.name, code)
			}
			boundCode, boundUnit := candidateAtlasProcess(t, path, saved, "restore", tc.raw)
			if boundCode != 0 || boundUnit != unit {
				t.Fatalf("fresh-process current-source restore %s/%s: exit=%d same-C=%v", producer.Name, tc.name, boundCode, boundUnit == unit)
			}
			for _, consumer := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				program, err := cdriver.BuildWindowsMSVCTarget(toolchain, consumer, unit, observer)
				if err != nil {
					t.Fatalf("restored C %s/%s/%s: %v", producer.Name, consumer.Name, tc.name, err)
				}
				p := filepath.Join(root, fmt.Sprintf("restored-%s-%s-%d.exe", producer.Name, consumer.Name, i))
				if err = os.WriteFile(p, program.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				if code, out := candidateAtlasProcess(t, p, nil); code != tc.exit || out != "" {
					t.Fatalf("restored execution %s/%s/%s: exit=%d want=%d stdout=%q", producer.Name, consumer.Name, tc.name, code, tc.exit, out)
				}
			}
			t.Logf("%s %s: canonical artifact %d bytes sha256=%x; fresh restore + both C profiles passed", producer.Name, tc.name, n, sha256.Sum256(saved))
			if i == 2 {
				t.Logf("%s: %d forged artifact refusals", producer.Name, checkNativeMIRMutations(t, path, saved))
			}
		}
		if _, err := reader.ReadByte(); err != io.EOF {
			t.Fatalf("trailing artifact output: %v", err)
		}
	}
}
