package ast

import "github.com/0disoft/fairyc/internal/syntax"

// PrivateSharedReborrow is inserted only for a type-checked call argument.
// It is not syntax and must retain its original source operand for Flow.
const PrivateSharedReborrow syntax.Kind = "shared-reborrow"
