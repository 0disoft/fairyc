package interpreter

import (
	"github.com/0disoft/fairyc/internal/mir"
)

type valueKind uint8

const (
	valueI32 valueKind = iota + 1
	valueI64
	valueU8
	valueU32
	valueUSize
	valueArray
	valueView
	valueNull
	valueResult
	valueOption
	valueEnum
	valueStruct
	valueResource
	valueVoid
)

type structState struct {
	definition *mir.StructType
	fields     []runtimeValue
}

type enumState struct {
	definition *mir.EnumType
	variant    string
	fields     []runtimeValue
}

type resourceState struct {
	definition *mir.ResourceType
	fields     []runtimeValue
	dropping   bool
	dropped    bool
}

type arrayState struct {
	elements []runtimeValue
}

type indexedViewState struct {
	staticData string
	array      *arrayState
	offset     uint64
	length     uint64
	mutable    bool
}

type taggedState struct {
	variant string
	payload *runtimeValue
	dropped bool
}

type scalarReference struct {
	array  *arrayState
	index  uint64
	static *indexedViewState
	fields *[]runtimeValue
	field  int
	locals map[mir.LocalID]runtimeValue
	local  mir.LocalID
}
