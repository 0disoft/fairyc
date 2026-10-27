package atlasmap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Build through the compiler and normal partition admission, outside timed queries.
// main -> branches -> left/right -> tail -> main exercises shared neighbors and cycles.
func wideQueryFixture(tb testing.TB, width int) (*QueryIndex, string) {
	tb.Helper()
	root := tb.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o700); err != nil {
		tb.Fatal(err)
	}
	manifest := "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.atlas-frontier\"\nname = \"atlas-frontier\"\nversion = \"1.0.0\"\nlanguage = \"fairyc-language/20\"\nsource = \"src\"\n"
	var source strings.Builder
	source.WriteString("module main; fn i32 main(){")
	for i := 0; i < width; i++ {
		fmt.Fprintf(&source, "discard branch_%d();", i)
	}
	source.WriteString("return 0;}")
	for i := 0; i < width; i++ {
		fmt.Fprintf(&source, "fn i32 branch_%d(){return left()+right();}", i)
	}
	source.WriteString("fn i32 left(){return tail();} fn i32 right(){return tail();} fn i32 tail(){return main();}")
	for path, contents := range map[string]string{"fairy.toml": manifest, "src/main.fy": source.String()} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(contents), 0o600); err != nil {
			tb.Fatal(err)
		}
	}
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		tb.Fatal(err)
	}
	input, err := workspace.SemanticInput()
	if err != nil {
		tb.Fatal(err)
	}
	checked, err := compiler.CheckPackages(input)
	if err != nil || !checked.Verified {
		tb.Fatalf("compile query graph: verified=%t, error=%v", checked.Verified, err)
	}
	build := testCompilerBuild()
	_, partitions, err := PrepareWorkspace(workspace, checked, build)
	if err != nil {
		tb.Fatal(err)
	}
	expectations, err := CurrentWorkspaceExpectations(workspace, checked, build)
	if err != nil {
		tb.Fatal(err)
	}
	for _, partition := range partitions {
		index, err := NewQueryIndex(partition, expectations[partition.Header().PackageInstance])
		if err != nil {
			tb.Fatal(err)
		}
		for _, node := range index.nodes {
			if node.declaration.Key.Name == "main" {
				return index, node.symbol.Visible
			}
		}
	}
	tb.Fatal("query graph has no main function")
	return nil, ""
}

func wideQueryRequest(width int, direction QueryDirection, depth int) RelationRequest {
	return RelationRequest{Direction: direction, Relations: []RelationKind{RelationCalls}, Budget: QueryBudget{
		MaxRecords: 6*width + 32, MaxRelations: 5*width + 32, MaxDepth: depth, MaxSelectionBytes: 64 << 20,
	}}
}

func TestAroundFrontierTracksEachDepthAndDirection(t *testing.T) {
	index, root := wideQueryFixture(t, 3)
	for _, test := range []struct {
		direction               QueryDirection
		depth, nodes, relations int
		limited                 bool
	}{
		{DirectionOutgoing, 2, 6, 9, true},
		{DirectionOutgoing, 4, 7, 12, false},
		{DirectionIncoming, 2, 4, 3, true},
		{DirectionIncoming, 4, 7, 12, false},
		{DirectionBoth, 2, 7, 12, false},
	} {
		t.Run(fmt.Sprintf("%s/depth=%d", test.direction, test.depth), func(t *testing.T) {
			result, err := index.Around(root, wideQueryRequest(3, test.direction, test.depth))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Nodes) != test.nodes || len(result.Relations) != test.relations || hasOmission(result.Omissions, OmissionDepthLimit) != test.limited {
				t.Fatalf("nodes=%d, relations=%d, omissions=%+v", len(result.Nodes), len(result.Relations), result.Omissions)
			}
			seen := make(map[string]bool)
			lastDepth := 0
			for _, node := range result.Nodes {
				if seen[node.Node.Symbol.Visible] || node.Depth < lastDepth {
					t.Fatalf("duplicate or out-of-order BFS node: %+v", node)
				}
				seen[node.Node.Symbol.Visible], lastDepth = true, node.Depth
			}
		})
	}
	request := wideQueryRequest(3, DirectionBoth, 4)
	request.Budget.MaxRelations = 0
	result, err := index.Around(root, request)
	if err != nil || len(result.Nodes) != 1 || len(result.Relations) != 0 || !hasOmission(result.Omissions, OmissionBudgetRelations) {
		t.Fatalf("zero-relation budget: %+v, %v", result, err)
	}
}

func BenchmarkAroundWideFrontier(b *testing.B) {
	for _, width := range []int{64, 512} {
		b.Run(fmt.Sprintf("width=%d", width), func(b *testing.B) {
			index, root := wideQueryFixture(b, width)
			request := wideQueryRequest(width, DirectionOutgoing, 4)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, err := index.Around(root, request)
				if err != nil || len(result.Nodes) != width+4 || len(result.Relations) != 3*width+3 {
					b.Fatalf("incomplete graph: nodes=%d, relations=%d, error=%v", len(result.Nodes), len(result.Relations), err)
				}
			}
		})
	}
}
