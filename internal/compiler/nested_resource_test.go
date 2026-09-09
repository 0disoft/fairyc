package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestPrivateVectorNestedResourceDrops(t *testing.T) {
	runPrivateVectorStorageNative(t, `module main;
@link_name("nested_record") unsafe extern C fn i32 record(i32 value);
@link_name("nested_mutate") unsafe extern C fn i32 mutate(ptr mut void value);
@link_name("nested_finished") unsafe extern C fn i32 finished();
resource Leaf { i32 value; cleanup { assert(unsafe { record(this.value) } == 1); } }
resource Box {
    Leaf first; Leaf last;
    cleanup {
        assert(unsafe { record(10) } == 1);
        assert(unsafe { mutate(ref mut this as ptr mut void) } == 1);
    }
}
resource Envelope { Box value; cleanup {} }
fn Box make() { return Box { .first = Leaf { .value = 1 }, .last = Leaf { .value = 2 } }; }
fn i32 exercise() {
    { let Envelope value = Envelope { .value = make() }; }
    { let Option<Box> empty = .none; }
    { let Option<Box> some = .some(make()); }
    { let Result<Box, i32> failed = .err(5);
      assert(switch (failed) { case .ok(_) : { false } case .err(code) : { code == 5 } }); }
    { let Result<Box, i32> passed = .ok(make());
      assert(switch (passed) { case .ok(_) : { true } case .err(_) : { false } }); }
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`, `
static const int32_t nested_expected[] = {10, 2, 99, 10, 2, 99, 10, 2, 99};
static size_t nested_position = 0;
int32_t nested_record(int32_t value) {
    if (nested_position >= sizeof nested_expected / sizeof nested_expected[0] ||
        nested_expected[nested_position] != value) return 0;
    nested_position++;
    return 1;
}
int32_t nested_mutate(void *value) {
    /* Box.first and Leaf.value are their respective first fields. */
    int32_t replacement = 99;
    memcpy(value, &replacement, sizeof replacement);
    return 1;
}
int32_t nested_finished(void) { return nested_position == 9 ? 1 : 0; }
`)
}

func TestPrivateVectorNestedResourceTokenOwnership(t *testing.T) {
	input, coordinates := vectorSDKInputWithCandidate(t, nestedTokenProgram, true)
	result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("nested token source: %v diagnostics=%+v unsupported=%+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	runVectorStorageMIRNativeLane(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", nestedTokenProbe, true)
}

const nestedTokenProgram = `module main;
import std::vec as v;
import std::bytes as bytes;
@link_name("token_fail_next") unsafe extern C fn i32 fail_next();
@link_name("token_record") unsafe extern C fn i32 record(i32 value);
@link_name("token_finished") unsafe extern C fn i32 finished();
resource Token {
    i32 index;
    bytes::Buffer cooked;
    cleanup { assert(unsafe { record(this.index) } == 1); }
}
fn Token make(i32 index) {
    let bytes::Buffer cooked = switch (bytes::copy(b"Fairy")) {
        case .ok(value) : { value }
        case .err(_) : { assert(false); bytes::empty() }
    };
    return Token { .index = index, .cooked = cooked };
}
fn bool valid(ref Token value, i32 index) {
    let ref u8[] cooked = bytes::view(ref value.cooked);
    return value.index == index && cooked.len == 5 && cooked[0] == 70 && cooked[4] == 121;
}
fn bool push(ref mut v::Vec<Token> tokens, Token value) {
    return switch (v::push<Token>(tokens, value)) { case .ok(_) : { true } case .err(_) : { false } };
}
fn bool retry(ref mut v::Vec<Token> tokens, Token value, i32 index) {
    assert(unsafe { fail_next() } == 1);
    return switch (v::push<Token>(tokens, value)) {
        case .ok(_) : { false }
        case .err(returned) : { assert(valid(ref returned, index)); push(tokens, returned) }
    };
}
fn i32 exercise() {
    var v::Vec<Token> tokens = v::empty<Token>();
    assert(retry(ref mut tokens, make(0), 0));
    for (var i32 index = 1; index < 8; index += 1) { assert(push(ref mut tokens, make(index))); }
    assert(retry(ref mut tokens, make(8), 8));
    assert(valid(v::at<Token>(ref tokens, 8), 8));
    let Token popped = switch (v::pop<Token>(ref mut tokens)) {
        case .some(value) : { value }
        case .none : { assert(false); make(-1) }
    };
    assert(valid(ref popped, 8));
    let usize capacity = v::capacity<Token>(ref tokens);
    assert(v::truncate<Token>(ref mut tokens, 100) == 0);
    assert(v::truncate<Token>(ref mut tokens, 8) == 0);
    assert(v::truncate<Token>(ref mut tokens, 4) == 4);
    assert(v::len<Token>(ref tokens) == 4);
    assert(v::capacity<Token>(ref tokens) == capacity);
    assert(valid(v::at<Token>(ref tokens, 3), 3));
    assert(v::truncate<Token>(ref mut tokens, 4) == 0);
    assert(v::clear<Token>(ref mut tokens) == 4);
    assert(v::truncate<Token>(ref mut tokens, 0) == 0);
    assert(push(ref mut tokens, make(9)));
    return 41;
}
fn i32 main() { return exercise() + unsafe { finished() }; }
`

const nestedTokenProbe = `
static const int32_t token_expected[] = {7, 6, 5, 4, 3, 2, 1, 0, 8, 9};
static size_t token_position = 0;
int32_t token_fail_next(void) { vector_fail = 1; return 1; }
int32_t token_record(int32_t value) {
    if (token_position >= sizeof token_expected / sizeof token_expected[0] ||
        token_expected[token_position] != value) return 0;
    token_position++;
    return 1;
}
int32_t token_finished(void) {
    return token_position == 10 && vector_live == 0 && vector_allocations == vector_frees &&
        vector_allocations > 10 && vector_failures == 2 ? 1 : 0;
}
`
