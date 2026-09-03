// Package diagnostic defines source-stable diagnostics shared by Stage 0 frontend packages.
package diagnostic

import "slices"

// CatalogVersion identifies the independently versioned Stage 0 diagnostic rule catalog.
const CatalogVersion = "fairyc.diagnostic-catalog/v0"

// CatalogEntry is the human-facing, source-independent explanation of one diagnostic code.
// It deliberately does not claim that the diagnostic is implemented or fixture-backed.
type CatalogEntry struct {
	Code     string
	Category string
	Meaning  string
}

const (
	categorySyntax        = "Syntax, modules, and imports"
	categoryResolution    = "Name resolution and source visibility"
	categoryType          = "Type and initialization"
	categoryResource      = "Resource ownership and drop"
	categoryBorrow        = "Borrowing and return origins"
	categoryRegion        = "Regions and arenas"
	categoryResult        = "Option, Result, and propagation"
	categoryUse           = "Return-value use obligations"
	categoryCompileTime   = "Bounded compile-time evaluation"
	categoryUnsafe        = "Unsafe and foreign boundaries"
	categoryABI           = "Target layout and ABI"
	categoryEffect        = "Effects and external capabilities"
	categoryState         = "Package-owned contract state"
	categoryCompatibility = "Compatibility and toolchain selection"
	categoryTest          = "Test declarations and harness"
	categoryAtlas         = "Atlas warnings or hard map-integrity errors"
	categoryVerifier      = "MIR verifier defects"
)

var catalogEntries = []CatalogEntry{
	{Code: "E0001", Category: categorySyntax, Meaning: "Source is not valid UTF-8 or identifier syntax is unsupported."},
	{Code: "E0002", Category: categorySyntax, Meaning: "Module declaration does not match its package-relative file identity."},
	{Code: "E0003", Category: categorySyntax, Meaning: "Import lacks an alias or uses a forbidden wildcard, selective, cyclic, or re-export form."},
	{Code: "E0004", Category: categorySyntax, Meaning: "Integer or floating literal has an invalid radix digit, separator, or excluded spelling."},
	{Code: "E0005", Category: categorySyntax, Meaning: "String-family literal has an unknown, truncated, out-of-range, or disallowed escape."},
	{Code: "E0006", Category: categorySyntax, Meaning: "C-string literal contains a decoded interior NUL."},
	{Code: "E0007", Category: categorySyntax, Meaning: "Parser inserted one required missing token at a zero-width byte span."},
	{Code: "E0008", Category: categorySyntax, Meaning: "Parser skipped an unexpected real token into an error node."},
	{Code: "E0009", Category: categorySyntax, Meaning: "Comparison operators are chained without explicit parentheses."},
	{Code: "E0010", Category: categorySyntax, Meaning: "Shared parser budget is exhausted before a complete bounded result is available."},
	{Code: "E0101", Category: categoryResolution, Meaning: "Name or qualified path cannot be resolved in its owning scope."},
	{Code: "E0102", Category: categoryResolution, Meaning: "Declaration duplicates a name in the same owning scope."},
	{Code: "E0103", Category: categoryResolution, Meaning: "Resolved item is not source-visible from the use module."},
	{Code: "E1001", Category: categoryType, Meaning: "Source local declaration has no initializer."},
	{Code: "E1002", Category: categoryType, Meaning: "Inert struct contains a resource."},
	{Code: "E1003", Category: categoryType, Meaning: "Named aggregate contains a safe borrow."},
	{Code: "E1004", Category: categoryType, Meaning: "Restricted user enum contains a resource or safe-borrow payload."},
	{Code: "E1005", Category: categoryType, Meaning: "Byte view is used as text without explicit validation."},
	{Code: "E1006", Category: categoryType, Meaning: "Expression type is incompatible with its expected type."},
	{Code: "E1007", Category: categoryType, Meaning: "Condition or logical operand is not `bool`."},
	{Code: "E1008", Category: categoryType, Meaning: "Assignment target is not a mutable place."},
	{Code: "E1009", Category: categoryType, Meaning: "Return expression is incompatible with the declared return type."},
	{Code: "E1010", Category: categoryType, Meaning: "Call argument count does not match the resolved declaration."},
	{Code: "E1012", Category: categoryType, Meaning: "`break` or `continue` has no enclosing loop."},
	{Code: "E1013", Category: categoryType, Meaning: "Integer literal is outside the range of its selected `i32`, `u8`, `u32`, or `usize` type."},
	{Code: "E1101", Category: categoryResource, Meaning: "Resource used after move or consumption."},
	{Code: "E1102", Category: categoryResource, Meaning: "Operation would create or execute a second cleanup obligation."},
	{Code: "E1103", Category: categoryResource, Meaning: "Resource field move is not supported."},
	{Code: "E1104", Category: categoryResource, Meaning: "Branch join leaves a resource `MaybeLive`."},
	{Code: "E1105", Category: categoryResource, Meaning: "Loop back edge changes the required ownership state."},
	{Code: "E1106", Category: categoryResource, Meaning: "Fallible `drop` or ambiguous close ownership."},
	{Code: "E1107", Category: categoryResource, Meaning: "`@abilities` is empty, duplicated, malformed, names an unknown resource ability, or declares `key` without `store`."},
	{Code: "E1108", Category: categoryResource, Meaning: "Compiler-scheduled implicit destruction reaches a resource whose explicit ability set omits `drop`."},
	{Code: "E1109", Category: categoryResource, Meaning: "Explicit `burn` is not authorized for this value, resource declaration, or module."},
	{Code: "E2101", Category: categoryBorrow, Meaning: "Access conflicts with an active shared or mutable loan."},
	{Code: "E2102", Category: categoryBorrow, Meaning: "Returned borrow originates from a local value."},
	{Code: "E2103", Category: categoryBorrow, Meaning: "Returned borrow has multiple possible input origins."},
	{Code: "E2104", Category: categoryBorrow, Meaning: "Borrow-returning function cannot become a function value or callback."},
	{Code: "E2105", Category: categoryBorrow, Meaning: "Borrow outlives or moves its owner."},
	{Code: "E3101", Category: categoryRegion, Meaning: "Region-derived value escapes the region."},
	{Code: "E3102", Category: categoryRegion, Meaning: "Plain region allocation contains a type that needs drop."},
	{Code: "E3103", Category: categoryRegion, Meaning: "Region capability is stored, captured, moved out, or returned."},
	{Code: "E3104", Category: categoryRegion, Meaning: "Source attempts to construct a compiler-owned `Region` capability."},
	{Code: "E4101", Category: categoryResult, Meaning: "`Result` value is not handled, returned, propagated, or explicitly discarded."},
	{Code: "E4102", Category: categoryResult, Meaning: "`?` is applied to a non-`Result` value."},
	{Code: "E4103", Category: categoryResult, Meaning: "`?` error type is incompatible with the enclosing return type."},
	{Code: "E4201", Category: categoryUse, Meaning: "`@must_use` is malformed, duplicated, misplaced, or applied to an ineligible result."},
	{Code: "E4202", Category: categoryUse, Meaning: "A resolved `@must_use` call result is ignored without explicit `discard`."},
	{Code: "E4301", Category: categoryCompileTime, Meaning: "A module constant or `@const_eval` declaration is malformed, misplaced, public, or otherwise ineligible."},
	{Code: "E4302", Category: categoryCompileTime, Meaning: "A well-typed expression is outside the admitted compile-time subset."},
	{Code: "E4303", Category: categoryCompileTime, Meaning: "Compile-time evaluation traps on arithmetic, division, shift, or cast semantics."},
	{Code: "E4304", Category: categoryCompileTime, Meaning: "A compile-time dependency cycle or deterministic evaluator budget prevents a result."},
	{Code: "E4305", Category: categoryCompileTime, Meaning: "A module-level `static_assert` evaluates to false."},
	{Code: "E4401", Category: categoryResolution, Meaning: "The reserved `source_location` intrinsic name is declared, bound, or used without its exact call form."},
	{Code: "E4402", Category: categorySyntax, Meaning: "A `source_location()` call has no valid canonical source path or exact retained line geometry."},
	{Code: "E5101", Category: categoryUnsafe, Meaning: "Raw-pointer or unchecked operation appears outside an unsafe boundary."},
	{Code: "E5102", Category: categoryUnsafe, Meaning: "Raw foreign signature contains a safe reference."},
	{Code: "E5103", Category: categoryUnsafe, Meaning: "Foreign unwind, exception, or `longjmp` may cross a Fairy C frame."},
	{Code: "E5104", Category: categoryUnsafe, Meaning: "Safe foreign callback is not supported in Stage 0."},
	{Code: "E5105", Category: categoryUnsafe, Meaning: "A retained foreign-pointer API received stack or region-borrowed storage."},
	{Code: "E5106", Category: categoryUnsafe, Meaning: "`@link_name` is duplicated or does not contain exactly one nonempty string literal."},
	{Code: "E6101", Category: categoryABI, Meaning: "Type has no supported ABI classification for the selected target."},
	{Code: "E6102", Category: categoryABI, Meaning: "Required target layout fact is unknown."},
	{Code: "E6103", Category: categoryABI, Meaning: "C ABI v0 excludes the requested calling, layout, callback, TLS, or variadic feature."},
	{Code: "E7101", Category: categoryEffect, Meaning: "`@effects` is empty, duplicated, malformed, or names an effect outside the Stage 0 vocabulary."},
	{Code: "E7102", Category: categoryEffect, Meaning: "A call or compiler-scheduled resource cleanup requires one or more effects not declared by the enclosing function."},
	{Code: "E7201", Category: categoryState, Meaning: "A package state declaration has malformed UID authority, invalid visibility, or more than one package owner."},
	{Code: "E7202", Category: categoryState, Meaning: "A package state field name is declared more than once."},
	{Code: "E7203", Category: categoryState, Meaning: "A package state field uses an unsupported or malformed storage codec declaration."},
	{Code: "E8001", Category: categoryCompatibility, Meaning: "Selected runtime ABI is incompatible with generated code."},
	{Code: "E8002", Category: categoryCompatibility, Meaning: "Requested language revision is unsupported by this compiler."},
	{Code: "E9001", Category: categoryTest, Meaning: "`@test` must appear exactly once without arguments on a private safe zero-argument free function returning `void` in isolated test source."},
	{Code: "A2001", Category: categoryAtlas, Meaning: "Dependency map is incompatible; affected Atlas relations are partial or unavailable while compilation remains valid."},
	{Code: "A9001", Category: categoryAtlas, Meaning: "Deterministic symbol identity collision or corrupt identity evidence; the map is rejected."},
	{Code: "V1001", Category: categoryVerifier, Meaning: "Every reachable block ends in exactly one terminator; each operation has its required shape and concrete type identity; every executable aggregate use agrees with one valid program-level definition; internal function symbols and call signature shapes are closed and consistent; and foreign declarations mapped to one link name have compatible signatures."},
	{Code: "V1002", Category: categoryVerifier, Meaning: "Every use is dominated by an initialized assignment on all incoming paths."},
	{Code: "V1101", Category: categoryVerifier, Meaning: "A resource root is moved, dropped, or burned at most once."},
	{Code: "V1102", Category: categoryVerifier, Meaning: "A live resource on a normal exiting edge has exactly one cleanup obligation."},
	{Code: "V1103", Category: categoryVerifier, Meaning: "No resource projection appears in a `Move` operand."},
	{Code: "V1104", Category: categoryVerifier, Meaning: "`Burn` discharges exactly one live direct resource root with its exact declared type and no cleanup metadata."},
	{Code: "V1201", Category: categoryVerifier, Meaning: "A drop flag is initialized before use and agrees with ownership transfer."},
	{Code: "V1202", Category: categoryVerifier, Meaning: "`Drop` clears its active flag before invoking cleanup."},
	{Code: "V1301", Category: categoryVerifier, Meaning: "No safe borrow outlives or conflicts with its verified origin and loan."},
	{Code: "V1302", Category: categoryVerifier, Meaning: "A safe-borrow return has exactly one permitted origin."},
	{Code: "V1401", Category: categoryVerifier, Meaning: "A region-derived value does not escape its region."},
	{Code: "V1402", Category: categoryVerifier, Meaning: "Plain-region allocation never receives a `NeedsDrop` type."},
	{Code: "V1403", Category: categoryVerifier, Meaning: "Normal control flow enters each lexical Region from one verified live mutable Arena owner, places into it, and resets it exactly once."},
	{Code: "V1501", Category: categoryVerifier, Meaning: "Every potentially trapping operation has one complete checked predecessor with matching marker, operands, operator, categories, success edge, and non-returning trap edge."},
	{Code: "V1502", Category: categoryVerifier, Meaning: "A Stage 0 call has no unwind successor."},
	{Code: "V1601", Category: categoryVerifier, Meaning: "`?`, implicit drop, source short-circuiting, and nested side effects do not remain."},
	{Code: "V1701", Category: categoryVerifier, Meaning: "Every unsafe instruction carries a source span and unsafe-scope identity."},
}

// Lookup returns the exact canonical catalog entry for code.
func Lookup(code string) (CatalogEntry, bool) {
	for _, entry := range catalogEntries {
		if entry.Code == code {
			return entry, true
		}
	}
	return CatalogEntry{}, false
}

// CatalogEntries returns a defensive copy of every reserved prototype entry.
func CatalogEntries() []CatalogEntry {
	return slices.Clone(catalogEntries)
}
