// Package layout computes target-owned scalar, array, enum, and aggregate layouts.
package layout

import (
	"errors"
	"fmt"
	"math"

	"github.com/0disoft/fairyc/internal/target"
)

const maxLayoutDepth = 128

// Kind identifies one closed Stage 0 layout input family.
type Kind string

const (
	KindLanguageScalar Kind = "language-scalar"
	KindCScalar        Kind = "c-scalar"
	KindPointer        Kind = "pointer"
	KindFunctionPtr    Kind = "function-pointer"
	KindArray          Kind = "array"
	KindStruct         Kind = "struct"
	KindCEnum          Kind = "c-enum"
)

// Type is one recursively closed target-layout input.
type Type struct {
	Kind    Kind    `json:"kind"`
	Name    string  `json:"name,omitempty"`
	Element *Type   `json:"element,omitempty"`
	Length  uint64  `json:"length,omitempty"`
	Fields  []Field `json:"fields,omitempty"`
}

// Field is one declaration-ordered aggregate field.
type Field struct {
	Name string `json:"name"`
	Type Type   `json:"type"`
}

// FieldResult is one computed top-level field placement.
type FieldResult struct {
	Name           string `json:"name"`
	OffsetBytes    uint64 `json:"offsetBytes"`
	SizeBytes      uint64 `json:"sizeBytes"`
	AlignmentBytes uint64 `json:"alignmentBytes"`
}

// Padding is one contiguous top-level padding range.
type Padding struct {
	OffsetBytes uint64 `json:"offsetBytes"`
	SizeBytes   uint64 `json:"sizeBytes"`
}

// Result is one complete target layout.
type Result struct {
	SizeBytes      uint64        `json:"sizeBytes"`
	AlignmentBytes uint64        `json:"alignmentBytes"`
	Fields         []FieldResult `json:"fields"`
	Padding        []Padding     `json:"padding"`
}

// ErrorCode identifies one stable fixture-visible layout rejection family.
type ErrorCode string

const (
	ErrorInvalidType        ErrorCode = "invalid-type"
	ErrorUnknownScalar      ErrorCode = "unknown-scalar"
	ErrorDuplicateField     ErrorCode = "duplicate-field"
	ErrorEmptyAggregate     ErrorCode = "empty-aggregate"
	ErrorInvalidArrayLength ErrorCode = "invalid-array-length"
	ErrorOverflow           ErrorCode = "overflow"
	ErrorDepthLimit         ErrorCode = "depth-limit"
)

// Error reports a fail-closed layout rejection without exposing host data.
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

// Engine binds layout computation to one validated immutable target description.
type Engine struct {
	targetID target.ID
	language map[string]target.Layout
	cScalars map[string]target.Layout
	pointer  target.Layout
	function target.Layout
}

// New creates one target-bound layout engine.
func New(spec target.Spec) (*Engine, error) {
	if err := target.Validate(spec); err != nil {
		return nil, fmt.Errorf("create layout engine: %w", err)
	}
	value := &Engine{
		targetID: spec.ID,
		language: make(map[string]target.Layout, len(spec.LanguageScalars)),
		cScalars: make(map[string]target.Layout, len(spec.CScalars)),
		pointer:  spec.Pointer,
		function: spec.FunctionPointer,
	}
	for _, scalar := range spec.LanguageScalars {
		value.language[scalar.Name] = scalar.Layout
	}
	for _, scalar := range spec.CScalars {
		value.cScalars[scalar.Name] = scalar.Layout
	}
	return value, nil
}

// Target returns the semantic target bound to this engine.
func (e *Engine) Target() target.ID {
	if e == nil {
		return ""
	}
	return e.targetID
}

// Of computes one type layout without consulting Go host layout or a local C compiler.
func (e *Engine) Of(value Type) (Result, error) {
	if e == nil {
		return Result{}, Error{Code: ErrorInvalidType, Detail: "layout engine is nil"}
	}
	return e.layout(value, 0)
}

func (e *Engine) layout(value Type, depth int) (Result, error) {
	if depth >= maxLayoutDepth {
		return Result{}, Error{Code: ErrorDepthLimit, Detail: fmt.Sprintf("layout nesting exceeds %d levels", maxLayoutDepth)}
	}
	switch value.Kind {
	case KindLanguageScalar:
		if err := requireLeaf(value, true); err != nil {
			return Result{}, err
		}
		return scalarResult(value.Name, e.language)
	case KindCScalar:
		if err := requireLeaf(value, true); err != nil {
			return Result{}, err
		}
		return scalarResult(value.Name, e.cScalars)
	case KindPointer:
		if err := requireLeaf(value, false); err != nil {
			return Result{}, err
		}
		return resultFromTargetLayout(e.pointer), nil
	case KindFunctionPtr:
		if err := requireLeaf(value, false); err != nil {
			return Result{}, err
		}
		return resultFromTargetLayout(e.function), nil
	case KindCEnum:
		if err := requireLeaf(value, true); err != nil {
			return Result{}, err
		}
		return scalarResult(value.Name, e.cScalars)
	case KindArray:
		if value.Name != "" || value.Element == nil || len(value.Fields) != 0 {
			return Result{}, Error{Code: ErrorInvalidType, Detail: "array must contain only an element and length"}
		}
		if value.Length == 0 {
			return Result{}, Error{Code: ErrorInvalidArrayLength, Detail: "fixed array length must be positive"}
		}
		element, err := e.layout(*value.Element, depth+1)
		if err != nil {
			return Result{}, err
		}
		size, err := checkedMultiply(element.SizeBytes, value.Length)
		if err != nil {
			return Result{}, err
		}
		return emptyResult(size, element.AlignmentBytes), nil
	case KindStruct:
		if value.Element != nil || value.Length != 0 {
			return Result{}, Error{Code: ErrorInvalidType, Detail: "struct cannot contain array metadata"}
		}
		if len(value.Fields) == 0 {
			return Result{}, Error{Code: ErrorEmptyAggregate, Detail: "Stage 0 does not lay out empty structs"}
		}
		return e.layoutStruct(value.Fields, depth)
	default:
		return Result{}, Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("unknown layout kind %q", value.Kind)}
	}
}

func (e *Engine) layoutStruct(fields []Field, depth int) (Result, error) {
	result := emptyResult(0, 1)
	result.Fields = make([]FieldResult, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	offset := uint64(0)
	for _, field := range fields {
		if field.Name == "" {
			return Result{}, Error{Code: ErrorInvalidType, Detail: "struct field name is empty"}
		}
		if _, duplicate := seen[field.Name]; duplicate {
			return Result{}, Error{Code: ErrorDuplicateField, Detail: fmt.Sprintf("struct field %q is repeated", field.Name)}
		}
		seen[field.Name] = struct{}{}
		fieldLayout, err := e.layout(field.Type, depth+1)
		if err != nil {
			return Result{}, fmt.Errorf("layout field %q: %w", field.Name, err)
		}
		aligned, err := alignUp(offset, fieldLayout.AlignmentBytes)
		if err != nil {
			return Result{}, err
		}
		if aligned > offset {
			result.Padding = append(result.Padding, Padding{OffsetBytes: offset, SizeBytes: aligned - offset})
		}
		end, err := checkedAdd(aligned, fieldLayout.SizeBytes)
		if err != nil {
			return Result{}, err
		}
		result.Fields = append(result.Fields, FieldResult{
			Name: field.Name, OffsetBytes: aligned,
			SizeBytes: fieldLayout.SizeBytes, AlignmentBytes: fieldLayout.AlignmentBytes,
		})
		offset = end
		result.AlignmentBytes = max(result.AlignmentBytes, fieldLayout.AlignmentBytes)
	}
	size, err := alignUp(offset, result.AlignmentBytes)
	if err != nil {
		return Result{}, err
	}
	if size > offset {
		result.Padding = append(result.Padding, Padding{OffsetBytes: offset, SizeBytes: size - offset})
	}
	result.SizeBytes = size
	return result, nil
}

func requireLeaf(value Type, named bool) error {
	if value.Element != nil || value.Length != 0 || len(value.Fields) != 0 {
		return Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("%s cannot contain aggregate metadata", value.Kind)}
	}
	if named && value.Name == "" {
		return Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("%s requires a scalar name", value.Kind)}
	}
	if !named && value.Name != "" {
		return Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("%s cannot contain a scalar name", value.Kind)}
	}
	return nil
}

func scalarResult(name string, values map[string]target.Layout) (Result, error) {
	value, found := values[name]
	if !found {
		return Result{}, Error{Code: ErrorUnknownScalar, Detail: fmt.Sprintf("scalar %q is not in the target", name)}
	}
	return resultFromTargetLayout(value), nil
}

func resultFromTargetLayout(value target.Layout) Result {
	return emptyResult(uint64(value.SizeBytes), uint64(value.AlignmentBytes))
}

func emptyResult(size, alignment uint64) Result {
	return Result{SizeBytes: size, AlignmentBytes: alignment, Fields: []FieldResult{}, Padding: []Padding{}}
}

func checkedAdd(left, right uint64) (uint64, error) {
	if left > math.MaxUint64-right {
		return 0, Error{Code: ErrorOverflow, Detail: "layout size exceeds uint64"}
	}
	return left + right, nil
}

func checkedMultiply(left, right uint64) (uint64, error) {
	if left != 0 && right > math.MaxUint64/left {
		return 0, Error{Code: ErrorOverflow, Detail: "array layout size exceeds uint64"}
	}
	return left * right, nil
}

func alignUp(value, alignment uint64) (uint64, error) {
	if alignment == 0 || alignment&(alignment-1) != 0 {
		return 0, Error{Code: ErrorInvalidType, Detail: fmt.Sprintf("invalid alignment %d", alignment)}
	}
	mask := alignment - 1
	if value > math.MaxUint64-mask {
		return 0, Error{Code: ErrorOverflow, Detail: "aligned layout size exceeds uint64"}
	}
	return (value + mask) &^ mask, nil
}

// Code extracts a fixture-visible layout error code.
func Code(err error) (ErrorCode, bool) {
	var value Error
	if !errors.As(err, &value) {
		return "", false
	}
	return value.Code, true
}
