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
    i32 left;
    bool ready;
    cleanup {}
}
fn i32 main() {
    let Pair pair = Pair { .left = 42, .ready = true };
    return pair.left;
}
`)
	if len(program.ResourceTypes) != 1 {
		t.Fatalf("resource types = %+v", program.ResourceTypes)
	}
	definition := program.ResourceTypes[0]
	if definition.Type != hir.ParseType("main::Pair") || definition.Entity != "resource-type:main-pair" || definition.DropKind != hir.ResourceDropEmpty || len(definition.Fields) != 2 {
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

func TestLowerVerifiesBorrowedResourceFieldReads(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Token {
    u32 value;
    cleanup {}
}
fn u32 read(ref Token token) { return token.value; }
fn u32 read_mut(ref mut Token token) { return token.value; }
fn i32 main() {
    var Token token = Token { .value = 42 };
    return if (read(ref token) == 42 && read_mut(ref mut token) == 42) { 42 } else { 1 };
}
`)
	owners := make(map[hir.Type]bool)
	for _, function := range program.Functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind == OperationFieldRead {
					owners[operation.OwnerType] = true
				}
			}
		}
	}
	if !owners[hir.SharedBorrow(hir.ParseType("main::Token"))] || !owners[hir.MutableBorrow(hir.ParseType("main::Token"))] {
		t.Fatalf("borrowed field-read owner types = %v", owners)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerLexicalRegionRetainsArenaOwnerOperand(t *testing.T) {
	t.Parallel()
	program := lowerTestSourceAtPath(t, "std/alloc.fy", `module std::alloc;
pub resource Arena {
    i32 opaque;
    cleanup {}
}
pub struct AllocError {
    u32 status;
}
fn i32 main() {
    var Arena arena = Arena { .opaque = 0 };
    region scratch in ref mut arena {
        discard scratch.place(42);
    }
    return 42;
}
`)
	var enter *Operation
	var place *Operation
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			operations := program.Functions[functionIndex].Blocks[blockIndex].Operations
			for operationIndex := range operations {
				if operations[operationIndex].Kind == OperationRegionEnter {
					enter = &operations[operationIndex]
				}
				if operations[operationIndex].Kind == OperationRegionPlace {
					place = &operations[operationIndex]
				}
			}
		}
	}
	arenaType := hir.ParseType("std::alloc::Arena")
	if enter == nil || len(enter.Operands) != 1 || enter.Operands[0].Kind != OperandLocal ||
		enter.OwnerType != arenaType || enter.Type != hir.MutableBorrow(arenaType) || enter.Operator != "&mut" {
		t.Fatalf("region enter = %+v", enter)
	}
	wantPlaceType := hir.Result(hir.SharedBorrow(hir.TypeI32), hir.ParseType("std::alloc::AllocError"))
	if place == nil || place.OwnerType != hir.TypeI32 || place.Type != wantPlaceType {
		t.Fatalf("region place = %+v", place)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			operations := mutated.Functions[functionIndex].Blocks[blockIndex].Operations
			for operationIndex := range operations {
				if operations[operationIndex].Kind == OperationRegionEnter {
					operations[operationIndex].Operands = nil
				}
			}
		}
	}
	failures := Verify(mutated)
	if len(failures) != 1 || failures[0].Code != "V1403" || failures[0].Detail != "region enter shape is invalid" {
		t.Fatalf("Verify(mutated) failures = %+v", failures)
	}
}

func TestLowerPreservesResourceDropEffectsOnDefinitionAndGlue(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@effects(filesystem)
resource Token {
    i32 value;
    cleanup { let i32 marker = this.value; }
}
@effects(filesystem)
fn i32 main() {
    let Token token = Token { .value = 42 };
    return 0;
}
`)
	if len(program.ResourceTypes) != 1 || !slices.Equal(program.ResourceTypes[0].DropEffects, []hir.Effect{hir.EffectFilesystem}) {
		t.Fatalf("resource types = %+v", program.ResourceTypes)
	}
	var glue *Function
	for index := range program.Functions {
		if program.Functions[index].DropGlueFor == program.ResourceTypes[0].Type {
			glue = &program.Functions[index]
		}
	}
	if glue == nil || !slices.Equal(glue.Effects, program.ResourceTypes[0].DropEffects) {
		t.Fatalf("drop glue = %+v", glue)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesReprCStructDefinitionConstructionAndFieldRead(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@repr(C)
struct Packet { u32 code; usize length; }
fn i32 main() {
    let Packet packet = Packet { .code = 42, .length = 0 };
    return if (packet.code == 42) { 0 } else { 1 };
}
`)
	if len(program.StructTypes) != 1 {
		t.Fatalf("struct types = %+v", program.StructTypes)
	}
	definition := program.StructTypes[0]
	if definition.Type != hir.ParseType("main::Packet") || definition.Name != "Packet" || definition.Entity != "struct-type:main-packet" || !definition.ReprC || len(definition.Fields) != 2 {
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

func TestLowerRejectsNonEmptyResourceWithoutDropGlue(t *testing.T) {
	t.Parallel()
	_, err := Lower(hir.Package{Modules: []hir.Module{{
		Path: "main",
		Resources: []hir.Resource{{
			Name: "Token", Type: hir.ParseType("main::Token"), DropKind: hir.ResourceDropNonEmpty,
			DropSymbol: hir.NoSymbol,
			Fields:     []hir.ResourceField{{Name: "value", Type: hir.TypeI32}},
		}},
	}}})
	if err == nil || !strings.Contains(err.Error(), "inconsistent drop symbol") {
		t.Fatalf("Lower() error = %v", err)
	}
}

func TestLowerPreservesForeignCallABIAndUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\n@link_name(\"fairyc_fixture_add_one\")\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return unsafe { add_one(41) }; }\n")
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
unsafe extern C fn c::int make_c_int() ;
@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn i32 consume_c_int(c::int value) ;
fn i32 main() { return unsafe { consume_c_int(make_c_int()) }; }
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
unsafe extern C fn c::size_t make_c_size_t() ;
@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn i32 consume_c_size_t(c::size_t value) ;
fn i32 main() { return unsafe { consume_c_size_t(make_c_size_t()) }; }
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
unsafe extern C fn c::long make_c_long() ;
@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn i32 consume_c_long(c::long value) ;
fn i32 main() { return unsafe { consume_c_long(make_c_long()) }; }
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
unsafe extern C fn c::ssize_t make_c_ssize_t() ;
@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn i32 consume_c_ssize_t(c::ssize_t value) ;
fn i32 main() { return unsafe { consume_c_ssize_t(make_c_ssize_t()) }; }
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
unsafe extern C fn %[2]s make_%[1]s();
unsafe extern C fn i32 consume_%[1]s(%[2]s value);
fn i32 main() { return unsafe { consume_%[1]s(make_%[1]s()) }; }
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
	program := lowerTestSource(t, "module main;\n@link_name(\"shared_symbol\")\nunsafe extern C fn i32 with_c_int(c::int value) ;\nfn i32 main() { return 0; }\n")
	conflict := program.ForeignFunctions[0]
	conflict.Symbol = hir.SymbolID(999)
	conflict.Name = "other::with_i32"
	conflict.ParameterTypes = []hir.Type{hir.TypeI32}
	conflict.Entity = "foreign:other.fy:0:1"
	program.ForeignFunctions = append(program.ForeignFunctions, conflict)
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeForeignDeclaration || failures[0].Entity != conflict.Entity || !strings.Contains(failures[0].Detail, "conflicting signatures") {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestVerifyRejectsConflictingForeignSignaturesForOneLinkName(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\n@link_name(\"shared_symbol\")\nunsafe extern C fn i32 with_value(i32 value) ;\nfn i32 main() { return 0; }\n")
	conflict := program.ForeignFunctions[0]
	conflict.Symbol = hir.SymbolID(999)
	conflict.Name = "other::without_value"
	conflict.ParameterTypes = nil
	conflict.Entity = "foreign:other.fy:0:1"
	program.ForeignFunctions = append(program.ForeignFunctions, conflict)
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeForeignDeclaration || failures[0].Entity != conflict.Entity || !strings.Contains(failures[0].Detail, "conflicting signatures") {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestVerifyRejectsEmptyForeignLinkName(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\nunsafe extern C fn i32 add_one(i32 value) ;\nfn i32 main() { return 0; }\n")
	entity := program.ForeignFunctions[0].Entity
	program.ForeignFunctions[0].LinkName = ""
	failures := Verify(program)
	if len(failures) != 1 || failures[0].Code != CodeForeignDeclaration || failures[0].Entity != entity || !strings.Contains(failures[0].Detail, "incomplete") {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesRawPointerAndTargetScalarForeignOperands(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\nunsafe extern C fn u32 foreign_probe(ptr u8 data, ptr mut u32 output, usize length, u8 marker) ;\nfn i32 main() { return if (unsafe { foreign_probe(null, null, 0, 7) } == 42) { 0 } else { 1 }; }\n")
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
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != CodeOperandShape {
		t.Fatalf("Verify() invalid null type = %+v", failures)
	}
}

func TestLowerPreservesRawAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@repr(C)
struct Packet { u32 code; }
unsafe extern C fn u32 sink(ptr mut Packet output) ;
fn i32 main() {
    var Packet output = Packet { .code = 0 };
    return if (unsafe { sink(ref mut output as ptr mut Packet) } == 0) { 0 } else { 1 };
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
	wantPointer := hir.RawPointer(hir.ParseType("main::Packet"), true)
	if address == nil || address.Type != wantPointer || address.OwnerType != hir.ParseType("main::Packet") || address.Operator != "&mut" || address.Entity == "" || address.UnsafeScope.Source != "main.fy" || len(address.Operands) != 1 || address.Operands[0].Kind != OperandLocal {
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
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() strengthened raw-address mutability = %+v", failures)
	}
}

func TestLowerPreservesOpaqueRawAddressOwnerType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@repr(C)
struct Packet { u32 code; }
unsafe extern C fn u32 sink(ptr mut void output) ;
fn i32 main() {
    var Packet output = Packet { .code = 0 };
    return if (unsafe { sink(ref mut output as ptr mut void) } == 0) { 42 } else { 1 };
}
`)
	var address *Operation
	for blockIndex := range program.Functions[0].Blocks {
		for operationIndex := range program.Functions[0].Blocks[blockIndex].Operations {
			operation := &program.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationRawAddress {
				address = operation
			}
		}
	}
	wantOwner := hir.ParseType("main::Packet")
	if address == nil || address.Type != hir.RawPointer(hir.TypeVoid, true) || address.OwnerType != wantOwner || address.Operator != "&mut" {
		t.Fatalf("opaque raw address = %+v", address)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationRawAddress {
				operation.Type = hir.RawPointer(hir.TypeU32, true)
			}
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() mismatched opaque raw address = %+v", failures)
	}
}

func TestLowerPreservesSafeBorrowToOpaquePointer(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Token { u32 value; cleanup {} }
unsafe extern C fn u32 sink(ptr mut void output) ;
fn u32 inspect(ref mut Token token) { return unsafe { sink(token as ptr mut void) }; }
fn i32 main() {
    var Token token = Token { .value = 42 };
    return if (inspect(ref mut token) == 0) { 42 } else { 1 };
}
`)
	var borrow *Operation
	var raw *Operation
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			for operationIndex := range program.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &program.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				switch operation.Kind {
				case OperationUnary:
					if operation.Operator == "&mut" && operation.OwnerType == hir.ParseType("main::Token") {
						borrow = operation
					}
				case OperationRawBorrowPointer:
					raw = operation
				}
			}
		}
	}
	wantBorrow := hir.MutableBorrow(hir.ParseType("main::Token"))
	if borrow == nil || borrow.Type != wantBorrow || len(borrow.Operands) != 1 || borrow.Operands[0].Kind != OperandLocal {
		t.Fatalf("safe resource borrow = %+v", borrow)
	}
	if raw == nil || raw.OwnerType != wantBorrow || raw.Type != hir.RawPointer(hir.TypeVoid, true) || raw.Operator != "as" || raw.Entity == "" {
		t.Fatalf("raw borrow pointer = %+v", raw)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationRawBorrowPointer {
					operation.Type = hir.RawPointer(hir.TypeU32, true)
				}
			}
		}
	}
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() mismatched raw borrow pointer = %+v", failures)
	}
}

func TestLowerPreservesCStringStaticStorageOperand(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let ref CStr name = c"fairy";
    return 42;
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
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != CodeOperandShape {
		t.Fatalf("Verify() unterminated C string failures = %+v", failures)
	}
}

func TestLowerShortCircuitMaterializesTypedBoolForUnaryNot(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let bool ready = true && false;
    return if (!ready) { 42 } else { 1 };
}
`)
	typedShortValue := false
	for _, block := range program.Functions[0].Blocks {
		for _, operation := range block.Operations {
			if operation.Kind == OperationAssign && len(operation.Operands) == 1 &&
				operation.Operands[0].Kind == OperandI32 && operation.Operands[0].Type == hir.TypeBool {
				typedShortValue = true
			}
		}
	}
	if !typedShortValue {
		t.Fatal("short-circuit lowering did not materialize a typed bool literal")
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesCStringDataAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_is_fairy")
unsafe extern C fn u32 is_fairy(ptr u8 value) ;
fn i32 main() {
    return if (unsafe { is_fairy(c"fairy" as ptr u8) } == 1) { 42 } else { 1 };
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
fn i32 main() {
    let ref u8[] bytes = b"\x00A";
    return 42;
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
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != CodeOperandShape {
		t.Fatalf("Verify() byte-string type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesBorrowedViewEmptiness(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let bool empty = b"".is_empty;
    return if (empty) { 42 } else { 1 };
}
`)
	var operation *Operation
	for blockIndex := range program.Functions[0].Blocks {
		for operationIndex := range program.Functions[0].Blocks[blockIndex].Operations {
			candidate := &program.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if candidate.Kind == OperationViewEmpty {
				operation = candidate
			}
		}
	}
	wantOwner := hir.SharedBorrow(hir.TypeByteSlice)
	if operation == nil || operation.OwnerType != wantOwner || operation.Type != hir.TypeBool ||
		operation.Field != "is_empty" || operation.DestinationLocal == NoLocal || len(operation.Operands) != 1 {
		t.Fatalf("view emptiness = %+v", operation)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			candidate := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if candidate.Kind == OperationViewEmpty {
				candidate.Type = hir.TypeUSize
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() view-empty type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedImmutableByteViewIndex(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let u8 value = b"\x00A"[1];
    return 42;
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
	if len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() index type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedImmutableByteViewSlice(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let ref u8[] bytes = b"fairy"[1..4];
    return 42;
}
`)
	function := &program.Functions[0]
	var check *Operation
	var slice *Operation
	var checkedSwitch *Terminator
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			switch operation.Kind {
			case OperationViewSliceCheck:
				check = operation
			case OperationViewSlice:
				slice = operation
			}
		}
		if block.Terminator.SwitchKind == "checked-view-slice" {
			checkedSwitch = &block.Terminator
		}
	}
	wantView := hir.SharedBorrow(hir.TypeByteSlice)
	if check == nil || check.OwnerType != wantView || check.IndexType != hir.TypeUSize || check.Type != wantView || check.Operator != ".." ||
		len(check.Operands) != 3 || check.Operands[0].Kind != OperandByteString ||
		check.Operands[1].Kind != OperandUSize || check.Operands[2].Kind != OperandUSize ||
		!slices.Equal(check.TrapCategories, []string{"bounds"}) {
		t.Fatalf("slice check = %+v", check)
	}
	if slice == nil || slice.OwnerType != wantView || slice.IndexType != hir.TypeUSize || slice.Type != wantView || slice.Operator != ".." ||
		len(slice.Operands) != 3 || len(slice.TrapCategories) != 0 {
		t.Fatalf("slice operation = %+v", slice)
	}
	if checkedSwitch == nil || checkedSwitch.Operator != ".." || len(checkedSwitch.Successors) != 2 ||
		!slices.Equal(checkedSwitch.SwitchOperands, check.Operands) {
		t.Fatalf("slice switch = %+v", checkedSwitch)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationViewSliceCheck {
				operation.TrapCategories = nil
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() missing slice bounds marker failures = %+v", failures)
	}

	mutated = Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationViewSlice {
				operation.Operands[1], operation.Operands[2] = operation.Operands[2], operation.Operands[1]
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() reordered slice bounds failures = %+v", failures)
	}
}

func TestLowerNormalizesOmittedImmutableByteViewSliceBounds(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let ref u8[] bytes = b"fairy"[..];
    return if (bytes.len == 5) { 42 } else { 1 };
}
`)
	function := &program.Functions[0]
	var length *Operation
	var check *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationViewLength && length == nil {
				length = operation
			}
			if operation.Kind == OperationViewSliceCheck {
				check = operation
			}
		}
	}
	if length == nil || len(length.Operands) != 1 || length.Operands[0].Kind != OperandByteString {
		t.Fatalf("omitted end length = %+v", length)
	}
	if check == nil || len(check.Operands) != 3 || check.Operands[1].Kind != OperandUSize || check.Operands[1].U64 != 0 ||
		check.Operands[2].Kind != OperandLocal || check.Operands[2].Local != length.DestinationLocal {
		t.Fatalf("normalized slice check = %+v, length = %+v", check, length)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerOmittedSliceEndDoesNotReevaluateReceiver(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn ref u8[] identity(ref u8[] bytes) { return bytes; }
fn i32 main() {
    let ref u8[] bytes = identity(b"fairy")[1..];
    return if (bytes.len == 4) { 42 } else { 1 };
}
`)
	var function *Function
	for index := range program.Functions {
		if program.Functions[index].Name == "main::main" {
			function = &program.Functions[index]
			break
		}
	}
	if function == nil {
		t.Fatal("main function is missing")
	}
	if len(function.Calls) != 1 {
		t.Fatalf("main calls = %+v, want one receiver evaluation", function.Calls)
	}
	var length *Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationViewLength {
				length = operation
				break
			}
		}
	}
	if length == nil || len(length.Operands) != 1 || length.Operands[0].Kind != OperandLocal {
		t.Fatalf("omitted-end length = %+v", length)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesValuePreservingIntegerCast(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let i32 value = b"*"[0] as i32;
    let c::int foreign = value as c::int;
    return foreign as i32;
}
`)
	function := &program.Functions[0]
	var casts []*Operation
	for blockIndex := range function.Blocks {
		for operationIndex := range function.Blocks[blockIndex].Operations {
			operation := &function.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationScalarCast {
				casts = append(casts, operation)
			}
		}
	}
	if len(casts) != 3 || casts[0].OwnerType != hir.TypeU8 || casts[0].Type != hir.TypeI32 ||
		casts[1].OwnerType != hir.TypeI32 || casts[1].Type != hir.TypeCInt ||
		casts[2].OwnerType != hir.TypeCInt || casts[2].Type != hir.TypeI32 {
		t.Fatalf("scalar casts = %+v", casts)
	}
	for _, cast := range casts {
		if cast.Operator != "as" || cast.DestinationLocal == NoLocal || len(cast.Operands) != 1 || cast.Operands[0].Kind != OperandLocal {
			t.Fatalf("scalar cast = %+v", cast)
		}
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	mutatedOne := false
	for blockIndex := range mutated.Functions[0].Blocks {
		for operationIndex := range mutated.Functions[0].Blocks[blockIndex].Operations {
			operation := &mutated.Functions[0].Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationScalarCast && !mutatedOne {
				operation.OwnerType = hir.TypeU32
				mutatedOne = true
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() cast source mutation failures = %+v", failures)
	}
}

func TestLowerPreservesTypedUnsignedOrderedComparison(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let usize size = 42;
    return if (size > 41) { 42 } else { 1 };
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
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() comparison owner mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedUSizeArithmeticType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let usize size = 41;
    let usize next = size + 1;
    return if (next == 42) { 42 } else { 1 };
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
fn i32 main() {
    let usize numerator = 84;
    let usize quotient = numerator / 2;
    return if (quotient == 42) { 42 } else { 1 };
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
fn i32 main() {
    let usize value = 84;
    let usize shifted = value >> 1;
    return if (shifted == 42) { 42 } else { 1 };
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
fn i32 main() {
    let usize value = 21;
    let usize shifted = value << 1;
    return if (shifted == 42) { 42 } else { 1 };
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
fn i32 main() {
    let u32 input = 4294967253;
    let u32 value = ~input;
    return if (value == 42) { 42 } else { 1 };
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
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() bitwise-not type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesByteStringDataAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_is_fairy_bytes")
unsafe extern C fn u32 is_fairy_bytes(ptr u8 value) ;
fn i32 main() {
    return if (unsafe { is_fairy_bytes(b"fairy" as ptr u8) } == 1) { 42 } else { 1 };
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
		Type: hir.ParseType("main::Token"), DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol,
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
		Type: hir.ParseType("main::Packet"), Name: "Packet", ReprC: true,
		Fields: []StructField{{Name: "code", Type: hir.TypeU32}},
	}}}
	cloned := Clone(original)
	cloned.StructTypes[0].Fields[0].Name = "changed"
	if original.StructTypes[0].Fields[0].Name != "code" {
		t.Fatalf("Clone() aliased struct fields: original = %+v, clone = %+v", original, cloned)
	}
}

func TestLowerResultIsOkDropsEitherOwnedPayload(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Token { i32 value; cleanup {} }
fn bool success(Result<Token, Token> input) { return input.is_ok; }
`)
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	var payloads, drops int
	for _, function := range program.Functions {
		if function.Name != "main::success" {
			continue
		}
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind == OperationResultPayload {
					payloads++
				}
				if operation.Kind == OperationDrop {
					drops++
				}
			}
		}
	}
	if payloads != 2 || drops != 2 {
		t.Fatalf("result is_ok payloads=%d drops=%d, want 2 each", payloads, drops)
	}
}

func TestLowerPropagateRebuildsDeclaredResultType(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Token {
    i32 value;
    cleanup {}
}
fn Result<i32, i32> read(Result<Token, i32> input) {
    let Token token = input?;
    return .ok(token.value);
}
fn i32 main() {
    return switch (read(.ok(Token { .value = 42 }))) {
        case .ok(value) : { value }
        case .err(error) : { -error }
    };
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
    Pair { i32 left; i32 right; };
}
fn i32 main() {
    let Message message = .Pair { .left = 20, .right = 22 };
    return switch (message) {
        case .Empty : { 0 }
        case .Pair { .left = first, .right = second } : { first + second }
    };
}
`)
	if len(program.EnumTypes) != 1 || program.EnumTypes[0].Type != hir.ParseType("main::Message") || len(program.EnumTypes[0].Variants) != 2 {
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
				if operation.OwnerType != hir.ParseType("main::Message") || operation.Variant != "Pair" || operation.Type != hir.TypeI32 {
					t.Fatalf("enum payload = %+v", operation)
				}
			}
		}
		if block.Terminator.Kind == TerminatorSwitch && block.Terminator.SwitchKind == "enum" {
			switches++
			if block.Terminator.SwitchType != hir.ParseType("main::Message") || !slices.Equal(block.Terminator.SwitchVariants, []string{"Empty", "Pair"}) || len(block.Terminator.Successors) != 2 {
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
enum Message { Empty; Value { i32 amount; }; }
fn i32 main() {
    let Message message = .Value { .amount = 42 };
    return switch (message) { case .Empty : { 0 } case .Value { .amount = value } : { value } };
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
			expectedCode := CodeAggregateUse
			if mutation.name == "switch-table" {
				expectedCode = CodeControlFlow
			}
			if len(failures) == 0 || failures[0].Code != expectedCode {
				t.Fatalf("Verify() failures = %+v, want the classified invariant failure", failures)
			}
		})
	}
}

func TestLowerPreservesUTF8StringStaticStorageOperand(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 main() {
    let ref str text = "λ";
    return 42;
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
	if failures := Verify(mutated); len(failures) != 1 || failures[0].Code != CodeOperandShape {
		t.Fatalf("Verify() UTF-8 string type mutation failures = %+v", failures)
	}
}

func TestLowerPreservesUTF8StringDataAddressUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@link_name("fairyc_fixture_is_fairy_text")
unsafe extern C fn u32 is_fairy_text(ptr u8 value) ;
fn i32 main() {
    let ref str text = "fairy";
    return if (unsafe { is_fairy_text(text as ptr u8) } == 1) { 42 } else { 1 };
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

func TestLowerPreservesAlwaysCheckedAssert(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\nfn i32 main() { let bool ready = true; assert(ready); return 42; }\n")
	function := &program.Functions[0]
	var assertion *Terminator
	for blockIndex := range function.Blocks {
		terminator := &function.Blocks[blockIndex].Terminator
		if terminator.SwitchKind == "assert" {
			assertion = terminator
			break
		}
	}
	if assertion == nil || assertion.SwitchType != hir.TypeBool || assertion.Operator != "assert" ||
		len(assertion.SwitchOperands) != 1 || assertion.SwitchOperands[0].Type != hir.TypeBool || len(assertion.Successors) != 2 {
		t.Fatalf("assert switch = %+v", assertion)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		terminator := &mutated.Functions[0].Blocks[blockIndex].Terminator
		if terminator.Kind == TerminatorTrap && slices.Equal(terminator.TrapCategories, []string{"assertion-failed"}) {
			terminator.TrapCategories = []string{"bounds"}
			break
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() assert trap mutation failures = %+v", failures)
	}

	mutated = Clone(program)
	for blockIndex := range mutated.Functions[0].Blocks {
		terminator := &mutated.Functions[0].Blocks[blockIndex].Terminator
		if terminator.SwitchKind == "assert" {
			terminator.SwitchOperands[0].Type = hir.TypeI32
			break
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeControlFlow {
		t.Fatalf("Verify() assert operand-type mutation failures = %+v", failures)
	}
}

func TestLowerLiteralFalseAssertHasNoContinuation(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, "module main;\nfn i32 main() { assert(false); }\n")
	function := &program.Functions[0]
	if len(function.Blocks) != 1 {
		t.Fatalf("literal false assertion blocks = %+v", function.Blocks)
	}
	terminator := function.Blocks[0].Terminator
	if terminator.Kind != TerminatorTrap || !slices.Equal(terminator.TrapCategories, []string{"assertion-failed"}) || len(terminator.Successors) != 0 {
		t.Fatalf("literal false assertion terminator = %+v", terminator)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedScalarFixedArrayIndex(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn u8 pick(u8[3] values, usize index) { return values[index]; }
fn i32 main() {
    let u8[3] values = [40, 41, 42];
    return pick(values, 2) as i32;
}

`)
	var construct, check, index *Operation
	var checked *Terminator
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			block := &program.Functions[functionIndex].Blocks[blockIndex]
			for operationIndex := range block.Operations {
				operation := &block.Operations[operationIndex]
				switch operation.Kind {
				case OperationArrayConstruct:
					construct = operation
				case OperationArrayIndexCheck:
					check = operation
				case OperationArrayIndex:
					index = operation
				}
			}
			if block.Terminator.SwitchKind == "checked-array-index" {
				checked = &block.Terminator
			}
		}
	}
	wantArray := hir.FixedArray(hir.TypeU8, 3)
	if construct == nil || construct.Type != wantArray || construct.OwnerType != hir.TypeU8 || len(construct.Operands) != 3 {
		t.Fatalf("array construction = %+v", construct)
	}
	if check == nil || check.OwnerType != wantArray || check.Type != hir.TypeU8 || !slices.Equal(check.TrapCategories, []string{"bounds"}) {
		t.Fatalf("array index check = %+v", check)
	}
	if index == nil || index.OwnerType != wantArray || index.Type != hir.TypeU8 || index.DestinationLocal == NoLocal {
		t.Fatalf("array index = %+v", index)
	}
	if checked == nil || len(checked.Successors) != 2 || len(checked.SwitchOperands) != 2 {
		t.Fatalf("checked array switch = %+v", checked)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationArrayIndexCheck {
					operation.TrapCategories = nil
				}
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != "V1501" {
		t.Fatalf("Verify() array-check mutation failures = %+v", failures)
	}
}

func TestLowerPreservesCheckedIndexedStores(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn u8 update_view(ref mut u8[] bytes, u8 value) {
    bytes[1] = value;
    return bytes[1];
}

fn i32 main() {
    var u8[3] values = [1, 2, 3];
    values[0] = 40;
    return update_view(ref mut values, 42) as i32;
}
`)
	var arrayStore, viewStore *Operation
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			for operationIndex := range program.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &program.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				switch operation.Kind {
				case OperationArrayIndexStore:
					arrayStore = operation
				case OperationViewIndexStore:
					viewStore = operation
				}
			}
		}
	}
	if arrayStore == nil || arrayStore.OwnerType != hir.FixedArray(hir.TypeU8, 3) || arrayStore.Type != hir.TypeU8 ||
		arrayStore.Operator != "[]=" || arrayStore.DestinationLocal != NoLocal || len(arrayStore.Operands) != 3 {
		t.Fatalf("array store = %+v", arrayStore)
	}
	if viewStore == nil || viewStore.OwnerType != hir.MutableBorrow(hir.TypeByteSlice) || viewStore.Type != hir.TypeU8 ||
		viewStore.Operator != "[]=" || viewStore.DestinationLocal != NoLocal || len(viewStore.Operands) != 3 {
		t.Fatalf("view store = %+v", viewStore)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationViewIndexStore {
					operation.Operator = "[]"
				}
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() store mutation failures = %+v", failures)
	}
}

func TestLowerPreservesScalarFixedArrayLength(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn u32[3] make() { return [40, 41, 42]; }
fn i32 main() { return if (make().len == 3) { 42 } else { 1 }; }
`)
	var mainFunction *Function
	for functionIndex := range program.Functions {
		if program.Functions[functionIndex].Name == "main::main" {
			mainFunction = &program.Functions[functionIndex]
			break
		}
	}
	if mainFunction == nil || len(mainFunction.Calls) != 1 {
		t.Fatalf("main function/calls = %+v, want one receiver evaluation", mainFunction)
	}
	var length *Operation
	for blockIndex := range mainFunction.Blocks {
		for operationIndex := range mainFunction.Blocks[blockIndex].Operations {
			operation := &mainFunction.Blocks[blockIndex].Operations[operationIndex]
			if operation.Kind == OperationArrayLength {
				length = operation
			}
		}
	}
	wantArray := hir.FixedArray(hir.TypeU32, 3)
	if length == nil || length.OwnerType != wantArray || length.Type != hir.TypeUSize || length.Field != "len" ||
		length.DestinationLocal == NoLocal || len(length.Operands) != 1 {
		t.Fatalf("array length = %+v", length)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationArrayLength {
					operation.Field = "size"
				}
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() array-length mutation failures = %+v", failures)
	}
}

func TestLowerPreservesFixedU8ArrayByteViewBorrow(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn i32 first(ref u8[] bytes) { return bytes[2] as i32; }
fn i32 main() {
    let u8[3] values = [40, 41, 42];
    return first(ref values);
}
`)
	var borrow *Operation
	for functionIndex := range program.Functions {
		if program.Functions[functionIndex].Name != "main::main" {
			continue
		}
		for blockIndex := range program.Functions[functionIndex].Blocks {
			for operationIndex := range program.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &program.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationArrayBorrowView {
					borrow = operation
				}
			}
		}
	}
	if borrow == nil || borrow.OwnerType != hir.FixedArray(hir.TypeU8, 3) || borrow.Type != hir.SharedBorrow(hir.TypeByteSlice) ||
		borrow.Operator != "&" || borrow.DestinationLocal == NoLocal || len(borrow.Operands) != 1 || borrow.Operands[0].Kind != OperandLocal {
		t.Fatalf("array byte-view borrow = %+v", borrow)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationArrayBorrowView {
					operation.Operator = "&mut"
				}
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() array byte-view mutation failures = %+v", failures)
	}
}

func TestLowerPreservesMutableFixedU8ArrayByteViewBorrow(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn usize length(ref mut u8[] bytes) { return bytes.len; }
fn i32 main() {
    var u8[3] values = [40, 41, 42];
    return if (length(ref mut values) == 3) { 42 } else { 1 };
}
`)
	var borrow *Operation
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			for operationIndex := range program.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &program.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationArrayBorrowView && operation.Operator == "&mut" {
					borrow = operation
				}
			}
		}
	}
	if borrow == nil || borrow.OwnerType != hir.FixedArray(hir.TypeU8, 3) || borrow.Type != hir.MutableBorrow(hir.TypeByteSlice) ||
		borrow.DestinationLocal == NoLocal || len(borrow.Operands) != 1 || borrow.Operands[0].Kind != OperandLocal {
		t.Fatalf("mutable array byte-view borrow = %+v", borrow)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	for functionIndex := range mutated.Functions {
		for blockIndex := range mutated.Functions[functionIndex].Blocks {
			for operationIndex := range mutated.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &mutated.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationArrayBorrowView && operation.Operator == "&mut" {
					operation.Type = hir.SharedBorrow(hir.TypeByteSlice)
				}
			}
		}
	}
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() mutable byte-view type mutation failures = %+v", failures)
	}
}

func TestLowerVerifiesBorrowedViewRawPartsUnsafeProvenance(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource String {
    ptr mut void data;
    usize length;
    cleanup {}
}
fn ref str as_str(ref String value) {
    return unsafe { str::from_raw_parts(value, value.data, value.length) };
}
`)
	var functionIndex = -1
	var constructor *Operation
	for index := range program.Functions {
		function := &program.Functions[index]
		if function.Name != "main::as_str" {
			continue
		}
		functionIndex = index
		for blockIndex := range function.Blocks {
			for operationIndex := range function.Blocks[blockIndex].Operations {
				operation := &function.Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind == OperationStrFromRawParts {
					constructor = operation
				}
			}
		}
	}
	if functionIndex < 0 || constructor == nil || constructor.Type != hir.SharedBorrow(hir.TypeStr) ||
		constructor.OwnerType != hir.SharedBorrow(hir.ParseType("main::String")) || len(constructor.Operands) != 3 || constructor.UnsafeScope.Source != "main.fy" {
		t.Fatalf("raw-parts constructor = %+v", constructor)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}

	mutated := Clone(program)
	mutated.Functions[functionIndex].UnsafeOperations = nil
	failures := Verify(mutated)
	if len(failures) != 1 || failures[0].Code != "V1701" {
		t.Fatalf("Verify() missing raw-parts provenance failures = %+v", failures)
	}
}

func TestLowerVerifiesBorrowedByteViewRawPartsMutability(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Buffer {
    ptr mut void data;
    usize length;
    cleanup {}
}
fn ref u8[] view(ref Buffer value) {
    return unsafe { byte_slice::from_raw_parts(value, value.data, value.length) };
}
fn ref mut u8[] view_mut(ref mut Buffer value) {
    return unsafe { byte_slice::from_raw_parts_mut(value, value.data, value.length) };
}
`)
	seen := map[OperationKind]bool{}
	mutableFunction := -1
	mutableBlock := -1
	mutableOperation := -1
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			for operationIndex := range program.Functions[functionIndex].Blocks[blockIndex].Operations {
				operation := &program.Functions[functionIndex].Blocks[blockIndex].Operations[operationIndex]
				if operation.Kind != OperationByteSliceFromRawParts && operation.Kind != OperationMutByteSliceFromRawParts {
					continue
				}
				seen[operation.Kind] = true
				if operation.Kind == OperationMutByteSliceFromRawParts {
					mutableFunction, mutableBlock, mutableOperation = functionIndex, blockIndex, operationIndex
				}
			}
		}
	}
	if !seen[OperationByteSliceFromRawParts] || !seen[OperationMutByteSliceFromRawParts] || mutableFunction < 0 {
		t.Fatalf("raw-parts operations = %v", seen)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
	mutated := Clone(program)
	mutated.Functions[mutableFunction].Blocks[mutableBlock].Operations[mutableOperation].OwnerType = hir.SharedBorrow(hir.ParseType("main::Buffer"))
	if failures := Verify(mutated); len(failures) == 0 || failures[0].Code != CodeOperationShape {
		t.Fatalf("Verify() shared-owner mutation failures = %+v", failures)
	}
}

func TestLowerMovesResourceThroughBlockTailInOptionArm(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Token {
    i32 value;
    cleanup {}
}
fn Token unwrap(Option<Token> input) {
    return switch (input) {
        case .some(token): { token }
        case .none: { Token { .value = 0 } }
    };
}
fn i32 main() {
    let Token token = unwrap(.some(Token { .value = 42 }));
    return token.value;
}
`)
	var unwrap *Function
	for index := range program.Functions {
		if program.Functions[index].Name == "main::unwrap" {
			unwrap = &program.Functions[index]
			break
		}
	}
	if unwrap == nil {
		t.Fatal("lowered program has no main::unwrap")
	}
	var transferred *MoveFact
	for index := range unwrap.Moves {
		if unwrap.Moves[index].Destination == "option-match-result" {
			transferred = &unwrap.Moves[index]
			break
		}
	}
	if transferred == nil || !transferred.CleanupObligationTransferred {
		t.Fatalf("option arm moves = %+v", unwrap.Moves)
	}
	for _, block := range unwrap.Blocks {
		for _, operation := range block.Operations {
			if operation.Kind == OperationDrop && operation.Span == transferred.Binding {
				t.Fatalf("moved option payload was dropped in its arm: %+v", operation)
			}
		}
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerMovesResourceParameterIntoLocalBinding(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
resource Token {
    i32 value;
    cleanup {}
}
fn i32 consume(Token input) {
    var Token value = input;
    return value.value;
}
fn i32 main() {
    return consume(Token { .value = 42 });
}
`)
	var consume *Function
	for index := range program.Functions {
		if program.Functions[index].Name == "main::consume" {
			consume = &program.Functions[index]
			break
		}
	}
	if consume == nil || len(consume.Resources) != 2 {
		t.Fatalf("consume resources = %+v", consume)
	}
	input := consume.Resources[0]
	value := consume.Resources[1]
	var inputMoves, inputDrops, valueDrops int
	for _, block := range consume.Blocks {
		for _, operation := range block.Operations {
			switch {
			case operation.Kind == OperationMove && operation.Symbol == input.Symbol:
				inputMoves++
				if operation.Destination != "binding-value" {
					t.Errorf("input move = %+v", operation)
				}
			case operation.Kind == OperationDrop && operation.Symbol == input.Symbol:
				inputDrops++
			case operation.Kind == OperationDrop && operation.Symbol == value.Symbol:
				valueDrops++
			}
		}
	}
	if inputMoves != 1 || inputDrops != 0 || valueDrops != 1 {
		t.Fatalf("consume ownership operations = input moves %d, input drops %d, value drops %d; moves=%+v", inputMoves, inputDrops, valueDrops, consume.Moves)
	}
	if len(consume.Moves) != 1 || consume.Moves[0].Destination != "binding-value" || !consume.Moves[0].CleanupObligationTransferred {
		t.Fatalf("consume moves = %+v", consume.Moves)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerCompoundAssignmentsReuseCheckedBinaryAndSinglePlaceEvaluation(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn u8 update_view(ref mut u8[] bytes) {
    bytes[1] += 2;
    return bytes[1];
}
fn i32 main() {
    var i32 value = 40;
    value += 2;
    var u8[3] values = [1, 40, 3];
    values[1] += 2;
    return update_view(ref mut values) as i32;
}
`)
	var binaryCount, arrayLoadCount, arrayStoreCount, viewLoadCount, viewStoreCount int
	for functionIndex := range program.Functions {
		for blockIndex := range program.Functions[functionIndex].Blocks {
			for _, operation := range program.Functions[functionIndex].Blocks[blockIndex].Operations {
				switch operation.Kind {
				case OperationBinary:
					if operation.Operator == "+" {
						binaryCount++
					}
				case OperationArrayIndex:
					arrayLoadCount++
				case OperationArrayIndexStore:
					arrayStoreCount++
				case OperationViewIndex:
					viewLoadCount++
				case OperationViewIndexStore:
					viewStoreCount++
				}
			}
		}
	}
	if binaryCount != 3 || arrayLoadCount < 1 || arrayStoreCount != 1 || viewLoadCount < 2 || viewStoreCount != 1 {
		t.Fatalf("compound MIR counts: binary=%d array-load=%d array-store=%d view-load=%d view-store=%d", binaryCount, arrayLoadCount, arrayStoreCount, viewLoadCount, viewStoreCount)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerLexicalDeferCapturesNowAndCallsOnReturn(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
fn void record(i32 value) { assert(value == 42); }
fn i32 main() {
    var i32 value = 40;
    defer record(value + 2);
    value = 0;
    return 42;
}
`)
	var mainFunction *Function
	for index := range program.Functions {
		if program.Functions[index].Name == "main::main" {
			mainFunction = &program.Functions[index]
		}
	}
	if mainFunction == nil || len(mainFunction.Calls) != 1 || len(mainFunction.CleanupPaths) != 1 ||
		len(mainFunction.CleanupPaths[0].Actions) != 1 || mainFunction.CleanupPaths[0].Actions[0].Kind != "defer-call" {
		t.Fatalf("deferred main function = %+v", mainFunction)
	}
	if len(mainFunction.EliminatedForms) == 0 || mainFunction.EliminatedForms[0].Form != "lexical-defer" {
		t.Fatalf("eliminated forms = %+v", mainFunction.EliminatedForms)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func TestLowerBurnDischargesResourceWithoutDrop(t *testing.T) {
	t.Parallel()
	program := lowerTestSource(t, `module main;
@abilities(store, key)
resource Coin { i32 amount; cleanup {} }
fn i32 main() {
    let Coin coin = Coin { .amount = 42 };
    burn coin;
    return 42;
}
`)
	var burns, drops int
	for _, block := range program.Functions[0].Blocks {
		for _, operation := range block.Operations {
			switch operation.Kind {
			case OperationBurn:
				burns++
				if operation.Type != program.ResourceTypes[0].Type || operation.DestinationLocal != NoLocal {
					t.Errorf("burn operation = %+v", operation)
				}
			case OperationDrop:
				drops++
			}
		}
	}
	if burns != 1 || drops != 0 {
		t.Fatalf("ownership operations = burns %d, drops %d", burns, drops)
	}
	if failures := Verify(program); len(failures) != 0 {
		t.Fatalf("Verify() failures = %+v", failures)
	}
}

func lowerTestSource(t *testing.T, text string) Program {
	t.Helper()
	return lowerTestSourceAtPath(t, "main.fy", text)
}

func lowerTestSourceAtPath(t *testing.T, path, text string) Program {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(text)), source.DefaultLimits())
	if err != nil {
		t.Fatalf("AnalyzeSyntax() error = %v", err)
	}
	if len(parsed.Tree.Diagnostics) != 0 || parsed.AST == nil {
		t.Fatalf("AnalyzeSyntax() diagnostics = %+v, AST = %+v", parsed.Tree.Diagnostics, parsed.AST)
	}
	result := semantic.Analyze([]semantic.File{{Path: path, Module: parsed.AST}})
	if result.CompletedThrough != semantic.PhaseFlow || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("Analyze() result = %+v", result)
	}
	program, err := Lower(result.HIR)
	if err != nil {
		t.Fatalf("Lower() error = %v", err)
	}
	return program
}
