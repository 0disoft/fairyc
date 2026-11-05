package cli

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestCandidateGenericAggregatePublicInterfaces(t *testing.T) {
	candidateAggregatePublicReplay(t, "public")
}

func TestCandidateGenericAggregateArtifactTransport(t *testing.T) {
	candidateAggregatePublicReplay(t, "transport")
}

func TestCandidateGenericAggregateArtifactRestore(t *testing.T) {
	candidateAggregatePublicReplay(t, "restore")
}

func candidateAggregatePublicReplay(t *testing.T, mode string) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows aggregate interface replay; Linux execution is batched")
	}
	base := candidateAggregateExecutions()[0].fixture
	base.name = "public concrete signatures"
	cases := []candidateAggregateLayoutFixture{
		base,
		{name: "reordered and rebased", root: base.root, reverse: true},
		{name: "concrete field edit", root: strings.ReplaceAll(base.root, ".value", ".payload"), maker: strings.ReplaceAll(strings.Replace(aggregateLayoutMaker, "T value;", "T payload;", 1), ".value", ".payload")},
		{name: "unused template edit", root: base.root, maker: strings.Replace(aggregateLayoutMaker, "pub struct Unused<T>{T value;}", "pub struct Unused<T>{T other;}", 1)},
		{name: "private body one", root: base.root, maker: aggregateLayoutMaker + "fn i32 hidden(){return 1;}"},
		{name: "private body two", root: base.root, maker: aggregateLayoutMaker + "fn i32 hidden(){return 2;}"},
		{name: "late public aggregate", root: base.root, maker: aggregateLayoutMaker + "pub struct Late{Cell<i32> nested;}"},
		{name: "late concrete signature", root: base.root, maker: aggregateLayoutMaker + "pub fn Cell<i64> late(Cell<i64> value){return value;}"},
		{name: "structural result", root: base.root, maker: aggregateLayoutMaker + "pub fn Option<Cell<i32>> optional(){return .none;}"},
		{name: "ordered scalar tuple", root: base.root, maker: aggregateLayoutMaker + "pub fn Pair<u8,i32> reversed(Pair<u8,i32> value){return value;}"},
		{name: "imported borrow", root: base.root + "pub fn ref m::Cell<i32> borrowed(ref m::Cell<i32> value){return value;}"},
		{name: "private instantiated aggregate", root: base.root, maker: aggregateLayoutMaker + "struct Hidden<T>{T value;}fn i32 hidden(){let p=Hidden<i32>{.value=1};return p.value;}"},
	}
	defaultMaker := aggregateLayoutMaker
	if mode == "restore" {
		defaultMaker += "pub fn Cell<bool> bool_cell(Cell<bool> p){return p;}fn i32 artifact_private(){return 1;}pub fn Pair<u8,i32> reversed(Pair<u8,i32> value){return value;}"
		for _, c := range candidateAggregateExecutions()[3:] {
			cases = append(cases, c.fixture)
		}
	}
	var wire bytes.Buffer
	wire.WriteString("FYGIP001")
	number := func(n uint64) {
		t.Helper()
		if err := binary.Write(&wire, binary.BigEndian, n); err != nil {
			t.Fatal(err)
		}
	}
	field := func(value []byte) { number(uint64(len(value))); wire.Write(value) }
	negative := []candidateAggregateLayoutFixture{
		{name: "private import", root: base.root, maker: strings.Replace(aggregateLayoutMaker, "pub struct Cell", "struct Cell", 1)},
		{name: "private exported instance", root: base.root, maker: aggregateLayoutMaker + "struct Hidden<T>{T value;}pub fn Hidden<i32> hidden(Hidden<i32> value){return value;}"},
		{name: "late unselected unresolved type", root: base.root, maker: aggregateLayoutMaker + "pub fn Missing late(){return 0;}"},
		{name: "invalid aggregate body", root: strings.Replace(base.root, ".value=40", ".missing=40", 1)},
	}
	number(uint64(len(cases) + len(negative)))
	var expected [][]candidateAggregateInterfaceExpectation
	for _, tc := range append(slices.Clone(cases), negative...) {
		maker := tc.maker
		if maker == "" {
			maker = defaultMaker
		}
		tc.maker = maker
		field([]byte(tc.name))
		shift := uint64(0)
		order := []int{0, 1, 2}
		if tc.reverse {
			shift = 19
			slices.Reverse(order)
		}
		number(shift)
		texts := []string{tc.root, maker, aggregatePlanHelper}
		for _, i := range order {
			number(uint64(i + 1))
			field([]byte(texts[i]))
		}
		if slices.ContainsFunc(negative, func(n candidateAggregateLayoutFixture) bool { return n.name == tc.name }) {
			if tc.name == "private exported instance" {
				number(6)
			} else {
				number(1)
			}
			continue
		}
		number(0)
		rows := candidateAggregateInterfaceExpectations(t, tc)
		expected = append(expected, rows)
		for _, row := range rows {
			number(row.owner)
			field(row.ordinary)
			field([]byte(row.ordinaryHash))
			field([]byte(row.sourceHash))
			number(uint64(len(row.specializations)))
			for _, id := range row.specializations {
				field([]byte(id))
			}
		}
	}
	if expected[0][1].ordinaryHash == expected[2][1].ordinaryHash || expected[0][1].sourceHash == expected[3][1].sourceHash {
		t.Fatal("field/template edits must change their owning interfaces")
	}
	for i := range expected[0] {
		if !bytes.Equal(expected[0][i].ordinary, expected[1][i].ordinary) || expected[0][i].sourceHash != expected[1][i].sourceHash || expected[4][i].sourceHash != expected[5][i].sourceHash {
			t.Fatal("source numbering/private bodies changed exported interfaces")
		}
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_public.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_public;", "module main;", 1) + "\n@effects(console) fn i32 main(){return run();}\n"
	if mode != "public" {
		file := "generic_aggregate_artifact.fy"
		if mode == "restore" {
			file = "generic_aggregate_restore.fy"
		}
		checks, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", file))
		if err != nil {
			t.Fatal(err)
		}
		var body []string
		for _, line := range strings.Split(string(checks), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "module ") {
				continue
			}
			if strings.HasPrefix(trimmed, "import ") {
				if !strings.Contains(main, trimmed) {
					main = strings.Replace(main, "module main;", "module main;\n"+trimmed, 1)
				}
				continue
			}
			body = append(body, line)
		}
		main += strings.Join(body, "\n")
		if mode == "transport" {
			main = strings.Replace(main, "let records=core::public_interfaces(ref complete);", "verify_artifact(ref complete,ref sources,ref owners,ref imports,ref table,k==0);let records=core::public_interfaces(ref complete);", 1)
		} else {
			main = strings.Replace(main, "let complete=build(ref sources,ref owners,ref imports,ref table);let records=core::public_interfaces(ref complete);", "let replay=verify_restore(build(ref sources,ref owners,ref imports,ref table),ref sources,ref owners,ref imports,ref table,k==0);let records=core::public_interfaces(complete_c::complete(ref replay));", 1)
		}
	}
	modules := []string{"module_project_artifact_encode", "module_project_artifact_decode", "module_cleanup_artifact_encode", "module_cleanup_artifact_decode", "module_identity_artifact_encode", "module_identity_artifact_decode"}
	if mode == "restore" {
		modules = append(modules, "module_project_complete_c")
	}
	candidateAtlasProbeSize(t, "project_compiler_core", main, 512<<10, func(t *testing.T, image string) {
		if mode != "restore" {
			t.Setenv("PATH", "")
			if code, out := candidateAtlasProcess(t, image, wire.Bytes()); code != 0 || out != "" {
				t.Fatalf("aggregate public interfaces: %d %q", code, out)
			}
			return
		}
		toolchain, err := cdriver.ResolveWindowsHostMSVC()
		if err != nil {
			t.Fatal(err)
		}
		support, err := cruntime.ScalarSource(80)
		if err != nil {
			t.Fatal(err)
		}
		support += "\n#include <stdio.h>\nvoid observe_cleanup(int32_t value){printf(\"%d,\",(int)value);}\n"
		t.Setenv("PATH", "")
		code, out := candidateAtlasProcess(t, image, wire.Bytes())
		if code != 0 {
			t.Fatalf("aggregate public interfaces: %d %q", code, out)
		}
		remaining := []byte(out)
		// One reordered restoration control precedes all same-source packets.
		want := []candidateAggregateExecution{{code: 42}}
		for _, c := range cases {
			value := candidateAggregateExecution{fixture: c, code: 42}
			for _, native := range candidateAggregateExecutions()[3:] {
				if native.fixture.name == c.name {
					value = native
				}
			}
			want = append(want, value)
		}
		for i, native := range want {
			end := bytes.IndexByte(remaining, '\n')
			if end < 0 {
				t.Fatalf("missing C packet %d", i)
			}
			n, err := strconv.Atoi(string(remaining[:end]))
			if err != nil || n <= 0 || n > 2<<20 || n > len(remaining)-end-1 {
				t.Fatalf("invalid C packet size %d: %v", n, err)
			}
			remaining = remaining[end+1:]
			packet, err := candidatebuild.Decode(remaining[:n])
			if err != nil {
				t.Fatal(err)
			}
			remaining = remaining[n:]
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				built, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, profile, packet.Header, packet.Literals, packet.Units, support)
				if err != nil {
					t.Fatalf("restored %d/%s: %v", i, profile.Name, err)
				}
				candidateAggregateExecuteImage(t, built.Bytes, native.code, native.output)
			}
		}
		if len(remaining) != 0 {
			t.Fatalf("unexpected trailing C bytes: %d", len(remaining))
		}
	}, modules...)
}
