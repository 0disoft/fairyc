package syntax

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

func TestParseBudgetFailsClosed(t *testing.T) {
	t.Parallel()
	input := []byte("module main;")
	limits := source.DefaultLimits()
	limits.MaxBytes = len(input) - 1
	tree := Parse(source.NewFile("main.fy", input), limits)
	if len(tree.Diagnostics) != 1 || tree.Diagnostics[0].Code != "E0010" {
		t.Fatalf("Parse() diagnostics = %+v, want one E0010", tree.Diagnostics)
	}
}

func TestParseMutableOrdinaryParameterLosslessly(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn close(mut file: File, status: u32) -> i32 { 0 }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	parameters := make([]Node, 0, 2)
	for _, node := range tree.Nodes {
		if node.Kind == KindParameter {
			parameters = append(parameters, node)
		}
	}
	if len(parameters) != 2 || parameters[0].Text != "file" || parameters[0].Cooked != "mut" ||
		parameters[1].Text != "status" || parameters[1].Cooked != "" {
		t.Fatalf("parameters = %+v", parameters)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestNestedTypeCloseKeepsShiftTokenLossless(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn wrap(value: Token) -> Option<Option<Token>> { value }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
	foundShiftToken := false
	for _, token := range tree.Tokens {
		if token.Text(tree.Source) == ">>" {
			foundShiftToken = true
			break
		}
	}
	if !foundShiftToken {
		t.Fatal("Parse() did not preserve the original >> token")
	}
	outer, inner := false, false
	for _, node := range tree.Nodes {
		if node.Kind != KindType {
			continue
		}
		switch node.Text {
		case "Option<Option<Token>>":
			outer = true
		case "Option<Token>":
			inner = true
		}
	}
	if !outer || !inner {
		t.Fatalf("Parse() nested type nodes = outer %v, inner %v", outer, inner)
	}
}

func TestTypeNodesRecordParserOwnedForms(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn inspect(value: Result<Option<&mut [u8]>, *void>) -> unsafe extern C fn(i32, *mut u8) -> u32 { callback }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	want := map[string]TypeForm{
		"Result<Option<&mut [u8]>, *void>": TypeFormNamed,
		"Option<&mut [u8]>":                TypeFormNamed,
		"&mut [u8]":                        TypeFormReferenceMut,
		"[u8]":                             TypeFormSlice,
		"*void":                            TypeFormPointer,
		"unsafe extern C fn(i32, *mut u8) -> u32": TypeFormForeignFunction,
		"*mut u8": TypeFormPointerMut,
	}
	found := make(map[string]TypeForm, len(want))
	for _, node := range tree.Nodes {
		if node.Kind == KindType {
			if _, tracked := want[node.Text]; tracked {
				found[node.Text] = TypeForm(node.Cooked)
			}
		}
	}
	for text, form := range want {
		if found[text] != form {
			t.Fatalf("type %q form = %q, want %q; found = %+v", text, found[text], form, found)
		}
	}
}

func TestParseCShapedTypeForms(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn inspect(values: ref mut u8[], callbacks: (ptr u8)[16]) -> ptr u8[16] { callback }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	want := map[string]TypeForm{
		"ref mut u8[]": TypeFormReferenceMut,
		"u8[]":         TypeFormSlice,
		"(ptr u8)":     TypeFormGrouped,
		"(ptr u8)[16]": TypeFormArray,
		"ptr u8":       TypeFormPointer,
		"ptr u8[16]":   TypeFormPointer,
		"u8[16]":       TypeFormArray,
	}
	found := make(map[string]TypeForm, len(want))
	for _, node := range tree.Nodes {
		if node.Kind == KindType {
			if _, tracked := want[node.Text]; tracked {
				found[node.Text] = TypeForm(node.Cooked)
			}
		}
	}
	for text, form := range want {
		if found[text] != form {
			t.Fatalf("type %q form = %q, want %q; found = %+v", text, found[text], form, found)
		}
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestParseCShapedDeclarationsAndNameSpans(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nconst usize LIMIT = 4;\nstruct Pair { i32 left; ptr u8 data; }\nenum Error { Os { i32 code; }; }\nfn Token Token(Token Token) { return Token; }\nunsafe extern C fn i32 c_add(i32 left, i32 right);\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var function *Node
	var parameter *Node
	for index := range tree.Nodes {
		node := &tree.Nodes[index]
		if node.Kind == KindFunction && node.Text == "Token" {
			function = node
		}
		if node.Kind == KindParameter && node.Text == "Token" {
			parameter = node
		}
	}
	functionName := strings.Index(string(input), "Token Token") + len("Token ")
	parameterName := strings.Index(string(input), "Token Token)") + len("Token ")
	if function == nil || function.NameSpan.StartByte != functionName || function.NameSpan.EndByte != functionName+len("Token") {
		t.Fatalf("function = %+v, want name span %d..%d", function, functionName, functionName+len("Token"))
	}
	if parameter == nil || parameter.NameSpan.StartByte != parameterName || parameter.NameSpan.EndByte != parameterName+len("Token") {
		t.Fatalf("parameter = %+v, want name span %d..%d", parameter, parameterName, parameterName+len("Token"))
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestCShapedFunctionRequiresExplicitReturn(t *testing.T) {
	t.Parallel()
	tree := Parse(source.NewFile("main.fy", []byte("module main;\nfn i32 main() { 0 }\n")), source.DefaultLimits())
	if len(tree.Diagnostics) != 1 || tree.Diagnostics[0].Code != "E0007" || tree.Diagnostics[0].Token != ";" {
		t.Fatalf("Parse() diagnostics = %+v, want missing statement terminator", tree.Diagnostics)
	}
}

func TestParseCShapedControlBorrowAndRecordDesignators(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nstruct Pair { i32 left; i32 right; }\nfn i32 main() { var u8[2] values = [1, 2]; let ref mut u8[] view = ref mut values; let Pair pair = Pair { .left = 20, .right = 22 }; while (view.len == 2) { break; } if (true) { return pair.left; } else { return 0; } }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var mutableBorrow bool
	var designatedFields int
	for _, node := range tree.Nodes {
		if node.Kind == KindPrefix && node.Text == "&mut" {
			mutableBorrow = true
		}
		if node.Kind == KindField && node.NameSpan.EndByte > node.NameSpan.StartByte && (node.Text == "left" || node.Text == "right") {
			designatedFields++
		}
	}
	if !mutableBorrow || designatedFields < 2 {
		t.Fatalf("mutable borrow = %v, designated fields = %d", mutableBorrow, designatedFields)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestParseCShapedSwitchAndLeadingDotVariants(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn Result<i32, i32> increment(Result<i32, i32> input) { return switch (input) { case .ok(value): { .ok(value + 1) } case .err(error): { .err(error) } }; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var switchCount int
	paths := make(map[string]int)
	for _, node := range tree.Nodes {
		if node.Kind == KindSwitch {
			switchCount++
		}
		if node.Kind == KindPath && strings.HasPrefix(node.Text, ".") {
			paths[node.Text]++
		}
	}
	if switchCount != 1 || paths[".ok"] != 2 || paths[".err"] != 2 {
		t.Fatalf("switches = %d, leading-dot paths = %+v", switchCount, paths)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestParseCShapedResourceCleanup(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nresource File { c::int handle; @effects(filesystem) cleanup { unsafe { close(this.handle); } } }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var cleanup *Node
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindCleanup {
			cleanup = &tree.Nodes[index]
			break
		}
	}
	if cleanup == nil || cleanup.Text != "cleanup" || len(cleanup.Children) != 2 {
		t.Fatalf("cleanup = %+v", cleanup)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestParseOwnerQualifiedFunction(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nstruct Counter { i32 value; }\nfn i32 Counter::double(i32 value) { return value * 2; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var owner *Node
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindOwner {
			owner = &tree.Nodes[index]
			break
		}
	}
	if owner == nil || owner.Text != "Counter" || owner.NameSpan.EndByte <= owner.NameSpan.StartByte {
		t.Fatalf("owner = %+v", owner)
	}
}

func TestNestedTypeCloseChargesContextualTokenBudget(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn wrap(value: Token) -> Option<Option<Token>> { value }\n")
	file := source.NewFile("main.fy", input)
	baseline := source.DefaultLimits()
	lexed := lexer.Lex(file, source.NewBudget(baseline, len(input)))
	if len(lexed.Diagnostics) != 0 {
		t.Fatalf("Lex() diagnostics = %+v", lexed.Diagnostics)
	}
	limits := source.DefaultLimits()
	limits.MaxTokens = len(lexed.Tokens) - 1
	tree := Parse(file, limits)
	if len(tree.Diagnostics) != 1 || tree.Diagnostics[0].Code != "E0010" {
		t.Fatalf("Parse() diagnostics = %+v, want one E0010", tree.Diagnostics)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestMissingSemicolonRecoversWithoutConsumingNextStatement(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 {\n  let value = 1\n  return value;\n}\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 1 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	diagnostic := tree.Diagnostics[0]
	if diagnostic.Code != "E0007" || diagnostic.Repair != "insert" || diagnostic.Token != ";" {
		t.Fatalf("Parse() diagnostic = %+v", diagnostic)
	}
	position := len("module main;\nfn main() -> i32 {\n  let value = 1")
	if diagnostic.Span.StartByte != position || diagnostic.Span.EndByte != position {
		t.Fatalf("Parse() span = %+v, want zero-width at %d", diagnostic.Span, position)
	}
}

func TestParseImmutableByteViewSlice(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> &[u8] { b\"fairy\"[1..4] }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var slice *Node
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindSlice {
			slice = &tree.Nodes[index]
			break
		}
	}
	if slice == nil || slice.Text != ".." || len(slice.Children) != 3 {
		t.Fatalf("slice node = %+v", slice)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestParseImmutableByteViewSliceWithOmittedBounds(t *testing.T) {
	t.Parallel()
	for _, bounds := range []string{"..4", "1..", ".."} {
		bounds := bounds
		t.Run(bounds, func(t *testing.T) {
			input := []byte("module main;\nfn main() -> &[u8] { b\"fairy\"[" + bounds + "] }\n")
			tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
			if len(tree.Diagnostics) != 0 {
				t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
			}
			var slice *Node
			for index := range tree.Nodes {
				if tree.Nodes[index].Kind == KindSlice {
					slice = &tree.Nodes[index]
				}
			}
			if slice == nil || len(slice.Children) != 3 {
				t.Fatalf("slice node = %+v", slice)
			}
			start, startFound := tree.Node(slice.Children[1])
			end, endFound := tree.Node(slice.Children[2])
			if !startFound || !endFound {
				t.Fatalf("slice bounds are missing from tree: start=%v end=%v", startFound, endFound)
			}
			if bounds[0] == '.' && (start.Kind != KindOmitBound || start.Text != "start") {
				t.Fatalf("slice start = %+v", start)
			}
			if bounds[len(bounds)-1] == '.' && (end.Kind != KindOmitBound || end.Text != "end") {
				t.Fatalf("slice end = %+v", end)
			}
			reconstructed, ok := tree.Reconstruct()
			if !ok || !slices.Equal(reconstructed, input) {
				t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
			}
		})
	}
}

func TestParseAlwaysCheckedAssertStatement(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 { assert(true); 42 }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var assertion *Node
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindAssert {
			assertion = &tree.Nodes[index]
			break
		}
	}
	if assertion == nil || len(assertion.Children) != 1 {
		t.Fatalf("assert node = %+v", assertion)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestParseDistinguishesFixedArrayLiteralForms(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn main() -> i32 { let list = [1, 2, 3]; let repeated = [4; 2]; let empty = []; list[0] }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	want := []string{"list", "repeat", "empty"}
	var got []string
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindArray {
			got = append(got, tree.Nodes[index].Text)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("array forms = %v, want %v", got, want)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}
