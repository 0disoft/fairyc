package hir

// ClosureTarget is a checked safe pure lifted function. CaptureCount includes
// only its leading immutable i32 parameters, excluding its one i32 value arg.
// Verifiers must recheck the actual declaration/body, not trust this fact alone.
type ClosureTarget struct{ CaptureCount int }
