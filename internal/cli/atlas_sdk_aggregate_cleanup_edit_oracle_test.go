package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/sourceedit"
)

// The cleanup still calls SDK Vec<i32>. Editing only leaf replaces the generic
// struct result and its constructor with bool implementations absent beforehand.
const atlasSDKAggregateCleanupEditBody = `{let check=Check<bool>{.value=true};return if(cell<bool>(true).value){43}else{1};}`

func atlasSDKAggregateCleanupEditedSource() string {
	return strings.Replace(atlasSDKAggregateCleanupSource, `{let check=Check<bool>{.value=true};return cell<i32>(42).value;}`, atlasSDKAggregateCleanupEditBody, 1)
}

func verifyAtlasSDKAggregateCleanupEditedOracle(t *testing.T, before, after preparedAtlasWorkspace) {
	t.Helper()
	want := map[string]int{"main::Cell<bool>": 1, "main::Check<bool>": 1, "main::cell<bool>": 1}
	for _, scalar := range []string{"u8", "i64", "i32", "u32", "usize"} {
		want["std::vec::Vec<"+scalar+">"] = 1
		want["std::vec::take_slot<"+scalar+">"] = 1
	}
	for _, name := range []string{"empty", "push", "at", "write_slot"} {
		want["std::vec::"+name+"<i32>"] = 1
	}
	counts := map[string]int{}
	var oldSDK, newSDK atlasmap.Partition
	var oldSource, newSource string
	for _, p := range before.partitions {
		if p.Header().PackageUID == "fairyc.std" {
			oldSDK = p
		} else if p.Header().PackageUID == "example.session.app" {
			oldSource = p.Header().PackageSource
		}
	}
	for _, p := range after.partitions {
		h := p.Header()
		t.Logf("edited Go source %s: provenance=%s declarations=%d functions=%d implementations=%d relations=%d", h.PackageUID, h.PackageSourceKind, len(p.Declarations()), len(p.Functions()), len(p.Specializations()), h.Counts.Relations)
		if h.PackageUID == "fairyc.std" {
			newSDK = p
		} else if h.PackageUID == "example.session.app" {
			newSource = h.PackageSource
		}
		for _, s := range p.Specializations() {
			if len(s.TypeArguments) != 1 {
				t.Fatalf("non-scalar edited implementation: %+v", s)
			}
			counts[s.Declaration.Module+"::"+s.Declaration.Name+"<"+s.TypeArguments[0].Name+">"]++
			if s.Declaration.Module == "main" && s.Declaration.Name == "Cell" && (s.Aggregate == nil || len(s.Aggregate.Fields) != 1 || s.Aggregate.Fields[0].Type.Name != "bool") {
				t.Fatalf("edited generic result lost bool layout: %+v", s)
			}
			if s.Declaration.Module == "main" && s.Declaration.Name == "Check" {
				if s.Cleanup == nil || len(s.Cleanup.Calls) != 3 {
					t.Fatalf("edited user cleanup lost SDK calls: %+v", s)
				}
				seen := map[string]int{}
				for _, call := range s.Cleanup.Calls {
					written := map[string]string{"empty": "v::empty<i32>()", "push": "v::push<i32>(ref mut values,42)", "at": "v::at<i32>(ref values,0)"}[call.Target.Name]
					start := strings.Index(atlasSDKAggregateCleanupEditedSource(), written)
					if written == "" || call.Target.Module != "std::vec" || call.Specialization == "" || call.Span.Source != "main.fy" || call.Span.StartByte != start || call.Span.EndByte != start+len(written) {
						t.Fatalf("edited cleanup endpoint/span: %+v", call)
					}
					seen[call.Target.Name]++
				}
				if !reflect.DeepEqual(seen, map[string]int{"empty": 1, "push": 1, "at": 1}) {
					t.Fatal("edited cleanup call inventory", seen)
				}
			}
		}
	}
	if !reflect.DeepEqual(counts, want) || oldSource == "" || newSource == "" || oldSource == newSource {
		t.Fatalf("edited implementation/source identity: got %v want %v source=%s -> %s", counts, want, oldSource, newSource)
	}
	if oldSDK.Header().PackageSourceKind != "sdk" || newSDK.Header().PackageSourceKind != "sdk" || oldSDK.Header().PackageSource != newSDK.Header().PackageSource || oldSDK.Header().InterfaceHash != newSDK.Header().InterfaceHash || oldSDK.Header().GenericInterfaceHash != newSDK.Header().GenericInterfaceHash || !reflect.DeepEqual(oldSDK.Declarations(), newSDK.Declarations()) || !reflect.DeepEqual(oldSDK.Symbols(), newSDK.Symbols()) || !reflect.DeepEqual(oldSDK.Specializations(), newSDK.Specializations()) {
		t.Fatal("root edit changed pinned SDK authority or concrete SDK facts")
	}
	index, err := atlasmap.NewWorkspaceCallIndexV3(after.partitions, after.expectations, atlasmap.WorkspaceCallBudget{MaxPackages: 8, MaxCalls: 8192})
	if err != nil {
		t.Fatal(err)
	}
	cleanupCalls := 0
	for _, call := range index.Calls() {
		if call.SourceBody == "cleanup" && call.SourceSpecialization != "" && call.Span.Source == "main.fy" {
			cleanupCalls++
			if call.TargetSpecialization == "" || call.Source.PackageInstance == call.Target.PackageInstance {
				t.Fatalf("edited cleanup lost cross-package specialization: %+v", call)
			}
		}
	}
	if cleanupCalls != 3 {
		t.Fatalf("edited cleanup joined calls=%d want 3", cleanupCalls)
	}
	t.Logf("edited resolved calls=%d; cross-package cleanup calls=%d", len(index.Calls()), cleanupCalls)
}

func TestAtlasSDKAggregateCleanupEditOracle(t *testing.T) {
	root, before := atlasSDKAggregateCleanupOracle(t)
	var symbol string
	for _, p := range before.partitions {
		for _, s := range p.Symbols() {
			if p.Header().PackageUID == "example.session.app" && s.Declaration.Name == "leaf" {
				symbol = s.Visible
			}
		}
	}
	if symbol == "" {
		t.Fatal("missing cleanup edit leaf")
	}
	writeWorkspaceFile(t, root, "edits/body.txt", atlasSDKAggregateCleanupEditBody)
	args := []string{"replace-body", "--package-uid", "example.session.app", "--symbol", symbol, "--body-file", "edits/body.txt", "--expected-source-sha256", sourceedit.Digest([]byte(atlasSDKAggregateCleanupSource))}
	var out, diagnostic bytes.Buffer
	untouched := remoteTree(t, root)
	code := runAtlasEdit(args, root, &out, &diagnostic)
	var report sourceedit.Report
	if code != 0 || diagnostic.Len() != 0 || json.Unmarshal(out.Bytes(), &report) != nil || !report.Verified || report.Applied || report.BodyStartByte < 0 || report.BodyEndByte < report.BodyStartByte || report.BodyEndByte > len(atlasSDKAggregateCleanupSource) || !reflect.DeepEqual(untouched, remoteTree(t, root)) {
		t.Fatalf("cleanup edit preview oracle: %d %s %s", code, &out, &diagnostic)
	}
	overlay := atlasSDKAggregateCleanupSource[:report.BodyStartByte] + atlasSDKAggregateCleanupEditBody + atlasSDKAggregateCleanupSource[report.BodyEndByte:]
	if overlay != atlasSDKAggregateCleanupEditedSource() || overlay == atlasSDKAggregateCleanupSource {
		t.Fatal("cleanup edit did not replace exactly the private leaf body")
	}
	writeWorkspaceFile(t, root, "src/main.fy", overlay)
	diagnostic.Reset()
	after, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, atlasmap.LogicalSchemaV3)
	if code != 0 || diagnostic.Len() != 0 {
		t.Fatalf("edited cleanup source oracle: %d %s", code, &diagnostic)
	}
	verifyAtlasSDKAggregateCleanupEditedOracle(t, before, after)
}
