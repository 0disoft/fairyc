package ast

// Function.SourceGeneric records the written declaration before specialization
// erases TypeParameters. Like lifting metadata, it is excluded from source JSON.

// LiftedClosure is private source-preparation metadata, excluded from AST JSON.
// A nonnil record distinguishes a zero-capture value target from an ordinary
// function. Captures precede the one written i32 value parameter.
type LiftedClosure struct{ CaptureCount int }

// ClosureValue names the actual lifted function in the prepared module.
// Its expression Children are captured scalar reads in written order.
type ClosureValue struct{ Target string }
