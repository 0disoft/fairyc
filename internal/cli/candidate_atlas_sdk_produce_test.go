package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/atlasmap"
)

func TestCandidateAtlasSDKProduction(t *testing.T) {
	type fixture struct {
		name, root string
		partitions map[string]atlasmap.Partition
	}
	var fixtures []fixture
	for _, source := range []struct{ name, text string }{
		{"core SDK", `module main;import std::num as n;import std::bytes as b;import std::alloc as a;
import std::text as text;import std::io as io;import std::fs as fs;import std::process as process;
@effects(console)fn i32 main(){
let buffer=switch(b::copy(b"42")){case .ok(v):{v}case .err(_):{return 1;}};
let arena=switch(a::create()){case .ok(v):{v}case .err(_):{return 1;}};
discard io::write_usize(process::arg_count());
return switch(n::parse_i64_decimal(b::view(ref buffer))){case .ok(v):{if(v==42){42}else{1}}case .err(_):{1}};}`},
	} {
		root := writeWorkspaceSessionFixture(t, source.text)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("SDK %s oracle: %d %s", source.name, code, &diagnostic)
		}
		f := fixture{name: source.name, root: root, partitions: map[string]atlasmap.Partition{}}
		for _, p := range prepared.partitions {
			data, err := atlasmap.EncodeV3(p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			p, err = atlasmap.DecodeV3(data, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			f.partitions[p.Header().PackageInstance] = p
		}
		fixtures = append(fixtures, f)
	}
	main := strings.Replace(candidateAtlasGenericProduceMain(), "join::join(admitted,8,128,", "join::join(admitted,8,8192,", 1)
	// Exporting maps for the independent Go decoder is extra test work. Keep its
	// bounded ledger separate from the actual ready/join production route.
	main = strings.Replace(main, "  let encoded=", "  var usize export_work=0;let encoded=", 1)
	main = strings.Replace(main, "total-external,external,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};", "total-external,external,ref mut export_work)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};", 1)
	// Exercise the producer and complete-current reader as separate bounded API
	// operations. SDK command admission still requires its own ledger integration.
	main = strings.Replace(main, " var admitted=", " var usize validation_work=0;var admitted=", 1)
	main = strings.Replace(main, "produce::ready(v::at<produce::Package>(ref packages,owner),total-external,external,ref mut meter)", "produce::ready(v::at<produce::Package>(ref packages,owner),total-external,external,ref mut validation_work)", 1)
	main = strings.Replace(main, "join::join(admitted,8,8192,ref mut meter)", "join::join(admitted,8,8192,ref mut validation_work)", 1)
	main = strings.Replace(main, `discard io::write("ready/");discard io::write_usize(code as usize);return 7;`, `discard io::write("ready/");discard io::write_usize(code as usize);discard io::write("/");discard io::write_usize(meter);discard io::write("\n");discard io::write_bytes(bytes::view(ref encoded));return 7;`, 1)
	main = strings.Replace(main, " let observed=", `
 if(bytes::equal(mode,b"facts-before") || bytes::equal(mode,b"facts-at") || bytes::equal(mode,b"facts-after") || bytes::equal(mode,b"facts-full")){
  meter=if(bytes::equal(mode,b"facts-before")){1048575}else if(bytes::equal(mode,b"facts-at")){1048576}else if(bytes::equal(mode,b"facts-after")){1048577}else{16777216};let start=meter;
  let f=switch(facts::collect(graph,ref lock,ref table,compiler::public_interfaces(ref complete),p,image::value(ref compiler_image),0,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}};
  discard io::write_usize(meter-start);discard io::write("\n");return if(io::write_bytes(facts::value(ref f,b"generic_interface_hash")).is_ok){0}else{4};
 }
 let observed=`, 1)
	candidateAtlasProbeSize(t, "project_atlas_source", main, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		ordinary := writeWorkspaceSessionFixture(t, workspaceSessionRootSource)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(ordinary, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != 0 {
			t.Fatalf("facts budget oracle: %d %s", code, &diagnostic)
		}
		wantHash := ""
		for _, p := range prepared.partitions {
			if p.Header().PackageUID == "example.session.app" {
				wantHash = p.Header().GenericInterfaceHash
			}
		}
		if wantHash == "" {
			t.Fatal("missing facts root oracle")
		}
		before := remoteTree(t, ordinary)
		cost := 0
		for _, mode := range []string{"facts-before", "facts-at", "facts-after"} {
			code, out, stderr := fetchCLIProcess(t, image, ordinary, []string{mode}, "")
			lines := strings.Split(out, "\n")
			if code != 0 || stderr != "" || len(lines) != 2 || lines[1] != wantHash {
				t.Fatalf("%s: %d %q %q, want hash %s", mode, code, out, stderr, wantHash)
			}
			measured, err := strconv.Atoi(lines[0])
			if err != nil || measured <= 0 || cost != 0 && measured != cost {
				t.Fatalf("facts cost differs: %d/%d %v", measured, cost, err)
			}
			cost = measured
		}
		code, _, stderr := fetchCLIProcess(t, image, ordinary, []string{"facts-full"}, "")
		if code != 7 || stderr != "" || !reflect.DeepEqual(before, remoteTree(t, ordinary)) {
			t.Fatal("facts exhausted ledger or read-only guarantee", code, stderr)
		}
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				// Full SDK map export adds codec/independent-decoder work to the
				// production path. This diagnostic ceiling is test-only.
				ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
				defer cancel()
				command := exec.CommandContext(ctx, image)
				command.Dir = f.root
				var stdout, diagnostic bytes.Buffer
				command.Stdout, command.Stderr = &stdout, &diagnostic
				if err := command.Run(); err != nil {
					head, data, _ := strings.Cut(stdout.String(), "\n")
					_, decodeErr := atlasmap.DecodeV3([]byte(data), 4<<20)
					t.Fatalf("SDK production: %v (%v) %s; independent Go decode: %v; %s", err, ctx.Err(), head, decodeErr, &diagnostic)
				}
				code, out, stderr := 0, stdout.String(), diagnostic.String()
				if code != 0 || stderr != "" {
					t.Fatalf("SDK production: %d %s %s", code, out, stderr)
				}
				var documents []json.RawMessage
				if err := json.Unmarshal([]byte(out), &documents); err != nil || len(documents) != len(f.partitions) {
					t.Fatalf("SDK partitions: %v %d, want %d", err, len(documents), len(f.partitions))
				}
				seen := map[string]bool{}
				for _, raw := range documents {
					got, err := atlasmap.DecodeV3(raw, 4<<20)
					if err != nil {
						t.Fatalf("Go rejected SDK map: %v", err)
					}
					h := got.Header()
					want, ok := f.partitions[h.PackageInstance]
					if !ok || seen[h.PackageInstance] {
						t.Fatal("unexpected or duplicate package", h.PackageInstance)
					}
					seen[h.PackageInstance] = true
					if !reflect.DeepEqual(got.Declarations(), want.Declarations()) || !reflect.DeepEqual(got.Symbols(), want.Symbols()) || h.InterfaceHash != want.Header().InterfaceHash || h.GenericInterfaceHash != want.Header().GenericInterfaceHash {
						t.Fatalf("SDK declarations/identities differ %s: got %+v want %+v", h.PackageUID, got.Declarations(), want.Declarations())
					}
					compareAtlasProducedFunctions(t, got.Functions(), want.Functions())
					if !reflect.DeepEqual(got.Specializations(), want.Specializations()) {
						t.Fatalf("SDK specializations differ %s: got %+v want %+v", h.PackageUID, got.Specializations(), want.Specializations())
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("SDK production changed workspace")
				}
			})
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}
