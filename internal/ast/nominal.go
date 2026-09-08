package ast

// TypeKindBoundNominal is an internal reference emitted by a checked generic
// expansion. Path is an opaque binding key, not a source module/type spelling.
// Ordinary semantic entry points reject it without the explicit binding table.
const TypeKindBoundNominal TypeKind = "bound-nominal"
