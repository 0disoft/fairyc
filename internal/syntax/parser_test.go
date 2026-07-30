package syntax

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/diagnostic"
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

func TestPreviousRevisionAcceptsRevisionOneCShapedGrammar(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"function":         "fn i32 answer() { return 42; }\n",
		"declarations":     "struct Pair { i32 value; }\nfn i32 read(Pair pair) { let i32 value = pair.value; return value; }\n",
		"resource cleanup": "resource File { i32 handle; cleanup { let i32 seen = this.handle; } }\n",
		"control":          "fn i32 choose(bool flag) { if (flag) { return 1; } else { return 0; } }\n",
		"switch":           "fn i32 choose(Result<i32, i32> value) { return switch (value) { case .ok(ok) : { ok } case .err(err) : { err } }; }\n",
		"classic for":      "fn i32 count() { var i32 value = 0; for (var i32 index = 0; index < 2; index = index + 1) { value = value + 1; } return value; }\n",
	}
	for name, body := range tests {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			input := "module main;\n" + body
			file := source.NewFile("main.fy", []byte(input))
			previous := ParsePreviousRevision(file, source.DefaultLimits())
			if len(previous.Diagnostics) != 0 {
				t.Fatalf("ParsePreviousRevision() diagnostics = %+v", previous.Diagnostics)
			}
			current := Parse(file, source.DefaultLimits())
			if len(current.Diagnostics) != 0 {
				t.Fatalf("Parse() diagnostics = %+v", current.Diagnostics)
			}
		})
	}
}

func TestParseCShapedParametersAndMigratedMutableLocalLosslessly(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 close(File input, u32 status) {\n    var File file = input; return 0; }\n")
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
	if len(parameters) != 2 || parameters[0].Text != "input" || parameters[0].Cooked != "" ||
		parameters[1].Text != "status" || parameters[1].Cooked != "" {
		t.Fatalf("parameters = %+v", parameters)
	}
	foundMutableLocal := false
	for _, node := range tree.Nodes {
		if node.Kind == KindBinding && node.Text == "file" && node.Cooked == "var" {
			foundMutableLocal = true
			break
		}
	}
	if !foundMutableLocal {
		t.Fatalf("nodes contain no migrated var local: %+v", tree.Nodes)
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestDeclarationDocumentationAttachmentIsExactAndConservative(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "single line", body: "/// Returns one.\nfn i32 one() { return 1; }\n", want: "/// Returns one."},
		{name: "owner qualified function", body: "/// Reads a value.\nfn i32 File::value(File this) { return 1; }\n", want: "/// Reads a value."},
		{name: "consecutive CRLF and attribute", body: "\t/// First.\r\n  /// Second.\r\n@effects(io)\r\npub fn i32 value() { return 1; }\r\n", want: "/// First.\r\n  /// Second."},
		{name: "blank line detaches", body: "/// Detached.\n\nfn i32 value() { return 1; }\n"},
		{name: "ordinary comment detaches", body: "/// Detached.\n// ordinary\nfn i32 value() { return 1; }\n"},
		{name: "block comment is not documentation", body: "/** not admitted */\nfn i32 value() { return 1; }\n"},
		{name: "same physical line is not leading documentation", body: "/// trailing after module\nfn i32 value() { return 1; }\n"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			prefix := "module main;\n"
			if test.name == "same physical line is not leading documentation" {
				prefix = "module main; "
			}
			input := []byte(prefix + test.body)
			tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
			if len(tree.Diagnostics) != 0 {
				t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
			}
			var function *Node
			for index := range tree.Nodes {
				if tree.Nodes[index].Kind == KindFunction {
					function = &tree.Nodes[index]
					break
				}
			}
			if function == nil {
				t.Fatal("function node was not produced")
			}
			got := ""
			if function.DocumentationSpan.EndByte > function.DocumentationSpan.StartByte {
				got = string(input[function.DocumentationSpan.StartByte:function.DocumentationSpan.EndByte])
			}
			if got != test.want {
				t.Fatalf("documentation = %q, want %q; span = %+v", got, test.want, function.DocumentationSpan)
			}
		})
	}
}

func TestDocumentationDoesNotAttachToUnsupportedTopLevelConst(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\n/// Constant text remains unattached.\nconst i32 value = 1;\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindConst {
			if tree.Nodes[index].DocumentationSpan != (diagnostic.Span{}) {
				t.Fatalf("const documentation = %+v", tree.Nodes[index].DocumentationSpan)
			}
			return
		}
	}
	t.Fatal("const node was not produced")
}

func TestDocumentationDoesNotAttachToNestedFields(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\n/// Type docs.\nstruct Pair {\n  /// Field text remains ordinary trivia.\n  i32 value;\n}\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var structure, field *Node
	for index := range tree.Nodes {
		switch tree.Nodes[index].Kind {
		case KindStruct:
			structure = &tree.Nodes[index]
		case KindField:
			field = &tree.Nodes[index]
		}
	}
	if structure == nil || structure.DocumentationSpan.EndByte <= structure.DocumentationSpan.StartByte {
		t.Fatalf("struct documentation = %+v", structure)
	}
	if field == nil || field.DocumentationSpan != (diagnostic.Span{}) {
		t.Fatalf("field documentation = %+v", field)
	}
}

func TestNestedTypeCloseKeepsShiftTokenLossless(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn Option<Option<Token>> wrap(Token value) { return value; }\n")
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
	input := []byte("module main;\nfn unsafe extern C fn u32(i32, ptr mut u8) inspect(Result<Option<ref mut u8[]>, ptr void> value) { return callback; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	want := map[string]TypeForm{
		"Result<Option<ref mut u8[]>, ptr void>": TypeFormNamed,
		"Option<ref mut u8[]>":                   TypeFormNamed,
		"ref mut u8[]":                           TypeFormReferenceMut,
		"u8[]":                                   TypeFormSlice,
		"ptr void":                               TypeFormPointer,
		"unsafe extern C fn u32(i32, ptr mut u8)": TypeFormForeignFunction,
		"ptr mut u8": TypeFormPointerMut,
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
	input := []byte("module main;\nfn ptr u8[16] inspect(ref mut u8[] values, (ptr u8)[16] callbacks) { return callback; }\n")
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

func TestParseContextualUserEnumVariants(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nenum Message { Empty; Pair { i32 left; i32 right; }; }\nfn Message make() { return .Pair { .left = 20, .right = 22 }; }\nfn i32 score(Message value) { return switch (value) { case .Empty: { 0 } case .Pair { .left = first, .right = second }: { first + second } }; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var contextualRecords int
	var contextualPatterns int
	for _, node := range tree.Nodes {
		if node.Kind == KindRecord && node.Text == ".Pair" {
			contextualRecords++
		}
		if node.Kind == KindPattern && node.Text == ".Pair" {
			contextualPatterns++
		}
	}
	if contextualRecords != 1 || contextualPatterns != 1 {
		t.Fatalf("contextual records = %d, patterns = %d", contextualRecords, contextualPatterns)
	}
}

func TestNestedTypeCloseChargesContextualTokenBudget(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn Option<Option<Token>> wrap(Token value) { return value; }\n")
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
	input := []byte("module main;\nfn i32 main() {\n  let value = 1\n  return value;\n}\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 1 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	diagnostic := tree.Diagnostics[0]
	if diagnostic.Code != "E0007" || diagnostic.Repair != "insert" || diagnostic.Token != ";" {
		t.Fatalf("Parse() diagnostic = %+v", diagnostic)
	}
	position := len("module main;\nfn i32 main() {\n  let value = 1")
	if diagnostic.Span.StartByte != position || diagnostic.Span.EndByte != position {
		t.Fatalf("Parse() span = %+v, want zero-width at %d", diagnostic.Span, position)
	}
}

func TestParseImmutableByteViewSlice(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn ref u8[] main() { return b\"fairy\"[1..4]; }\n")
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
			input := []byte("module main;\nfn ref u8[] main() { return b\"fairy\"[" + bounds + "]; }\n")
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
	input := []byte("module main;\nfn i32 main() { assert(true); return 42; }\n")
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
	input := []byte("module main;\nfn i32 main() { let list = [1, 2, 3]; let repeated = [4; 2]; let empty = []; return list[0]; }\n")
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

func TestParseClassicForClausesLosslessly(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { for (var i32 index = 0; index < 6; index = index + 1) { if (index == 2) { continue; }; } return 42; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var loop *Node
	for index := range tree.Nodes {
		if tree.Nodes[index].Kind == KindFor {
			loop = &tree.Nodes[index]
			break
		}
	}
	if loop == nil || len(loop.Children) != 4 {
		t.Fatalf("for node = %+v", loop)
	}
	want := []Kind{KindForInit, KindForCond, KindForPost, KindBlock}
	for index, childID := range loop.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != want[index] {
			t.Fatalf("for child %d = %+v, found %v; want %s", index, child, found, want[index])
		}
	}
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want exact input", reconstructed, ok)
	}
}

func TestParseCompoundAssignmentsPreservesOperators(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { var i32 value = 1; value += 2; value <<= 1; for (var i32 index = 0; index < 2; index += 1) { value ^= index; } return value; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	var operators []string
	for _, node := range tree.Nodes {
		if node.Kind == KindAssignment {
			operators = append(operators, node.Text)
		}
	}
	if !slices.Equal(operators, []string{"+=", "<<=", "+=", "^="}) {
		t.Fatalf("assignment operators = %v", operators)
	}
	if reconstructed, ok := tree.Reconstruct(); !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want exact input", reconstructed, ok)
	}
}

func TestPreviousRevisionAcceptsCompoundAssignment(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { var i32 value = 40; value += 2; return value; }\n")
	tree := ParsePreviousRevision(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("ParsePreviousRevision() diagnostics = %+v", tree.Diagnostics)
	}
	if reconstructed, ok := tree.Reconstruct(); !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want exact rejected input", reconstructed, ok)
	}
}

func TestParseStaticAssertAsCurrentContextualItem(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nstatic_assert((40 + 2) == 42);\nfn i32 static_assert() { return 42; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	assertions := 0
	for _, node := range tree.Nodes {
		if node.Kind == KindStaticAssert {
			assertions++
		}
	}
	if assertions != 1 {
		t.Fatalf("static assertion nodes = %d, want 1", assertions)
	}
	if reconstructed, ok := tree.Reconstruct(); !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want exact input", reconstructed, ok)
	}
	previous := ParsePreviousRevision(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(previous.Diagnostics) == 0 {
		t.Fatal("ParsePreviousRevision() accepted current-only static_assert item")
	}
}

func TestParseLexicalDeferLosslessly(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn void record(i32 value) {}\nfn i32 main() { defer record(42); return 42; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	found := false
	for _, node := range tree.Nodes {
		found = found || node.Kind == KindDefer
	}
	if !found {
		t.Fatal("Parse() produced no defer node")
	}
	if reconstructed, ok := tree.Reconstruct(); !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want exact input", reconstructed, ok)
	}
	previous := ParsePreviousRevision(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(previous.Diagnostics) == 0 {
		t.Fatal("ParsePreviousRevision() accepted current-only defer")
	}
}

func TestParseClassicForPreservesOmittedClauses(t *testing.T) {
	t.Parallel()
	input := []byte("module main;\nfn i32 main() { for (;;) { break; } return 42; }\n")
	tree := Parse(source.NewFile("main.fy", input), source.DefaultLimits())
	if len(tree.Diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", tree.Diagnostics)
	}
	for _, node := range tree.Nodes {
		if node.Kind == KindForInit || node.Kind == KindForCond || node.Kind == KindForPost {
			if len(node.Children) != 0 {
				t.Fatalf("omitted clause %s children = %v", node.Kind, node.Children)
			}
		}
	}
}
