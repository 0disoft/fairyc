package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

const candidateAtlasSDKVectorUnusedSource = `module main;import std::vec as v;fn i32 main(){return 0;}`

const candidateAtlasSDKVectorUsedSource = `module main;import std::vec as v;fn i32 main(){var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);return *v::at<i32>(ref values,0);}`

// pop also materializes a callable take_slot<i32> frame. Its cleanup inline
// observation must join that same source implementation without a duplicate.
const candidateAtlasSDKVectorPopSource = `module main;import std::vec as v;fn i32 main(){var values=v::empty<i32>();assert(v::push<i32>(ref mut values,42).is_ok);return switch(v::pop<i32>(ref mut values)){case .some(value):{value}case .none:{0}};}`

func candidateAtlasSDKVectorProduceMain(t *testing.T) string {
	t.Helper()
	main := candidateAtlasGenericProduceMain()
	replace := func(old, replacement string) {
		t.Helper()
		if strings.Count(main, old) != 1 {
			t.Fatalf("SDK vector probe requires one exact port: %q", old)
		}
		main = strings.Replace(main, old, replacement, 1)
	}
	replace(`join::join(admitted,8,128,`, `join::join(admitted,8,8192,`)
	replace(`if(bytes::equal(mode,b"budget")){meter=16777216;}`, `if(bytes::equal(mode,b"budget") || bytes::equal(mode,b"source-budget")){meter=16777216;}`)
	// Keep source, export, and current/join work in separate bounded ledgers.
	replace(`  let encoded=`, `  var usize export_work=if(bytes::equal(mode,b"export-budget")){16777216}else{0};let encoded=`)
	replace(`total-external,external,ref mut meter)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}}`, `total-external,external,ref mut export_work)){case .ok(p):{p}case .err(code):{discard io::write_usize(code as usize);return 7;}}`)
	replace(` var admitted=`, ` var usize validation_work=if(bytes::equal(mode,b"validation-budget")){16777216}else{0};var admitted=`)
	replace(`produce::ready(v::at<produce::Package>(ref packages,owner),total-external,external,ref mut meter)`, `produce::ready(v::at<produce::Package>(ref packages,owner),total-external,external,ref mut validation_work)`)
	replace(`join::join(admitted,8,8192,ref mut meter)`, `join::join(admitted,8,8192,ref mut validation_work)`)
	replace(` let joined=switch(join::join(admitted,8,8192,ref mut validation_work))`, ` if(bytes::equal(mode,b"join-budget")){validation_work=16777216;}let joined=switch(join::join(admitted,8,8192,ref mut validation_work))`)
	tag := func(begin, end, phase string) {
		t.Helper()
		start := strings.Index(main, begin)
		if start < 0 {
			t.Fatalf("missing SDK vector diagnostic phase %s", phase)
		}
		stop := strings.Index(main[start:], end)
		if stop < 0 {
			t.Fatalf("missing SDK vector diagnostic end %s", phase)
		}
		stop += start
		part := main[start:stop]
		old := `case .err(code):{discard io::write_usize(code as usize);return 7;}`
		if strings.Count(part, old) != 1 {
			t.Fatalf("SDK vector requires one diagnostic error port: %s", phase)
		}
		part = strings.Replace(part, old, `case .err(code):{discard io::write("`+phase+`/");discard io::write_usize(code as usize);return 7;}`, 1)
		main = main[:start] + part + main[stop:]
	}
	tag(" let observed=", " var packages=", "instances")
	tag("  let f=", "  let part=", "facts")
	tag("  let part=", "  total+=", "package")
	tag("  var usize export_work=", "  let ready=", "encode")
	return main
}

func TestCandidateAtlasSDKVectorProduction(t *testing.T) {
	type fixture struct {
		name, root string
		partitions map[string]atlasmap.Partition
	}
	var fixtures []fixture
	for _, source := range []struct{ name, text string }{
		{"unused", candidateAtlasSDKVectorUnusedSource},
		{"used", candidateAtlasSDKVectorUsedSource},
		{"pop", candidateAtlasSDKVectorPopSource},
	} {
		root := writeWorkspaceSessionFixture(t, source.text)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
		if code != ExitSuccess {
			t.Fatalf("SDK vector oracle %s: %d %s", source.name, code, &diagnostic)
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
		verifyCandidateAtlasSDKVectorOracle(t, f.name, f.partitions)
		fixtures = append(fixtures, f)
	}
	unusedSDK := candidateAtlasSDKVectorSDK(t, fixtures[0].partitions)
	for _, f := range fixtures[1:] {
		usedSDK := candidateAtlasSDKVectorSDK(t, f.partitions)
		if !reflect.DeepEqual(unusedSDK.Declarations(), usedSDK.Declarations()) || unusedSDK.Header().GenericInterfaceHash != usedSDK.Header().GenericInterfaceHash {
			t.Fatal("root vector use changed original SDK declarations or generic interface")
		}
	}

	main := candidateAtlasSDKVectorProduceMain(t)
	candidateAtlasProbeSize(t, "project_atlas_source", main, 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				before := remoteTree(t, f.root)
				code, out, stderr := candidateAtlasSDKCLIProcess(t, image, f.root, nil)
				if code != 0 || stderr != "" {
					t.Fatalf("SDK vector production: %d %s %s", code, out, stderr)
				}
				var documents []json.RawMessage
				if err := json.Unmarshal([]byte(out), &documents); err != nil || len(documents) != len(f.partitions) {
					t.Fatalf("SDK vector partitions: %v %d, want %d", err, len(documents), len(f.partitions))
				}
				global := candidateAtlasSDKVectorGlobalOracle(t, f.partitions)
				seen := map[string]bool{}
				for _, raw := range documents {
					got, err := atlasmap.DecodeV3(raw, 4<<20)
					if err != nil {
						t.Fatalf("Go rejected produced SDK vector map: %v", err)
					}
					h := got.Header()
					want, ok := f.partitions[h.PackageInstance]
					if !ok || seen[h.PackageInstance] {
						t.Fatalf("unexpected or duplicate SDK vector partition %s", h.PackageInstance)
					}
					seen[h.PackageInstance] = true
					verifyCandidateAtlasSDKVectorPartition(t, got, want)
					verifyCandidateAtlasSDKVectorCompleteness(t, got, want, global)
				}
				// These API boundaries are shared by all three fixtures. Exercise
				// each once per compiler profile. All source fixtures still have
				// their own complete map comparison.
				var controls []struct{ mode, output string }
				if f.name == "unused" {
					controls = []struct{ mode, output string }{
						{"source-budget", "instances/3"},
						{"export-budget", "encode/3"},
						{"validation-budget", "ready/3"},
						{"join-budget", "join/3"},
						{"missing-coordinate", "instances/7"},
					}
				}
				for _, bad := range controls {
					code, out, stderr := candidateAtlasSDKCLIProcess(t, image, f.root, []string{bad.mode})
					if code != 7 || out != bad.output || stderr != "" {
						t.Fatalf("%s: %d %q %q, want 7 %q", bad.mode, code, out, stderr, bad.output)
					}
				}
				if !reflect.DeepEqual(before, remoteTree(t, f.root)) {
					t.Fatal("SDK vector production changed workspace")
				}
			})
		}
	}, "project_compiler_core", "project_workspace_lock", "project_coordinate_graph")
}

func candidateAtlasSDKVectorSDK(t *testing.T, partitions map[string]atlasmap.Partition) atlasmap.Partition {
	t.Helper()
	var sdk atlasmap.Partition
	found := false
	for _, p := range partitions {
		if p.Header().PackageUID == "fairyc.std" {
			if found {
				t.Fatal("duplicate SDK package UID")
			}
			sdk, found = p, true
		}
	}
	if !found {
		t.Fatal("missing SDK partition")
	}
	return sdk
}

func verifyCandidateAtlasSDKVectorOracle(t *testing.T, name string, partitions map[string]atlasmap.Partition) {
	t.Helper()
	sdk := candidateAtlasSDKVectorSDK(t, partitions)
	want := map[string]bool{}
	for _, scalar := range []string{"u8", "i64", "i32", "u32", "usize"} {
		want["Vec<"+scalar+">"] = true
		want["take_slot<"+scalar+">"] = true
	}
	explicit := map[string]bool{}
	observed := map[string]bool{}
	var vector semanticsnapshot.SpecializationSummary
	for _, s := range sdk.Specializations() {
		if s.Declaration.Module != "std::vec" || len(s.TypeArguments) != 1 || s.TypeArguments[0].Kind != semanticsnapshot.TypeBuiltin {
			t.Fatalf("SDK vector specialization family differs: %+v", s)
		}
		if !strings.HasPrefix(s.ID, "g0_") || observed[s.Declaration.Name+"<"+s.TypeArguments[0].Name+">"] {
			t.Fatalf("SDK vector specialization identity differs: %+v", s)
		}
		key := s.Declaration.Name + "<" + s.TypeArguments[0].Name + ">"
		observed[key] = true
		if s.TypeArguments[0].Name == "i32" && (s.Declaration.Name == "empty" || s.Declaration.Name == "push" || s.Declaration.Name == "at" || s.Declaration.Name == "pop" || s.Declaration.Name == "write_slot") {
			explicit[key] = true
		}
		if s.Declaration.Name == "Vec" && s.TypeArguments[0].Name == "i32" {
			vector = s
		}
	}
	if name == "used" || name == "pop" {
		operation := "at"
		if name == "pop" {
			operation = "pop"
		}
		for _, suffix := range []string{"empty", "push", operation, "write_slot"} {
			want[suffix+"<i32>"] = true
		}
	}
	if !reflect.DeepEqual(observed, want) || len(sdk.Specializations()) != len(want) {
		t.Fatalf("SDK vector closure %s: got %v want %v", name, observed, want)
	}
	if name == "unused" {
		if len(explicit) != 0 {
			t.Fatal("unused std::vec acquired root-requested implementations", explicit)
		}
	} else if len(explicit) != 4 {
		t.Fatalf("used std::vec root-requested implementations: %v", explicit)
	}
	declarations := map[string]semanticsnapshot.Declaration{}
	for _, d := range sdk.Declarations() {
		declarations[d.Key.Name] = d
	}
	for _, scalar := range []string{"u8", "i64", "i32", "u32", "usize"} {
		for _, prefix := range []string{"view_", "view_mut_"} {
			name := prefix + scalar
			d, ok := declarations[name]
			if !ok || len(d.Parameters) != 1 || !candidateAtlasSDKVectorType(d.Parameters[0].Type, scalar) {
				t.Fatalf("ordinary SDK Vec-typed signature %s lost %s", name, scalar)
			}
		}
	}
	if vector.Aggregate == nil || len(vector.Aggregate.Fields) != 6 {
		t.Fatalf("SDK Vec concrete fields missing: %+v", vector.Aggregate)
	}
	wantFields := []string{"data", "bytes", "alignment", "length", "element_size", "element_alignment"}
	for i, field := range vector.Aggregate.Fields {
		if field.Name != wantFields[i] {
			t.Fatalf("SDK Vec field %d: got %q want %q", i, field.Name, wantFields[i])
		}
		if i == 0 {
			if field.Type.Kind != semanticsnapshot.TypePointer || !field.Type.Mutable || len(field.Type.Arguments) != 1 || field.Type.Arguments[0].Name != "void" {
				t.Fatalf("SDK Vec storage pointer changed: %+v", field.Type)
			}
		} else if field.Type.Kind != semanticsnapshot.TypeBuiltin || field.Type.Name != "usize" {
			t.Fatalf("SDK Vec storage metadata changed: %+v", field.Type)
		}
	}
	if vector.Cleanup == nil || len(vector.Cleanup.Calls) == 0 {
		t.Fatal("SDK Vec cleanup call facts missing")
	}
}

func candidateAtlasSDKVectorType(value semanticsnapshot.TypeRef, scalar string) bool {
	if value.Kind == semanticsnapshot.TypeNominal && value.Name == "std::vec::Vec" && len(value.Arguments) == 1 &&
		value.Arguments[0].Kind == semanticsnapshot.TypeBuiltin && value.Arguments[0].Name == scalar {
		return true
	}
	for _, argument := range value.Arguments {
		if candidateAtlasSDKVectorType(argument, scalar) {
			return true
		}
	}
	return false
}

type candidateAtlasSDKVectorGlobal struct {
	calls, external, templates, implementations int
}

func candidateAtlasSDKVectorGlobalOracle(t *testing.T, partitions map[string]atlasmap.Partition) candidateAtlasSDKVectorGlobal {
	t.Helper()
	var result candidateAtlasSDKVectorGlobal
	set := false
	for _, p := range partitions {
		calls := semanticsnapshot.Completeness{}
		templates := semanticsnapshot.Completeness{}
		for _, c := range p.Completeness() {
			if c.Relation == "calls" {
				calls = c
			}
			if c.Relation == "generic-template-bodies" {
				templates = c
			}
		}
		if calls.Relation == "" || templates.Relation == "" {
			t.Fatalf("Go calls oracle missing for %s", p.Header().PackageInstance)
		}
		if !set {
			result.calls, result.external, result.templates = calls.Resolved, calls.External, templates.Unknown
			set = true
		} else if calls.Resolved != result.calls || calls.External != result.external || templates.Unknown != result.templates {
			t.Fatalf("Go global completeness changed across partitions")
		}
		result.implementations += len(p.Specializations())
	}
	return result
}

func verifyCandidateAtlasSDKVectorPartition(t *testing.T, got, want atlasmap.Partition) {
	t.Helper()
	h, expected := got.Header(), want.Header()
	if h.Counts.Relations != candidateAtlasProducedRelationCount(want) {
		t.Fatalf("SDK source relation count: %d want %d", h.Counts.Relations, candidateAtlasProducedRelationCount(want))
	}
	if !reflect.DeepEqual(got.Declarations(), want.Declarations()) || !reflect.DeepEqual(got.Symbols(), want.Symbols()) ||
		h.InterfaceHash != expected.InterfaceHash || h.GenericInterfaceHash != expected.GenericInterfaceHash {
		t.Fatalf("SDK vector declarations/IDs/interfaces differ %s", h.PackageUID)
	}
	compareAtlasProducedFunctions(t, got.Functions(), want.Functions())
	actual, expectedSpecializations := got.Specializations(), want.Specializations()
	if len(actual) != len(expectedSpecializations) || h.Counts.Specializations != len(expectedSpecializations) {
		t.Fatalf("SDK vector implementation count differs %s: %d/%d", h.PackageUID, len(actual), len(expectedSpecializations))
	}
	for i, a := range actual {
		w := expectedSpecializations[i]
		if a.ID != w.ID || a.Declaration != w.Declaration || !reflect.DeepEqual(a.TypeArguments, w.TypeArguments) {
			t.Fatalf("SDK vector implementation identity differs: got %+v want %+v", a, w)
		}
		if !reflect.DeepEqual(a.Aggregate, w.Aggregate) {
			t.Fatalf("SDK vector concrete fields differ: got %+v want %+v", a.Aggregate, w.Aggregate)
		}
		if (a.Cleanup == nil) != (w.Cleanup == nil) {
			t.Fatalf("SDK vector cleanup presence differs: got %+v want %+v", a.Cleanup, w.Cleanup)
		}
		if w.Aggregate == nil {
			compareAtlasProducedFunctions(t, []semanticsnapshot.FunctionSummary{a.Function}, []semanticsnapshot.FunctionSummary{w.Function})
		} else if !reflect.DeepEqual(a.Function, semanticsnapshot.FunctionSummary{}) {
			t.Fatal("SDK vector aggregate published as an ordinary function")
		}
		if a.Cleanup != nil {
			compareAtlasProducedFunctions(t, []semanticsnapshot.FunctionSummary{*a.Cleanup}, []semanticsnapshot.FunctionSummary{*w.Cleanup})
		}
	}
}

func verifyCandidateAtlasSDKVectorCompleteness(t *testing.T, got, want atlasmap.Partition, global candidateAtlasSDKVectorGlobal) {
	t.Helper()
	h := got.Header()
	if h.GenerationStatus != atlasmap.GenerationPartial || len(h.Omissions) == 0 {
		t.Fatalf("SDK vector generation status/omissions: %+v", h)
	}
	wantRows := map[string]semanticsnapshot.Completeness{}
	callStatus, callReason := semanticsnapshot.CompletenessExact, ""
	if global.external != 0 {
		callStatus, callReason = semanticsnapshot.CompletenessPartial, "external-ffi"
	}
	wantRows["calls"] = semanticsnapshot.Completeness{Relation: "calls", Status: callStatus, Resolved: global.calls, External: global.external}
	wantRows["drops"] = semanticsnapshot.Completeness{Relation: "drops", Status: semanticsnapshot.CompletenessExact}
	wantRows["may-trap"] = semanticsnapshot.Completeness{Relation: "may-trap", Status: semanticsnapshot.CompletenessExact}
	wantRows["uses-unsafe"] = semanticsnapshot.Completeness{Relation: "uses-unsafe", Status: semanticsnapshot.CompletenessExact}
	wantRows["generic-specializations"] = semanticsnapshot.Completeness{Relation: "generic-specializations", Status: semanticsnapshot.CompletenessExact, Resolved: global.implementations}
	wantRows["generic-template-bodies"] = semanticsnapshot.Completeness{Relation: "generic-template-bodies", Status: semanticsnapshot.CompletenessUnavailable, Unknown: global.templates, Reasons: []string{"bodies-checked-only-after-specialization"}}
	if callReason != "" {
		row := wantRows["calls"]
		row.Reasons = []string{callReason}
		wantRows["calls"] = row
	}
	for _, relation := range []string{"allocates", "borrows", "effects", "moves", "reads", "returns-error", "test-calls", "writes"} {
		wantRows[relation] = semanticsnapshot.Completeness{Relation: relation, Status: semanticsnapshot.CompletenessUnavailable, Reasons: []string{"candidate-atlas-not-collected"}}
	}
	actual := got.Completeness()
	if len(actual) != len(wantRows) {
		t.Fatalf("SDK vector completeness rows: %d, want %d", len(actual), len(wantRows))
	}
	for _, row := range actual {
		w, ok := wantRows[row.Relation]
		if !ok || row.Status != w.Status || row.Resolved != w.Resolved || row.External != w.External || row.Unknown != w.Unknown || !slices.Equal(row.Reasons, w.Reasons) {
			t.Fatalf("SDK vector completeness %s: got %+v want %+v", row.Relation, row, w)
		}
	}
}
