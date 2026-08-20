package contractabi

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"unicode"
)

var tsReserved = map[string]struct{}{
	"await": {}, "break": {}, "constructor": {}, "case": {}, "catch": {}, "class": {}, "const": {}, "continue": {},
	"debugger": {}, "default": {}, "delete": {}, "do": {}, "else": {}, "enum": {}, "export": {},
	"extends": {}, "false": {}, "finally": {}, "for": {}, "function": {}, "if": {}, "implements": {},
	"import": {}, "in": {}, "instanceof": {}, "interface": {}, "let": {}, "new": {}, "null": {},
	"package": {}, "private": {}, "protected": {}, "public": {}, "return": {}, "static": {}, "super": {},
	"switch": {}, "this": {}, "throw": {}, "true": {}, "try": {}, "typeof": {}, "var": {}, "void": {},
	"while": {}, "with": {}, "yield": {},
}

var goReserved = map[string]struct{}{
	"break": {}, "default": {}, "func": {}, "interface": {}, "select": {}, "case": {}, "defer": {},
	"go": {}, "map": {}, "struct": {}, "chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {},
	"const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {}, "continue": {}, "for": {},
	"import": {}, "return": {}, "var": {},
}

func tsLocalName(value string) string {
	if _, reserved := tsReserved[value]; reserved {
		return value + "_"
	}
	return value
}

func goLocalName(value string) string {
	if value == "ctx" {
		return "ctx_"
	}
	if _, reserved := goReserved[value]; reserved {
		return value + "_"
	}
	return value
}

func exportedName(value string) string {
	parts := splitIdentifierWords(value)
	if len(parts) == 0 {
		return "Unnamed"
	}
	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		builder.WriteString(string(runes))
	}
	if builder.Len() == 0 {
		return "Unnamed"
	}
	return builder.String()
}

func splitIdentifierWords(value string) []string {
	value = strings.ReplaceAll(value, "::", "_")
	return strings.FieldsFunc(value, func(r rune) bool { return r == '_' || r == '-' })
}

func externalTypeName(path string) string {
	segments := strings.Split(path, "::")
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		parts = append(parts, exportedName(segment))
	}
	base := strings.Join(parts, "_")
	if len(segments) == 1 {
		return base
	}
	digest := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%s_%x", base, digest[:4])
}

func allocateTypeScriptParameterNames(parameters []Parameter, reserveOptions bool) []string {
	used := make(map[string]struct{}, len(parameters)+1)
	if reserveOptions {
		used["options"] = struct{}{}
	}
	result := make([]string, len(parameters))
	for index, parameter := range parameters {
		name := tsLocalName(parameter.Name)
		for {
			if _, exists := used[name]; !exists {
				break
			}
			name += "_"
		}
		used[name] = struct{}{}
		result[index] = name
	}
	return result
}

func allocateGoParameterNames(parameters []Parameter, reserveOptions bool) []string {
	used := make(map[string]struct{}, len(parameters)+2)
	used["ctx"] = struct{}{}
	if reserveOptions {
		used["options"] = struct{}{}
	}
	result := make([]string, len(parameters))
	for index, parameter := range parameters {
		name := goLocalName(parameter.Name)
		for {
			if _, exists := used[name]; !exists {
				break
			}
			name += "_"
		}
		used[name] = struct{}{}
		result[index] = name
	}
	return result
}
