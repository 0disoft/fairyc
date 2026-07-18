// Package cabi classifies the closed Stage 0 C-representable type subset for a declared target.
package cabi

import (
	"errors"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	maxClassificationDepth = 128
	eightbyteSize          = uint64(8)
)

// Class is one SysV AMD64 eightbyte class admitted by the Stage 0 C subset.
type Class string

const (
	ClassInteger Class = "integer"
	ClassSSE     Class = "sse"
	ClassMemory  Class = "memory"
	classNone    Class = ""
)

// ErrorCode is one stable fixture-visible C ABI classification rejection.
type ErrorCode string

const (
	ErrorUnsupportedTarget ErrorCode = "unsupported-target"
	ErrorLayoutRejected    ErrorCode = "layout-rejected"
	ErrorInvalidCEnumBase  ErrorCode = "invalid-c-enum-base"
	ErrorDepthLimit        ErrorCode = "depth-limit"
	ErrorInvalidType       ErrorCode = "invalid-type"
)

// Error reports a fail-closed classifier rejection.
type Error struct {
	Code   ErrorCode
	Detail string
}

func (e Error) Error() string {
	if e.Detail == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Detail
}

// Code extracts a fixture-visible classifier error code.
func Code(err error) (ErrorCode, bool) {
	var value Error
	if !errors.As(err, &value) {
		return "", false
	}
	return value.Code, true
}

// Classification records the complete eightbyte classification of one value.
type Classification struct {
	SizeBytes      uint64  `json:"sizeBytes"`
	AlignmentBytes uint64  `json:"alignmentBytes"`
	Classes        []Class `json:"classes"`
}

// LocationKind identifies one closed call-boundary transport.
type LocationKind string

const (
	LocationVoid      LocationKind = "void"
	LocationRegisters LocationKind = "registers"
	LocationStack     LocationKind = "stack"
	LocationIndirect  LocationKind = "indirect"
)

// Register is one canonical SysV AMD64 register spelling used by fixtures.
type Register string

// ReturnLocation describes result transport, including the hidden pointer for memory returns.
type ReturnLocation struct {
	Kind                  LocationKind   `json:"kind"`
	Classification        Classification `json:"classification"`
	Registers             []Register     `json:"registers"`
	HiddenPointerRegister Register       `json:"hiddenPointerRegister,omitempty"`
	PointerResultRegister Register       `json:"pointerResultRegister,omitempty"`
}

// ArgumentLocation describes one declaration-ordered parameter transport.
type ArgumentLocation struct {
	Index          int            `json:"index"`
	Kind           LocationKind   `json:"kind"`
	Classification Classification `json:"classification"`
	Registers      []Register     `json:"registers"`
}

// Signature is one fixed, non-variadic C call boundary. A nil result means void.
type Signature struct {
	Result     *layout.Type
	Parameters []layout.Type
}

// Plan is one complete SysV AMD64 result and argument assignment.
type Plan struct {
	Return    ReturnLocation     `json:"return"`
	Arguments []ArgumentLocation `json:"arguments"`
}

// Engine binds C ABI classification to one immutable target and its layout engine.
type Engine struct {
	targetID target.ID
	layout   *layout.Engine
}

var (
	integerArgumentRegisters = []Register{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
	sseArgumentRegisters     = []Register{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7"}
	integerReturnRegisters   = []Register{"rax", "rdx"}
	sseReturnRegisters       = []Register{"xmm0", "xmm1"}
)

// New creates a host-independent classifier for the exact Stage 0 SysV AMD64 profile.
func New(spec target.Spec) (*Engine, error) {
	if err := target.Validate(spec); err != nil {
		return nil, Error{Code: ErrorUnsupportedTarget, Detail: err.Error()}
	}
	if spec.ID != target.X8664LinuxGNU || spec.CallingConvention != "sysv-amd64" ||
		spec.AggregateLayout != "sysv-amd64-lp64" || spec.ReturnClassifier != "sysv-amd64" {
		return nil, Error{Code: ErrorUnsupportedTarget, Detail: fmt.Sprintf("target %q does not declare the Stage 0 SysV AMD64 profile", spec.ID)}
	}
	layoutEngine, err := layout.New(spec)
	if err != nil {
		return nil, Error{Code: ErrorUnsupportedTarget, Detail: err.Error()}
	}
	return &Engine{targetID: spec.ID, layout: layoutEngine}, nil
}

// Target returns the semantic target bound to this classifier.
func (e *Engine) Target() target.ID {
	if e == nil {
		return ""
	}
	return e.targetID
}

// Classify applies the closed Stage 0 subset of the SysV AMD64 eightbyte algorithm.
func (e *Engine) Classify(value layout.Type) (Classification, error) {
	if e == nil || e.layout == nil {
		return Classification{}, Error{Code: ErrorUnsupportedTarget, Detail: "C ABI classifier is nil"}
	}
	computed, err := e.layout.Of(value)
	if err != nil {
		return Classification{}, Error{Code: ErrorLayoutRejected, Detail: err.Error()}
	}
	result := Classification{
		SizeBytes: computed.SizeBytes, AlignmentBytes: computed.AlignmentBytes,
	}
	if computed.SizeBytes > 2*eightbyteSize {
		result.Classes = []Class{ClassMemory}
		return result, nil
	}
	result.Classes = make([]Class, (computed.SizeBytes+eightbyteSize-1)/eightbyteSize)
	if err := e.classifyInto(value, 0, result.Classes, 0); err != nil {
		return Classification{}, err
	}
	for index, class := range result.Classes {
		if class == classNone {
			return Classification{}, Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("eightbyte %d contains only unclassified storage", index)}
		}
	}
	return result, nil
}

func (e *Engine) classifyInto(value layout.Type, baseOffset uint64, classes []Class, depth int) error {
	if depth >= maxClassificationDepth {
		return Error{Code: ErrorDepthLimit, Detail: fmt.Sprintf("classification nesting exceeds %d levels", maxClassificationDepth)}
	}
	computed, err := e.layout.Of(value)
	if err != nil {
		return Error{Code: ErrorLayoutRejected, Detail: err.Error()}
	}
	switch value.Kind {
	case layout.KindLanguageScalar:
		class := ClassInteger
		if value.Name == "f32" || value.Name == "f64" {
			class = ClassSSE
		}
		return mergeRange(classes, baseOffset, computed.SizeBytes, class)
	case layout.KindCScalar:
		class := ClassInteger
		if value.Name == "c.float" || value.Name == "c.double" {
			class = ClassSSE
		}
		return mergeRange(classes, baseOffset, computed.SizeBytes, class)
	case layout.KindCEnum:
		if value.Name == "c.float" || value.Name == "c.double" {
			return Error{Code: ErrorInvalidCEnumBase, Detail: fmt.Sprintf("C enum base %q is not an integer scalar", value.Name)}
		}
		return mergeRange(classes, baseOffset, computed.SizeBytes, ClassInteger)
	case layout.KindPointer, layout.KindFunctionPtr:
		return mergeRange(classes, baseOffset, computed.SizeBytes, ClassInteger)
	case layout.KindArray:
		elementLayout, err := e.layout.Of(*value.Element)
		if err != nil {
			return Error{Code: ErrorLayoutRejected, Detail: err.Error()}
		}
		for index := uint64(0); index < value.Length; index++ {
			if err := e.classifyInto(*value.Element, baseOffset+index*elementLayout.SizeBytes, classes, depth+1); err != nil {
				return err
			}
		}
		return nil
	case layout.KindStruct:
		if len(computed.Fields) != len(value.Fields) {
			return Error{Code: ErrorInvalidType, Detail: "layout field identity drifted during classification"}
		}
		for index, field := range value.Fields {
			placement := computed.Fields[index]
			if placement.Name != field.Name || placement.OffsetBytes%placement.AlignmentBytes != 0 {
				return Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("field %q has invalid ABI placement", field.Name)}
			}
			if err := e.classifyInto(field.Type, baseOffset+placement.OffsetBytes, classes, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("layout kind %q is not C ABI classifiable", value.Kind)}
	}
}

func mergeRange(classes []Class, offset, size uint64, incoming Class) error {
	if size == 0 || incoming == classNone || incoming == ClassMemory {
		return Error{Code: ErrorInvalidType, Detail: "invalid scalar classification range"}
	}
	first := offset / eightbyteSize
	last := (offset + size - 1) / eightbyteSize
	if last >= uint64(len(classes)) {
		return Error{Code: ErrorInvalidType, Detail: "classification range exceeds aggregate layout"}
	}
	for index := first; index <= last; index++ {
		classes[index] = merge(classes[index], incoming)
	}
	return nil
}

func merge(current, incoming Class) Class {
	if current == incoming || incoming == classNone {
		return current
	}
	if current == classNone {
		return incoming
	}
	if current == ClassMemory || incoming == ClassMemory {
		return ClassMemory
	}
	if current == ClassInteger || incoming == ClassInteger {
		return ClassInteger
	}
	return ClassSSE
}

// Plan assigns return and parameter locations with aggregate rollback on register exhaustion.
func (e *Engine) Plan(signature Signature) (Plan, error) {
	if e == nil || e.layout == nil {
		return Plan{}, Error{Code: ErrorUnsupportedTarget, Detail: "C ABI classifier is nil"}
	}
	result := Plan{Arguments: make([]ArgumentLocation, 0, len(signature.Parameters))}
	integerUsed := 0
	sseUsed := 0
	if signature.Result == nil {
		result.Return = ReturnLocation{Kind: LocationVoid, Registers: []Register{}, Classification: Classification{Classes: []Class{}}}
	} else {
		classification, err := e.Classify(*signature.Result)
		if err != nil {
			return Plan{}, fmt.Errorf("classify return: %w", err)
		}
		result.Return = ReturnLocation{Classification: cloneClassification(classification), Registers: []Register{}}
		if isMemory(classification) {
			result.Return.Kind = LocationIndirect
			result.Return.HiddenPointerRegister = "rdi"
			result.Return.PointerResultRegister = "rax"
			integerUsed = 1
		} else {
			result.Return.Kind = LocationRegisters
			result.Return.Registers = assignRegisters(classification.Classes, integerReturnRegisters, sseReturnRegisters, 0, 0)
		}
	}
	for index, parameter := range signature.Parameters {
		classification, err := e.Classify(parameter)
		if err != nil {
			return Plan{}, fmt.Errorf("classify parameter %d: %w", index, err)
		}
		argument := ArgumentLocation{
			Index: index, Classification: cloneClassification(classification), Registers: []Register{},
		}
		integerNeed, sseNeed := classCounts(classification.Classes)
		if isMemory(classification) || integerUsed+integerNeed > len(integerArgumentRegisters) || sseUsed+sseNeed > len(sseArgumentRegisters) {
			argument.Kind = LocationStack
		} else {
			argument.Kind = LocationRegisters
			argument.Registers = assignRegisters(classification.Classes, integerArgumentRegisters, sseArgumentRegisters, integerUsed, sseUsed)
			integerUsed += integerNeed
			sseUsed += sseNeed
		}
		result.Arguments = append(result.Arguments, argument)
	}
	return result, nil
}

func assignRegisters(classes []Class, integers, vectors []Register, integerStart, vectorStart int) []Register {
	result := make([]Register, 0, len(classes))
	integerIndex := integerStart
	vectorIndex := vectorStart
	for _, class := range classes {
		switch class {
		case ClassInteger:
			result = append(result, integers[integerIndex])
			integerIndex++
		case ClassSSE:
			result = append(result, vectors[vectorIndex])
			vectorIndex++
		}
	}
	return result
}

func classCounts(classes []Class) (int, int) {
	integers := 0
	vectors := 0
	for _, class := range classes {
		if class == ClassInteger {
			integers++
		} else if class == ClassSSE {
			vectors++
		}
	}
	return integers, vectors
}

func isMemory(value Classification) bool {
	return len(value.Classes) == 1 && value.Classes[0] == ClassMemory
}

func cloneClassification(value Classification) Classification {
	value.Classes = slices.Clone(value.Classes)
	return value
}
