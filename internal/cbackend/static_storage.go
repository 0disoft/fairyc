package cbackend

import (
	"crypto/sha256"
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

type staticStorageSpec struct {
	operandKind  mir.OperandKind
	valueType    hir.Type
	label        string
	symbolPrefix string
	valid        func(string) bool
}

func collectStaticStorage(functions []mir.Function, spec staticStorageSpec) (map[string]string, error) {
	result := make(map[string]string)
	// Literal contents can recur across operands and functions, including in
	// separately decoded strings. Cache only within this collection; still check
	// each operand's type before reusing content validation and its digest.
	seen := make(map[string]struct{})
	record := func(function string, operand mir.Operand) error {
		if operand.Kind != spec.operandKind {
			return nil
		}
		if operand.Type != spec.valueType {
			return fmt.Errorf("collect %s storage in %s: operand has invalid static storage", spec.label, function)
		}
		if _, found := seen[operand.StaticData]; found {
			return nil
		}
		if spec.valid != nil && !spec.valid(operand.StaticData) {
			return fmt.Errorf("collect %s storage in %s: operand has invalid static storage", spec.label, function)
		}
		symbol := staticStorageSymbol(spec.symbolPrefix, operand.StaticData)
		if previous, found := result[symbol]; found && previous != operand.StaticData {
			return fmt.Errorf("collect %s storage in %s: digest collision for %s", spec.label, function, symbol)
		}
		result[symbol] = operand.StaticData
		seen[operand.StaticData] = struct{}{}
		return nil
	}
	recordAll := func(function string, operands []mir.Operand) error {
		for _, operand := range operands {
			if err := record(function, operand); err != nil {
				return err
			}
		}
		return nil
	}
	for _, function := range functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if err := recordAll(function.Name, operation.Operands); err != nil {
					return nil, err
				}
			}
			if err := recordAll(function.Name, block.Terminator.Arguments); err != nil {
				return nil, err
			}
			if err := recordAll(function.Name, block.Terminator.SwitchOperands); err != nil {
				return nil, err
			}
			if err := record(function.Name, block.Terminator.ReturnValue); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func staticStorageSymbol(prefix, value string) string {
	var digest [sha256.Size]byte
	if len(value) <= 32 {
		// Keep the compiler's small, stack-backed []byte conversion fast path.
		digest = sha256.Sum256([]byte(value))
	} else {
		digest = chunkedStaticStorageDigest(value)
	}
	return fmt.Sprintf("%s%x", prefix, digest)
}

// Bound temporary storage instead of copying the entire literal to []byte.
// Keep this frame separate so short literals do not need the scratch buffer.
func chunkedStaticStorageDigest(value string) [sha256.Size]byte {
	hasher := sha256.New()
	var buffer [4096]byte
	for len(value) != 0 {
		count := copy(buffer[:], value)
		hasher.Write(buffer[:count])
		value = value[count:]
	}
	var digest [sha256.Size]byte
	hasher.Sum(digest[:0])
	return digest
}

func emitStaticStorage(output *strings.Builder, storage map[string]string, emptySentinel bool) {
	emitStaticStorageLinkage(output, storage, emptySentinel, "static ")
}

func emitStaticStorageLinkage(output *strings.Builder, storage map[string]string, emptySentinel bool, linkage string) {
	if len(storage) == 0 {
		return
	}
	symbols := make([]string, 0, len(storage))
	for symbol := range storage {
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	for _, symbol := range symbols {
		value := storage[symbol]
		storageLength := len(value)
		if emptySentinel && storageLength == 0 {
			storageLength = 1
		}
		fmt.Fprintf(output, "%sconst uint8_t %s[%d] = { ", linkage, symbol, storageLength)
		if len(value) == 0 {
			output.WriteString("UINT8_C(0)")
		} else {
			emitStaticStorageBytes(output, value)
		}
		output.WriteString(" };\n")
	}
	output.WriteByte('\n')
}

// Batch decimal byte literals without invoking the general formatter per byte.
// The fixed scratch buffer bounds temporary storage independently of the input.
func emitStaticStorageBytes(output *strings.Builder, value string) {
	const maxLiteral = len(", UINT8_C(255)")
	const suffix = len(" };\n\n")
	// Reserve the byte literals and closing syntax without integer overflow.
	if len(value) <= (int(^uint(0)>>1)-output.Len()-suffix)/maxLiteral {
		output.Grow(len(value)*maxLiteral + suffix)
	}
	var scratch [4096]byte
	buffer := scratch[:0]
	for index := 0; index < len(value); index++ {
		if cap(buffer)-len(buffer) < maxLiteral {
			output.Write(buffer)
			buffer = buffer[:0]
		}
		if index != 0 {
			buffer = append(buffer, ',', ' ')
		}
		buffer = append(buffer, "UINT8_C("...)
		buffer = strconv.AppendUint(buffer, uint64(value[index]), 10)
		buffer = append(buffer, ')')
	}
	output.Write(buffer)
}

func collectStringStorage(functions []mir.Function) (map[string]string, error) {
	return collectStaticStorage(functions, staticStorageSpec{
		operandKind: mir.OperandString, valueType: hir.SharedBorrow(hir.TypeStr), label: "UTF-8 string",
		symbolPrefix: "fy_static_str_", valid: utf8.ValidString,
	})
}

func stringStorageSymbol(value string) string {
	return staticStorageSymbol("fy_static_str_", value)
}

func (e *emitter) emitStringStorage(output *strings.Builder) {
	emitStaticStorage(output, e.analysis.stringStorage, true)
}

func collectByteStringStorage(functions []mir.Function) (map[string]string, error) {
	return collectStaticStorage(functions, staticStorageSpec{
		operandKind: mir.OperandByteString, valueType: hir.SharedBorrow(hir.TypeByteSlice),
		label: "byte-string", symbolPrefix: "fy_static_bytes_",
	})
}

func byteStringStorageSymbol(value string) string {
	return staticStorageSymbol("fy_static_bytes_", value)
}

func (e *emitter) emitByteStringStorage(output *strings.Builder) {
	emitStaticStorage(output, e.analysis.byteStringStorage, true)
}

func collectCStringStorage(functions []mir.Function) (map[string]string, error) {
	return collectStaticStorage(functions, staticStorageSpec{
		operandKind: mir.OperandCString, valueType: hir.SharedBorrow(hir.TypeCStr),
		label: "C string", symbolPrefix: "fy_static_cstr_", valid: validCStringStorage,
	})
}

func validCStringStorage(value string) bool {
	return len(value) > 0 && value[len(value)-1] == 0 && !strings.Contains(value[:len(value)-1], "\x00")
}

func cStringStorageSymbol(value string) string {
	return staticStorageSymbol("fy_static_cstr_", value)
}

func (e *emitter) emitCStringStorage(output *strings.Builder) {
	emitStaticStorage(output, e.analysis.cStringStorage, false)
}
