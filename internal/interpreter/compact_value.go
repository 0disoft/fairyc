package interpreter

import "github.com/0disoft/fairyc/internal/hir"

// Scalars stay inline. Compound values retain exactly one existing state pointer;
// the closed payload interface cannot box scalar values or unrelated objects.
type runtimeValue struct {
	kind    valueKind
	typ     hir.Type
	scalar  uint64
	payload valuePayload
}

type valuePayload interface{ runtimeValuePayload() }

func (*taggedState) runtimeValuePayload()      {}
func (*enumState) runtimeValuePayload()        {}
func (*structState) runtimeValuePayload()      {}
func (*resourceState) runtimeValuePayload()    {}
func (*arrayState) runtimeValuePayload()       {}
func (*indexedViewState) runtimeValuePayload() {}
func (*scalarReference) runtimeValuePayload()  {}
func (*runtimeRegion) runtimeValuePayload()    {}

func i32Bits(value int32) uint64   { return uint64(uint32(value)) }
func i64Bits(value int64) uint64   { return uint64(value) }
func (v runtimeValue) i32() int32  { return int32(v.scalar) }
func (v runtimeValue) i64() int64  { return int64(v.scalar) }
func (v runtimeValue) u64() uint64 { return v.scalar }

func (v runtimeValue) tagged() *taggedState     { state, _ := v.payload.(*taggedState); return state }
func (v runtimeValue) enum() *enumState         { state, _ := v.payload.(*enumState); return state }
func (v runtimeValue) structure() *structState  { state, _ := v.payload.(*structState); return state }
func (v runtimeValue) resource() *resourceState { state, _ := v.payload.(*resourceState); return state }
func (v runtimeValue) array() *arrayState       { state, _ := v.payload.(*arrayState); return state }
func (v runtimeValue) view() *indexedViewState {
	state, _ := v.payload.(*indexedViewState)
	return state
}
func (v runtimeValue) reference() *scalarReference {
	state, _ := v.payload.(*scalarReference)
	return state
}
