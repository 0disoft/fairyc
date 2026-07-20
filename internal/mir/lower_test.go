package mir

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestLowerPreservesResourceDefinitionAndFieldTypes(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Pair {
    left: i32;
    ready: bool;
    drop fn(self: &mut Self) {}
}
fn main() -> i32 {
    let pair: Pair = Pair { left: 42, ready: true };
    pair.left
}
`)
	if len(program.ResourceTypes) != 1 {
		t.Fatalf("resource types = %+v", program.ResourceTypes)
	}
	definition := program.ResourceTypes[0]
	if definition.Type != hir.Type("main::Pair") || definition.Entity != "resource-type:main-pair" || definition.DropKind != hir.ResourceDropEmpty || len(definition.Fields) != 2 {
		t.Fatalf("resource definition = %+v", definition)
	}
	if definition.Fields[0].Name != "left" || definition.Fields[0].Type != hir.TypeI32 || definition.Fields[1].Name != "ready" || definition.Fields[1].Type != hir.TypeBool {
		t.Fatalf("resource fields = %+v", definition.Fields)
	}

	var construct, read *Operation
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			operations := program.Functions[functionIndex].Blocks[blockIndex].Operations
			for operationIndex := range operations {
				operation := &operations[operationIndex]
				switch operation.Kind {
				case OperationResourceConstruct:
					construct = operation
				case OperationFieldRead:
					read = operation
				}
			}
		}
	}
	if construct == nil || construct.Type != definition.Type || !slices.Equal(construct.Fields, []string{"left", "ready"}) {
		t.Fatalf("resource construction = %+v", construct)
	}
	if read == nil || read.OwnerType != definition.Type || read.Type != hir.TypeI32 || read.Field != "left" {
		t.Fatalf("field read = %+v", read)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesReprCStructDefinitionConstructionAndFieldRead(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@repr(C)
struct Packet { code: u32; length: usize; }
fn main() -> i32 {
    let packet: Packet = Packet { code: 42, length: 0 };
    if packet.code == 42 { 0 } else { 1 }
}
`)
	if len(program.StructTypes) != 1 {
		t.Fatalf("struct types = %+v", program.StructTypes)
	}
	definition := program.StructTypes[0]
	if definition.Type != hir.Type("main::Packet") || definition.Name != "Packet" || definition.Entity != "struct-type:main-packet" || !definition.ReprC || len(definition.Fields) != 2 {
		t.Fatalf("struct definition = %+v", definition)
	}
	if definition.Fields[0].Name != "code" || definition.Fields[0].Type != hir.TypeU32 || definition.Fields[1].Name != "length" || definition.Fields[1].Type != hir.TypeUSize {
		t.Fatalf("struct fields = %+v", definition.Fields)
	}
	var construct, read *Operation
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			operations := program.Functions[functionIndex].Blocks[blockIndex].Operations
			for operationIndex := range operations {
				operation := &operations[operationIndex]
				switch operation.Kind {
				case OperationStructConstruct:
					construct = operation
				case OperationFieldRead:
					read = operation
				}
			}
		}
	}
	if construct == nil || construct.Type != definition.Type || !slices.Equal(construct.Fields, []string{"code", "length"}) {
		t.Fatalf("struct construction = %+v", construct)
	}
	if read == nil || read.OwnerType != definition.Type || read.Type != hir.TypeU32 || read.Field != "code" {
		t.Fatalf("field read = %+v", read)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerRejectsNonEmptyResourceDropKind(t *testing.T) {
	t.Parallel()
	_, err := Lower(hir.Package{Modules: []hir.Module{{
		Path: "main",
		Resources: []hir.Resource{{
			Name: "Token", Type: hir.Type("main::Token"), DropKind: hir.ResourceDropNonEmpty,
			Fields: []hir.ResourceField{{Name: "value", Type: hir.TypeI32}},
		}},
	}}})
	if err == nil || !strings.Contains(err.Error(), "unsupported drop kind") {
		t.Fatalf("Lower() error = %v", err)
	}
}

func TestLowerPreservesForeignCallABIAndUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\n@link_name(\"fairyc_fixture_add_one\")\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { unsafe { add_one(41) } }\n")
	if len(program.ForeignFunctions) != 1 {
		t.Fatalf("foreign functions = %+v", program.ForeignFunctions)
	}
	foreign := program.ForeignFunctions[0]
	if foreign.Name != "main::add_one" || foreign.LinkName != "fairyc_fixture_add_one" || foreign.ABI != "C" || foreign.ResultType != hir.TypeI32 || !slices.Equal(foreign.ParameterTypes, []hir.Type{hir.TypeI32}) {
		t.Fatalf("foreign function = %+v", foreign)
	}
	function := program.Functions[0]
	var call *Terminator
	for blockIndex := range function.Blocks {
		if function.Blocks[blockIndex].Terminator.Kind == TerminatorCall {
			call = &function.Blocks[blockIndex].Terminator
			break
		}
	}
	if call == nil || call.Callee != foreign.Symbol || call.ForeignABI != "C" || call.UnsafeScope.Source != "main.fy" || call.Entity == "" {
		t.Fatalf("foreign call = %+v", call)
	}
	if len(function.UnsafeOperations) != 1 || function.UnsafeOperations[0].Entity != call.Entity || function.UnsafeOperations[0].SourceSpanEntity == "" || function.UnsafeOperations[0].ScopeEntity == "" {
		t.Fatalf("unsafe operations = %+v", function.UnsafeOperations)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	wantEntity := call.Entity
	for _, test := range []struct {
		name   string
		mutate func(*Program)
	}{
		{
			name: "missing unsafe operation",
			mutate: func(mutated *Program) {
				mutated.Functions[0].UnsafeOperations = nil
			},
		},
		{
			name: "forged source span entity",
			mutate: func(mutated *Program) {
				mutated.Functions[0].UnsafeOperations[0].SourceSpanEntity += "-forged"
			},
		},
		{
			name: "scope does not contain call",
			mutate: func(mutated *Program) {
				for blockIndex := range mutated.Functions[0].Blocks {
					terminator := &mutated.Functions[0].Blocks[blockIndex].Terminator
					if terminator.Kind == TerminatorCall {
						terminator.UnsafeScope.StartByte = terminator.Span.StartByte + 1
						return
					}
				}
				t.Fatal("cloned fixture has no foreign call")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := Clone(program)
			test.mutate(&mutated)
			failures := Verify(mutated)
			if len(failures) != 1 || failures[0].Code != "V1701" || failures[0].Entity != wantEntity {
				t.Fatalf("Verify() after %s = %+v", test.name, failures)
			}
		})
	}
}

func TestLowerPreservesCIntAcrossForeignCallResultAndArgument(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn make_c_int() -> c::int;
@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn consume_c_int(value: c::int) -> i32;
fn main() -> i32 { unsafe { consume_c_int(make_c_int()) } }
`)
	if len(program.ForeignFunctions) != 2 {
		t.Fatalf("foreign functions = %+v", program.ForeignFunctions)
	}
	var producer, consumer ForeignFunction
	for _, foreign := range program.ForeignFunctions {
		switch foreign.Name {
		case "main::make_c_int":
			producer = foreign
		case "main::consume_c_int":
			consumer = foreign
		}
	}
	if producer.ResultType != hir.TypeCInt || len(producer.ParameterTypes) != 0 || consumer.ResultType != hir.TypeI32 || !slices.Equal(consumer.ParameterTypes, []hir.Type{hir.TypeCInt}) {
		t.Fatalf("foreign signatures = %+v", program.ForeignFunctions)
	}
	var producerCall, consumerCall *Terminator
	for blockIndex := range program.Functions[0].Blocks {
		terminator := &program.Functions[0].Blocks[blockIndex].Terminator
		if terminator.Kind != TerminatorCall {
			continue
		}
		switch terminator.Callee {
		case producer.Symbol:
			producerCall = terminator
		case consumer.Symbol:
			consumerCall = terminator
		}
	}
	if producerCall == nil || consumerCall == nil || producerCall.DestinationLocal == NoLocal || len(consumerCall.Arguments) != 1 || consumerCall.Arguments[0].Kind != OperandLocal || consumerCall.Arguments[0].Local != producerCall.DestinationLocal {
		t.Fatalf("producer call = %+v, consumer call = %+v", producerCall, consumerCall)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesCSizeTAcrossForeignCallResultAndArgument(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_make_c_size_t")
unsafe extern C fn make_c_size_t() -> c::size_t;
@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn consume_c_size_t(value: c::size_t) -> i32;
fn main() -> i32 { unsafe { consume_c_size_t(make_c_size_t()) } }
`)
	var producer, consumer ForeignFunction
	for _, foreign := range program.ForeignFunctions {
		switch foreign.Name {
		case "main::make_c_size_t":
			producer = foreign
		case "main::consume_c_size_t":
			consumer = foreign
		}
	}
	if producer.ResultType != hir.TypeCSizeT || consumer.ResultType != hir.TypeI32 || !slices.Equal(consumer.ParameterTypes, []hir.Type{hir.TypeCSizeT}) {
		t.Fatalf("foreign signatures = %+v", program.ForeignFunctions)
	}
	var producerCall, consumerCall *Terminator
	for blockIndex := range program.Functions[0].Blocks {
		terminator := &program.Functions[0].Blocks[blockIndex].Terminator
		if terminator.Kind != TerminatorCall {
			continue
		}
		switch terminator.Callee {
		case producer.Symbol:
			producerCall = terminator
		case consumer.Symbol:
			consumerCall = terminator
		}
	}
	if producerCall == nil || consumerCall == nil || producerCall.DestinationLocal == NoLocal || len(consumerCall.Arguments) != 1 || consumerCall.Arguments[0].Kind != OperandLocal || consumerCall.Arguments[0].Local != producerCall.DestinationLocal {
		t.Fatalf("producer call = %+v, consumer call = %+v", producerCall, consumerCall)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesCLongAcrossForeignCallResultAndArgument(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_make_c_long")
unsafe extern C fn make_c_long() -> c::long;
@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn consume_c_long(value: c::long) -> i32;
fn main() -> i32 { unsafe { consume_c_long(make_c_long()) } }
`)
	var producer, consumer ForeignFunction
	for _, foreign := range program.ForeignFunctions {
		switch foreign.Name {
		case "main::make_c_long":
			producer = foreign
		case "main::consume_c_long":
			consumer = foreign
		}
	}
	if producer.ResultType != hir.TypeCLong || consumer.ResultType != hir.TypeI32 || !slices.Equal(consumer.ParameterTypes, []hir.Type{hir.TypeCLong}) {
		t.Fatalf("foreign signatures = %+v", program.ForeignFunctions)
	}
	var producerCall, consumerCall *Terminator
	for blockIndex := range program.Functions[0].Blocks {
		terminator := &program.Functions[0].Blocks[blockIndex].Terminator
		if terminator.Kind != TerminatorCall {
			continue
		}
		switch terminator.Callee {
		case producer.Symbol:
			producerCall = terminator
		case consumer.Symbol:
			consumerCall = terminator
		}
	}
	if producerCall == nil || consumerCall == nil || producerCall.DestinationLocal == NoLocal || len(consumerCall.Arguments) != 1 || consumerCall.Arguments[0].Kind != OperandLocal || consumerCall.Arguments[0].Local != producerCall.DestinationLocal {
		t.Fatalf("producer call = %+v, consumer call = %+v", producerCall, consumerCall)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesCSSizeTAcrossForeignCallResultAndArgument(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_make_c_ssize_t")
unsafe extern C fn make_c_ssize_t() -> c::ssize_t;
@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn consume_c_ssize_t(value: c::ssize_t) -> i32;
fn main() -> i32 { unsafe { consume_c_ssize_t(make_c_ssize_t()) } }
`)
	var producer, consumer ForeignFunction
	for _, foreign := range program.ForeignFunctions {
		switch foreign.Name {
		case "main::make_c_ssize_t":
			producer = foreign
		case "main::consume_c_ssize_t":
			consumer = foreign
		}
	}
	if producer.ResultType != hir.TypeCSSizeT || consumer.ResultType != hir.TypeI32 || !slices.Equal(consumer.ParameterTypes, []hir.Type{hir.TypeCSSizeT}) {
		t.Fatalf("foreign signatures = %+v", program.ForeignFunctions)
	}
	var producerCall, consumerCall *Terminator
	for blockIndex := range program.Functions[0].Blocks {
		terminator := &program.Functions[0].Blocks[blockIndex].Terminator
		if terminator.Kind != TerminatorCall {
			continue
		}
		switch terminator.Callee {
		case producer.Symbol:
			producerCall = terminator
		case consumer.Symbol:
			consumerCall = terminator
		}
	}
	if producerCall == nil || consumerCall == nil || producerCall.DestinationLocal == NoLocal || len(consumerCall.Arguments) != 1 || consumerCall.Arguments[0].Kind != OperandLocal || consumerCall.Arguments[0].Local != producerCall.DestinationLocal {
		t.Fatalf("producer call = %+v, consumer call = %+v", producerCall, consumerCall)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesRemainingCScalarsAcrossForeignCalls(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		typ  hir.Type
	}{
		{name: "bool", typ: hir.TypeCBool},
		{name: "char", typ: hir.TypeCChar},
		{name: "double", typ: hir.TypeCDouble},
		{name: "float", typ: hir.TypeCFloat},
		{name: "longlong", typ: hir.TypeCLongLong},
		{name: "schar", typ: hir.TypeCSChar},
		{name: "short", typ: hir.TypeCShort},
		{name: "uchar", typ: hir.TypeCUChar},
		{name: "uint", typ: hir.TypeCUInt},
		{name: "ulong", typ: hir.TypeCULong},
		{name: "ulonglong", typ: hir.TypeCULongLong},
		{name: "ushort", typ: hir.TypeCUShort},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			text := fmt.Sprintf(`module main;
unsafe extern C fn make_%[1]s() -> %[2]s;
unsafe extern C fn consume_%[1]s(value: %[2]s) -> i32;
fn main() -> i32 { unsafe { consume_%[1]s(make_%[1]s()) } }
`, test.name, test.typ)
			program := lowerTestSource(t, text)
			if len(program.ForeignFunctions) != 2 {
				t.Fatalf("foreign functions = %+v", program.ForeignFunctions)
			}
			var producer, consumer ForeignFunction
			for _, foreign := range program.ForeignFunctions {
				switch foreign.Name {
				case "main::make_" + test.name:
					producer = foreign
				case "main::consume_" + test.name:
					consumer = foreign
				}
			}
			if producer.ResultType != test.typ || len(producer.ParameterTypes) != 0 ||
				consumer.ResultType != hir.TypeI32 || !slices.Equal(consumer.ParameterTypes, []hir.Type{test.typ}) {
				t.Fatalf("foreign signatures = %+v", program.ForeignFunctions)
			}
			var producerCall, consumerCall *Terminator
			for blockIndex := range program.Functions[0].Blocks {
				terminator := &program.Functions[0].Blocks[blockIndex].Terminator
				if terminator.Kind != TerminatorCall {
					continue
				}
				switch terminator.Callee {
				case producer.Symbol:
					producerCall = terminator
				case consumer.Symbol:
					consumerCall = terminator
				}
			}
			if producerCall == nil || consumerCall == nil || producerCall.DestinationLocal == NoLocal ||
				len(consumerCall.Arguments) != 1 || consumerCall.Arguments[0].Kind != OperandLocal ||
				consumerCall.Arguments[0].Local != producerCall.DestinationLocal {
				t.Fatalf("producer call = %+v, consumer call = %+v", producerCall, consumerCall)
			}
			if failures := Verify(program); len(failures) != 0 {
				t.Fatalf("Verify() failures = %+v", failures)
			}
		})
	}
}

func TestVerifyKeepsI32AndCIntForeignSignaturesDistinct(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\n@link_name(\"shared_symbol\")\nunsafe extern C fn with_c_int(value: c::int) -> i32;\nfn main() -> i32 { 0 }\n")
	conflict := program.ForeignFunctions[0]
	conflict.Symbol = hir.SymbolID(999)
	conflict.Name = "other::with_i32"
	conflict.ParameterTypes = []hir.Type{hir.TypeI32}
	conflict.Entity = "foreign:other.fy:0:1"
	program.ForeignFunctions = append(program.ForeignFunctions, conflict)
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != "V1001" || failures[0].Entity != conflict.Entity || !strings.Contains(failures[0].Detail, "conflicting signatures") {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestVerifyRejectsConflictingForeignSignaturesForOneLinkName(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\n@link_name(\"shared_symbol\")\nunsafe extern C fn with_value(value: i32) -> i32;\nfn main() -> i32 { 0 }\n")
	conflict := program.ForeignFunctions[0]
	conflict.Symbol = hir.SymbolID(999)
	conflict.Name = "other::without_value"
	conflict.ParameterTypes = nil
	conflict.Entity = "foreign:other.fy:0:1"
	program.ForeignFunctions = append(program.ForeignFunctions, conflict)
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != "V1001" || failures[0].Entity != conflict.Entity || !strings.Contains(failures[0].Detail, "conflicting signatures") {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestVerifyRejectsEmptyForeignLinkName(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\nunsafe extern C fn add_one(value: i32) -> i32;\nfn main() -> i32 { 0 }\n")
	entity := program.ForeignFunctions[0].Entity
	program.ForeignFunctions[0].LinkName = ""
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != "V1001" || failures[0].Entity != entity || !strings.Contains(failures[0].Detail, "incomplete") {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesRawPointerAndTargetScalarForeignOperands(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\nunsafe extern C fn foreign_probe(data: *u8, output: *mut u32, length: usize, marker: u8) -> u32;\nfn main() -> i32 { if unsafe { foreign_probe(null, null, 0, 7) } == 42 { 0 } else { 1 } }\n")
	foreign := program.ForeignFunctions[0]
	wantParameters := []hir.Type{
		hir.RawPointer(hir.TypeU8, false),
		hir.RawPointer(hir.TypeU32, true),
		hir.TypeUSize,
		hir.TypeU8,
	}
	if foreign.ResultType != hir.TypeU32 || !slices.Equal(foreign.ParameterTypes, wantParameters) {
		t.Fatalf("foreign signature = %+v", foreign)
	}
	var call *Terminator
	for blockIndex := range program.Functions[0].Blocks {
		terminator := &program.Functions[0].Blocks[blockIndex].Terminator
		if terminator.Kind == TerminatorCall {
			call = terminator
			break
		}
	}
	if call == nil || len(call.Arguments) != 4 {
		t.Fatalf("foreign call = %+v", call)
	}
	wantArguments := []Operand{
		{Kind: OperandNull, Type: hir.RawPointer(hir.TypeU8, false)},
		{Kind: OperandNull, Type: hir.RawPointer(hir.TypeU32, true)},
		{Kind: OperandUSize, U64: 0},
		{Kind: OperandU8, U64: 7},
	}
	if !slices.Equal(call.Arguments, wantArguments) {
		t.Fatalf("foreign arguments = %+v, want %+v", call.Arguments, wantArguments)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		terminator := &mutated.Functions[0].Blocks[blockIndex].Terminator
		if terminator.Kind == TerminatorCall {
			terminator.Arguments[0].Type = hir.TypeI32
			break
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() invalid null type = %+v", failures)
	}
}

func TestLowerPreservesRawAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@repr(C)
struct Packet { code: u32; }
unsafe extern C fn sink(output: *mut Packet) -> u32;
fn main() -> i32 {
    var output: Packet = Packet { code: 0 };
    if unsafe { sink(&mut output as *mut Packet) } == 0 { 0 } else { 1 }
}
`)
	function := program.Functions[0]
	var address *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationRawAddress {
				address = operation
			}
		}
	}
	wantPointer := hir.RawPointer(hir.Type("main::Packet"), true)
	if address == nil || address.Type != wantPointer || address.OwnerType != hir.Type("main::Packet") || address.Operator != "&mut" || address.Entity == "" || address.UnsafeScope.Source != "main.fy" || len(address.Operands) != 1 || address.Operands[0].Kind != OperandLocal {
		t.Fatalf("raw address = %+v", address)
	}
	if len(function.UnsafeOperations) != 2 {
		t.Fatalf("unsafe operations = %+v", function.UnsafeOperations)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationRawAddress {
				operation.UnsafeScope.StartByte = operation.Span.StartByte + 1
			}
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != "V1701" {
		t.Fatalf("Verify() invalid raw-address scope = %+v", failures)
	}
	mutated = Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationRawAddress {
				operation.Operator = "&"
			}
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() strengthened raw-address mutability = %+v", failures)
	}
}

func TestLowerPreservesCStringStaticStorageOperand(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let name: &CStr = c"fairy";
    42
}
`)
	function := program.Functions[0]
	var literal *Operand
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationAssign && len(operation.Operands) == 1 && operation.Operands[0].Kind == OperandCString {
				literal = &operation.Operands[0]
			}
		}
	}
	wantType := hir.SharedBorrow(hir.TypeCStr)
	if literal == nil || literal.Type != wantType || literal.StaticData != "fairy\x00" {
		t.Fatalf("C string operand = %+v, want type %q and exact terminated bytes", literal, wantType)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationAssign && len(operation.Operands) == 1 && operation.Operands[0].Kind == OperandCString {
				operation.Operands[0].StaticData = "fairy"
			}
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() unterminated C string failures = %+v", failures)
	}
}

func TestLowerPreservesCStringDataAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_is_fairy")
unsafe extern C fn is_fairy(value: *u8) -> u32;
fn main() -> i32 {
    if unsafe { is_fairy(c"fairy" as *u8) } == 1 { 42 } else { 1 }
}
`)
	function := program.Functions[0]
	var address *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationCStrDataAddress {
				address = operation
			}
		}
	}
	wantPointer := hir.RawPointer(hir.TypeU8, false)
	if address == nil || address.Type != wantPointer || address.OwnerType != hir.SharedBorrow(hir.TypeCStr) || address.Operator != "as" || address.Entity == "" || address.UnsafeScope.Source != "main.fy" || len(address.Operands) != 1 || address.Operands[0].Kind != OperandCString || address.Operands[0].StaticData != "fairy\x00" {
		t.Fatalf("C string data address = %+v", address)
	}
	if len(function.UnsafeOperations) != 2 {
		t.Fatalf("unsafe operations = %+v", function.UnsafeOperations)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesByteStringStaticStorageOperand(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let bytes: &[u8] = b"\x00A";
    42
}
`)
	function := program.Functions[0]
	var literal *Operand
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationAssign && len(operation.Operands) == 1 && operation.Operands[0].Kind == OperandByteString {
				literal = &operation.Operands[0]
			}
		}
	}
	wantType := hir.SharedBorrow(hir.TypeByteSlice)
	if literal == nil || literal.Type != wantType || literal.StaticData != "\x00A" {
		t.Fatalf("byte-string operand = %+v, want type %q and exact bytes", literal, wantType)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationAssign && len(operation.Operands) == 1 && operation.Operands[0].Kind == OperandByteString {
				operation.Operands[0].Type = hir.SharedBorrow(hir.TypeCStr)
			}
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() byte-string type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedImmutableByteViewIndex(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let value: u8 = b"\x00A"[1];
    42
}
`)
	function := &program.Functions[0]
	var check *Operation
	var load *Operation
	var checkedSwitch *Terminator
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			switch operation.Kind {
			case OperationViewIndexCheck:
				check = operation
			case OperationViewIndex:
				load = operation
			}
		}
		if block.Terminator.SwitchKind == "checked-view-index" {
			checkedSwitch = &block.Terminator
		}
	}
	wantOwner := hir.SharedBorrow(hir.TypeByteSlice)
	if check == nil || check.OwnerType != wantOwner || check.IndexType != hir.TypeUSize || check.Type != hir.TypeU8 || check.Operator != "[]" ||
		len(check.Operands) != 2 || check.Operands[0].Kind != OperandByteString || check.Operands[1].Kind != OperandUSize ||
		!slices.Equal(check.TrapCategories, []string{"bounds"}) {
		t.Fatalf("index check = %+v", check)
	}
	if load == nil || load.OwnerType != wantOwner || load.IndexType != hir.TypeUSize || load.Type != hir.TypeU8 || load.Operator != "[]" ||
		len(load.Operands) != 2 || len(load.TrapCategories) != 0 {
		t.Fatalf("index load = %+v", load)
	}
	if checkedSwitch == nil || checkedSwitch.Operator != "[]" || len(checkedSwitch.Successors) != 2 ||
		!slices.Equal(checkedSwitch.SwitchOperands, check.Operands) {
		t.Fatalf("index switch = %+v", checkedSwitch)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationViewIndexCheck {
				operation.TrapCategories = nil
			}
		}
	}
	failures := Verify(mutated)
	if len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() missing bounds marker failures = %+v", failures)
	}

	mutated = Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationViewIndex {
				operation.IndexType = hir.TypeI32
			}
		}
	}
	failures = Verify(mutated)
	if len(failures) == 0 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() index type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesValuePreservingIntegerCast(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 { b"*"[0] as i32 }
`)
	function := &program.Functions[0]
	var cast *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationScalarCast {
				cast = operation
			}
		}
	}
	if cast == nil || cast.OwnerType != hir.TypeU8 || cast.Type != hir.TypeI32 || cast.Operator != "as" ||
		cast.DestinationLocal == NoLocal || len(cast.Operands) != 1 || cast.Operands[0].Kind != OperandLocal {
		t.Fatalf("scalar cast = %+v", cast)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationScalarCast {
				operation.OwnerType = hir.TypeU32
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() cast source mutation failures = %+v", failures)
	}
}

func TestLowerPreservesTypedUnsignedOrderedComparison(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let size: usize = 42;
    if size > 41 { 42 } else { 1 }
}
`)
	function := &program.Functions[0]
	var comparison *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationBinary && operation.Operator == ">" {
				comparison = operation
			}
		}
	}
	if comparison == nil || comparison.OwnerType != hir.TypeUSize || comparison.Type != hir.TypeBool ||
		comparison.DestinationLocal == NoLocal || len(comparison.Operands) != 2 {
		t.Fatalf("typed comparison = %+v", comparison)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationBinary && operation.Operator == ">" {
				operation.OwnerType = hir.TypeBool
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() comparison owner mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedUSizeArithmeticType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let size: usize = 41;
    let next: usize = size + 1;
    if next == 42 { 42 } else { 1 }
}

`)
	function := &program.Functions[0]
	var marker *Operation
	var result *Operation
	var check *Terminator
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			if operation.Kind == OperationArithmeticCheck && operation.Operator == "+" {
				marker = operation
				check = &block.Terminator
			}
			if operation.Kind == OperationBinary && operation.Operator == "+" {
				result = operation
			}
		}
	}
	if marker == nil || result == nil || check == nil || marker.OwnerType != hir.TypeUSize || marker.Type != hir.TypeUSize ||
		result.OwnerType != hir.TypeUSize || result.Type != hir.TypeUSize || check.SwitchKind != "checked-arithmetic" ||
		check.SwitchType != hir.TypeUSize {
		t.Fatalf("marker = %+v, result = %+v, check = %+v", marker, result, check)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationArithmeticCheck && operation.Operator == "+" {
				operation.OwnerType = hir.TypeU32
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() arithmetic marker type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedUSizeDivisionType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let numerator: usize = 84;
    let quotient: usize = numerator / 2;
    if quotient == 42 { 42 } else { 1 }
}
`)
	function := &program.Functions[0]
	var marker *Operation
	var result *Operation
	var check *Terminator
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			if operation.Kind == OperationCheckedDivision {
				marker = operation
				check = &block.Terminator
			}
			if operation.Kind == OperationBinary && operation.Operator == "/" {
				result = operation
			}
		}
	}
	if marker == nil || result == nil || check == nil || marker.OwnerType != hir.TypeUSize || marker.Type != hir.TypeUSize ||
		!slices.Equal(marker.TrapCategories, []string{"division-by-zero"}) || result.OwnerType != hir.TypeUSize ||
		result.Type != hir.TypeUSize || check.SwitchKind != "checked-division" || check.SwitchType != hir.TypeUSize {
		t.Fatalf("marker = %+v, result = %+v, check = %+v", marker, result, check)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationCheckedDivision {
				operation.OwnerType = hir.TypeU32
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() division marker type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedUSizeRightShiftType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let value: usize = 84;
    let shifted: usize = value >> 1;
    if shifted == 42 { 42 } else { 1 }
}
`)
	function := &program.Functions[0]
	var marker *Operation
	var result *Operation
	var check *Terminator
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			if operation.Kind == OperationArithmeticCheck && operation.Operator == ">>" {
				marker = operation
				check = &block.Terminator
			}
			if operation.Kind == OperationBinary && operation.Operator == ">>" {
				result = operation
			}
		}
	}
	if marker == nil || result == nil || check == nil || marker.OwnerType != hir.TypeUSize || marker.Type != hir.TypeUSize ||
		!slices.Equal(marker.TrapCategories, []string{"invalid-shift"}) || result.OwnerType != hir.TypeUSize ||
		result.Type != hir.TypeUSize || check.SwitchKind != "checked-shift" || check.SwitchType != hir.TypeUSize {
		t.Fatalf("marker = %+v, result = %+v, check = %+v", marker, result, check)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationArithmeticCheck && operation.Operator == ">>" {
				operation.OwnerType = hir.TypeU32
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() shift marker type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedUSizeLeftShiftTypeAndTrapOrder(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let value: usize = 21;
    let shifted: usize = value << 1;
    if shifted == 42 { 42 } else { 1 }
}
`)
	function := &program.Functions[0]
	var marker *Operation
	var result *Operation
	var check *Terminator
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			if operation.Kind == OperationArithmeticCheck && operation.Operator == "<<" {
				marker = operation
				check = &block.Terminator
			}
			if operation.Kind == OperationBinary && operation.Operator == "<<" {
				result = operation
			}
		}
	}
	if marker == nil || result == nil || check == nil || marker.OwnerType != hir.TypeUSize || marker.Type != hir.TypeUSize ||
		!slices.Equal(marker.TrapCategories, []string{"invalid-shift", "integer-overflow"}) ||
		result.OwnerType != hir.TypeUSize || result.Type != hir.TypeUSize || check.SwitchKind != "checked-shift" ||
		check.SwitchType != hir.TypeUSize || check.Operator != "<<" {
		t.Fatalf("marker = %+v, result = %+v, check = %+v", marker, result, check)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationArithmeticCheck && operation.Operator == "<<" {
				operation.TrapCategories = []string{"integer-overflow", "invalid-shift"}
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() left-shift trap-order mutation failures = %+v", failures)
	}
}

func TestLowerPreservesUnsignedBitwiseNotType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let input: u32 = 4294967253;
    let value: u32 = ~input;
    if value == 42 { 42 } else { 1 }
}
`)
	function := &program.Functions[0]
	var bitwiseNot *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationUnary && operation.Operator == "~" {
				bitwiseNot = operation
			}
		}
	}
	if bitwiseNot == nil || bitwiseNot.OwnerType != hir.TypeU32 || bitwiseNot.Type != hir.TypeU32 ||
		bitwiseNot.DestinationLocal == NoLocal || len(bitwiseNot.Operands) != 1 {
		t.Fatalf("bitwise not = %+v", bitwiseNot)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationUnary && operation.Operator == "~" {
				operation.Type = hir.TypeUSize
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() bitwise-not type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesByteStringDataAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_is_fairy_bytes")
unsafe extern C fn is_fairy_bytes(value: *u8) -> u32;
fn main() -> i32 {
    if unsafe { is_fairy_bytes(b"fairy" as *u8) } == 1 { 42 } else { 1 }
}
`)
	function := program.Functions[0]
	var address *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationByteSliceDataAddress {
				address = operation
			}
		}
	}
	wantPointer := hir.RawPointer(hir.TypeU8, false)
	if address == nil || address.Type != wantPointer || address.OwnerType != hir.SharedBorrow(hir.TypeByteSlice) || address.Operator != "as" || address.Entity == "" || address.UnsafeScope.Source != "main.fy" || len(address.Operands) != 1 || address.Operands[0].Kind != OperandByteString || address.Operands[0].StaticData != "fairy" {
		t.Fatalf("byte-string data address = %+v", address)
	}
	if len(function.UnsafeOperations) != 2 {
		t.Fatalf("unsafe operations = %+v", function.UnsafeOperations)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestCloneDeepCopiesResourceDefinitions(t *testing.T) {
	t.Parallel()
	original := Program{ResourceTypes: []ResourceType{{
		Type: hir.Type("main::Token"), DropKind: hir.ResourceDropEmpty,
		Fields: []ResourceField{{Name: "value", Type: hir.TypeI32}},
	}}}
	cloned := Clone(original)
	cloned.ResourceTypes[0].Fields[0].Name = "changed"
	if original.ResourceTypes[0].Fields[0].Name != "value" {
		t.Fatalf("Clone() aliased resource fields: original = %+v, clone = %+v", original, cloned)
	}
}

func TestCloneDeepCopiesStructDefinitions(t *testing.T) {
	t.Parallel()
	original := Program{StructTypes: []StructType{{
		Type: hir.Type("main::Packet"), Name: "Packet", ReprC: true,
		Fields: []StructField{{Name: "code", Type: hir.TypeU32}},
	}}}
	cloned := Clone(original)
	cloned.StructTypes[0].Fields[0].Name = "changed"
	if original.StructTypes[0].Fields[0].Name != "code" {
		t.Fatalf("Clone() aliased struct fields: original = %+v, clone = %+v", original, cloned)
	}
}

func TestLowerPropagateRebuildsDeclaredResultType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn read(input: Result<Token, i32>) -> Result<i32, i32> {
    let token: Token = input?;
    Result::Ok(token.value)
}
fn main() -> i32 {
    match read(Result::Ok(Token { value: 42 })) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    }
}
`)
	wantResultType := hir.Result(hir.TypeI32, hir.TypeI32)
	var read *Function
	for index := range program.Functions {
		if program.Functions[index].Name == "main::read" {
			read = &program.Functions[index]
			break
		}
	}
	if read == nil {
		t.Fatal("lowered program has no main::read")
	}
	found := false
	for _, block := range read.Blocks {
		for _, operation := range block.Operations {
			if operation.Kind != OperationResultConstruct || operation.Variant != "err" {
				continue
			}
			if operation.Type != wantResultType || len(operation.Operands) != 1 {
				t.Fatalf("propagated Result construction = %+v", operation)
			}
			if block.Terminator.Kind != TerminatorReturn || block.Terminator.ReturnValue != (Operand{Kind: OperandLocal, Local: operation.DestinationLocal}) {
				t.Fatalf("propagated Result return = %+v, construction = %+v", block.Terminator, operation)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("propagate error path did not rebuild the declared Result type")
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerRestrictedEnumConstructionPayloadAndSwitch(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
enum Message {
    Empty;
    Pair { left: i32; right: i32; };
}
fn main() -> i32 {
    let message: Message = Message::Pair { left: 20, right: 22 };
    match message {
        Message::Empty => 0,
        Message::Pair { left: first, right: second } => first + second,
    }
}
`)
	if len(program.EnumTypes) != 1 || program.EnumTypes[0].Type != hir.Type("main::Message") || len(program.EnumTypes[0].Variants) != 2 {
		t.Fatalf("enum types = %+v", program.EnumTypes)
	}
	function := program.Functions[0]
	constructs := 0
	payloads := 0
	switches := 0
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			switch operation.Kind {
			case OperationEnumConstruct:
				constructs++
				if operation.Variant != "Pair" || !slices.Equal(operation.Fields, []string{"left", "right"}) || len(operation.Operands) != 2 {
					t.Fatalf("enum construct = %+v", operation)
				}
			case OperationEnumPayload:
				payloads++
				if operation.OwnerType != hir.Type("main::Message") || operation.Variant != "Pair" || operation.Type != hir.TypeI32 {
					t.Fatalf("enum payload = %+v", operation)
				}
			}
		}
		if block.Terminator.Kind == TerminatorSwitch && block.Terminator.SwitchKind == "enum" {
			switches++
			if block.Terminator.SwitchType != hir.Type("main::Message") || !slices.Equal(block.Terminator.SwitchVariants, []string{"Empty", "Pair"}) || len(block.Terminator.Successors) != 2 {
				t.Fatalf("enum switch = %+v", block.Terminator)
			}
		}
	}
	if constructs != 1 || payloads != 2 || switches != 1 {
		t.Fatalf("constructs=%d payloads=%d switches=%d", constructs, payloads, switches)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestVerifyRejectsMutatedRestrictedEnumOperations(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
enum Message { Empty; Value { amount: i32; }; }
fn main() -> i32 {
    let message: Message = Message::Value { amount: 42 };
    match message { Message::Empty => 0, Message::Value { amount: value } => value, }
}
`)
	mutations := []struct {
		name   string
		mutate func(*Program)
	}{
		{
			name: "construct-variant",
			mutate: func(value *Program) {
				for blockIndex := range value.Functions[0].Blocks {
					for operationIndex := range value.Functions[0].Blocks[blockIndex].Operations {
						operation := &value.Functions[0].Blocks[blockIndex].Operations[operationIndex]
						if operation.Kind == OperationEnumConstruct {
							operation.Variant = "Forged"
							return
						}
					}
				}
			},
		},
		{
			name: "payload-field",
			mutate: func(value *Program) {
				for blockIndex := range value.Functions[0].Blocks {
					for operationIndex := range value.Functions[0].Blocks[blockIndex].Operations {
						operation := &value.Functions[0].Blocks[blockIndex].Operations[operationIndex]
						if operation.Kind == OperationEnumPayload {
							operation.Field = "forged"
							return
						}
					}
				}
			},
		},
		{
			name: "switch-table",
			mutate: func(value *Program) {
				for blockIndex := range value.Functions[0].Blocks {
					terminator := &value.Functions[0].Blocks[blockIndex].Terminator
					if terminator.Kind == TerminatorSwitch && terminator.SwitchKind == "enum" {
						terminator.SwitchVariants[0] = "Forged"
						return
					}
				}
			},
		},
	}
	for _, mutation := range mutations {
		mutation := mutation
		t.Run(mutation.name, func(t *testing.T) {
			t.Parallel()
			mutated := Clone(program)
			mutation.mutate(&mutated)
			failures := Verify(mutated)
			if len(failures) == 0 || failures[0].Code != "V1001" {
				t.Fatalf("Verify() failures = %+v, want V1001", failures)
			}
		})
	}
}

func TestLowerPreservesUTF8StringStaticStorageOperand(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn main() -> i32 {
    let text: &str = "λ";
    42
}
`)
	function := program.Functions[0]
	var literal *Operand
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationAssign && len(operation.Operands) == 1 && operation.Operands[0].Kind == OperandString {
				literal = &operation.Operands[0]
			}
		}
	}
	wantType := hir.SharedBorrow(hir.TypeStr)
	if literal == nil || literal.Type != wantType || literal.StaticData != "λ" {
		t.Fatalf("UTF-8 string operand = %+v, want type %q and exact bytes", literal, wantType)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationAssign && len(operation.Operands) == 1 && operation.Operands[0].Kind == OperandString {
				operation.Operands[0].Type = hir.SharedBorrow(hir.TypeCStr)
			}
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != "V1001" {
		t.Fatalf("Verify() UTF-8 string type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesUTF8StringDataAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_is_fairy_text")
unsafe extern C fn is_fairy_text(value: *u8) -> u32;
fn main() -> i32 {
    let text: &str = "fairy";
    if unsafe { is_fairy_text(text as *u8) } == 1 { 42 } else { 1 }
}
`)
	function := program.Functions[0]
	var address *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationStrDataAddress {
				address = operation
			}
		}
	}
	wantPointer := hir.RawPointer(hir.TypeU8, false)
	if address == nil || address.Type != wantPointer || address.OwnerType != hir.SharedBorrow(hir.TypeStr) || address.Operator != "as" || address.Entity == "" || address.UnsafeScope.Source != "main.fy" || len(address.Operands) != 1 || address.Operands[0].Kind != OperandLocal {
		t.Fatalf("UTF-8 string data address = %+v", address)
	}
	if len(function.UnsafeOperations) != 2 {
		t.Fatalf("unsafe operations = %+v", function.UnsafeOperations)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func lowerTestSource(t *testing.T, text string) Program {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
		t.Fatalf("AnalyzeSyntax() diagnostics = %+v, AST = %+v", parsed.Tree.Diagnostics, parsed.AST)
	}
	result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("Analyze() result = %+v", result)
	}
	program, err := Lower(result.HIR)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	return program
}
