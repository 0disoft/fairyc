package cbackendnative

const cStringLiteralBuildSmokeSource = `module main;
fn i32 main() {
    return if (unsafe { c"fairy" as ptr u8 } != null) { 42 } else { 1 };
}
`

const byteStringLiteralBuildSmokeSource = `module main;
fn i32 main() {
    let ref u8[] empty = b"";
    let ref u8[] bytes = b"fairy";
    return if (unsafe { bytes as ptr u8 } != null) { 42 } else { 1 };
}
`

const utf8StringLiteralBuildSmokeSource = `module main;
fn i32 main() {
    let ref str empty = "";
    let ref str text = "Fairy λ";
    return if (unsafe { text as ptr u8 } != null) { 42 } else { 1 };
}
`

const utf8StringLiteralSourceCallSmokeSource = `module main;
@link_name("fairyc_fixture_is_fairy_utf8")
unsafe extern C fn u32 is_fairy_utf8(ptr u8 value) ;
fn i32 main() {
    let ref str text = "Fairy λ";
    return if (unsafe { is_fairy_utf8(text as ptr u8) } == 1) { 42 } else { 1 };
}
`

const utf8StringLiteralSourceCallCompanion = `#include <stdint.h>

uint32_t fairyc_fixture_is_fairy_utf8(const uint8_t *value) {
    static const uint8_t expected[8] = {
        UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114),
        UINT8_C(121), UINT8_C(32), UINT8_C(206), UINT8_C(187)
    };
    if (value == 0) {
        return UINT32_C(0);
    }
    for (uint32_t index = 0; index < UINT32_C(8); index++) {
        if (value[index] != expected[index]) {
            return UINT32_C(0);
        }
    }
    return UINT32_C(1);
}
`

var utf8StringLiteralSourceCallExpectedCFragments = []string{
	"typedef struct fy_str",
	"static const uint8_t fy_static_str_",
	"UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(32), UINT8_C(206), UINT8_C(187)",
	".length = ((uintptr_t)UINT64_C(8))",
	"extern uint32_t fairyc_fixture_is_fairy_utf8(const uint8_t * fy_a_0);",
	".data;",
	"fairyc_fixture_is_fairy_utf8(fy_l_",
}

const byteStringLiteralSourceCallSmokeSource = `module main;
@link_name("fairyc_fixture_is_fairy_bytes")
unsafe extern C fn u32 is_fairy_bytes(ptr u8 value) ;
fn i32 main() {
    let ref u8[] bytes = b"fairy";
    return if (unsafe { is_fairy_bytes(bytes as ptr u8) } == 1) { 42 } else { 1 };
}
`

const byteStringLiteralSourceCallCompanion = `#include <stdint.h>

uint32_t fairyc_fixture_is_fairy_bytes(const uint8_t *value) {
    static const uint8_t expected[5] = {
        UINT8_C(102), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)
    };
    if (value == 0) {
        return UINT32_C(0);
    }
    for (uint32_t index = 0; index < UINT32_C(5); index++) {
        if (value[index] != expected[index]) {
            return UINT32_C(0);
        }
    }
    return UINT32_C(1);
}
`

var byteStringLiteralSourceCallExpectedCFragments = []string{
	"typedef struct fy_byte_slice",
	"static const uint8_t fy_static_bytes_",
	"UINT8_C(102), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)",
	".length = ((uintptr_t)UINT64_C(5))",
	"extern uint32_t fairyc_fixture_is_fairy_bytes(const uint8_t * fy_a_0);",
	".data;",
	"fairyc_fixture_is_fairy_bytes(fy_l_",
}

const cStringLiteralSourceCallSmokeSource = `module main;
@link_name("fairyc_fixture_is_fairy")
unsafe extern C fn u32 is_fairy(ptr u8 value) ;
fn i32 main() {
    return if (unsafe { is_fairy(c"fairy" as ptr u8) } == 1) { 42 } else { 1 };
}
`

const cStringLiteralSourceCallCompanion = `#include <stdint.h>

uint32_t fairyc_fixture_is_fairy(const uint8_t *value) {
    static const uint8_t expected[6] = {
        UINT8_C(102), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(0)
    };
    if (value == 0) {
        return UINT32_C(0);
    }
    for (uint32_t index = 0; index < UINT32_C(6); index++) {
        if (value[index] != expected[index]) {
            return UINT32_C(0);
        }
    }
    return UINT32_C(1);
}
`

var cStringLiteralSourceCallExpectedCFragments = []string{
	"static const uint8_t fy_static_cstr_",
	"UINT8_C(102), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(0)",
	"extern uint32_t fairyc_fixture_is_fairy(const uint8_t * fy_a_0);",
	" = fy_static_cstr_",
	"fairyc_fixture_is_fairy(fy_l_",
}
