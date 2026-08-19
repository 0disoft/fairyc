package contractabi

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	gotoken "go/token"
	"go/types"
	"strings"
	"testing"
)

func TestRenderJSONIsDeterministicAndStructured(t *testing.T) {
	t.Parallel()
	document := mustParseInterface(t, validTokenInterface)
	first, err := RenderJSON(document)
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	second, err := RenderJSON(document)
	if err != nil {
		t.Fatalf("RenderJSON() second error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("RenderJSON() is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if len(first) == 0 || first[len(first)-1] != '\n' {
		t.Fatalf("RenderJSON() has no trailing newline: %q", first)
	}
	var decoded Document
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("RenderJSON() output is invalid JSON: %v", err)
	}
	if decoded.Schema != SchemaVersion || decoded.Contracts[0].Functions[0].Effects[0] != externalCallEffect {
		t.Fatalf("decoded IDL = %+v", decoded)
	}
}

func TestRenderTypeScriptProducesTypedTransportClients(t *testing.T) {
	t.Parallel()
	document := mustParseInterface(t, validTokenInterface)
	generated, err := RenderTypeScript(document)
	if err != nil {
		t.Fatalf("RenderTypeScript() error = %v", err)
	}
	text := string(generated)
	for _, fragment := range []string{
		"export interface ContractTransport",
		"export class TokenClient",
		"public balance_of(owner: Address): Promise<Result<bigint, ContractError>>",
		"public transfer(to: Address, amount: bigint, options: PayableCallOptions)",
		"contract: \"Token\"",
		"method: \"transfer\"",
		"view: true",
		"payable: true",
		"effects: [\"external_call\"]",
		"export interface TransferEvent",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("generated TypeScript does not contain %q:\n%s", fragment, text)
		}
	}
	if strings.Contains(text, "ethers") || strings.Contains(text, "borsh") || strings.Contains(text, "scale") {
		t.Fatalf("generated TypeScript leaked a chain codec:\n%s", text)
	}
}

func TestRenderGoProducesTypeCheckedBindings(t *testing.T) {
	t.Parallel()
	document := mustParseInterface(t, validTokenInterface)
	generated, err := RenderGo(document, "contractbindings")
	if err != nil {
		t.Fatalf("RenderGo() error = %v", err)
	}
	text := string(generated)
	for _, fragment := range []string{
		"type ContractTransport interface",
		"type TokenClient struct",
		"func (client *TokenClient) BalanceOf",
		"func (client *TokenClient) Transfer",
		"type TransferEvent struct",
		"type FairyU256 [32]byte",
		"Contract:  \"Token\"",
		"Method:    \"transfer\"",
		"Effects:   []string{\"external_call\"}",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("generated Go does not contain %q:\n%s", fragment, text)
		}
	}
	typeCheckGeneratedGo(t, generated)
}

func TestRenderersAvoidGeneratedParameterNameCollisions(t *testing.T) {
	t.Parallel()
	source := `module main;
extern contract Keywords {
    @payable
    fn Result<void, Error> call(Address options, Address class, Address class_);
}
`
	document := mustParseInterface(t, source)
	typescript, err := RenderTypeScript(document)
	if err != nil {
		t.Fatalf("RenderTypeScript() error = %v", err)
	}
	if !strings.Contains(string(typescript), "options_: Address, class_: Address, class__: Address, options: PayableCallOptions") {
		t.Fatalf("generated TypeScript parameter allocation is unstable:\n%s", typescript)
	}
	goSource, err := RenderGo(document, "bindings")
	if err != nil {
		t.Fatalf("RenderGo() error = %v", err)
	}
	typeCheckGeneratedGo(t, goSource)
}

func TestRenderGoRejectsInvalidPackageName(t *testing.T) {
	t.Parallel()
	document := mustParseInterface(t, validTokenInterface)
	if _, err := RenderGo(document, "Contract-Bindings"); err == nil {
		t.Fatal("RenderGo() accepted an invalid package name")
	}
}

func mustParseInterface(t *testing.T, source string) Document {
	t.Helper()
	document, diagnostics := Parse([]byte(source))
	if len(diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", diagnostics)
	}
	return document
}

func typeCheckGeneratedGo(t *testing.T, source []byte) {
	t.Helper()
	files := gotoken.NewFileSet()
	parsed, err := parser.ParseFile(files, "bindings.go", source, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated Go parse error = %v\n%s", err, source)
	}
	config := types.Config{Importer: importer.Default()}
	if _, err := config.Check("generated/contractbindings", files, []*ast.File{parsed}, nil); err != nil {
		t.Fatalf("generated Go type error = %v\n%s", err, source)
	}
}

func TestRenderersRejectGeneratedIdentifierCollisions(t *testing.T) {
	t.Parallel()
	source := `module main;
extern contract foo_bar { fn void call(); }
extern contract foo__bar { fn void invoke(); }
`
	document := mustParseInterface(t, source)
	if _, err := RenderGo(document, "bindings"); err == nil {
		t.Fatal("RenderGo() accepted colliding generated client names")
	}
	if _, err := RenderTypeScript(document); err == nil {
		t.Fatal("RenderTypeScript() accepted colliding generated client names")
	}
}
