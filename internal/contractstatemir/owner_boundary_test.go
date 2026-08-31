package contractstatemir

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestHIRTypeContainsRawPointerTraversesOwnerTypeGraph(t *testing.T) {
	structType := hir.Type("ledger::PointerStruct")
	enumType := hir.Type("ledger::PointerEnum")
	resourceType := hir.Type("ledger::PointerResource")
	cycleType := hir.Type("ledger::Cycle")
	input := hir.Package{Modules: []hir.Module{{
		Structs: []hir.Struct{
			{Type: structType, Fields: []hir.StructField{{Type: hir.RawPointer(hir.TypeU8, false)}}},
			{Type: cycleType, Fields: []hir.StructField{{Type: cycleType}}},
		},
		Enums:     []hir.Enum{{Type: enumType, Variants: []hir.EnumVariant{{Fields: []hir.EnumField{{Type: hir.RawPointer(hir.TypeU32, true)}}}}}},
		Resources: []hir.Resource{{Type: resourceType, Fields: []hir.ResourceField{{Type: hir.RawPointer(hir.TypeU8, true)}}}},
	}}}
	tests := []struct {
		name      string
		typeValue hir.Type
		want      bool
	}{
		{name: "direct", typeValue: hir.RawPointer(hir.TypeU8, false), want: true},
		{name: "borrow element", typeValue: hir.SharedBorrow(hir.RawPointer(hir.TypeU8, false)), want: true},
		{name: "fixed array", typeValue: hir.FixedArray(hir.RawPointer(hir.TypeU8, false), 2), want: true},
		{name: "option", typeValue: hir.Option(hir.RawPointer(hir.TypeU8, false)), want: true},
		{name: "result", typeValue: hir.Result(hir.TypeU32, hir.RawPointer(hir.TypeU8, false)), want: true},
		{name: "struct field", typeValue: structType, want: true},
		{name: "enum field", typeValue: enumType, want: true},
		{name: "resource field", typeValue: resourceType, want: true},
		{name: "safe borrow", typeValue: hir.MutableBorrow(hir.TypeU32), want: false},
		{name: "pointer-free cycle", typeValue: cycleType, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hirTypeContainsRawPointer(input, test.typeValue, make(map[hir.Type]bool)); got != test.want {
				t.Fatalf("hirTypeContainsRawPointer(%q) = %t, want %t", test.typeValue, got, test.want)
			}
		})
	}
}

func TestHIRTypeContainsOwnedResourceTraversesValueGraph(t *testing.T) {
	resourceType := hir.Type("ledger::Handle")
	structType := hir.Type("ledger::ResourceStruct")
	enumType := hir.Type("ledger::ResourceEnum")
	cycleType := hir.Type("ledger::Cycle")
	input := hir.Package{Modules: []hir.Module{{
		Structs: []hir.Struct{
			{Type: structType, Fields: []hir.StructField{{Type: resourceType}}},
			{Type: cycleType, Fields: []hir.StructField{{Type: cycleType}}},
		},
		Enums:     []hir.Enum{{Type: enumType, Variants: []hir.EnumVariant{{Fields: []hir.EnumField{{Type: resourceType}}}}}},
		Resources: []hir.Resource{{Type: resourceType, DropKind: hir.ResourceDropEmpty}},
	}}}
	tests := []struct {
		name string
		typ  hir.Type
		want bool
	}{
		{name: "resource", typ: resourceType, want: true},
		{name: "fixed array", typ: hir.FixedArray(resourceType, 2), want: true},
		{name: "option", typ: hir.Option(resourceType), want: true},
		{name: "result", typ: hir.Result(hir.TypeU32, resourceType), want: true},
		{name: "struct field", typ: structType, want: true},
		{name: "enum field", typ: enumType, want: true},
		{name: "shared resource borrow", typ: hir.SharedBorrow(resourceType)},
		{name: "mutable resource borrow", typ: hir.MutableBorrow(resourceType)},
		{name: "scalar", typ: hir.TypeU32},
		{name: "pointer-free cycle", typ: cycleType},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hirTypeContainsOwnedResource(input, test.typ, make(map[hir.Type]bool)); got != test.want {
				t.Fatalf("hirTypeContainsOwnedResource(%q) = %t, want %t", test.typ, got, test.want)
			}
		})
	}
}

func TestHIRStateOwnerTrapBoundaryClassifiesClosedForms(t *testing.T) {
	tests := []struct {
		name      string
		function  hir.Function
		wantError bool
	}{
		{name: "unary minus", function: trapOwnerExpression("prefix", "-"), wantError: true},
		{name: "bitwise complement", function: trapOwnerExpression("prefix", "~")},
		{name: "array index", function: trapOwnerExpression("array-index", "[]"), wantError: true},
		{name: "view index", function: trapOwnerExpression("view-index", "[]"), wantError: true},
		{name: "view slice", function: trapOwnerExpression("view-slice", ".."), wantError: true},
		{name: "scalar cast", function: trapOwnerExpression("scalar-cast", "as")},
	}
	for _, operator := range []string{"+", "-", "*", "/", "%", "<<", ">>"} {
		tests = append(tests, struct {
			name      string
			function  hir.Function
			wantError bool
		}{name: "binary " + operator, function: trapOwnerExpression("binary", operator), wantError: true})
	}
	for _, operator := range []string{"==", "!=", "<", "<=", ">", ">=", "&", "|", "^"} {
		tests = append(tests, struct {
			name      string
			function  hir.Function
			wantError bool
		}{name: "safe binary " + operator, function: trapOwnerExpression("binary", operator)})
	}
	for _, operator := range []string{"+=", "-=", "*=", "/=", "%=", "<<=", ">>="} {
		tests = append(tests, struct {
			name      string
			function  hir.Function
			wantError bool
		}{name: "compound " + operator, function: trapOwnerAssignment(operator), wantError: true})
	}
	for _, operator := range []string{"=", "&=", "|=", "^="} {
		tests = append(tests, struct {
			name      string
			function  hir.Function
			wantError bool
		}{name: "safe assignment " + operator, function: trapOwnerAssignment(operator)})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyHIRStateOwnerTrapBoundary(test.function)
			if (err != nil) != test.wantError {
				t.Fatalf("verifyHIRStateOwnerTrapBoundary() error = %v, wantError = %t", err, test.wantError)
			}
		})
	}
}

func TestHIRStateOwnerLifetimeBoundaryClassifiesOwnedValuesAndRegions(t *testing.T) {
	resourceType := hir.Type("ledger::Handle")
	input := hir.Package{Modules: []hir.Module{{Resources: []hir.Resource{{Type: resourceType, DropKind: hir.ResourceDropEmpty}}}}}
	tests := []struct {
		name     string
		function hir.Function
		contains string
	}{
		{name: "owned parameter", function: hir.Function{Parameters: []hir.Parameter{{Type: resourceType}}}, contains: "owned resource values are not admitted"},
		{name: "owned result", function: hir.Function{ResultType: hir.Option(resourceType)}, contains: "owned resource values are not admitted"},
		{name: "owned local", function: hir.Function{Locals: []hir.Local{{Type: resourceType}}}, contains: "owned resource values are not admitted"},
		{name: "nested owned expression", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "group", Children: []hir.Expression{{Kind: "path", Type: resourceType}}}}}, contains: "owned resource values are not admitted"},
		{name: "nested lexical region", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "group", Children: []hir.Expression{{Kind: "region"}}}}}, contains: "lexical regions are not admitted"},
		{name: "borrowed resource", function: hir.Function{Parameters: []hir.Parameter{{Type: hir.SharedBorrow(resourceType)}}}},
		{name: "static resource namespace", function: hir.Function{OwnerType: resourceType}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyHIRStateOwnerLifetimeBoundary(input, test.function)
			if test.contains == "" && err != nil {
				t.Fatalf("verifyHIRStateOwnerLifetimeBoundary() error = %v", err)
			}
			if test.contains != "" && (err == nil || !strings.Contains(err.Error(), test.contains)) {
				t.Fatalf("verifyHIRStateOwnerLifetimeBoundary() error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestHIRStateOwnerClosedFormBoundaryUsesPositiveAllowlists(t *testing.T) {
	allowedExpressions := []hir.Expression{
		{Kind: "literal"}, {Kind: "path"}, {Kind: "group"}, {Kind: "record"}, {Kind: "array"},
		{Kind: "array-view-borrow"}, {Kind: "array-length"}, {Kind: "field-expression"},
		{Kind: "view-length"}, {Kind: "view-empty"}, {Kind: "scalar-cast"},
		{Kind: "result-variant"}, {Kind: "option-variant"}, {Kind: "enum-variant"},
		{Kind: "block", Block: &hir.Block{}},
	}
	for _, operator := range []string{"+", "!", "~", "&", "&mut", "*"} {
		allowedExpressions = append(allowedExpressions, hir.Expression{Kind: "prefix", Operator: operator})
	}
	for _, operator := range []string{"==", "!=", "<", "<=", ">", ">=", "&", "|", "^"} {
		allowedExpressions = append(allowedExpressions, hir.Expression{Kind: "binary", Operator: operator})
	}
	for _, expression := range allowedExpressions {
		if err := verifyHIRStateOwnerClosedFormBoundary(hir.Function{Body: hir.Block{Tail: &expression}}); err != nil {
			t.Errorf("allowed expression %q/%q rejected: %v", expression.Kind, expression.Operator, err)
		}
	}
	for _, statement := range []hir.Statement{
		{Kind: "binding"}, {Kind: "return"}, {Kind: "discard"}, {Kind: "expression-statement"},
		{Kind: "assignment"}, {Kind: "assignment", Operator: "="}, {Kind: "assignment", Operator: "&="},
		{Kind: "assignment", Operator: "|="}, {Kind: "assignment", Operator: "^="},
	} {
		if err := verifyHIRStateOwnerClosedFormBoundary(hir.Function{Body: hir.Block{Statements: []hir.Statement{statement}}}); err != nil {
			t.Errorf("allowed statement %q/%q rejected: %v", statement.Kind, statement.Operator, err)
		}
	}
	rejected := []struct {
		name     string
		function hir.Function
		contains string
	}{
		{name: "future statement", function: hir.Function{Body: hir.Block{Statements: []hir.Statement{{Kind: "future-statement"}}}}, contains: "statement kind \"future-statement\" is not admitted"},
		{name: "future assignment", function: hir.Function{Body: hir.Block{Statements: []hir.Statement{{Kind: "assignment", Operator: "??="}}}}, contains: "assignment operator \"??=\" is not admitted"},
		{name: "hidden loop payload", function: hir.Function{Body: hir.Block{Statements: []hir.Statement{{Kind: "binding", For: &hir.ForLoop{}}}}}, contains: "carries loop state"},
		{name: "future expression", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "future-expression"}}}, contains: "expression kind \"future-expression\" is not admitted"},
		{name: "future prefix", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "prefix", Operator: "future"}}}, contains: "prefix operator \"future\" is not admitted"},
		{name: "future binary", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "binary", Operator: "future"}}}, contains: "binary operator \"future\" is not admitted"},
		{name: "expression effects", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "literal", Effects: []hir.Effect{hir.EffectStorageRead}}}}, contains: "carries expression-local effects"},
		{name: "nested child", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "group", Children: []hir.Expression{{Kind: "future-child"}}}}}, contains: "future-child"},
		{name: "nested block", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "block", Block: &hir.Block{Tail: &hir.Expression{Kind: "future-block"}}}}}, contains: "future-block"},
		{name: "nested arm", function: hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: "enum-variant", Arms: []hir.MatchArm{{Expression: hir.Expression{Kind: "future-arm"}}}}}}, contains: "future-arm"},
	}
	for _, test := range rejected {
		t.Run(test.name, func(t *testing.T) {
			err := verifyHIRStateOwnerClosedFormBoundary(test.function)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("verifyHIRStateOwnerClosedFormBoundary() error = %v, want %q", err, test.contains)
			}
		})
	}
}

func trapOwnerExpression(kind, operator string) hir.Function {
	return hir.Function{Body: hir.Block{Tail: &hir.Expression{Kind: kind, Operator: operator}}}
}

func trapOwnerAssignment(operator string) hir.Function {
	return hir.Function{Body: hir.Block{Statements: []hir.Statement{{Kind: "assignment", Operator: operator}}}}
}
