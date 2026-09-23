package syntax

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestParseByteViewForPreservesHeaderAndSource(t *testing.T) {
	input := []byte("module main; fn void show(ref u8[] view) { for (let u8 byte in view) { if (byte == 0) { continue; } } }")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", tree.Diagnostics)
	}
	var loop *Node
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindByteFor {
			loop = &tree.Nodes[index]
			break
		}
	}
	if loop == nil || loop.Text != "byte" || len(loop.Children) != 3 {
		t.Fatalf("byte for = %+v", loop)
	}
	if got := string(input[loop.NameSpan.StartByte:loop.NameSpan.EndByte]); got != "byte" {
		t.Fatalf("name span = %q", got)
	}
	for index, kind := range []Kind{KindType, KindPath, KindBlock} {
		child, ok := tree.Node(loop.Children[index])
		if !ok || child.Kind != kind {
			t.Fatalf("child %d = %+v, %v; want %s", index, child, ok, kind)
		}
	}
	if reconstructed, ok := tree.Reconstruct(); !ok || !bytes.Equal(reconstructed, input) {
		t.Fatalf("reconstruction = %q, %v", reconstructed, ok)
	}
	if previous := ParsePreviousRevision(source.NewFile("main.fy", input), source.DefaultLimits()); len(previous.Diagnostics) == 0 {
		t.Fatal("previous revision accepted byte-view for")
	}
}

func TestParseByteViewForKeepsClassicInitializerDistinct(t *testing.T) {
	input := "module main; fn i32 main() { for (let i32 index = 0; index < 2; index += 1) {} return 0; }"
	tree := Parse(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", tree.Diagnostics)
	}
	classic, bytes := 0, 0
	for _, node := range tree.Nodes {
		if node.Kind == KindFor {
			classic++
		}
		if node.Kind == KindByteFor {
			bytes++
		}
	}
	if classic != 1 || bytes != 0 {
		t.Fatalf("classic=%d byte=%d", classic, bytes)
	}
	for _, malformed := range []string{
		"for (let u8 byte in ) {}",
		"for (let u8 byte in view {}",
	} {
		sourceText := "module main; fn void main(ref u8[] view) { " + malformed + " }"
		if got := Parse(source.NewFile("main.fy", []byte(sourceText)), source.DefaultLimits()); len(got.Diagnostics) == 0 {
			t.Fatalf("accepted malformed %q", malformed)
		}
	}
}
