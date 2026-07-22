package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const atlasPackRelationSource = "module main;\nimport answer as answer;\nfn i32 leaf() { return 40; }\nfn i32 middle() { return leaf() + 1; }\nfn i32 main() { return middle() + answer::value() - 40; }\n"

func TestExperimentalAtlasPackShowAndAroundEmitDeterministicCurrentFacts(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasPackRelationSource)
	middle := atlasFunctionSymbol(t, root, "middle")

	showArguments := atlasPackRelationArguments("show", "example.session.app", middle.Visible)
	code, first, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, showArguments...), root)
	if code != ExitSuccess || stderr != "" || !json.Valid([]byte(first)) || strings.HasSuffix(first, "\n") {
		t.Fatalf("show pack = (%d, %q, %q)", code, first, stderr)
	}
	show := decodeAtlasPackRelation(t, first)
	if show.PackVersion != "fairyc.context-pack/v1" || show.Pack.Query.Kind != "show" ||
		show.Pack.Query.Subject != middle.Visible || show.Pack.Query.Direction != "outgoing" ||
		len(show.Pack.Query.Relations) != 1 || show.Pack.Query.Relations[0] != "calls" ||
		show.Pack.Facts.Show == nil || show.Pack.Facts.Show.Subject.Symbol.Visible != middle.Visible ||
		show.Pack.Facts.Around != nil || len(show.Pack.Relations) != 1 ||
		show.Pack.Relations[0].Kind != "calls" || show.Pack.Relations[0].Target == nil ||
		show.Pack.Budget.MaxDepth != 0 || show.Pack.Usage.Records == 0 ||
		len(show.Pack.Documentation) != 0 || len(show.Pack.SourceExcerpts) != 0 {
		t.Fatalf("show pack = %+v", show)
	}
	if strings.Contains(first, root) || strings.Contains(first, atlasPackRelationSource) {
		t.Fatal("show pack leaked a host path or source body")
	}

	locked := append(atlasPackRelationArguments("show", "example.session.app", middle.Visible), "--locked")
	code, lockedOutput, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, locked...), root)
	if code != ExitSuccess || stderr != "" || lockedOutput != first {
		t.Fatalf("locked show pack = (%d, equal %t, %q)", code, lockedOutput == first, stderr)
	}

	aroundArguments := atlasPackRelationArguments("around", "example.session.app", middle.Full)
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, aroundArguments...), root)
	if code != ExitSuccess || stderr != "" || !json.Valid([]byte(output)) || strings.HasSuffix(output, "\n") {
		t.Fatalf("around pack = (%d, %q, %q)", code, output, stderr)
	}
	around := decodeAtlasPackRelation(t, output)
	if around.Pack.Query.Kind != "around" || around.Pack.Query.Subject != middle.Visible ||
		around.Pack.Facts.Show != nil || around.Pack.Facts.Around == nil ||
		len(around.Pack.Facts.Around.Nodes) < 2 || around.Pack.Facts.Around.Nodes[0].Depth != 0 ||
		around.Pack.Facts.Around.Nodes[0].Node.Symbol.Visible != middle.Visible ||
		around.Pack.Budget.MaxDepth != 2 || around.Pack.Usage.Depth == 0 || len(around.Pack.Relations) == 0 {
		t.Fatalf("around pack = %+v", around)
	}
}

func TestExperimentalAtlasPackRelationsSeparateInvocationAndOutputFailures(t *testing.T) {
	root := writeWorkspaceSessionFixture(t, atlasPackRelationSource)
	middle := atlasFunctionSymbol(t, root, "middle")

	for _, test := range []struct {
		name      string
		arguments []string
		contains  string
	}{
		{name: "unknown package", arguments: atlasPackRelationArguments("show", "example.missing", middle.Visible), contains: "no package UID"},
		{name: "unknown symbol", arguments: atlasPackRelationArguments("show", "example.session.app", "fysym1-sha256-128-"+strings.Repeat("0", 32)), contains: "not found"},
		{name: "tiny selection", arguments: replaceAtlasPackArgument(atlasPackRelationArguments("show", "example.session.app", middle.Visible), "--max-selection-bytes", "1"), contains: "cannot contain"},
		{name: "tiny output", arguments: replaceAtlasPackArgument(atlasPackRelationArguments("show", "example.session.app", middle.Visible), "--max-output-bytes", "1"), contains: "exceeds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, stdout, stderr := runForTest(append([]string{"experimental", "atlas-pack"}, test.arguments...), root)
			if code != ExitInvalidInvocation || stdout != "" || !strings.Contains(stderr, test.contains) {
				t.Fatalf("result = (%d, %q, %q), want %q", code, stdout, stderr, test.contains)
			}
		})
	}

	var errorText bytes.Buffer
	code := Run(append([]string{"experimental", "atlas-pack"}, atlasPackRelationArguments("show", "example.session.app", middle.Visible)...), root, failingWriter{}, &errorText)
	if code != ExitInternalFailure || !strings.Contains(errorText.String(), "write output failed") {
		t.Fatalf("output failure = (%d, %q)", code, errorText.String())
	}
}

func TestParseAtlasPackRelationOptionsRejectsIncompleteDuplicateAndNoncanonicalPolicy(t *testing.T) {
	t.Parallel()
	validShow := atlasPackRelationArguments("show", "example.session.app", "fysym1-sha256-128-"+strings.Repeat("0", 32))[1:]
	validAround := atlasPackRelationArguments("around", "example.session.app", "fysym1-sha256-128-"+strings.Repeat("0", 32))[1:]
	for _, test := range []struct {
		name      string
		query     string
		arguments []string
		contains  string
	}{
		{name: "missing", query: "show", arguments: nil, contains: "requires --package-uid"},
		{name: "bad uid names command", query: "show", arguments: replaceAtlasPackArgument(validShow, "--package-uid", "Example"), contains: `"atlas-pack show"`},
		{name: "bad symbol", query: "show", arguments: replaceAtlasPackArgument(validShow, "--symbol", "middle"), contains: "canonical visible or full SymbolId"},
		{name: "bad direction", query: "show", arguments: replaceAtlasPackArgument(validShow, "--direction", "sideways"), contains: "does not support direction"},
		{name: "missing relation", query: "show", arguments: removeAtlasRelationFlag(validShow, "--relation"), contains: "at least one --relation"},
		{name: "duplicate relation", query: "show", arguments: append(validShow, "--relation", "calls"), contains: "repeats relation"},
		{name: "unknown relation", query: "show", arguments: replaceAtlasPackArgument(validShow, "--relation", "guesses"), contains: "does not support relation"},
		{name: "show depth", query: "show", arguments: append(validShow, "--max-depth", "1"), contains: "does not accept flag"},
		{name: "around missing depth", query: "around", arguments: removeAtlasRelationFlag(validAround, "--max-depth"), contains: "requires --max-depth"},
		{name: "around noncanonical depth", query: "around", arguments: replaceAtlasPackArgument(validAround, "--max-depth", "02"), contains: "canonical decimal"},
		{name: "around depth overflow", query: "around", arguments: replaceAtlasPackArgument(validAround, "--max-depth", "65"), contains: "0 through 64"},
		{name: "zero records", query: "show", arguments: replaceAtlasPackArgument(validShow, "--max-records", "0"), contains: "1 through 65536"},
		{name: "zero relations", query: "show", arguments: replaceAtlasPackArgument(validShow, "--max-relations", "0"), contains: "1 through 65536"},
		{name: "duplicate flag", query: "show", arguments: append(validShow, "--direction", "outgoing"), contains: "more than once"},
		{name: "duplicate locked", query: "show", arguments: append(validShow, "--locked", "--locked"), contains: "more than once"},
		{name: "unknown flag", query: "show", arguments: append(validShow, "--pretty"), contains: "does not accept flag"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseAtlasPackRelationOptions(test.query, test.arguments); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("parse error = %v, want containing %q", err, test.contains)
			}
		})
	}

	zeroDepth, err := parseAtlasPackRelationOptions("around", replaceAtlasPackArgument(validAround, "--max-depth", "0"))
	if err != nil || zeroDepth.maxDepth != 0 {
		t.Fatalf("zero-depth around options = (%+v, %v)", zeroDepth, err)
	}
}

func atlasFunctionSymbol(t *testing.T, root, name string) atlasSymbolReference {
	t.Helper()
	arguments := []string{
		"list", "--package-uid", "example.session.app", "--kind", "function",
		"--max-symbols", "16", "--max-selection-bytes", "1048576", "--max-output-bytes", "2097152",
	}
	code, output, stderr := runForTest(append([]string{"experimental", "atlas-symbols"}, arguments...), root)
	if code != ExitSuccess || stderr != "" {
		t.Fatalf("symbol catalog = (%d, %q, %q)", code, output, stderr)
	}
	catalog := decodeAtlasSymbols(t, output)
	for _, entry := range catalog.Catalog.Symbols {
		if entry.Name == name {
			return atlasSymbolReference{Visible: entry.Symbol.Visible, Full: entry.Symbol.Full}
		}
	}
	t.Fatalf("symbol catalog has no function %q", name)
	return atlasSymbolReference{}
}

func atlasPackRelationArguments(query, packageUID, symbol string) []string {
	result := []string{
		query, "--package-uid", packageUID, "--symbol", symbol,
		"--direction", "outgoing", "--relation", "calls",
		"--max-records", "64", "--max-relations", "64",
		"--max-selection-bytes", "1048576", "--max-output-bytes", "2097152",
	}
	if query == "around" {
		result = append(result, "--max-depth", "2")
	}
	return result
}

func removeAtlasRelationFlag(arguments []string, flag string) []string {
	result := make([]string, 0, len(arguments)-2)
	for position := 0; position < len(arguments); position++ {
		if arguments[position] == flag && position+1 < len(arguments) {
			position++
			continue
		}
		result = append(result, arguments[position])
	}
	return result
}

type atlasSymbolReference struct {
	Visible string
	Full    string
}

type atlasPackRelationEnvelope struct {
	PackVersion string `json:"pack_version"`
	Pack        struct {
		Query struct {
			Kind      string   `json:"kind"`
			Subject   string   `json:"subject"`
			Direction string   `json:"direction"`
			Relations []string `json:"relations"`
		} `json:"query"`
		Budget struct {
			MaxDepth int `json:"max_depth"`
		} `json:"budget"`
		Usage struct {
			Records int `json:"records"`
			Depth   int `json:"depth"`
		} `json:"usage"`
		Facts struct {
			Show *struct {
				Subject atlasPackNode `json:"subject"`
			} `json:"show"`
			Around *struct {
				Nodes []struct {
					Depth int           `json:"depth"`
					Node  atlasPackNode `json:"node"`
				} `json:"nodes"`
			} `json:"around"`
		} `json:"facts"`
		Relations []struct {
			Kind   string `json:"kind"`
			Target *struct {
				Symbol string `json:"symbol"`
			} `json:"target"`
		} `json:"relations"`
		Documentation  []json.RawMessage `json:"documentation"`
		SourceExcerpts []json.RawMessage `json:"source_excerpts"`
	} `json:"pack"`
}

type atlasPackNode struct {
	Symbol struct {
		Visible string `json:"Visible"`
		Full    string `json:"Full"`
	} `json:"symbol"`
}

func decodeAtlasPackRelation(t *testing.T, output string) atlasPackRelationEnvelope {
	t.Helper()
	var result atlasPackRelationEnvelope
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
