// Package editors projects compiler vocabulary into editor-specific presentation assets.
package editors

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/lexer"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type rule = map[string]any

// Assets is deterministic; it never changes the compiler grammar or semantic admission.
func Assets(root string) (map[string][]byte, error) {
	grammar, err := os.ReadFile(filepath.Join(root, "spec/grammar/stage0.ebnf"))
	if err != nil {
		return nil, err
	}
	keywords, operators := lexer.EditorVocabulary()
	contextual := []string{"burn", "cell", "map", "state", "static_assert"}
	types := []string{"void", "bool", "u8", "i32", "i64", "u32", "usize", "str", "CStr"}
	metadata := rule{"schema": "fairyc.editor-language/v1", "languageId": "fairyc", "displayName": "Fairy C",
		"extensions": []string{".fy"}, "scopeName": "source.fairyc", "reservedKeywords": keywords,
		"contextualKeywords": contextual, "builtinTypes": types, "operators": operators,
		"grammarSHA256": fmt.Sprintf("%x", sha256.Sum256(grammar)),
		"lineComment":   "//", "blockComment": []string{"/*", "*/"}, "nestedBlockComments": true,
		"identifierPattern": "[A-Za-z_][A-Za-z0-9_]*"}
	words := func(values []string) string { return "\\b(?:" + strings.Join(values, "|") + ")\\b" }
	identifier := `[A-Za-z_][A-Za-z0-9_]*`
	typePath := `(?!(?:` + strings.Join(keywords, "|") + `)\b)` + identifier + `(?:\s*::\s*` + identifier + `)*`
	// Only syntax positions identify nominal types; capitalization is not a type test.
	typeExpression := `(?:(?:ref|ptr|mut)\s+)*` + typePath + `(?:\s*<[^;{}\n]*>)?(?:\s*\[[^\]\n]*\])*`
	typeCapture := rule{"name": "meta.type.fairyc", "patterns": []rule{{"include": "#type"}}}
	escape := rule{"name": "constant.character.escape.fairyc", "match": `\\(?:[\\"nrt0]|x[0-9a-fA-F]{2}|u\{[0-9a-fA-F]{1,6}\})`}
	stringRule := rule{"name": "string.quoted.double.fairyc", "begin": `\b[bc]"|"`, "end": `"`,
		"patterns": []rule{escape, {"name": "invalid.illegal.escape.fairyc", "match": `\\.`}}}
	block := rule{"name": "comment.block.fairyc", "begin": `/\*`, "end": `\*/`, "patterns": []rule{{"include": "#block-comment"}}}
	slices.SortStableFunc(operators, func(a, b string) int { return len(b) - len(a) })
	quoted := make([]string, len(operators))
	for i, op := range operators {
		quoted[i] = regexp.QuoteMeta(op)
	}
	patterns := []rule{
		{"include": "#block-comment"},
		{"name": "comment.line.documentation.fairyc", "begin": "///", "end": "$"},
		{"name": "comment.line.double-slash.fairyc", "begin": "//", "end": "$"},
		{"include": "#string"},
		{"match": `(@)(` + identifier + `(?:::` + identifier + `)*)`,
			"captures": rule{"1": rule{"name": "punctuation.definition.annotation.fairyc"}, "2": rule{"name": "entity.other.attribute-name.fairyc"}}},
		{"match": `\b(var|let|const|fn|pub)\s+(` + typeExpression + `)(?=\s+` + identifier + `\b)`,
			"captures": rule{"1": rule{"name": "keyword.control.fairyc"}, "2": typeCapture}},
		{"match": `(?:^|(?<=[(,;{]))[ \t]*(` + typeExpression + `)(?=\s+` + identifier + `\b)`,
			"captures": rule{"1": typeCapture}},
		{"name": "constant.language.fairyc", "match": words([]string{"true", "false", "null"})},
		{"name": "keyword.control.fairyc", "match": words(keywords)},
		{"name": "keyword.other.contextual.fairyc", "match": `\bstatic_assert(?=\s*\()|\bstate(?=\s*\{)|\b(?:cell|map|burn)(?=\s)`},
		{"name": "storage.type.fairyc", "match": words(types)},
		{"name": "entity.name.function.fairyc", "match": `\b[A-Za-z_][A-Za-z0-9_]*(?=\s*(?:<[^;{}\n]*>)?\()`},
		{"name": "constant.numeric.fairyc", "match": `\b(?:0x[0-9a-fA-F](?:_?[0-9a-fA-F])*|0b[01](?:_?[01])*|[0-9](?:_?[0-9])*(?:\.[0-9](?:_?[0-9])*)?(?:[eE][+-]?[0-9](?:_?[0-9])*)?)\b`},
		{"name": "keyword.operator.fairyc", "match": strings.Join(quoted, "|")},
	}
	tm := rule{"name": "Fairy C", "scopeName": "source.fairyc", "patterns": patterns,
		"repository": rule{"block-comment": block, "string": stringRule,
			"type": rule{"patterns": []rule{
				{"include": "#block-comment"},
				{"include": "#string"},
				{"name": "keyword.control.fairyc", "match": words([]string{"ref", "ptr", "mut"})},
				{"name": "storage.type.fairyc", "match": words(types)},
				{"name": "entity.name.namespace.fairyc", "match": `\b` + identifier + `(?=\s*::)`},
				{"name": "entity.name.type.fairyc", "match": `\b` + identifier + `\b`},
				{"name": "keyword.operator.fairyc", "match": `::|<|>|\[|\]|,`},
				{"name": "constant.numeric.fairyc", "match": `\b[0-9](?:_?[0-9])*\b`},
			}}}}
	assets := map[string][]byte{}
	for name, value := range map[string]any{"editors/language.json": metadata, "editors/vscode/syntaxes/fairyc.tmLanguage.json": tm} {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, err
		}
		assets[name] = append(data, '\n')
	}
	return assets, nil
}
