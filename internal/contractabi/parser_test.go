package contractabi

import (
	"slices"
	"strings"
	"testing"
)

const validTokenInterface = `module wallet::token;

extern contract Token {
    @view
    fn Result<u64, ContractError> balance_of(Address owner);

    @payable
    fn Result<void, ContractError> transfer(Address to, u256 amount);
}

@event
struct Transfer {
    Address from;
    Address to;
    u256 amount;
}
`

func TestParseContractInterface(t *testing.T) {
	t.Parallel()
	document, diagnostics := Parse([]byte(validTokenInterface))
	if len(diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", diagnostics)
	}
	if document.Schema != SchemaVersion || document.Module != "wallet::token" {
		t.Fatalf("Parse() document header = %+v", document)
	}
	if len(document.Contracts) != 1 || document.Contracts[0].Name != "Token" {
		t.Fatalf("Parse() contracts = %+v", document.Contracts)
	}
	functions := document.Contracts[0].Functions
	if len(functions) != 2 {
		t.Fatalf("Parse() functions = %+v", functions)
	}
	if !functions[0].View || functions[0].Payable || functions[0].Name != "balance_of" {
		t.Fatalf("balance_of = %+v", functions[0])
	}
	if functions[1].View || !functions[1].Payable || functions[1].Name != "transfer" {
		t.Fatalf("transfer = %+v", functions[1])
	}
	for _, function := range functions {
		if !slices.Equal(function.Effects, []string{externalCallEffect}) {
			t.Fatalf("%s effects = %v", function.Name, function.Effects)
		}
	}
	if len(document.Events) != 1 || document.Events[0].Name != "Transfer" || len(document.Events[0].Fields) != 3 {
		t.Fatalf("Parse() events = %+v", document.Events)
	}
	if got := document.Events[0].Fields[2].Type.String(); got != "u256" {
		t.Fatalf("amount type = %q", got)
	}
}

func TestParseContractInterfaceRejectsUnsafeBoundaryTypes(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"reference":         "ref Address owner",
		"pointer":           "ptr u8 bytes",
		"slice":             "u8[] bytes",
		"target integer":    "usize count",
		"void parameter":    "void value",
		"generic nominal":   "Map<Address,u64> value",
		"zero length array": "u8[0] bytes",
	}
	for name, parameter := range tests {
		name, parameter := name, parameter
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := "module main; extern contract C { fn void call(" + parameter + "); }"
			_, diagnostics := Parse([]byte(source))
			if !hasDiagnostic(diagnostics, "CIDL1007") {
				t.Fatalf("Parse() diagnostics = %+v, want CIDL1007", diagnostics)
			}
		})
	}
}

func TestParseContractInterfaceRejectsConflictingAndDuplicateDeclarations(t *testing.T) {
	t.Parallel()
	source := `module main;
extern contract Token {
    @view @payable
    fn void send(Address owner, Address owner);
    fn void send(Address other);
}
@event struct Token { u64 value; u64 value; }
`
	_, diagnostics := Parse([]byte(source))
	for _, code := range []string{"CIDL1005", "CIDL1008"} {
		if !hasDiagnostic(diagnostics, code) {
			t.Fatalf("Parse() diagnostics = %+v, want %s", diagnostics, code)
		}
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Line < 1 || diagnostic.Column < 1 || diagnostic.StartByte < 0 || diagnostic.EndByte < diagnostic.StartByte {
			t.Fatalf("invalid diagnostic coordinates: %+v", diagnostic)
		}
	}
}

func TestParseContractInterfaceRequiresEventAttribute(t *testing.T) {
	t.Parallel()
	_, diagnostics := Parse([]byte("module main; struct Transfer { u64 amount; }"))
	if !hasDiagnostic(diagnostics, "CIDL1006") {
		t.Fatalf("Parse() diagnostics = %+v, want CIDL1006", diagnostics)
	}
}

func TestParseContractInterfaceRejectsReservedIdentifiersAndMalformedLengths(t *testing.T) {
	t.Parallel()
	tests := []string{
		"module main; extern contract contract {}",
		"module main; extern contract C { fn void fn(); }",
		"module main; @event struct E { u8[1__2] bytes; }",
		"module main; @event struct E { u8[12_] bytes; }",
	}
	for _, source := range tests {
		_, diagnostics := Parse([]byte(source))
		if !hasDiagnostic(diagnostics, "CIDL1002") {
			t.Fatalf("Parse(%q) diagnostics = %+v, want CIDL1002", source, diagnostics)
		}
	}
}

func TestParseContractInterfacePreservesDeclarationOrder(t *testing.T) {
	t.Parallel()
	source := `module main;
extern contract Zebra { fn u64 z(); fn u64 a(); }
extern contract Alpha { fn u64 b(); }
@event struct Later { u64 z; u64 a; }
`
	document, diagnostics := Parse([]byte(source))
	if len(diagnostics) != 0 {
		t.Fatalf("Parse() diagnostics = %+v", diagnostics)
	}
	if got := []string{document.Contracts[0].Name, document.Contracts[1].Name}; !slices.Equal(got, []string{"Zebra", "Alpha"}) {
		t.Fatalf("contract order = %v", got)
	}
	if got := []string{document.Contracts[0].Functions[0].Name, document.Contracts[0].Functions[1].Name}; !slices.Equal(got, []string{"z", "a"}) {
		t.Fatalf("function order = %v", got)
	}
	if got := []string{document.Events[0].Fields[0].Name, document.Events[0].Fields[1].Name}; !slices.Equal(got, []string{"z", "a"}) {
		t.Fatalf("field order = %v", got)
	}
}

func TestParseContractInterfaceReportsUnexpectedUnicodeToken(t *testing.T) {
	t.Parallel()
	_, diagnostics := Parse([]byte("module main; extern contract 토큰 {}"))
	if len(diagnostics) == 0 || !strings.HasPrefix(diagnostics[0].Code, "CIDL") {
		t.Fatalf("Parse() diagnostics = %+v", diagnostics)
	}
}

func hasDiagnostic(values []Diagnostic, code string) bool {
	for _, value := range values {
		if value.Code == code {
			return true
		}
	}
	return false
}

func TestParseContractInterfaceRejectsRecursiveEventTypes(t *testing.T) {
	t.Parallel()
	source := `module main;
@event struct Node { Option<Node> next; }
`
	_, diagnostics := Parse([]byte(source))
	if !hasDiagnostic(diagnostics, "CIDL1007") {
		t.Fatalf("Parse() diagnostics = %+v, want CIDL1007", diagnostics)
	}
}
