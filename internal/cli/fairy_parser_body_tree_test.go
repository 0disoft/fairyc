package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"github.com/0disoft/fairyc/internal/target"
)

// This is a disposable comparison format, not a serialized compiler AST.
func referenceBodyTree(input string) (string, error) {
	return referenceComparisonTree(input, false)
}

func referenceComparisonTree(input string, header bool) (string, error) {
	if len(input) > 32768 {
		return "", fmt.Errorf("source budget")
	}
	r, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(input)), source.DefaultLimits())
	if err != nil || r.Tree == nil || len(r.Tree.Diagnostics) != 0 || r.AST == nil || len(r.AST.Functions) != 1 {
		return "", fmt.Errorf("body input is not one error-free function")
	}
	root, ok := r.Tree.Node(r.Tree.Root)
	if !ok || len(root.Children) != 2 {
		return "", fmt.Errorf("expected module and one function only")
	}
	fn := r.AST.Functions[0]
	type attributeOrigin struct {
		name  diagnostic.Span
		kinds []int
	}
	var attributeOrigins []attributeOrigin
	var publicSpan, unsafeSpan diagnostic.Span
	var genericSpan diagnostic.Span
	var genericNames []diagnostic.Span
	if header {
		decl, found := r.Tree.Node(root.Children[1])
		if !found || decl.Kind != syntax.KindFunction {
			return "", fmt.Errorf("expected ordinary function")
		}
		modifierStart := decl.TokenStart
		// The AST retains generic names, while their declaration origins live in
		// the owning CST list. Do not search unrelated type uses by spelling.
		for _, id := range decl.Children {
			list, ok := r.Tree.Node(id)
			if ok && list.Kind == syntax.KindAttribute {
				index := len(attributeOrigins)
				if index >= len(fn.Attributes) || len(list.Children) == 0 {
					return "", fmt.Errorf("attribute origin missing")
				}
				attribute := fn.Attributes[index]
				name, found := r.Tree.Node(list.Children[0])
				if !found || name.Kind != syntax.KindPath || name.Text != attribute.Name || len(list.Children)-1 != len(attribute.Arguments) {
					return "", fmt.Errorf("attribute disagrees with declaration origin")
				}
				origin := attributeOrigin{name: name.Span}
				for i, argID := range list.Children[1:] {
					arg, found := r.Tree.Node(argID)
					if !found || arg.Text != attribute.Arguments[i].Raw || arg.Span != attribute.Arguments[i].Span {
						return "", fmt.Errorf("attribute argument origin mismatch")
					}
					kind := 0
					if arg.Kind == syntax.KindPath {
						kind = 401
					} else {
						for _, token := range r.Tree.Tokens[arg.TokenStart:arg.TokenEnd] {
							if token.Kind == lexer.Trivia {
								continue
							}
							if token.Kind == lexer.Integer {
								kind = 402
							}
							if token.Kind == lexer.String {
								kind = 403
							}
							break
						}
					}
					if kind == 0 {
						return "", fmt.Errorf("unsupported attribute argument")
					}
					origin.kinds = append(origin.kinds, kind)
				}
				attributeOrigins = append(attributeOrigins, origin)
				modifierStart = max(modifierStart, list.TokenEnd)
			}
			if !ok || list.Kind != syntax.KindTypeParameters {
				continue
			}
			genericSpan = list.Span
			for _, nameID := range list.Children {
				name, ok := r.Tree.Node(nameID)
				i := len(genericNames)
				if !ok || name.Kind != syntax.KindPath || i >= len(fn.TypeParameters) || name.Text != fn.TypeParameters[i] {
					return "", fmt.Errorf("generic parameter names disagree with declaration origins")
				}
				genericNames = append(genericNames, name.NameSpan)
			}
		}
		if len(genericNames) != len(fn.TypeParameters) {
			return "", fmt.Errorf("generic parameter count disagrees with declaration origins")
		}
		if len(attributeOrigins) != len(fn.Attributes) {
			return "", fmt.Errorf("attribute count mismatch")
		}
		// Modifier origins belong to the declaration prefix, not its result type.
		// Keep the independently read origins consistent with the lowered flags.
		for _, token := range r.Tree.Tokens[modifierStart:decl.TokenEnd] {
			if token.Kind != lexer.Keyword {
				continue
			}
			word := token.Text(r.Tree.Source)
			if word == "fn" {
				break
			}
			if word == "pub" {
				publicSpan = token.Span
			}
			if word == "unsafe" {
				unsafeSpan = token.Span
			}
		}
		if fn.Public != (publicSpan.EndByte > publicSpan.StartByte) || fn.Unsafe != (unsafeSpan.EndByte > unsafeSpan.StartByte) {
			return "", fmt.Errorf("function modifier flags disagree with declaration origins")
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "1 %d ", len(input))
	for _, b := range []byte(input) {
		fmt.Fprintf(&out, "%d ", b)
	}
	count := 0
	var failure error
	record := func(kind int, span diagnostic.Span, children int, name diagnostic.Span, depth int, operator string) bool {
		count++
		if depth >= 64 || count > 256 {
			failure = fmt.Errorf("body export budget")
			return false
		}
		fmt.Fprintf(&out, "%d %d %d %d %d %d ", kind, span.StartByte, span.EndByte, children, name.StartByte, name.EndByte)
		fmt.Fprintf(&out, "%d ", len(operator))
		for _, b := range []byte(operator) {
			fmt.Fprintf(&out, "%d ", b)
		}
		return true
	}
	var block func(*ast.Block, int) bool
	var expr func(*ast.Expression, int) bool
	var stmt func(ast.Statement, int) bool
	// Surface AST keeps the generic path spelling, but its precise path-only
	// span belongs to the CST. Read that origin instead of reparsing Type.Text.
	paths := map[[2]int]diagnostic.Span{}
	fieldNames := map[[2]int]diagnostic.Span{}
	patternKinds := map[[2]int]int{}
	forInitializers := map[[2]int]diagnostic.Span{}
	regionNames := map[[2]int]diagnostic.Span{}
	for _, node := range r.Tree.Nodes {
		if node.Kind == syntax.KindRegion {
			for _, token := range r.Tree.Tokens[node.TokenStart:node.TokenEnd] {
				if token.Kind == lexer.Identifier {
					regionNames[[2]int{node.Span.StartByte, node.Span.EndByte}] = token.Span
					break
				}
			}
		}
		if node.Kind == syntax.KindFor && len(node.Children) > 0 {
			if clause, ok := r.Tree.Node(node.Children[0]); ok && clause.Kind == syntax.KindForInit {
				forInitializers[[2]int{node.Span.StartByte, node.Span.EndByte}] = clause.Span
			}
		}
		if node.Kind == syntax.KindField {
			fieldNames[[2]int{node.Span.StartByte, node.Span.EndByte}] = node.NameSpan
			if node.NameSpan.EndByte == node.NameSpan.StartByte {
				for _, token := range r.Tree.Tokens[node.TokenStart:node.TokenEnd] {
					if token.Kind == lexer.Identifier {
						fieldNames[[2]int{node.Span.StartByte, node.Span.EndByte}] = token.Span
						break
					}
				}
			}
		}
		if node.Kind == syntax.KindPattern && node.Text != "arm" && node.TokenEnd > node.TokenStart {
			kind := 225
			switch r.Tree.Tokens[node.TokenEnd-1].Text(r.Tree.Source) {
			case ")":
				kind = 226
			case "}":
				kind = 227
			}
			patternKinds[[2]int{node.Span.StartByte, node.Span.EndByte}] = kind
		}
		if node.Kind == syntax.KindType && node.Cooked == string(syntax.TypeFormNamed) && len(node.Children) != 0 {
			if path, ok := r.Tree.Node(node.Children[0]); ok && path.Kind == syntax.KindPath {
				paths[[2]int{node.Span.StartByte, node.Span.EndByte}] = path.Span
			}
		}
	}
	var typeTree func(*ast.TypeReference, int) bool
	typeTree = func(typ *ast.TypeReference, d int) bool {
		if typ == nil {
			return false
		}
		k, n := 0, 0
		switch typ.Kind {
		case ast.TypeKindNamed:
			k = 301
			if len(typ.Arguments) != 0 {
				k, n = 306, len(typ.Arguments)+1
			}
		case ast.TypeKindReference:
			k, n = 302, 1
			if typ.Mutable {
				k = 303
			}
		case ast.TypeKindRawPointer:
			k, n = 311, 1
			if typ.Mutable {
				k = 312
			}
		case ast.TypeKindSlice:
			k, n = 304, 1
		case ast.TypeKindGrouped:
			k, n = 305, 1
		case ast.TypeKindArray:
			k, n = 308, 2
		case ast.TypeKindForeignFunction:
			if typ.ABI != "C" || typ.Result == nil {
				return false
			}
			k, n = 315, len(typ.Parameters)+1
		default:
			return false
		}
		if !record(k, typ.Span, n, diagnostic.Span{}, d, "") {
			return false
		}
		if k == 301 {
			return true
		}
		if k == 306 {
			span, ok := paths[[2]int{typ.Span.StartByte, typ.Span.EndByte}]
			if !ok || !record(301, span, 0, diagnostic.Span{}, d+1, "") {
				return false
			}
			for i := range typ.Arguments {
				if !typeTree(&typ.Arguments[i], d+1) {
					return false
				}
			}
			return true
		}
		if k == 315 {
			if !typeTree(typ.Result, d+1) {
				return false
			}
			for i := range typ.Parameters {
				if !typeTree(&typ.Parameters[i], d+1) {
					return false
				}
			}
			return true
		}
		if !typeTree(typ.Element, d+1) {
			return false
		}
		if k == 308 {
			if typ.Length == nil {
				return false
			}
			lengthKind := 309
			if typ.Length.Kind == syntax.KindPath {
				lengthKind = 310
			} else if typ.Length.Kind != syntax.KindLiteral {
				return false
			}
			return record(lengthKind, typ.Length.Span, 0, diagnostic.Span{}, d+1, "")
		}
		return true
	}
	patternTree := func(p *ast.Expression, d int) bool {
		if p == nil || p.Kind != syntax.KindPattern || len(p.Children) == 0 {
			return false
		}
		kind := patternKinds[[2]int{p.Span.StartByte, p.Span.EndByte}]
		if kind == 0 || (kind == 225 && len(p.Children) != 1) || (kind == 226 && len(p.Children) != 2) {
			return false
		}
		if !record(kind, p.Span, len(p.Children), diagnostic.Span{}, d, "") {
			return false
		}
		variant := p.Children[0]
		if variant.Kind != syntax.KindPath || !record(201, variant.Span, 0, diagnostic.Span{}, d+1, "") {
			return false
		}
		for _, child := range p.Children[1:] {
			if kind == 226 {
				if child.Kind != syntax.KindPath || !record(229, child.Span, 0, diagnostic.Span{}, d+1, "") {
					return false
				}
			} else {
				name, ok := fieldNames[[2]int{child.Span.StartByte, child.Span.EndByte}]
				if child.Kind != syntax.KindField || !ok || name.EndByte <= name.StartByte || child.Cooked == "" || len(child.Children) != 0 {
					return false
				}
				if !record(228, child.Span, 1, name, d+1, "") {
					return false
				}
				binding := diagnostic.Span{StartByte: child.Span.EndByte - len(child.Cooked), EndByte: child.Span.EndByte}
				if !record(229, binding, 0, diagnostic.Span{}, d+2, "") {
					return false
				}
			}
		}
		return true
	}
	block = func(b *ast.Block, d int) bool {
		if b == nil {
			return false
		}
		n := len(b.Statements)
		if b.Tail != nil {
			n++
		}
		if !record(100, b.Span, n, diagnostic.Span{}, d, "") {
			return false
		}
		for _, s := range b.Statements {
			if !stmt(s, d+1) {
				return false
			}
		}
		return b.Tail == nil || expr(b.Tail, d+1)
	}
	expr = func(e *ast.Expression, d int) bool {
		if e == nil || (e.Type != nil && e.Kind != syntax.KindCast) || (len(e.TypeArguments) != 0 && e.Kind != syntax.KindSpecialize && e.Kind != syntax.KindRecord) {
			return false
		}
		k := map[syntax.Kind]int{syntax.KindLiteral: 200, syntax.KindPath: 201, syntax.KindGroup: 202, syntax.KindPrefix: 203, syntax.KindBinary: 204, syntax.KindCall: 205, syntax.KindBlock: 206, syntax.KindIf: 207, syntax.KindCast: 208, syntax.KindSpecialize: 209, syntax.KindFieldExpr: 210, syntax.KindIndex: 211, syntax.KindSlice: 212}[e.Kind]
		if e.Kind == syntax.KindOmitBound {
			if e.Text == "start" {
				k = 213
			} else if e.Text == "end" {
				k = 214
			}
		}
		if e.Kind == syntax.KindArray {
			if e.Text == "empty" || e.Text == "list" {
				k = 215
			} else if e.Text == "repeat" {
				k = 218
			}
		}
		if e.Kind == syntax.KindRecord {
			k = 216
		}
		if e.Kind == syntax.KindField {
			k = 217
		}
		if e.Kind == syntax.KindPrefix && e.Text == "&" {
			k = 219
		}
		if e.Kind == syntax.KindPrefix && e.Text == "&mut" {
			k = 220
		}
		if e.Kind == syntax.KindPropagate {
			k = 221
		}
		if e.Kind == syntax.KindUnsafe {
			k = 222
		}
		if e.Kind == syntax.KindSwitch {
			k = 223
		}
		if k == 0 {
			failure = fmt.Errorf("unhandled expression %s at %d", e.Kind, e.Span.StartByte)
			return false
		}
		n := len(e.Children)
		if k >= 219 && k <= 222 && n != 1 {
			return false
		}
		if k == 216 {
			if n == 0 {
				return false
			}
			n += len(e.TypeArguments)
		}
		if k == 208 {
			if e.Type == nil || n != 1 {
				return false
			}
			n++
		}
		if k == 209 {
			if n != 1 || len(e.TypeArguments) == 0 {
				return false
			}
			n += len(e.TypeArguments)
		}
		if k == 206 {
			n = 1
		}
		operator := ""
		if k == 203 || k == 204 {
			operator = e.Text
		}
		name := diagnostic.Span{}
		if k == 210 {
			name = diagnostic.Span{StartByte: e.Span.EndByte - len(e.Text), EndByte: e.Span.EndByte}
		}
		if k == 217 {
			var ok bool
			name, ok = fieldNames[[2]int{e.Span.StartByte, e.Span.EndByte}]
			if !ok || name.EndByte <= name.StartByte || n != 1 {
				return false
			}
		}
		if !record(k, e.Span, n, name, d, operator) {
			return false
		}
		if k == 206 {
			return block(e.Block, d+1)
		}
		if k == 222 {
			if e.Children[0].Kind != syntax.KindBlock {
				return false
			}
			return block(e.Children[0].Block, d+1)
		}
		if k == 223 {
			if n == 0 || !expr(e.Children[0], d+1) {
				return false
			}
			for _, arm := range e.Children[1:] {
				if arm.Kind != syntax.KindPattern || arm.Text != "arm" || len(arm.Children) != 2 || arm.Children[1].Kind != syntax.KindBlock {
					return false
				}
				if !record(224, arm.Span, 2, diagnostic.Span{}, d+1, "") || !patternTree(arm.Children[0], d+2) || !block(arm.Children[1].Block, d+2) {
					return false
				}
			}
			return true
		}
		if k == 216 {
			if !expr(e.Children[0], d+1) {
				return false
			}
			for i := range e.TypeArguments {
				if !typeTree(&e.TypeArguments[i], d+1) {
					return false
				}
			}
			for _, field := range e.Children[1:] {
				if field.Kind != syntax.KindField || !expr(field, d+1) {
					return false
				}
			}
			return true
		}
		for i, c := range e.Children {
			if k == 207 && i > 0 && c.Kind == syntax.KindBlock {
				if !block(c.Block, d+1) {
					return false
				}
			} else if !expr(c, d+1) {
				return false
			}
		}
		if k == 208 {
			return typeTree(e.Type, d+1)
		}
		for i := range e.TypeArguments {
			if !typeTree(&e.TypeArguments[i], d+1) {
				return false
			}
		}
		return true
	}
	stmt = func(s ast.Statement, d int) bool {
		if s.Kind == syntax.KindFor {
			loop := s.For
			if loop == nil || loop.Body == nil || !record(111, s.Span, 4, diagnostic.Span{}, d, "") {
				return false
			}
			if loop.Initializer != nil {
				initial := *loop.Initializer
				if initial.Kind != syntax.KindBinding && initial.Kind != syntax.KindAssignment {
					return false
				}
				span, ok := forInitializers[[2]int{s.Span.StartByte, s.Span.EndByte}]
				if !ok {
					return false
				}
				// Include the written semicolon via its CST clause; leave the AST immutable.
				initial.Span = span
				if !stmt(initial, d+1) {
					return false
				}
			} else if !record(112, diagnostic.Span{}, 0, diagnostic.Span{}, d+1, "") {
				return false
			}
			if loop.Condition != nil {
				if !expr(loop.Condition, d+1) {
					return false
				}
			} else if !record(113, diagnostic.Span{}, 0, diagnostic.Span{}, d+1, "") {
				return false
			}
			if loop.Post != nil {
				if loop.Post.Kind != syntax.KindAssignment || !stmt(*loop.Post, d+1) {
					return false
				}
			} else if !record(114, diagnostic.Span{}, 0, diagnostic.Span{}, d+1, "") {
				return false
			}
			return block(loop.Body, d+1)
		}
		k := map[syntax.Kind]int{syntax.KindBinding: 101, syntax.KindReturn: 103, syntax.KindAssignment: 104, syntax.KindExprStmt: 105, syntax.KindWhile: 106, syntax.KindBreak: 107, syntax.KindContinue: 108, syntax.KindAssert: 115, syntax.KindDefer: 116, syntax.KindDiscard: 117, syntax.KindBurn: 118, syntax.KindRegion: 119}[s.Kind]
		if k == 0 || s.For != nil {
			failure = fmt.Errorf("unhandled statement %s at %d", s.Kind, s.Span.StartByte)
			return false
		}
		var children []*ast.Expression
		var writtenType *ast.TypeReference
		name := diagnostic.Span{}
		if k == 101 {
			if s.Binding == nil {
				return false
			}
			if s.Binding.Mutable {
				k = 102
			}
			name = s.Binding.NameSpan
			writtenType = s.Binding.Type
			if writtenType != nil {
				k += 8
			}
			children = append(children, s.Binding.Expression)
		} else if k == 104 {
			children = append(children, s.Target, s.Expression)
		} else if k == 106 || k == 119 {
			if s.Expression == nil || len(s.Expression.Children) != 2 || s.Expression.Children[1].Kind != syntax.KindBlock {
				return false
			}
			children = s.Expression.Children
			if k == 119 {
				var ok bool
				name, ok = regionNames[[2]int{s.Span.StartByte, s.Span.EndByte}]
				if !ok || name.EndByte <= name.StartByte {
					return false
				}
			}
		} else if k != 107 && k != 108 && s.Expression != nil {
			children = append(children, s.Expression)
		}
		n := len(children)
		if k >= 115 && k <= 118 && n != 1 {
			return false
		}
		if writtenType != nil {
			n++
		}
		if !record(k, s.Span, n, name, d, s.Operator) {
			return false
		}
		if writtenType != nil && !typeTree(writtenType, d+1) {
			return false
		}
		for i, c := range children {
			if (k == 106 || k == 119) && i == 1 {
				if !block(c.Block, d+1) {
					return false
				}
			} else if !expr(c, d+1) {
				return false
			}
		}
		return true
	}
	bodyDepth := 0
	if header {
		bodyDepth = 1
		headerSpans := []diagnostic.Span{publicSpan, unsafeSpan, fn.OwnerSpan}
		children := len(fn.Parameters) + 2 + len(fn.Attributes)
		if len(genericNames) != 0 {
			children++
		}
		for _, span := range headerSpans {
			if span.EndByte > span.StartByte {
				children++
			}
		}
		if !record(90, fn.Span, children, fn.NameSpan, 0, "") {
			return "", failure
		}
		for i, attribute := range fn.Attributes {
			origin := attributeOrigins[i]
			if !record(400, attribute.Span, len(attribute.Arguments), origin.name, 1, "") {
				return "", failure
			}
			for j, argument := range attribute.Arguments {
				if !record(origin.kinds[j], argument.Span, 0, diagnostic.Span{}, 2, "") {
					return "", failure
				}
			}
		}
		for i, span := range headerSpans {
			if span.EndByte > span.StartByte && !record(92+i, span, 0, diagnostic.Span{}, 1, "") {
				return "", failure
			}
		}
		if len(genericNames) != 0 {
			if !record(95, genericSpan, len(genericNames), diagnostic.Span{}, 1, "") {
				return "", failure
			}
			for _, span := range genericNames {
				if !record(96, span, 0, diagnostic.Span{}, 2, "") {
					return "", failure
				}
			}
		}
		if !typeTree(&fn.ResultType, 1) {
			return "", fmt.Errorf("function result export: %v", failure)
		}
		for _, p := range fn.Parameters {
			if !record(91, p.Span, 1, p.NameSpan, 1, "") || !typeTree(&p.Type, 2) {
				return "", fmt.Errorf("function parameter export: %v", failure)
			}
		}
	}
	if !block(fn.Body, bodyDepth) {
		if failure == nil {
			failure = fmt.Errorf("unsupported body-tree node")
		}
		return "", failure
	}
	return out.String(), nil
}

func TestWindowsFairyParserBodyTree(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	t.Run("function-header", testWindowsFairyFunctionTree)
	inputs := []string{
		"module main; fn i32 f(i32 a){var x=1;{let y=2;call(x,y);}if(a>0){x+=a;}else if(false){}else{x=3;}while(x<a){if(false){continue;}x+=1;break;}return x;}",
		"module main; /* 한글 */\r\nfn void f(){let x={1};if(a){call((x),-2);}else{3}return;}",
		"module main;fn void f(){}",
		"module main;fn void f(){" + strings.Repeat("x;", 127) + "return;}",
		"module main;fn void f(){let i32 x=1;var pkg::Thing y=x;{let ref mut ptr u8[] z=y;}}",
		"module main; /* 타입 */\r\nfn void f(){let Box /* gap */ <Pair<u8,i32>,ref T> x=seed;var (ref u8)[0x10][cfg /* gap */ :: N][] y=x;let ptr void p=x;}",
		"module main;fn void f(){let unsafe extern C fn void() x=callback;var (unsafe /* gap */ extern C fn ptr mut u8(i32,ref u8[],))[] y=x;let Box<unsafe extern C fn i32(Box<u8>,unsafe extern C fn void(u32))> z=y;}",
		"module main;fn void f(){" + strings.Repeat("let i32 x=1;", 85) + "}",
		"module main;fn void f(){let i32" + strings.Repeat("[]", 61) + " x=1;}",
		"module main; /* 변환 */\r\nfn void f(){let x=(a+b) as i32 as u8;var ptr mut u8 p=value as ptr mut u8;return callback as unsafe extern C fn i32(ref u8[]);}",
		"module main;fn void f(){let x=pkg::choose /* gap */ <Box<Pair<u8,i32>>,ref mut T>(first<i32>(1),second as u8);if(a<b){call(x);}return empty<unsafe extern C fn void(),u8[0x10]>();}",
		"module main;fn void f(){" + strings.Repeat("f<i32>();", 51) + "}",
		"module main;fn void f(){x as i32" + strings.Repeat("[]", 60) + ";}",
		"module main;fn void f(){f<i32" + strings.Repeat("[]", 59) + ">();}",
		"module main; /* 접근 */\r\nfn void f(){obj /* gap */ .field[at(1)].value+=2;return factory<i32>(x).items[0].run(y);}",
		"module main;fn void f(){a[..];a[0..];a[..0];a[0..0];a[start..end];a[ /* L */ .. /* R */ ];a[1.. /* R */ ];return a[..n][i].field;}",
		"module main;fn void f(){" + strings.Repeat("a[..];", 51) + "}",
		"module main;fn void f(){a" + strings.Repeat("[..]", 61) + ";}",
		"module main;fn void f(){a" + strings.Repeat(".field", 61) + ";}",
		"module main;fn void f(){[];[1];[first(),2,third(),];[[1,2],[]];[value;0];[value;cfg::N];}",
		"module main; /* 생성 */\r\nfn void f(){Empty{};let x=Pair{.z=3,.a=[1,2],};return .Pair{.right=call(x),.left=0};}",
		"module main;fn void f(){return pkg::Box<Pair<u8,i32>,ref T>{. /* name */ z=Inner{.value=[1;2]},.a=2,.z=3};}",
		"module main;fn void f(){[" + strings.Repeat("1,", 253) + "];}",
		"module main;fn void f(){" + strings.Repeat("R{.x=1};", 51) + "}",
		"module main; /* 참조 */\r\nfn void f(){let a=ref object.field;let b=ref /* gap */ mut values[i];use(ref a,ref mut b);return ref ref mut a;}",
		"module main;fn void f(){let a=fetch<i32>()?.items[0]?;return (ref mut a)?;}",
		"module main;fn void f(){let a=unsafe /* gap */ {};let b=unsafe{let p=ref mut a;unsafe{call(p)?}};return {unsafe{ref b}};}",
		"module main;fn void f(){" + strings.Repeat("unsafe{ref mut x?;};", 36) + "x;return;}",
		"module main;fn void f(){" + strings.Repeat("ref ", 61) + "x;}",
		"module main;fn void f(){x" + strings.Repeat("?", 61) + ";}",
		"module main;fn void f(){return " + strings.Repeat("unsafe{", 30) + "x?" + strings.Repeat("}", 30) + ";}",
		"module main;fn void f(){return switch(value){case .None:{}case .Some(value):{value}case .Some(_):{0}};}",
		"module main; /* 패턴 */\r\nfn void f(){return switch(value){case .Pair{. /* field */ right = b,.left=_,.right=c,}:{call(b,c)}case .Empty{}:{}case .Empty:{}};}",
		"module main;fn void f(){let v=.Pair{.left=1,.right=2};return switch(v){case .Pair{.right=y,.left=x}:{switch(x){case .Some(z):{unsafe{z?}}}}};}",
		"module main;fn void f(){return switch(x){};}",
		"module main;fn void f(){return switch(x){" + strings.Repeat("case .N:{}", 63) + "};}",
		"module main;fn void f(){return " + strings.Repeat("switch(x){case .N:{", 20) + "x?" + strings.Repeat("}}", 20) + ";}",
		"module main; /* 반복 */\r\nfn void f(){for(let i=0 /* gap */ ;i<n;i+=1){while(flag){continue;}break;}for(i=1 /* end */ ;;i*=2){call(i);}}",
		"module main;fn void f(){for(var i32 i=0;i<2;i+=1){for(j=0;j<3;j-=1){if(flag){continue;}break;}}}",
		"module main;fn void f(){" + strings.Repeat("for(;;){}", 51) + "}",
		"module main;fn void f(){" + strings.Repeat("for(;;){", 30) + "ref x;" + strings.Repeat("}", 30) + "}",
		"module main; /* 문장 */\r\nfn void f(){assert /* gap */ (a<b);defer /* gap */ release_handle(ref mut x);discard fetch<i32>()?.value;burn object.field;}",
		"module main;fn void f(){for(var i=0;i<2;i+=1){assert(i>=0);defer release_handle(i);discard switch(x){case .Some(v):{v}case .None:{0}};unsafe{burn item;}continue;}}",
		"module main;fn void f(){" + strings.Repeat("assert(x);defer x;discard x;burn x;", 31) + "assert(x);defer x;discard x;return;}",
		"module main;fn void f(){assert(" + strings.Repeat("ref ", 61) + "x);}",
		"module main;fn void f(){defer " + strings.Repeat("ref ", 61) + "x;}",
		"module main;fn void f(){discard " + strings.Repeat("ref ", 61) + "x;}",
		"module main;fn void f(){burn " + strings.Repeat("ref ", 61) + "x;}",
		"module main;fn void f(){region r in source {}}",
		"module main; /* 영역 */\r\nfn void f(){region /* name */ outer in pools[i].view(){let x=1;region inner in select<Pool>(outer){use(inner,x);}region outer in source{use(outer);}}}",
		"module main;fn void f(){for(;;){region r in source{discard switch(x){case .Some(v):{use(r,v)}case .None:{0}};defer release_handle(r);break;}}}",
		"module main;fn void f(){" + strings.Repeat("region r in src{}", 85) + "}",
		"module main;fn void f(){" + strings.Repeat("region r in src{", 30) + "ref x;" + strings.Repeat("}", 30) + "}",
	}
	// All eight presence combinations retain the same four structural roles.
	for mask := 0; mask < 8; mask++ {
		initial, condition, post := "", "", ""
		if mask&1 != 0 {
			initial = "var i32 i=0"
		}
		if mask&2 != 0 {
			condition = "i<2"
		}
		if mask&4 != 0 {
			post = "i+=1"
		}
		inputs = append(inputs, "module main;fn void f(){for("+initial+";"+condition+";"+post+"){break;}}")
	}
	inputs = append(inputs, bodyTreeGrammarAuditInputs()...)
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	for _, name := range []string{"syntax.fy", "integrity.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if name == "syntax.fy" {
			// Test-only origin access: do not expose Parsed.source or change the shipped API.
			text += "\npub fn scanner::TokenInfo comparison_token(ref Parsed p,usize index){return scanner::token_info(ref p.source,index);}\n"
		}
		writeTestSource(t, root, "src/"+name, text)
	}
	program := parserCorpusProgram
	a := strings.Index(program, "@effects(console) fn i32 report(")
	b := strings.Index(program, "@effects(console) fn i32 main()")
	if a < 0 || b < a {
		t.Fatal("corpus driver boundary drift")
	}
	program = program[:a] + candidateBodyTreeWriter + program[b:]
	a = strings.Index(program, " return switch(parsed) {")
	if a < 0 {
		t.Fatal("corpus result boundary drift")
	}
	program = program[:a] + " return switch(parsed){case .ok(value):{report(value)}case .err(_):{4}};\n}\n"
	writeTestSource(t, root, "src/main.fy", program)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		for _, input := range inputs {
			want, err := referenceBodyTree(input)
			if err != nil {
				t.Fatalf("reference for %q: %v", input, err)
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewBufferString(input))
			if err != nil || out.ExitCode != 0 || out.Stderr != "" || out.TrapControl != "" || out.Stdout != want {
				t.Fatalf("body tree mismatch %v: err=%v output=%+v want=%s", profile, err, out, want)
			}
		}
		t.Logf("profile=%v complete canonical bodies=%d", profile, len(inputs))
		for _, input := range []string{
			"module main;fn void f(){let ref mut =1;}",
			"module main;fn void f(){}fn void g(){}",
			"module main;import lib as lib;fn void f(){}",
			strings.Repeat(" ", 32769) + "module main;fn void f(){}",
			"module main;fn void f(){return 1}",
			"module main;fn void f(){" + strings.Repeat("x;", 128) + "}",
			"module main;const i32 X=1;fn void f(){}",
			"module main;fn void f(){" + strings.Repeat("let i32 x=1;", 86) + "}",
			"module main;fn void f(){let i32" + strings.Repeat("[]", 62) + " x=1;}",
			"module main;fn void f(){x as ;}",
			"module main;fn void f(){f<i32>(;}",
			"module main;fn void f(){" + strings.Repeat("f<i32>();", 52) + "}",
			"module main;fn void f(){x as i32" + strings.Repeat("[]", 61) + ";}",
			"module main;fn void f(){f<i32" + strings.Repeat("[]", 60) + ">();}",
			"module main;fn void f(){a.;}",
			"module main;fn void f(){a[];}",
			"module main;fn void f(){a[1..2;}",
			"module main;fn void f(){" + strings.Repeat("a[..];", 52) + "}",
			"module main;fn void f(){a" + strings.Repeat("[..]", 62) + ";}",
			"module main;fn void f(){a" + strings.Repeat(".field", 62) + ";}",
			"module main;fn void f(){[1,,2];}",
			"module main;fn void f(){R{.x=};}",
			"module main;fn void f(){[" + strings.Repeat("1,", 254) + "];}",
			"module main;fn void f(){" + strings.Repeat("R{.x=1};", 52) + "}",
			"module main;fn void f(){ref mut ;}",
			"module main;fn void f(){?x;}",
			"module main;fn void f(){unsafe x;}",
			"module main;fn void f(){" + strings.Repeat("unsafe{ref mut x?;};", 36) + "x;x;}",
			"module main;fn void f(){" + strings.Repeat("ref ", 62) + "x;}",
			"module main;fn void f(){x" + strings.Repeat("?", 62) + ";}",
			"module main;fn void f(){return " + strings.Repeat("unsafe{", 30) + "ref x?" + strings.Repeat("}", 30) + ";}",
			"module main;fn void f(){return switch(x){case .Some():{}};}",
			"module main;fn void f(){return switch(x){case .Pair{.left=}:{}};}",
			"module main;fn void f(){return switch(x){case .None {}};}",
			"module main;fn void f(){return switch(x){" + strings.Repeat("case .N:{}", 62) + "case .N:{0}};}",
			"module main;fn void f(){return " + strings.Repeat("switch(x){case .N:{", 20) + "ref x?" + strings.Repeat("}}", 20) + ";}",
			"module main;fn void f(){for(var i=0 i<2;i+=1){}}",
			"module main;fn void f(){for(;;var i=0){}}",
			"module main;fn void f(){for(;i<;){}}",
			"module main;fn void f(){for(;;)}",
			"module main;fn void f(){" + strings.Repeat("for(;;){}", 51) + "return;}",
			"module main;fn void f(){" + strings.Repeat("for(;;){", 30) + "ref ref x;" + strings.Repeat("}", 30) + "}",
			"module main;fn void f(){assert();}",
			"module main;fn void f(){defer ;}",
			"module main;fn void f(){discard ;}",
			"module main;fn void f(){burn ;}",
			"module main;fn void f(){" + strings.Repeat("assert(x);defer x;discard x;burn x;", 31) + "assert(x);defer x;discard x;burn x;}",
			"module main;fn void f(){assert(" + strings.Repeat("ref ", 62) + "x);}",
			"module main;fn void f(){defer " + strings.Repeat("ref ", 62) + "x;}",
			"module main;fn void f(){discard " + strings.Repeat("ref ", 62) + "x;}",
			"module main;fn void f(){burn " + strings.Repeat("ref ", 62) + "x;}",
			"module main;fn void f(){region in source{}}",
			"module main;fn void f(){region r source{}}",
			"module main;fn void f(){region r in {}}",
			"module main;fn void f(){region r in source;}",
			"module main;fn void f(){" + strings.Repeat("region r in src{}", 85) + "return;}",
			"module main;fn void f(){" + strings.Repeat("region r in src{", 30) + "ref ref x;" + strings.Repeat("}", 30) + "}",
		} {
			if text, err := referenceBodyTree(input); err == nil || text != "" {
				t.Fatalf("reference accepted unsupported/invalid/budget input %q", input)
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewBufferString(input))
			if err != nil || out.ExitCode == 0 || out.Stdout != "" || out.Stderr != "" || out.TrapControl != "" {
				t.Fatalf("partial or successful rejected export: %+v %v", out, err)
			}
		}
	}
}
