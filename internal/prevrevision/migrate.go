// Package prevrevision migrates the immediately preceding Fairy C source revision without
// exposing that grammar through the product compiler entry point.
package prevrevision

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Result is one in-memory migration. Callers own persistence and atomic replacement.
type Result struct {
	Source  []byte
	Changed bool
}

type edit struct {
	start       int
	end         int
	replacement string
}

// Migrate converts every implemented previous-revision spelling and verifies that the result is
// accepted by the current product parser.
func Migrate(path string, input []byte) (Result, error) {
	declarations, err := MigrateDeclarations(path, input)
	if err != nil {
		return Result{}, err
	}
	tree := parseMigrationSource(path, declarations.Source)
	if len(tree.Diagnostics) != 0 {
		return Result{}, fmt.Errorf("surface migration input produced %d diagnostics", len(tree.Diagnostics))
	}
	edits, err := surfaceEdits(tree)
	if err != nil {
		return Result{}, err
	}
	output := declarations.Source
	changed := declarations.Changed
	if len(edits) != 0 {
		output, err = applyEdits(declarations.Source, edits)
		if err != nil {
			return Result{}, err
		}
		changed = true
	}
	verified := parseCurrentSource(path, output)
	if len(verified.Diagnostics) != 0 {
		return Result{}, fmt.Errorf("migrated surface parse produced %d diagnostics: %+v\n%s", len(verified.Diagnostics), verified.Diagnostics, output)
	}
	return Result{Source: output, Changed: changed}, nil
}

// MigrateDeclarations converts declaration order, structural type spelling, resource cleanup,
// and function tail returns. It refuses syntax-invalid input and overlapping edits.
func MigrateDeclarations(path string, input []byte) (Result, error) {
	tree := parseMigrationSource(path, input)
	if len(tree.Diagnostics) != 0 {
		return Result{}, fmt.Errorf("previous revision parse produced %d diagnostics", len(tree.Diagnostics))
	}
	edits, err := declarationEdits(tree)
	if err != nil {
		return Result{}, err
	}
	if len(edits) == 0 {
		return Result{Source: slices.Clone(input)}, nil
	}
	output, err := applyEdits(input, edits)
	if err != nil {
		return Result{}, err
	}
	verified := parseMigrationSource(path, output)
	if len(verified.Diagnostics) != 0 {
		return Result{}, fmt.Errorf("migrated declaration parse produced %d diagnostics", len(verified.Diagnostics))
	}
	return Result{Source: output, Changed: true}, nil
}

func parseMigrationSource(path string, input []byte) *syntax.Tree {
	lexed := lexer.Lex(source.NewFile(path, input), source.NewBudget(source.DefaultLimits(), len(input)))
	module := ""
	for index := 0; index < len(lexed.Tokens); index++ {
		if lexed.Tokens[index].Text(lexed.Source) != "module" {
			continue
		}
		for index++; index < len(lexed.Tokens); index++ {
			token := lexed.Tokens[index]
			text := token.Text(lexed.Source)
			if text == ";" {
				break
			}
			if token.Kind != lexer.Trivia {
				module += text
			}
		}
		break
	}
	virtualPath := migrationVirtualPathForModule(path, module)
	return syntax.ParsePreviousRevision(source.NewFile(virtualPath, input), source.DefaultLimits())
}

func parseCurrentSource(path string, input []byte) *syntax.Tree {
	return syntax.Parse(source.NewFile(migrationVirtualPath(path, input), input), source.DefaultLimits())
}

func migrationVirtualPath(path string, input []byte) string {
	lexed := lexer.Lex(source.NewFile(path, input), source.NewBudget(source.DefaultLimits(), len(input)))
	module := ""
	for index := 0; index < len(lexed.Tokens); index++ {
		if lexed.Tokens[index].Text(lexed.Source) != "module" {
			continue
		}
		for index++; index < len(lexed.Tokens); index++ {
			token := lexed.Tokens[index]
			text := token.Text(lexed.Source)
			if text == ";" {
				break
			}
			if token.Kind != lexer.Trivia {
				module += text
			}
		}
		break
	}
	return migrationVirtualPathForModule(path, module)
}

func migrationVirtualPathForModule(path, module string) string {
	if module == "" {
		return path
	}
	return strings.ReplaceAll(module, "::", "/") + ".fy"
}

func declarationEdits(tree *syntax.Tree) ([]edit, error) {
	var edits []edit
	for index := range tree.Nodes {
		node := tree.Nodes[index]
		switch node.Kind {
		case syntax.KindFunction:
			if node.Text == "drop" {
				block, found := childOfKind(tree, node, syntax.KindBlock)
				if !found {
					return nil, errors.New("drop declaration has no body")
				}
				edits = append(edits, edit{start: node.Span.StartByte, end: block.Span.StartByte, replacement: "cleanup "})
				continue
			}
			legacy, fnToken := legacyFunction(tree, node)
			if !legacy {
				continue
			}
			block, found := childOfKind(tree, node, syntax.KindBlock)
			if !found {
				return nil, fmt.Errorf("function %q has no body", node.Text)
			}
			if commentsBetween(tree, fnToken+1, block.TokenStart) {
				return nil, fmt.Errorf("function %q header contains comments and requires manual migration", node.Text)
			}
			header, err := renderFunctionHeader(tree, node)
			if err != nil {
				return nil, err
			}
			edits = append(edits, edit{start: tree.Tokens[fnToken].Span.EndByte, end: block.Span.StartByte, replacement: header})
			for _, childID := range node.Children {
				parameter, ok := tree.Node(childID)
				if !ok || parameter.Kind != syntax.KindParameter || parameter.Cooked != "mut" {
					continue
				}
				parameterType, ok := childOfKind(tree, parameter, syntax.KindType)
				if !ok {
					return nil, fmt.Errorf("mutable parameter %q has no type", parameter.Text)
				}
				edits = append(edits, edit{
					start: block.Span.StartByte + 1, end: block.Span.StartByte + 1,
					replacement: "\n    var " + renderType(tree, parameterType) + " " + parameter.Text + " = input;",
				})
			}
			if tail, found := childOfKind(tree, block, syntax.KindTail); found && len(tail.Children) == 1 {
				expression, ok := tree.Node(tail.Children[0])
				if ok {
					edits = append(edits,
						edit{start: expression.Span.StartByte, end: expression.Span.StartByte, replacement: "return "},
						edit{start: expression.Span.EndByte, end: expression.Span.EndByte, replacement: ";"},
					)
				}
			}
		case syntax.KindForeign:
			legacy, fnToken := legacyFunction(tree, node)
			if !legacy {
				continue
			}
			if commentsBetween(tree, fnToken+1, node.TokenEnd) {
				return nil, fmt.Errorf("foreign function %q header contains comments and requires manual migration", node.Text)
			}
			header, err := renderFunctionHeader(tree, node)
			if err != nil {
				return nil, err
			}
			semicolon := tokenIndex(tree, node, ";")
			if semicolon < 0 {
				return nil, fmt.Errorf("foreign function %q has no terminator", node.Text)
			}
			edits = append(edits, edit{start: tree.Tokens[fnToken].Span.EndByte, end: tree.Tokens[semicolon].Span.StartByte, replacement: header})
		case syntax.KindField, syntax.KindConst, syntax.KindBinding:
			typeNode, found := childOfKind(tree, node, syntax.KindType)
			if !found || node.NameSpan.StartByte >= typeNode.Span.StartByte {
				continue
			}
			start := node.NameSpan.StartByte
			prefix := ""
			if node.Kind == syntax.KindConst || node.Kind == syntax.KindBinding {
				keyword := tokenIndex(tree, node, "const", "let", "var")
				if keyword < 0 {
					continue
				}
				start = tree.Tokens[keyword].Span.EndByte
				prefix = " "
			}
			edits = append(edits, edit{
				start: start, end: typeNode.Span.EndByte,
				replacement: prefix + renderType(tree, typeNode) + " " + node.Text,
			})
		}
	}
	return edits, nil
}

func surfaceEdits(tree *syntax.Tree) ([]edit, error) {
	var edits []edit
	localEnums := make(map[string]struct{})
	for _, node := range tree.Nodes {
		if node.Kind == syntax.KindEnum {
			localEnums[node.Text] = struct{}{}
		}
	}
	for index := range tree.Nodes {
		node := tree.Nodes[index]
		switch node.Kind {
		case syntax.KindPrefix:
			if node.Text != "&" && node.Text != "&mut" {
				continue
			}
			operator := tokenIndex(tree, node, "&")
			if operator < 0 {
				continue
			}
			end := tree.Tokens[operator].Span.EndByte
			replacement := "ref "
			if node.Text == "&mut" {
				mutable := nextSignificant(tree, operator+1, node.TokenEnd)
				if mutable >= 0 && tree.Tokens[mutable].Text(tree.Source) == "mut" {
					end = tree.Tokens[mutable].Span.EndByte
				}
				replacement = "ref mut"
			}
			edits = append(edits, edit{start: tree.Tokens[operator].Span.StartByte, end: end, replacement: replacement})
		case syntax.KindCast:
			typeNode, found := childOfKind(tree, node, syntax.KindType)
			if found && strings.TrimSpace(typeNode.Text) != renderType(tree, typeNode) {
				edits = append(edits, edit{start: typeNode.Span.StartByte, end: typeNode.Span.EndByte, replacement: renderType(tree, typeNode)})
			}
		case syntax.KindIf, syntax.KindWhile:
			condition, found := firstExpressionChild(tree, node)
			if !found || controlHeaderParenthesized(tree, node) {
				continue
			}
			edits = append(edits,
				edit{start: condition.Span.StartByte, end: condition.Span.StartByte, replacement: "("},
				edit{start: condition.Span.EndByte, end: condition.Span.EndByte, replacement: ")"},
			)
		case syntax.KindRecord:
			for _, childID := range node.Children[1:] {
				field, ok := tree.Node(childID)
				if !ok || field.Kind != syntax.KindField || len(field.Children) != 1 {
					continue
				}
				first := nextSignificant(tree, field.TokenStart, field.TokenEnd)
				if first >= 0 && tree.Tokens[first].Text(tree.Source) != "." {
					start := tree.Tokens[first].Span.StartByte
					edits = append(edits, edit{start: start, end: start, replacement: "."})
				}
				separator := tokenIndex(tree, field, ":")
				if separator >= 0 {
					value, valueOK := tree.Node(field.Children[0])
					end := tree.Tokens[separator].Span.EndByte
					if valueOK {
						end = value.Span.StartByte
					}
					edits = append(edits, edit{
						start: tree.Tokens[separator].Span.StartByte, end: end, replacement: " = ",
					})
				}
			}
		case syntax.KindPattern:
			if node.Text == "arm" {
				continue
			}
			for _, childID := range node.Children[1:] {
				field, ok := tree.Node(childID)
				if !ok || field.Kind != syntax.KindField || field.Cooked == "" {
					continue
				}
				first := nextSignificant(tree, field.TokenStart, field.TokenEnd)
				if first >= 0 && tree.Tokens[first].Text(tree.Source) != "." {
					start := tree.Tokens[first].Span.StartByte
					edits = append(edits, edit{start: start, end: start, replacement: "."})
				}
				separator := tokenIndex(tree, field, ":")
				if separator < 0 {
					continue
				}
				binding := nextSignificant(tree, separator+1, field.TokenEnd)
				end := tree.Tokens[separator].Span.EndByte
				if binding >= 0 {
					end = tree.Tokens[binding].Span.StartByte
				}
				edits = append(edits, edit{start: tree.Tokens[separator].Span.StartByte, end: end, replacement: " = "})
			}
		case syntax.KindMatch:
			matchToken := tokenIndex(tree, node, "match")
			if matchToken < 0 || len(node.Children) < 2 {
				return nil, errors.New("malformed previous match")
			}
			scrutinee, ok := tree.Node(node.Children[0])
			if !ok {
				return nil, errors.New("match has no scrutinee")
			}
			edits = append(edits,
				edit{start: tree.Tokens[matchToken].Span.StartByte, end: tree.Tokens[matchToken].Span.EndByte, replacement: "switch"},
				edit{start: scrutinee.Span.StartByte, end: scrutinee.Span.StartByte, replacement: "("},
				edit{start: scrutinee.Span.EndByte, end: scrutinee.Span.EndByte, replacement: ")"},
			)
			for _, armID := range node.Children[1:] {
				arm, ok := tree.Node(armID)
				if !ok || arm.Kind != syntax.KindPattern || len(arm.Children) != 2 {
					return nil, errors.New("malformed previous match arm")
				}
				pattern, patternOK := tree.Node(arm.Children[0])
				body, bodyOK := tree.Node(arm.Children[1])
				if !patternOK || !bodyOK {
					return nil, errors.New("match arm has missing pattern or body")
				}
				arrow := tokenIndex(tree, arm, "=>")
				if arrow < 0 {
					return nil, errors.New("match arm has no arrow")
				}
				edits = append(edits,
					edit{start: pattern.Span.StartByte, end: pattern.Span.StartByte, replacement: "case "},
					edit{start: tree.Tokens[arrow].Span.StartByte, end: tree.Tokens[arrow].Span.EndByte, replacement: ":"},
				)
				if body.Kind != syntax.KindBlock {
					edits = append(edits,
						edit{start: body.Span.StartByte, end: body.Span.StartByte, replacement: "{ "},
						edit{start: body.Span.EndByte, end: body.Span.EndByte, replacement: " }"},
					)
				}
				comma := lastSignificantToken(tree, arm)
				if comma >= 0 && tree.Tokens[comma].Text(tree.Source) != "," {
					comma = -1
				}
				if comma >= 0 {
					edits = append(edits, edit{start: tree.Tokens[comma].Span.StartByte, end: tree.Tokens[comma].Span.EndByte})
				}
			}
		case syntax.KindPath:
			if node.Text == "self" {
				edits = append(edits, edit{start: node.Span.StartByte, end: node.Span.EndByte, replacement: "this"})
				continue
			}
			replacement, ok := migratedVariantPath(node.Text, localEnums)
			if ok {
				edits = append(edits, edit{start: node.Span.StartByte, end: node.Span.EndByte, replacement: replacement})
			}
		}
	}
	return edits, nil
}

func firstExpressionChild(tree *syntax.Tree, node syntax.Node) (syntax.Node, bool) {
	if len(node.Children) == 0 {
		return syntax.Node{}, false
	}
	return tree.Node(node.Children[0])
}

func controlHeaderParenthesized(tree *syntax.Tree, node syntax.Node) bool {
	keyword := tokenIndex(tree, node, string(node.Kind))
	if keyword < 0 {
		return false
	}
	open := nextSignificant(tree, keyword+1, node.TokenEnd)
	if open < 0 || tree.Tokens[open].Text(tree.Source) != "(" {
		return false
	}
	depth := 0
	for index := open; index >= 0 && index < node.TokenEnd; index = nextSignificant(tree, index+1, node.TokenEnd) {
		switch tree.Tokens[index].Text(tree.Source) {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				next := nextSignificant(tree, index+1, node.TokenEnd)
				return next >= 0 && tree.Tokens[next].Text(tree.Source) == "{"
			}
		}
	}
	return false
}

func migratedVariantPath(value string, localEnums map[string]struct{}) (string, bool) {
	switch value {
	case "Result::Ok":
		return ".ok", true
	case "Result::Err":
		return ".err", true
	case "Option::Some":
		return ".some", true
	case "Option::None":
		return ".none", true
	}
	parts := strings.Split(value, "::")
	if len(parts) == 2 {
		if _, ok := localEnums[parts[0]]; ok {
			return "." + parts[1], true
		}
	}
	return "", false
}

func legacyFunction(tree *syntax.Tree, node syntax.Node) (bool, int) {
	fn := tokenIndex(tree, node, "fn")
	if fn < 0 {
		return false, -1
	}
	next := nextSignificant(tree, fn+1, node.TokenEnd)
	after := nextSignificant(tree, next+1, node.TokenEnd)
	return next >= 0 && after >= 0 && tree.Tokens[next].Kind == lexer.Identifier &&
		tree.Tokens[next].Text(tree.Source) == node.Text && tree.Tokens[after].Text(tree.Source) == "(", fn
}

func renderFunctionHeader(tree *syntax.Tree, node syntax.Node) (string, error) {
	resultType, found := childOfKind(tree, node, syntax.KindType)
	if !found {
		return "", fmt.Errorf("function %q has no result type", node.Text)
	}
	parameters := make([]string, 0, 4)
	for _, childID := range node.Children {
		child, ok := tree.Node(childID)
		if !ok || child.Kind != syntax.KindParameter {
			continue
		}
		parameterType, ok := childOfKind(tree, child, syntax.KindType)
		if !ok {
			return "", fmt.Errorf("parameter %q has no type", child.Text)
		}
		name := child.Text
		if child.Cooked == "mut" {
			name = "input"
		}
		parameters = append(parameters, renderType(tree, parameterType)+" "+name)
	}
	return " " + renderType(tree, resultType) + " " + node.Text + "(" + strings.Join(parameters, ", ") + ") ", nil
}

func renderType(tree *syntax.Tree, node syntax.Node) string {
	children := make([]syntax.Node, 0, len(node.Children))
	for _, id := range node.Children {
		child, ok := tree.Node(id)
		if ok {
			children = append(children, child)
		}
	}
	childType := func(index int) string {
		if index < 0 || index >= len(children) || children[index].Kind != syntax.KindType {
			return ""
		}
		return renderType(tree, children[index])
	}
	switch syntax.TypeForm(node.Cooked) {
	case syntax.TypeFormReference:
		return "ref " + childType(0)
	case syntax.TypeFormReferenceMut:
		return "ref mut " + childType(0)
	case syntax.TypeFormPointer:
		return "ptr " + childType(0)
	case syntax.TypeFormPointerMut:
		return "ptr mut " + childType(0)
	case syntax.TypeFormSlice:
		return childType(0) + "[]"
	case syntax.TypeFormArray:
		length := ""
		if len(children) > 1 {
			length = strings.TrimSpace(children[1].Text)
		}
		return childType(0) + "[" + length + "]"
	case syntax.TypeFormGrouped:
		return "(" + childType(0) + ")"
	case syntax.TypeFormForeignFunction:
		if len(children) == 0 {
			return strings.TrimSpace(node.Text)
		}
		parameters := make([]string, 0, len(children)-1)
		for index := 0; index < len(children)-1; index++ {
			parameters = append(parameters, renderType(tree, children[index]))
		}
		return "unsafe extern C fn " + renderType(tree, children[len(children)-1]) + "(" + strings.Join(parameters, ", ") + ")"
	case syntax.TypeFormNamed:
		if len(children) == 0 {
			return strings.TrimSpace(node.Text)
		}
		name := children[0].Text
		if len(children) == 1 {
			return name
		}
		arguments := make([]string, 0, len(children)-1)
		for index := 1; index < len(children); index++ {
			arguments = append(arguments, renderType(tree, children[index]))
		}
		return name + "<" + strings.Join(arguments, ", ") + ">"
	default:
		return strings.TrimSpace(node.Text)
	}
}

func childOfKind(tree *syntax.Tree, node syntax.Node, kind syntax.Kind) (syntax.Node, bool) {
	for _, id := range node.Children {
		child, ok := tree.Node(id)
		if ok && child.Kind == kind {
			return child, true
		}
	}
	return syntax.Node{}, false
}

func tokenIndex(tree *syntax.Tree, node syntax.Node, values ...string) int {
	wanted := make(map[string]struct{}, len(values))
	for _, value := range values {
		wanted[value] = struct{}{}
	}
	for index := max(node.TokenStart, 0); index < min(node.TokenEnd, len(tree.Tokens)); index++ {
		if _, ok := wanted[tree.Tokens[index].Text(tree.Source)]; ok {
			return index
		}
	}
	return -1
}

func nextSignificant(tree *syntax.Tree, start, end int) int {
	for index := max(start, 0); index < min(end, len(tree.Tokens)); index++ {
		if tree.Tokens[index].Kind != lexer.Trivia {
			return index
		}
	}
	return -1
}

func lastSignificantToken(tree *syntax.Tree, node syntax.Node) int {
	for index := min(node.TokenEnd, len(tree.Tokens)) - 1; index >= max(node.TokenStart, 0); index-- {
		if tree.Tokens[index].Kind != lexer.Trivia {
			return index
		}
	}
	return -1
}

func commentsBetween(tree *syntax.Tree, start, end int) bool {
	for index := max(start, 0); index < min(end, len(tree.Tokens)); index++ {
		if tree.Tokens[index].Kind != lexer.Trivia {
			continue
		}
		text := tree.Tokens[index].Text(tree.Source)
		if strings.Contains(text, "//") || strings.Contains(text, "/*") {
			return true
		}
	}
	return false
}

func applyEdits(input []byte, edits []edit) ([]byte, error) {
	slices.SortFunc(edits, func(left, right edit) int {
		if left.start != right.start {
			return right.start - left.start
		}
		return right.end - left.end
	})
	output := slices.Clone(input)
	previousStart := len(input) + 1
	for _, current := range edits {
		if current.start < 0 || current.end < current.start || current.end > len(input) {
			return nil, fmt.Errorf("invalid migration edit %d..%d", current.start, current.end)
		}
		if current.end > previousStart {
			return nil, fmt.Errorf("overlapping migration edit %d..%d", current.start, current.end)
		}
		output = append(output[:current.start], append([]byte(current.replacement), output[current.end:]...)...)
		previousStart = current.start
	}
	return output, nil
}
