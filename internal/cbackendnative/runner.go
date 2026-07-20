// Package cbackendnative compiles and executes constrained C fixtures on the declared Linux target.
package cbackendnative

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cabifixture"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cbackendfixture"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	reportSchemaVersion = "fairyc.native-c-report/v31"
	trapExitBase        = 192
	maxProcessOutput    = 64 * 1024
	executionTimeout    = 10 * time.Second
)

var nativeProfiles = []cdriver.Profile{
	cdriver.LowProfile,
	cdriver.OptimizedProfile,
}

const buildSmokeSource = `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn choose() -> Option<Token> { Option::Some(Token { value: 41 }) }
fn increment(input: Result<Token, i32>) -> Result<Token, i32> {
	let token: Token = input?;
	Result::Ok(Token { value: token.value + 1 })
}
fn main() -> i32 {
	let token: Token = match choose() {
		Option::Some(token) => token,
		Option::None => Token { value: 0 },
	};
	match increment(Result::Ok(token)) {
		Result::Ok(token) => token.value,
		Result::Err(error) => -error,
    }
}
`

const cStringLiteralBuildSmokeSource = `module main;
fn main() -> i32 {
    if unsafe { c"fairy" as *u8 } != null { 42 } else { 1 }
}
`

const byteStringLiteralBuildSmokeSource = `module main;
fn main() -> i32 {
    let empty: &[u8] = b"";
    let bytes: &[u8] = b"fairy";
    if unsafe { bytes as *u8 } != null { 42 } else { 1 }
}
`

const utf8StringLiteralBuildSmokeSource = `module main;
fn main() -> i32 {
    let empty: &str = "";
    let text: &str = "Fairy λ";
    if unsafe { text as *u8 } != null { 42 } else { 1 }
}
`

const utf8StringLiteralSourceCallSmokeSource = `module main;
@link_name("fairyc_fixture_is_fairy_utf8")
unsafe extern C fn is_fairy_utf8(value: *u8) -> u32;
fn main() -> i32 {
    let text: &str = "Fairy λ";
    if unsafe { is_fairy_utf8(text as *u8) } == 1 { 42 } else { 1 }
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
unsafe extern C fn is_fairy_bytes(value: *u8) -> u32;
fn main() -> i32 {
    let bytes: &[u8] = b"fairy";
    if unsafe { is_fairy_bytes(bytes as *u8) } == 1 { 42 } else { 1 }
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
unsafe extern C fn is_fairy(value: *u8) -> u32;
fn main() -> i32 {
    if unsafe { is_fairy(c"fairy" as *u8) } == 1 { 42 } else { 1 }
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

const foreignCallSmokeSource = `module main;
@link_name("fairyc_fixture_add_one")
unsafe extern C fn add_one(value: i32) -> i32;
fn main() -> i32 { unsafe { add_one(41) } }
`

const foreignCallCompanionSource = `#include <stdint.h>

int32_t fairyc_fixture_add_one(int32_t value) {
    return value + INT32_C(1);
}
`

const foreignCIntCallSmokeSource = `module main;
@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn make_c_int() -> c::int;
@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn consume_c_int(value: c::int) -> i32;
fn main() -> i32 { unsafe { consume_c_int(make_c_int()) } }
`

const foreignCIntCallCompanionSource = `#include <stdint.h>

int fairyc_fixture_make_c_int(void) {
    return 41;
}

int32_t fairyc_fixture_consume_c_int(int value) {
    return value == 41 ? INT32_C(42) : INT32_C(1);
}
`

const foreignCLongCallSmokeSource = `module main;
@link_name("fairyc_fixture_make_c_long")
unsafe extern C fn make_c_long() -> c::long;
@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn consume_c_long(value: c::long) -> i32;
fn main() -> i32 { unsafe { consume_c_long(make_c_long()) } }
`

const foreignCLongCallCompanionSource = `#include <stdint.h>

long fairyc_fixture_make_c_long(void) {
    return -41L;
}

int32_t fairyc_fixture_consume_c_long(long value) {
    return value == -41L && value < 0L ? INT32_C(42) : INT32_C(1);
}
`

const foreignCSizeTCallSmokeSource = `module main;
@link_name("fairyc_fixture_make_c_size_t")
unsafe extern C fn make_c_size_t() -> c::size_t;
@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn consume_c_size_t(value: c::size_t) -> i32;
fn main() -> i32 { unsafe { consume_c_size_t(make_c_size_t()) } }
`

const foreignCSizeTCallCompanionSource = `#include <stddef.h>
#include <stdint.h>

size_t fairyc_fixture_make_c_size_t(void) {
    return (size_t)41;
}

int32_t fairyc_fixture_consume_c_size_t(size_t value) {
    return value == (size_t)41 ? INT32_C(42) : INT32_C(1);
}
`

const foreignCSSizeTCallSmokeSource = `module main;
@link_name("fairyc_fixture_make_c_ssize_t")
unsafe extern C fn make_c_ssize_t() -> c::ssize_t;
@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn consume_c_ssize_t(value: c::ssize_t) -> i32;
fn main() -> i32 { unsafe { consume_c_ssize_t(make_c_ssize_t()) } }
`

const foreignCSSizeTCallCompanionSource = `#define _POSIX_C_SOURCE 200809L

#include <stdint.h>
#include <sys/types.h>

ssize_t fairyc_fixture_make_c_ssize_t(void) {
    return (ssize_t)-41;
}

int32_t fairyc_fixture_consume_c_ssize_t(ssize_t value) {
    return value == (ssize_t)-41 && value < (ssize_t)0 ? INT32_C(42) : INT32_C(1);
}
`

const foreignCScalarFamilyCallSmokeSource = `module main;
unsafe extern C fn fairyc_fixture_make_c_bool() -> c::bool;
unsafe extern C fn fairyc_fixture_consume_c_bool(value: c::bool) -> i32;
unsafe extern C fn fairyc_fixture_make_c_char() -> c::char;
unsafe extern C fn fairyc_fixture_consume_c_char(value: c::char) -> i32;
unsafe extern C fn fairyc_fixture_make_c_double() -> c::double;
unsafe extern C fn fairyc_fixture_consume_c_double(value: c::double) -> i32;
unsafe extern C fn fairyc_fixture_make_c_float() -> c::float;
unsafe extern C fn fairyc_fixture_consume_c_float(value: c::float) -> i32;
unsafe extern C fn fairyc_fixture_make_c_longlong() -> c::longlong;
unsafe extern C fn fairyc_fixture_consume_c_longlong(value: c::longlong) -> i32;
unsafe extern C fn fairyc_fixture_make_c_schar() -> c::schar;
unsafe extern C fn fairyc_fixture_consume_c_schar(value: c::schar) -> i32;
unsafe extern C fn fairyc_fixture_make_c_short() -> c::short;
unsafe extern C fn fairyc_fixture_consume_c_short(value: c::short) -> i32;
unsafe extern C fn fairyc_fixture_make_c_uchar() -> c::uchar;
unsafe extern C fn fairyc_fixture_consume_c_uchar(value: c::uchar) -> i32;
unsafe extern C fn fairyc_fixture_make_c_uint() -> c::uint;
unsafe extern C fn fairyc_fixture_consume_c_uint(value: c::uint) -> i32;
unsafe extern C fn fairyc_fixture_make_c_ulong() -> c::ulong;
unsafe extern C fn fairyc_fixture_consume_c_ulong(value: c::ulong) -> i32;
unsafe extern C fn fairyc_fixture_make_c_ulonglong() -> c::ulonglong;
unsafe extern C fn fairyc_fixture_consume_c_ulonglong(value: c::ulonglong) -> i32;
unsafe extern C fn fairyc_fixture_make_c_ushort() -> c::ushort;
unsafe extern C fn fairyc_fixture_consume_c_ushort(value: c::ushort) -> i32;

fn check_bool() -> i32 { unsafe { fairyc_fixture_consume_c_bool(fairyc_fixture_make_c_bool()) } }
fn check_char() -> i32 { unsafe { fairyc_fixture_consume_c_char(fairyc_fixture_make_c_char()) } }
fn check_double() -> i32 { unsafe { fairyc_fixture_consume_c_double(fairyc_fixture_make_c_double()) } }
fn check_float() -> i32 { unsafe { fairyc_fixture_consume_c_float(fairyc_fixture_make_c_float()) } }
fn check_longlong() -> i32 { unsafe { fairyc_fixture_consume_c_longlong(fairyc_fixture_make_c_longlong()) } }
fn check_schar() -> i32 { unsafe { fairyc_fixture_consume_c_schar(fairyc_fixture_make_c_schar()) } }
fn check_short() -> i32 { unsafe { fairyc_fixture_consume_c_short(fairyc_fixture_make_c_short()) } }
fn check_uchar() -> i32 { unsafe { fairyc_fixture_consume_c_uchar(fairyc_fixture_make_c_uchar()) } }
fn check_uint() -> i32 { unsafe { fairyc_fixture_consume_c_uint(fairyc_fixture_make_c_uint()) } }
fn check_ulong() -> i32 { unsafe { fairyc_fixture_consume_c_ulong(fairyc_fixture_make_c_ulong()) } }
fn check_ulonglong() -> i32 { unsafe { fairyc_fixture_consume_c_ulonglong(fairyc_fixture_make_c_ulonglong()) } }
fn check_ushort() -> i32 { unsafe { fairyc_fixture_consume_c_ushort(fairyc_fixture_make_c_ushort()) } }

fn main() -> i32 {
    let failures: i32 = check_bool() + check_char() + check_double() + check_float() +
        check_longlong() + check_schar() + check_short() + check_uchar() + check_uint() +
        check_ulong() + check_ulonglong() + check_ushort();
    if failures == 0 { 42 } else { 1 }
}
`

const foreignCScalarFamilyCallCompanionSource = `#include <stdint.h>

_Bool fairyc_fixture_make_c_bool(void) { return (_Bool)1; }
int32_t fairyc_fixture_consume_c_bool(_Bool value) { return value == (_Bool)1 ? INT32_C(0) : INT32_C(1); }
char fairyc_fixture_make_c_char(void) { return (char)-41; }
int32_t fairyc_fixture_consume_c_char(char value) { return value == (char)-41 && value < (char)0 ? INT32_C(0) : INT32_C(1); }
double fairyc_fixture_make_c_double(void) { return -1.5; }
int32_t fairyc_fixture_consume_c_double(double value) { return value == -1.5 ? INT32_C(0) : INT32_C(1); }
float fairyc_fixture_make_c_float(void) { return -1.5F; }
int32_t fairyc_fixture_consume_c_float(float value) { return value == -1.5F ? INT32_C(0) : INT32_C(1); }
long long fairyc_fixture_make_c_longlong(void) { return -41LL; }
int32_t fairyc_fixture_consume_c_longlong(long long value) { return value == -41LL ? INT32_C(0) : INT32_C(1); }
signed char fairyc_fixture_make_c_schar(void) { return (signed char)-41; }
int32_t fairyc_fixture_consume_c_schar(signed char value) { return value == (signed char)-41 ? INT32_C(0) : INT32_C(1); }
short fairyc_fixture_make_c_short(void) { return (short)-41; }
int32_t fairyc_fixture_consume_c_short(short value) { return value == (short)-41 ? INT32_C(0) : INT32_C(1); }
unsigned char fairyc_fixture_make_c_uchar(void) { return (unsigned char)41; }
int32_t fairyc_fixture_consume_c_uchar(unsigned char value) { return value == (unsigned char)41 ? INT32_C(0) : INT32_C(1); }
unsigned int fairyc_fixture_make_c_uint(void) { return 41U; }
int32_t fairyc_fixture_consume_c_uint(unsigned int value) { return value == 41U ? INT32_C(0) : INT32_C(1); }
unsigned long fairyc_fixture_make_c_ulong(void) { return 41UL; }
int32_t fairyc_fixture_consume_c_ulong(unsigned long value) { return value == 41UL ? INT32_C(0) : INT32_C(1); }
unsigned long long fairyc_fixture_make_c_ulonglong(void) { return 41ULL; }
int32_t fairyc_fixture_consume_c_ulonglong(unsigned long long value) { return value == 41ULL ? INT32_C(0) : INT32_C(1); }
unsigned short fairyc_fixture_make_c_ushort(void) { return (unsigned short)41; }
int32_t fairyc_fixture_consume_c_ushort(unsigned short value) { return value == (unsigned short)41 ? INT32_C(0) : INT32_C(1); }
`

var foreignCScalarFamilyExpectedCFragments = []string{
	"_Static_assert(sizeof(_Bool) == UINT64_C(1), \"fairyc target c.bool size\");",
	"_Static_assert(((char)-1) < (char)0, \"fairyc target c.char signed\");",
	"extern double fairyc_fixture_make_c_double(void);",
	"extern int32_t fairyc_fixture_consume_c_float(float fy_a_0);",
	"extern long long fairyc_fixture_make_c_longlong(void);",
	"extern int32_t fairyc_fixture_consume_c_schar(signed char fy_a_0);",
	"extern short fairyc_fixture_make_c_short(void);",
	"extern int32_t fairyc_fixture_consume_c_uchar(unsigned char fy_a_0);",
	"extern unsigned int fairyc_fixture_make_c_uint(void);",
	"extern int32_t fairyc_fixture_consume_c_ulong(unsigned long fy_a_0);",
	"extern unsigned long long fairyc_fixture_make_c_ulonglong(void);",
	"extern int32_t fairyc_fixture_consume_c_ushort(unsigned short fy_a_0);",
}

const foreignPointerCallSmokeSource = `module main;
unsafe extern C fn fairyc_fixture_pointer_scalars(data: *u8, output: *mut u32, length: usize, marker: u8) -> u32;
fn main() -> i32 {
    if unsafe { fairyc_fixture_pointer_scalars(null, null, 0, 7) } == 42 { 42 } else { 1 }
}
`

const foreignPointerCallCompanionSource = `#include <stddef.h>
#include <stdint.h>

uint32_t fairyc_fixture_pointer_scalars(
    const uint8_t *data,
    uint32_t *output,
    uintptr_t length,
    uint8_t marker
) {
    if (data != NULL || output != NULL || length != (uintptr_t)0 || marker != UINT8_C(7)) {
        return UINT32_C(0);
    }
    return UINT32_C(42);
}
`

const foreignCStructResultCallSmokeSource = `module main;
@repr(C)
struct fairyc_fixture_status_v0 {
    status: u32;
    allocation_status: u32;
}
@repr(C)
struct fairyc_fixture_large_status_v0 {
    first: u32;
    second: u32;
    third: u32;
    fourth: u32;
    fifth: u32;
}
@link_name("fairyc_fixture_make_status_v0")
unsafe extern C fn make_status() -> fairyc_fixture_status_v0;
@link_name("fairyc_fixture_make_large_status_v0")
unsafe extern C fn make_large_status() -> fairyc_fixture_large_status_v0;
fn main() -> i32 {
    let result: fairyc_fixture_status_v0 = unsafe { make_status() };
    let large: fairyc_fixture_large_status_v0 = unsafe { make_large_status() };
    if result.status == 41 && result.allocation_status == 1 &&
        large.first == 1 && large.second == 2 && large.third == 3 &&
        large.fourth == 4 && large.fifth == 5 {
        42
    } else {
        0
    }
}
`

const foreignCStructResultCallCompanionSource = `#include <stdint.h>

struct fairyc_fixture_status_v0 {
    uint32_t status;
    uint32_t allocation_status;
};

struct fairyc_fixture_large_status_v0 {
    uint32_t first;
    uint32_t second;
    uint32_t third;
    uint32_t fourth;
    uint32_t fifth;
};

struct fairyc_fixture_status_v0 fairyc_fixture_make_status_v0(void) {
    struct fairyc_fixture_status_v0 result = {UINT32_C(41), UINT32_C(1)};
    return result;
}

struct fairyc_fixture_large_status_v0 fairyc_fixture_make_large_status_v0(void) {
    struct fairyc_fixture_large_status_v0 result = {
        UINT32_C(1), UINT32_C(2), UINT32_C(3), UINT32_C(4), UINT32_C(5)
    };
    return result;
}
`

var foreignCStructResultCallExpectedCFragments = []string{
	"struct fairyc_fixture_status_v0 {",
	"uint32_t status;",
	"uint32_t allocation_status;",
	"_Static_assert(sizeof(struct fairyc_fixture_status_v0) == UINT64_C(8)",
	"_Static_assert(_Alignof(struct fairyc_fixture_status_v0) == UINT64_C(4)",
	"offsetof(struct fairyc_fixture_status_v0, status) == UINT64_C(0)",
	"offsetof(struct fairyc_fixture_status_v0, allocation_status) == UINT64_C(4)",
	"extern struct fairyc_fixture_status_v0 fairyc_fixture_make_status_v0(void);",
	" = fairyc_fixture_make_status_v0();",
	"struct fairyc_fixture_large_status_v0 {",
	"_Static_assert(sizeof(struct fairyc_fixture_large_status_v0) == UINT64_C(20)",
	"offsetof(struct fairyc_fixture_large_status_v0, fifth) == UINT64_C(16)",
	"extern struct fairyc_fixture_large_status_v0 fairyc_fixture_make_large_status_v0(void);",
	" = fairyc_fixture_make_large_status_v0();",
}

const textValidationSourceCallSmokeSource = `module main;
@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn utf8_valid(data: *u8, length: usize) -> u32;
fn main() -> i32 {
    if unsafe { utf8_valid(null, 0) } == 1 { 42 } else { 1 }
}
`

const textViewOutputSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_text_view_v0 {
    data: *u8;
    length: usize;
}
@link_name("fy_std_text_validate_utf8_v0")
unsafe extern C fn validate_utf8(
    data: *u8,
    length: usize,
    output: *mut fy_std_text_view_v0
) -> u32;
fn main() -> i32 {
    var output: fy_std_text_view_v0 = fy_std_text_view_v0 { data: null, length: 99 };
    let status: u32 = unsafe {
        validate_utf8(
            null,
            0,
            &mut output as *mut fy_std_text_view_v0
        )
    };
    if status == 0 && output.length == 0 { 42 } else { 1 }
}
`

const systemHeapSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_system_heap_block_v0 {
    data: *mut void;
    size: c::size_t;
    alignment: c::size_t;
}

@link_name("fairyc_fixture_zero_size_t")
unsafe extern C fn zero_size_t() -> c::size_t;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn eight_size_t() -> c::size_t;
@link_name("fy_std_system_heap_allocate_v0")
unsafe extern C fn allocate(
    size: c::size_t,
    alignment: c::size_t,
    output: *mut fy_std_system_heap_block_v0
) -> u32;
@link_name("fairyc_fixture_block_is_allocated")
unsafe extern C fn block_is_allocated(block: *fy_std_system_heap_block_v0) -> u32;
@link_name("fy_std_system_heap_deallocate_v0")
unsafe extern C fn deallocate(block: *mut fy_std_system_heap_block_v0) -> u32;
@link_name("fairyc_fixture_block_is_empty")
unsafe extern C fn block_is_empty(block: *fy_std_system_heap_block_v0) -> u32;

fn main() -> i32 {
    let zero: c::size_t = unsafe { zero_size_t() };
    var block: fy_std_system_heap_block_v0 = fy_std_system_heap_block_v0 {
        data: null,
        size: zero,
        alignment: zero,
    };
    let size: c::size_t = unsafe { eight_size_t() };
    let alignment: c::size_t = unsafe { eight_size_t() };
    let allocated: u32 = unsafe {
        allocate(size, alignment, &mut block as *mut fy_std_system_heap_block_v0)
    };
    let observed: u32 = unsafe {
        block_is_allocated(&block as *fy_std_system_heap_block_v0)
    };
    let deallocated: u32 = unsafe {
        deallocate(&mut block as *mut fy_std_system_heap_block_v0)
    };
    let cleared: u32 = unsafe {
        block_is_empty(&block as *fy_std_system_heap_block_v0)
    };
    if allocated == 0 && observed == 1 && deallocated == 0 && cleared == 1 { 42 } else { 1 }
}
`

const systemHeapSourceCallBridge = `
size_t fairyc_fixture_zero_size_t(void) { return (size_t)0; }
size_t fairyc_fixture_eight_size_t(void) { return (size_t)8; }

uint32_t fairyc_fixture_block_is_allocated(const struct fy_std_system_heap_block_v0 *block) {
    return block != NULL && block->data != NULL && block->size == (size_t)8 &&
           block->alignment == (size_t)8 ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_block_is_empty(const struct fy_std_system_heap_block_v0 *block) {
    return block != NULL && block->data == NULL && block->size == (size_t)0 &&
           block->alignment == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}
`

var systemHeapSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
	"struct fy_std_system_heap_block_v0 {",
	"void * data;",
	"size_t size;",
	"size_t alignment;",
	"extern uint32_t fy_std_system_heap_allocate_v0(size_t fy_a_0, size_t fy_a_1, struct fy_std_system_heap_block_v0 * fy_a_2);",
	"extern uint32_t fairyc_fixture_block_is_allocated(const struct fy_std_system_heap_block_v0 * fy_a_0);",
	"extern uint32_t fy_std_system_heap_deallocate_v0(struct fy_std_system_heap_block_v0 * fy_a_0);",
	" = &fy_l_",
}

const bufferSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_buffer_v0 {
    block: fy_std_system_heap_block_v0;
    length: c::size_t;
}
@repr(C)
struct fy_std_system_heap_block_v0 {
    data: *mut void;
    size: c::size_t;
    alignment: c::size_t;
}

@link_name("fairyc_fixture_zero_size_t")
unsafe extern C fn zero_size_t() -> c::size_t;
@link_name("fairyc_fixture_four_size_t")
unsafe extern C fn four_size_t() -> c::size_t;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn eight_size_t() -> c::size_t;
@link_name("fy_std_buffer_initialize_v0")
unsafe extern C fn initialize(
    capacity: c::size_t,
    output: *mut fy_std_buffer_v0
) -> u32;
@link_name("fy_std_buffer_resize_zeroed_v0")
unsafe extern C fn resize_zeroed(
    buffer: *mut fy_std_buffer_v0,
    new_length: c::size_t
) -> u32;
@link_name("fairyc_fixture_buffer_is_zeroed")
unsafe extern C fn buffer_is_zeroed(buffer: *fy_std_buffer_v0) -> u32;
@link_name("fy_std_buffer_drop_v0")
unsafe extern C fn drop_buffer(buffer: *mut fy_std_buffer_v0) -> u32;
@link_name("fairyc_fixture_buffer_is_empty")
unsafe extern C fn buffer_is_empty(buffer: *fy_std_buffer_v0) -> u32;

fn main() -> i32 {
    let zero: c::size_t = unsafe { zero_size_t() };
    let empty_block: fy_std_system_heap_block_v0 = fy_std_system_heap_block_v0 {
        data: null,
        size: zero,
        alignment: zero,
    };
    var buffer: fy_std_buffer_v0 = fy_std_buffer_v0 {
        block: empty_block,
        length: zero,
    };
    let capacity: c::size_t = unsafe { eight_size_t() };
    let initialized: u32 = unsafe {
        initialize(capacity, &mut buffer as *mut fy_std_buffer_v0)
    };
    let length: c::size_t = unsafe { four_size_t() };
    let resized: u32 = unsafe {
        resize_zeroed(&mut buffer as *mut fy_std_buffer_v0, length)
    };
    let observed: u32 = unsafe {
        buffer_is_zeroed(&buffer as *fy_std_buffer_v0)
    };
    let dropped: u32 = unsafe {
        drop_buffer(&mut buffer as *mut fy_std_buffer_v0)
    };
    let cleared: u32 = unsafe {
        buffer_is_empty(&buffer as *fy_std_buffer_v0)
    };
    if initialized == 0 && resized == 0 && observed == 1 && dropped == 0 && cleared == 1 {
        42
    } else {
        1
    }
}
`

const bufferSourceCallBridge = `
size_t fairyc_fixture_zero_size_t(void) { return (size_t)0; }
size_t fairyc_fixture_four_size_t(void) { return (size_t)4; }
size_t fairyc_fixture_eight_size_t(void) { return (size_t)8; }

uint32_t fairyc_fixture_buffer_is_zeroed(const struct fy_std_buffer_v0 *buffer) {
    if (buffer == NULL || buffer->block.data == NULL ||
        buffer->block.size != (size_t)8 || buffer->block.alignment != (size_t)1 ||
        buffer->length != (size_t)4) {
        return UINT32_C(0);
    }
    const uint8_t *bytes = (const uint8_t *)buffer->block.data;
    return bytes[0] == UINT8_C(0) && bytes[1] == UINT8_C(0) &&
           bytes[2] == UINT8_C(0) && bytes[3] == UINT8_C(0) ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_buffer_is_empty(const struct fy_std_buffer_v0 *buffer) {
    return buffer != NULL && buffer->block.data == NULL &&
           buffer->block.size == (size_t)0 && buffer->block.alignment == (size_t)0 &&
           buffer->length == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}
`

var bufferSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
	"struct fy_std_system_heap_block_v0 {",
	"void * data;",
	"size_t size;",
	"size_t alignment;",
	"struct fy_std_buffer_v0 {",
	"struct fy_std_system_heap_block_v0 block;",
	"size_t length;",
	"_Static_assert(sizeof(struct fy_std_buffer_v0) == UINT64_C(32)",
	"offsetof(struct fy_std_buffer_v0, length) == UINT64_C(24)",
	"extern uint32_t fy_std_buffer_initialize_v0(size_t fy_a_0, struct fy_std_buffer_v0 * fy_a_1);",
	"extern uint32_t fy_std_buffer_resize_zeroed_v0(struct fy_std_buffer_v0 * fy_a_0, size_t fy_a_1);",
	"extern uint32_t fy_std_buffer_drop_v0(struct fy_std_buffer_v0 * fy_a_0);",
	" = &fy_l_",
}

const arenaSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_arena_v0 {
    head: *mut fy_std_arena_node_v0;
    generation: c::ulong;
    active_region: u8;
}
@repr(C)
struct fy_std_region_v0 {
    arena: *mut fy_std_arena_v0;
    mark: *mut fy_std_arena_node_v0;
    generation: c::ulong;
    active: u8;
}
@repr(C)
struct fy_std_arena_node_v0 {
    storage: fy_std_system_heap_block_v0;
    next: *mut fy_std_arena_node_v0;
}
@repr(C)
struct fy_std_system_heap_block_v0 {
    data: *mut void;
    size: c::size_t;
    alignment: c::size_t;
}

@link_name("fairyc_fixture_zero_c_ulong")
unsafe extern C fn zero_c_ulong() -> c::ulong;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn eight_size_t() -> c::size_t;
@link_name("fy_std_arena_initialize_v0")
unsafe extern C fn initialize_arena(output: *mut fy_std_arena_v0) -> u32;
@link_name("fy_std_region_begin_v0")
unsafe extern C fn begin_region(
    arena: *mut fy_std_arena_v0,
    output: *mut fy_std_region_v0
) -> u32;
@link_name("fy_std_region_allocate_zeroed_v0")
unsafe extern C fn allocate_zeroed(
    scope: *mut fy_std_region_v0,
    size: c::size_t,
    alignment: c::size_t,
    output: *mut *mut void
) -> u32;
@link_name("fairyc_fixture_arena_allocation_is_zeroed")
unsafe extern C fn allocation_is_zeroed(
    arena: *mut fy_std_arena_v0,
    scope: *mut fy_std_region_v0,
    data: *mut void
) -> u32;
@link_name("fy_std_region_reset_v0")
unsafe extern C fn reset_region(scope: *mut fy_std_region_v0) -> u32;
@link_name("fairyc_fixture_arena_region_is_reset")
unsafe extern C fn region_is_reset(
    arena: *mut fy_std_arena_v0,
    scope: *mut fy_std_region_v0
) -> u32;
@link_name("fy_std_arena_drop_v0")
unsafe extern C fn drop_arena(arena: *mut fy_std_arena_v0) -> u32;
@link_name("fairyc_fixture_arena_is_dropped")
unsafe extern C fn arena_is_dropped(arena: *mut fy_std_arena_v0) -> u32;

fn main() -> i32 {
    let zero_generation: c::ulong = unsafe { zero_c_ulong() };
    var arena: fy_std_arena_v0 = fy_std_arena_v0 {
        head: null,
        generation: zero_generation,
        active_region: 0,
    };
    var scope: fy_std_region_v0 = fy_std_region_v0 {
        arena: null,
        mark: null,
        generation: zero_generation,
        active: 0,
    };
    let initialized: u32 = unsafe {
        initialize_arena(&mut arena as *mut fy_std_arena_v0)
    };
    let began: u32 = unsafe {
        begin_region(
            &mut arena as *mut fy_std_arena_v0,
            &mut scope as *mut fy_std_region_v0
        )
    };
    let size: c::size_t = unsafe { eight_size_t() };
    let alignment: c::size_t = unsafe { eight_size_t() };
    var data: *mut void = null;
    let allocated: u32 = unsafe {
        allocate_zeroed(
            &mut scope as *mut fy_std_region_v0,
            size,
            alignment,
            &mut data as *mut *mut void
        )
    };
    let observed: u32 = unsafe {
        allocation_is_zeroed(
            &mut arena as *mut fy_std_arena_v0,
            &mut scope as *mut fy_std_region_v0,
            data
        )
    };
    let reset: u32 = unsafe {
        reset_region(&mut scope as *mut fy_std_region_v0)
    };
    let cleared: u32 = unsafe {
        region_is_reset(
            &mut arena as *mut fy_std_arena_v0,
            &mut scope as *mut fy_std_region_v0
        )
    };
    let dropped: u32 = unsafe {
        drop_arena(&mut arena as *mut fy_std_arena_v0)
    };
    let finalized: u32 = unsafe {
        arena_is_dropped(&mut arena as *mut fy_std_arena_v0)
    };
    if initialized == 0 && began == 0 && allocated == 0 && observed == 1 &&
        reset == 0 && cleared == 1 && dropped == 0 && finalized == 1 {
        42
    } else {
        1
    }
}
`

const arenaSourceCallBridge = `
_Static_assert(
    _Generic((uint64_t)0, unsigned long: 1, default: 0),
    "fairyc target uint64_t is unsigned long"
);

unsigned long fairyc_fixture_zero_c_ulong(void) { return 0UL; }
size_t fairyc_fixture_eight_size_t(void) { return (size_t)8; }

uint32_t fairyc_fixture_arena_allocation_is_zeroed(
    struct fy_std_arena_v0 *arena,
    struct fy_std_region_v0 *region,
    void *data
) {
    if (arena == NULL || region == NULL || data == NULL || arena->head == NULL ||
        arena->generation != UINT64_C(1) || arena->active_region != UINT8_C(1) ||
        region->arena != arena || region->mark != NULL ||
        region->generation != UINT64_C(1) || region->active != UINT8_C(1) ||
        (uintptr_t)data % (uintptr_t)8 != (uintptr_t)0) {
        return UINT32_C(0);
    }
    const uint8_t *bytes = (const uint8_t *)data;
    for (size_t index = 0; index < (size_t)8; ++index) {
        if (bytes[index] != UINT8_C(0)) {
            return UINT32_C(0);
        }
    }
    return UINT32_C(1);
}

uint32_t fairyc_fixture_arena_region_is_reset(
    struct fy_std_arena_v0 *arena,
    struct fy_std_region_v0 *region
) {
    return arena != NULL && region != NULL && arena->head == NULL &&
           arena->generation == UINT64_C(2) && arena->active_region == UINT8_C(0) &&
           region->arena == NULL && region->mark == NULL &&
           region->generation == UINT64_C(0) && region->active == UINT8_C(0) ?
           UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_arena_is_dropped(struct fy_std_arena_v0 *arena) {
    return arena != NULL && arena->head == NULL && arena->generation == UINT64_C(3) &&
           arena->active_region == UINT8_C(0) ? UINT32_C(1) : UINT32_C(0);
}
`

var arenaSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(unsigned long) == UINT64_C(8), \"fairyc target c.ulong size\");",
	"struct fy_std_system_heap_block_v0 {",
	"struct fy_std_arena_node_v0 {",
	"struct fy_std_system_heap_block_v0 storage;",
	"struct fy_std_arena_node_v0 * next;",
	"_Static_assert(sizeof(struct fy_std_arena_node_v0) == UINT64_C(32)",
	"_Static_assert(_Alignof(struct fy_std_arena_node_v0) == UINT64_C(8)",
	"offsetof(struct fy_std_arena_node_v0, storage) == UINT64_C(0)",
	"offsetof(struct fy_std_arena_node_v0, next) == UINT64_C(24)",
	"struct fy_std_arena_v0 {",
	"struct fy_std_arena_node_v0 * head;",
	"unsigned long generation;",
	"uint8_t active_region;",
	"_Static_assert(sizeof(struct fy_std_arena_v0) == UINT64_C(24)",
	"offsetof(struct fy_std_arena_v0, head) == UINT64_C(0)",
	"offsetof(struct fy_std_arena_v0, generation) == UINT64_C(8)",
	"offsetof(struct fy_std_arena_v0, active_region) == UINT64_C(16)",
	"struct fy_std_region_v0 {",
	"struct fy_std_arena_v0 * arena;",
	"struct fy_std_arena_node_v0 * mark;",
	"_Static_assert(sizeof(struct fy_std_region_v0) == UINT64_C(32)",
	"offsetof(struct fy_std_region_v0, arena) == UINT64_C(0)",
	"offsetof(struct fy_std_region_v0, mark) == UINT64_C(8)",
	"offsetof(struct fy_std_region_v0, generation) == UINT64_C(16)",
	"offsetof(struct fy_std_region_v0, active) == UINT64_C(24)",
	"extern uint32_t fy_std_arena_initialize_v0(struct fy_std_arena_v0 * fy_a_0);",
	"extern uint32_t fy_std_region_begin_v0(struct fy_std_arena_v0 * fy_a_0, struct fy_std_region_v0 * fy_a_1);",
	"extern uint32_t fy_std_region_allocate_zeroed_v0(struct fy_std_region_v0 * fy_a_0, size_t fy_a_1, size_t fy_a_2, void * * fy_a_3);",
	"extern uint32_t fy_std_region_reset_v0(struct fy_std_region_v0 * fy_a_0);",
	"extern uint32_t fy_std_arena_drop_v0(struct fy_std_arena_v0 * fy_a_0);",
	" = &fy_l_",
}

const cStrValidationSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_cstr_view_v0 {
    data: *u8;
    length: c::size_t;
}

@link_name("fairyc_fixture_cstr_valid_storage")
unsafe extern C fn valid_storage() -> *u8;
@link_name("fairyc_fixture_cstr_valid_storage_length")
unsafe extern C fn valid_storage_length() -> c::size_t;
@link_name("fairyc_fixture_cstr_interior_nul_storage")
unsafe extern C fn interior_nul_storage() -> *u8;
@link_name("fairyc_fixture_cstr_interior_nul_storage_length")
unsafe extern C fn interior_nul_storage_length() -> c::size_t;
@link_name("fairyc_fixture_cstr_unterminated_storage")
unsafe extern C fn unterminated_storage() -> *u8;
@link_name("fairyc_fixture_cstr_unterminated_storage_length")
unsafe extern C fn unterminated_storage_length() -> c::size_t;
@link_name("fy_std_cstr_validate_v0")
unsafe extern C fn validate_cstr(
    storage: *u8,
    storage_length: c::size_t,
    output: *mut fy_std_cstr_view_v0
) -> u32;
@link_name("fairyc_fixture_cstr_view_is_valid")
unsafe extern C fn view_is_valid(view: *fy_std_cstr_view_v0) -> u32;

fn main() -> i32 {
    let valid: *u8 = unsafe { valid_storage() };
    let valid_length: c::size_t = unsafe { valid_storage_length() };
    var output: fy_std_cstr_view_v0 = fy_std_cstr_view_v0 {
        data: null,
        length: valid_length,
    };
    let accepted: u32 = unsafe {
        validate_cstr(valid, valid_length, &mut output as *mut fy_std_cstr_view_v0)
    };
    let valid_observed: u32 = unsafe {
        view_is_valid(&output as *fy_std_cstr_view_v0)
    };

    let interior_nul: *u8 = unsafe { interior_nul_storage() };
    let interior_nul_length: c::size_t = unsafe { interior_nul_storage_length() };
    let interior_nul_status: u32 = unsafe {
        validate_cstr(
            interior_nul,
            interior_nul_length,
            &mut output as *mut fy_std_cstr_view_v0
        )
    };
    let interior_failure_preserved_output: u32 = unsafe {
        view_is_valid(&output as *fy_std_cstr_view_v0)
    };

    let unterminated: *u8 = unsafe { unterminated_storage() };
    let unterminated_length: c::size_t = unsafe { unterminated_storage_length() };
    let unterminated_status: u32 = unsafe {
        validate_cstr(
            unterminated,
            unterminated_length,
            &mut output as *mut fy_std_cstr_view_v0
        )
    };
    let unterminated_failure_preserved_output: u32 = unsafe {
        view_is_valid(&output as *fy_std_cstr_view_v0)
    };

    if accepted == 0 && valid_observed == 1 &&
        interior_nul_status == 4 && interior_failure_preserved_output == 1 &&
        unterminated_status == 3 && unterminated_failure_preserved_output == 1 {
        42
    } else {
        1
    }
}
`

const cStrValidationSourceCallBridge = `
static const uint8_t fairyc_fixture_cstr_valid_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(0)
};
static const uint8_t fairyc_fixture_cstr_interior_nul_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(0), UINT8_C(105), UINT8_C(114), UINT8_C(121), UINT8_C(0)
};
static const uint8_t fairyc_fixture_cstr_unterminated_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)
};

const uint8_t *fairyc_fixture_cstr_valid_storage(void) {
    return fairyc_fixture_cstr_valid_bytes;
}

size_t fairyc_fixture_cstr_valid_storage_length(void) {
    return sizeof(fairyc_fixture_cstr_valid_bytes);
}

const uint8_t *fairyc_fixture_cstr_interior_nul_storage(void) {
    return fairyc_fixture_cstr_interior_nul_bytes;
}

size_t fairyc_fixture_cstr_interior_nul_storage_length(void) {
    return sizeof(fairyc_fixture_cstr_interior_nul_bytes);
}

const uint8_t *fairyc_fixture_cstr_unterminated_storage(void) {
    return fairyc_fixture_cstr_unterminated_bytes;
}

size_t fairyc_fixture_cstr_unterminated_storage_length(void) {
    return sizeof(fairyc_fixture_cstr_unterminated_bytes);
}

uint32_t fairyc_fixture_cstr_view_is_valid(const struct fy_std_cstr_view_v0 *view) {
    return view != NULL && view->data == fairyc_fixture_cstr_valid_bytes &&
           view->length == sizeof(fairyc_fixture_cstr_valid_bytes) - (size_t)1 &&
           view->data[0] == UINT8_C(70) && view->data[1] == UINT8_C(97) &&
           view->data[2] == UINT8_C(105) && view->data[3] == UINT8_C(114) &&
           view->data[4] == UINT8_C(121) ? UINT32_C(1) : UINT32_C(0);
}
`

var cStrValidationSourceCallExpectedCFragments = []string{
	"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
	"struct fy_std_cstr_view_v0 {",
	"const uint8_t * data;",
	"size_t length;",
	"_Static_assert(sizeof(struct fy_std_cstr_view_v0) == UINT64_C(16)",
	"offsetof(struct fy_std_cstr_view_v0, data) == UINT64_C(0)",
	"offsetof(struct fy_std_cstr_view_v0, length) == UINT64_C(8)",
	"extern uint32_t fy_std_cstr_validate_v0(const uint8_t * fy_a_0, size_t fy_a_1, struct fy_std_cstr_view_v0 * fy_a_2);",
	"extern uint32_t fairyc_fixture_cstr_view_is_valid(const struct fy_std_cstr_view_v0 * fy_a_0);",
	"fy_std_cstr_validate_v0(fy_l_",
	" = &fy_l_",
}

const ownedTextSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_system_heap_block_v0 {
    data: *mut void;
    size: c::size_t;
    alignment: c::size_t;
}
@repr(C)
struct fy_std_buffer_v0 {
    block: fy_std_system_heap_block_v0;
    length: c::size_t;
}
@repr(C)
struct fy_std_text_view_v0 {
    data: *u8;
    length: c::size_t;
}
@repr(C)
struct fy_std_string_v0 {
    bytes: fy_std_buffer_v0;
}
@repr(C)
struct fy_std_cstr_view_v0 {
    data: *u8;
    length: c::size_t;
}
@repr(C)
struct fy_std_cstring_v0 {
    bytes: fy_std_buffer_v0;
}
@repr(C)
struct fy_std_owned_text_result_v0 {
    status: u32;
    allocation_status: u32;
}

@link_name("fairyc_fixture_owned_text_zero_size_t")
unsafe extern C fn zero_size_t() -> c::size_t;
@link_name("fairyc_fixture_owned_text_storage")
unsafe extern C fn text_storage() -> *u8;
@link_name("fairyc_fixture_owned_text_storage_length")
unsafe extern C fn text_storage_length() -> c::size_t;

@link_name("fy_std_string_copy_v0")
unsafe extern C fn string_copy(
    input: *fy_std_text_view_v0,
    output: *mut fy_std_string_v0
) -> fy_std_owned_text_result_v0;
@link_name("fy_std_string_as_view_v0")
unsafe extern C fn string_as_view(
    input: *fy_std_string_v0,
    output: *mut fy_std_text_view_v0
) -> fy_std_owned_text_result_v0;
@link_name("fy_std_string_append_v0")
unsafe extern C fn string_append(
    value: *mut fy_std_string_v0,
    suffix: *fy_std_text_view_v0
) -> fy_std_owned_text_result_v0;
@link_name("fy_std_string_drop_v0")
unsafe extern C fn string_drop(value: *mut fy_std_string_v0) -> fy_std_owned_text_result_v0;

@link_name("fy_std_cstring_from_text_v0")
unsafe extern C fn cstring_from_text(
    input: *fy_std_text_view_v0,
    output: *mut fy_std_cstring_v0
) -> fy_std_owned_text_result_v0;
@link_name("fy_std_cstring_as_view_v0")
unsafe extern C fn cstring_as_view(
    input: *fy_std_cstring_v0,
    output: *mut fy_std_cstr_view_v0
) -> fy_std_owned_text_result_v0;
@link_name("fy_std_cstring_drop_v0")
unsafe extern C fn cstring_drop(value: *mut fy_std_cstring_v0) -> fy_std_owned_text_result_v0;

@link_name("fairyc_fixture_string_view_is_fairy")
unsafe extern C fn string_view_is_fairy(view: *fy_std_text_view_v0) -> u32;
@link_name("fairyc_fixture_string_view_is_fairy_twice")
unsafe extern C fn string_view_is_fairy_twice(view: *fy_std_text_view_v0) -> u32;
@link_name("fairyc_fixture_string_is_empty")
unsafe extern C fn string_is_empty(value: *fy_std_string_v0) -> u32;
@link_name("fairyc_fixture_cstr_view_is_fairy")
unsafe extern C fn cstr_view_is_fairy(view: *fy_std_cstr_view_v0) -> u32;
@link_name("fairyc_fixture_cstring_is_empty")
unsafe extern C fn cstring_is_empty(value: *fy_std_cstring_v0) -> u32;

fn main() -> i32 {
    let zero: c::size_t = unsafe { zero_size_t() };
    let storage: *u8 = unsafe { text_storage() };
    let storage_length: c::size_t = unsafe { text_storage_length() };
    let input: fy_std_text_view_v0 = fy_std_text_view_v0 {
        data: storage,
        length: storage_length,
    };

    let string_block: fy_std_system_heap_block_v0 = fy_std_system_heap_block_v0 {
        data: null,
        size: zero,
        alignment: zero,
    };
    let string_buffer: fy_std_buffer_v0 = fy_std_buffer_v0 {
        block: string_block,
        length: zero,
    };
    var owned_string: fy_std_string_v0 = fy_std_string_v0 { bytes: string_buffer };
    var string_view: fy_std_text_view_v0 = fy_std_text_view_v0 {
        data: null,
        length: zero,
    };
    let copied: fy_std_owned_text_result_v0 = unsafe {
        string_copy(
            &input as *fy_std_text_view_v0,
            &mut owned_string as *mut fy_std_string_v0
        )
    };
    let viewed_once: fy_std_owned_text_result_v0 = unsafe {
        string_as_view(
            &owned_string as *fy_std_string_v0,
            &mut string_view as *mut fy_std_text_view_v0
        )
    };
    let observed_once: u32 = unsafe {
        string_view_is_fairy(&string_view as *fy_std_text_view_v0)
    };
    let appended: fy_std_owned_text_result_v0 = unsafe {
        string_append(
            &mut owned_string as *mut fy_std_string_v0,
            &input as *fy_std_text_view_v0
        )
    };
    let viewed_twice: fy_std_owned_text_result_v0 = unsafe {
        string_as_view(
            &owned_string as *fy_std_string_v0,
            &mut string_view as *mut fy_std_text_view_v0
        )
    };
    let observed_twice: u32 = unsafe {
        string_view_is_fairy_twice(&string_view as *fy_std_text_view_v0)
    };
    let dropped_string: fy_std_owned_text_result_v0 = unsafe {
        string_drop(&mut owned_string as *mut fy_std_string_v0)
    };
    let string_empty: u32 = unsafe {
        string_is_empty(&owned_string as *fy_std_string_v0)
    };

    let cstring_block: fy_std_system_heap_block_v0 = fy_std_system_heap_block_v0 {
        data: null,
        size: zero,
        alignment: zero,
    };
    let cstring_buffer: fy_std_buffer_v0 = fy_std_buffer_v0 {
        block: cstring_block,
        length: zero,
    };
    var owned_cstring: fy_std_cstring_v0 = fy_std_cstring_v0 { bytes: cstring_buffer };
    var cstr_view: fy_std_cstr_view_v0 = fy_std_cstr_view_v0 {
        data: null,
        length: zero,
    };
    let created_cstring: fy_std_owned_text_result_v0 = unsafe {
        cstring_from_text(
            &input as *fy_std_text_view_v0,
            &mut owned_cstring as *mut fy_std_cstring_v0
        )
    };
    let viewed_cstring: fy_std_owned_text_result_v0 = unsafe {
        cstring_as_view(
            &owned_cstring as *fy_std_cstring_v0,
            &mut cstr_view as *mut fy_std_cstr_view_v0
        )
    };
    let observed_cstring: u32 = unsafe {
        cstr_view_is_fairy(&cstr_view as *fy_std_cstr_view_v0)
    };
    let dropped_cstring: fy_std_owned_text_result_v0 = unsafe {
        cstring_drop(&mut owned_cstring as *mut fy_std_cstring_v0)
    };
    let cstring_empty: u32 = unsafe {
        cstring_is_empty(&owned_cstring as *fy_std_cstring_v0)
    };

    if copied.status == 0 && copied.allocation_status == 0 &&
        viewed_once.status == 0 && viewed_once.allocation_status == 0 && observed_once == 1 &&
        appended.status == 0 && appended.allocation_status == 0 &&
        viewed_twice.status == 0 && viewed_twice.allocation_status == 0 && observed_twice == 1 &&
        dropped_string.status == 0 && dropped_string.allocation_status == 0 && string_empty == 1 &&
        created_cstring.status == 0 && created_cstring.allocation_status == 0 &&
        viewed_cstring.status == 0 && viewed_cstring.allocation_status == 0 && observed_cstring == 1 &&
        dropped_cstring.status == 0 && dropped_cstring.allocation_status == 0 && cstring_empty == 1 {
        42
    } else {
        0
    }
}
`

const ownedTextSourceCallBridge = `
static const uint8_t fairyc_fixture_owned_text_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121)
};

size_t fairyc_fixture_owned_text_zero_size_t(void) { return (size_t)0; }

const uint8_t *fairyc_fixture_owned_text_storage(void) {
    return fairyc_fixture_owned_text_bytes;
}

size_t fairyc_fixture_owned_text_storage_length(void) {
    return sizeof(fairyc_fixture_owned_text_bytes);
}

static uint32_t fairyc_fixture_bytes_are_fairy(const uint8_t *data, size_t length) {
    return data != NULL && length == (size_t)5 &&
           data[0] == UINT8_C(70) && data[1] == UINT8_C(97) &&
           data[2] == UINT8_C(105) && data[3] == UINT8_C(114) &&
           data[4] == UINT8_C(121) ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_string_view_is_fairy(const struct fy_std_text_view_v0 *view) {
    return view != NULL ? fairyc_fixture_bytes_are_fairy(view->data, view->length) : UINT32_C(0);
}

uint32_t fairyc_fixture_string_view_is_fairy_twice(const struct fy_std_text_view_v0 *view) {
    return view != NULL && view->data != NULL && view->length == (size_t)10 &&
           fairyc_fixture_bytes_are_fairy(view->data, (size_t)5) == UINT32_C(1) &&
           fairyc_fixture_bytes_are_fairy(view->data + (size_t)5, (size_t)5) == UINT32_C(1) ?
           UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_string_is_empty(const struct fy_std_string_v0 *value) {
    return value != NULL && value->bytes.block.data == NULL &&
           value->bytes.block.size == (size_t)0 && value->bytes.block.alignment == (size_t)0 &&
           value->bytes.length == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_cstr_view_is_fairy(const struct fy_std_cstr_view_v0 *view) {
    return view != NULL && fairyc_fixture_bytes_are_fairy(view->data, view->length) == UINT32_C(1) &&
           view->data[view->length] == UINT8_C(0) ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_cstring_is_empty(const struct fy_std_cstring_v0 *value) {
    return value != NULL && value->bytes.block.data == NULL &&
           value->bytes.block.size == (size_t)0 && value->bytes.block.alignment == (size_t)0 &&
           value->bytes.length == (size_t)0 ? UINT32_C(1) : UINT32_C(0);
}
`

var ownedTextSourceCallExpectedCFragments = []string{
	"struct fy_std_owned_text_result_v0 {",
	"uint32_t status;",
	"uint32_t allocation_status;",
	"_Static_assert(sizeof(struct fy_std_owned_text_result_v0) == UINT64_C(8)",
	"struct fy_std_string_v0 {",
	"struct fy_std_buffer_v0 bytes;",
	"struct fy_std_cstring_v0 {",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_copy_v0(const struct fy_std_text_view_v0 * fy_a_0, struct fy_std_string_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_as_view_v0(const struct fy_std_string_v0 * fy_a_0, struct fy_std_text_view_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_append_v0(struct fy_std_string_v0 * fy_a_0, const struct fy_std_text_view_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_string_drop_v0(struct fy_std_string_v0 * fy_a_0);",
	"extern struct fy_std_owned_text_result_v0 fy_std_cstring_from_text_v0(const struct fy_std_text_view_v0 * fy_a_0, struct fy_std_cstring_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_cstring_as_view_v0(const struct fy_std_cstring_v0 * fy_a_0, struct fy_std_cstr_view_v0 * fy_a_1);",
	"extern struct fy_std_owned_text_result_v0 fy_std_cstring_drop_v0(struct fy_std_cstring_v0 * fy_a_0);",
	" = fy_std_string_copy_v0(",
	" = fy_std_cstring_from_text_v0(",
}

const linuxFileSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_file_result_v0 {
    status: u32;
    error_number: i32;
    count: c::size_t;
}
@repr(C)
struct fy_std_linux_file_v0 {
    descriptor: c::int;
    open: u8;
}
@repr(C)
struct fy_std_linux_open_options_v0 {
    read: u8;
    write: u8;
    create: u8;
    truncate: u8;
    mode: u32;
}

@link_name("fairyc_fixture_file_prepare_path")
unsafe extern C fn prepare_path() -> u32;
@link_name("fairyc_fixture_file_path")
unsafe extern C fn file_path() -> *u8;
@link_name("fairyc_fixture_file_path_storage_length")
unsafe extern C fn file_path_storage_length() -> c::size_t;
@link_name("fairyc_fixture_file_payload")
unsafe extern C fn file_payload() -> *u8;
@link_name("fairyc_fixture_file_payload_length")
unsafe extern C fn file_payload_length() -> c::size_t;
@link_name("fairyc_fixture_file_three_size_t")
unsafe extern C fn three_size_t() -> c::size_t;
@link_name("fairyc_fixture_file_zero_size_t")
unsafe extern C fn zero_size_t() -> c::size_t;
@link_name("fairyc_fixture_file_closed_descriptor")
unsafe extern C fn closed_descriptor() -> c::int;
@link_name("fairyc_fixture_file_read_segment_one")
unsafe extern C fn read_segment_one() -> *mut u8;
@link_name("fairyc_fixture_file_read_segment_two")
unsafe extern C fn read_segment_two() -> *mut u8;
@link_name("fairyc_fixture_file_read_segment_three")
unsafe extern C fn read_segment_three() -> *mut u8;

@link_name("fy_std_linux_file_open_v0")
unsafe extern C fn linux_file_open(
    path: *u8,
    path_storage_length: c::size_t,
    options: *fy_std_linux_open_options_v0,
    output: *mut fy_std_linux_file_v0
) -> fy_std_file_result_v0;
@link_name("fy_std_linux_file_read_v0")
unsafe extern C fn linux_file_read(
    file: *mut fy_std_linux_file_v0,
    buffer: *mut u8,
    length: c::size_t
) -> fy_std_file_result_v0;
@link_name("fy_std_linux_file_write_all_v0")
unsafe extern C fn linux_file_write_all(
    file: *mut fy_std_linux_file_v0,
    buffer: *u8,
    length: c::size_t
) -> fy_std_file_result_v0;
@link_name("fy_std_linux_file_sync_v0")
unsafe extern C fn linux_file_sync(
    file: *mut fy_std_linux_file_v0
) -> fy_std_file_result_v0;
@link_name("fy_std_linux_file_close_v0")
unsafe extern C fn linux_file_close(
    file: *mut fy_std_linux_file_v0
) -> fy_std_file_result_v0;

@link_name("fairyc_fixture_file_result_matches")
unsafe extern C fn result_matches(
    result: *fy_std_file_result_v0,
    status: u32,
    error_number: i32,
    count: c::size_t
) -> u32;
@link_name("fairyc_fixture_file_is_closed")
unsafe extern C fn file_is_closed(file: *fy_std_linux_file_v0) -> u32;
@link_name("fairyc_fixture_file_buffers_match_and_cleanup")
unsafe extern C fn buffers_match_and_cleanup() -> u32;

fn main() -> i32 {
    let prepared: u32 = unsafe { prepare_path() };
    let path: *u8 = unsafe { file_path() };
    let path_length: c::size_t = unsafe { file_path_storage_length() };
    let payload: *u8 = unsafe { file_payload() };
    let payload_length: c::size_t = unsafe { file_payload_length() };
    let three: c::size_t = unsafe { three_size_t() };
    let zero: c::size_t = unsafe { zero_size_t() };
    let closed: c::int = unsafe { closed_descriptor() };

    var file: fy_std_linux_file_v0 = fy_std_linux_file_v0 {
        descriptor: closed,
        open: 0,
    };
    let write_options: fy_std_linux_open_options_v0 = fy_std_linux_open_options_v0 {
        read: 0,
        write: 1,
        create: 0,
        truncate: 1,
        mode: 384,
    };
    let opened_for_write: fy_std_file_result_v0 = unsafe {
        linux_file_open(
            path,
            path_length,
            &write_options as *fy_std_linux_open_options_v0,
            &mut file as *mut fy_std_linux_file_v0
        )
    };
    let opened_for_write_ok: u32 = unsafe {
        result_matches(&opened_for_write as *fy_std_file_result_v0, 0, 0, zero)
    };
    let written: fy_std_file_result_v0 = unsafe {
        linux_file_write_all(&mut file as *mut fy_std_linux_file_v0, payload, payload_length)
    };
    let written_ok: u32 = unsafe {
        result_matches(&written as *fy_std_file_result_v0, 0, 0, payload_length)
    };
    let synced: fy_std_file_result_v0 = unsafe {
        linux_file_sync(&mut file as *mut fy_std_linux_file_v0)
    };
    let synced_ok: u32 = unsafe {
        result_matches(&synced as *fy_std_file_result_v0, 0, 0, zero)
    };
    let closed_after_write: fy_std_file_result_v0 = unsafe {
        linux_file_close(&mut file as *mut fy_std_linux_file_v0)
    };
    let closed_after_write_ok: u32 = unsafe {
        result_matches(&closed_after_write as *fy_std_file_result_v0, 0, 0, zero)
    };
    let closed_state_after_write: u32 = unsafe {
        file_is_closed(&file as *fy_std_linux_file_v0)
    };
    let closed_twice: fy_std_file_result_v0 = unsafe {
        linux_file_close(&mut file as *mut fy_std_linux_file_v0)
    };
    let closed_twice_ok: u32 = unsafe {
        result_matches(&closed_twice as *fy_std_file_result_v0, 3, 0, zero)
    };

    let read_options: fy_std_linux_open_options_v0 = fy_std_linux_open_options_v0 {
        read: 1,
        write: 0,
        create: 0,
        truncate: 0,
        mode: 0,
    };
    let opened_for_read: fy_std_file_result_v0 = unsafe {
        linux_file_open(
            path,
            path_length,
            &read_options as *fy_std_linux_open_options_v0,
            &mut file as *mut fy_std_linux_file_v0
        )
    };
    let opened_for_read_ok: u32 = unsafe {
        result_matches(&opened_for_read as *fy_std_file_result_v0, 0, 0, zero)
    };
    let first_buffer: *mut u8 = unsafe { read_segment_one() };
    let second_buffer: *mut u8 = unsafe { read_segment_two() };
    let third_buffer: *mut u8 = unsafe { read_segment_three() };
    let read_first: fy_std_file_result_v0 = unsafe {
        linux_file_read(&mut file as *mut fy_std_linux_file_v0, first_buffer, three)
    };
    let read_first_ok: u32 = unsafe {
        result_matches(&read_first as *fy_std_file_result_v0, 0, 0, three)
    };
    let read_second: fy_std_file_result_v0 = unsafe {
        linux_file_read(&mut file as *mut fy_std_linux_file_v0, second_buffer, three)
    };
    let read_second_ok: u32 = unsafe {
        result_matches(&read_second as *fy_std_file_result_v0, 0, 0, three)
    };
    let read_third: fy_std_file_result_v0 = unsafe {
        linux_file_read(&mut file as *mut fy_std_linux_file_v0, third_buffer, three)
    };
    let read_third_ok: u32 = unsafe {
        result_matches(&read_third as *fy_std_file_result_v0, 0, 0, three)
    };
    let read_eof: fy_std_file_result_v0 = unsafe {
        linux_file_read(&mut file as *mut fy_std_linux_file_v0, first_buffer, three)
    };
    let read_eof_ok: u32 = unsafe {
        result_matches(&read_eof as *fy_std_file_result_v0, 0, 0, zero)
    };
    let closed_after_read: fy_std_file_result_v0 = unsafe {
        linux_file_close(&mut file as *mut fy_std_linux_file_v0)
    };
    let closed_after_read_ok: u32 = unsafe {
        result_matches(&closed_after_read as *fy_std_file_result_v0, 0, 0, zero)
    };
    let closed_state_after_read: u32 = unsafe {
        file_is_closed(&file as *fy_std_linux_file_v0)
    };
    let observed: u32 = unsafe { buffers_match_and_cleanup() };

    if prepared == 1 && opened_for_write_ok == 1 && written_ok == 1 && synced_ok == 1 &&
        closed_after_write_ok == 1 && closed_state_after_write == 1 && closed_twice_ok == 1 &&
        opened_for_read_ok == 1 && read_first_ok == 1 && read_second_ok == 1 &&
        read_third_ok == 1 && read_eof_ok == 1 && closed_after_read_ok == 1 &&
        closed_state_after_read == 1 && observed == 1 {
        42
    } else {
        0
    }
}
`

const linuxFileSourceCallBridge = `
#include <stdlib.h>

static char fairyc_fixture_file_path_storage[] = "/tmp/fairyc-file-source-XXXXXX";
static const uint8_t fairyc_fixture_file_payload_bytes[] = {
    UINT8_C(70), UINT8_C(97), UINT8_C(105), UINT8_C(114), UINT8_C(121),
    UINT8_C(70), UINT8_C(105), UINT8_C(108), UINT8_C(101)
};
static uint8_t fairyc_fixture_file_read_bytes[9];

uint32_t fairyc_fixture_file_prepare_path(void) {
    int descriptor = mkstemp(fairyc_fixture_file_path_storage);
    if (descriptor < 0) {
        return UINT32_C(0);
    }
    if (close(descriptor) != 0) {
        (void)unlink(fairyc_fixture_file_path_storage);
        return UINT32_C(0);
    }
    return UINT32_C(1);
}

const uint8_t *fairyc_fixture_file_path(void) {
    return (const uint8_t *)fairyc_fixture_file_path_storage;
}

size_t fairyc_fixture_file_path_storage_length(void) {
    return sizeof(fairyc_fixture_file_path_storage);
}

const uint8_t *fairyc_fixture_file_payload(void) {
    return fairyc_fixture_file_payload_bytes;
}

size_t fairyc_fixture_file_payload_length(void) {
    return sizeof(fairyc_fixture_file_payload_bytes);
}

size_t fairyc_fixture_file_three_size_t(void) { return (size_t)3; }
size_t fairyc_fixture_file_zero_size_t(void) { return (size_t)0; }
int fairyc_fixture_file_closed_descriptor(void) { return -1; }
uint8_t *fairyc_fixture_file_read_segment_one(void) { return fairyc_fixture_file_read_bytes; }
uint8_t *fairyc_fixture_file_read_segment_two(void) { return fairyc_fixture_file_read_bytes + 3; }
uint8_t *fairyc_fixture_file_read_segment_three(void) { return fairyc_fixture_file_read_bytes + 6; }

uint32_t fairyc_fixture_file_result_matches(
    const struct fy_std_file_result_v0 *result,
    uint32_t status,
    int32_t error_number,
    size_t count
) {
    return result != NULL && result->status == status && result->error_number == error_number &&
           result->count == count ? UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_file_is_closed(const struct fy_std_linux_file_v0 *file) {
    return file != NULL && file->descriptor == -1 && file->open == UINT8_C(0) ?
           UINT32_C(1) : UINT32_C(0);
}

uint32_t fairyc_fixture_file_buffers_match_and_cleanup(void) {
    uint32_t matches = UINT32_C(1);
    for (size_t index = 0; index < sizeof(fairyc_fixture_file_payload_bytes); ++index) {
        if (fairyc_fixture_file_read_bytes[index] != fairyc_fixture_file_payload_bytes[index]) {
            matches = UINT32_C(0);
        }
    }
    if (unlink(fairyc_fixture_file_path_storage) != 0) {
        return UINT32_C(0);
    }
    return matches;
}
`

var linuxFileSourceCallExpectedCFragments = []string{
	"struct fy_std_file_result_v0 {",
	"uint32_t status;",
	"int32_t error_number;",
	"size_t count;",
	"_Static_assert(sizeof(struct fy_std_file_result_v0) == UINT64_C(16)",
	"struct fy_std_linux_file_v0 {",
	"int descriptor;",
	"uint8_t open;",
	"struct fy_std_linux_open_options_v0 {",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_open_v0(const uint8_t * fy_a_0, size_t fy_a_1, const struct fy_std_linux_open_options_v0 * fy_a_2, struct fy_std_linux_file_v0 * fy_a_3);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_read_v0(struct fy_std_linux_file_v0 * fy_a_0, uint8_t * fy_a_1, size_t fy_a_2);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_write_all_v0(struct fy_std_linux_file_v0 * fy_a_0, const uint8_t * fy_a_1, size_t fy_a_2);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_sync_v0(struct fy_std_linux_file_v0 * fy_a_0);",
	"extern struct fy_std_file_result_v0 fy_std_linux_file_close_v0(struct fy_std_linux_file_v0 * fy_a_0);",
	" = fy_std_linux_file_open_v0(",
	" = fy_std_linux_file_read_v0(",
	" = fy_std_linux_file_write_all_v0(",
	" = fy_std_linux_file_sync_v0(",
	" = fy_std_linux_file_close_v0(",
}

// Compiler records the bounded toolchain fingerprint used by one native run.
type Compiler struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Report is the deterministic success record emitted by the native fixture command.
type Report struct {
	SchemaVersion                       string     `json:"schemaVersion"`
	Target                              string     `json:"target"`
	Libc                                string     `json:"libc"`
	Compilers                           []Compiler `json:"compilers"`
	Profiles                            []string   `json:"profiles"`
	Cases                               []string   `json:"cases"`
	Executions                          int        `json:"executions"`
	DropTraceExecutions                 int        `json:"dropTraceExecutions"`
	LayoutCases                         []string   `json:"layoutCases"`
	LayoutProbeExecutions               int        `json:"layoutProbeExecutions"`
	Sanitizer                           string     `json:"sanitizer"`
	SanitizerExecutions                 int        `json:"sanitizerExecutions"`
	ABICases                            []string   `json:"abiCases"`
	ABIDirections                       []string   `json:"abiDirections"`
	ABIProbeExecutions                  int        `json:"abiProbeExecutions"`
	SystemHeapExecutions                int        `json:"systemHeapExecutions"`
	SystemHeapSourceCallExecutions      int        `json:"systemHeapSourceCallExecutions"`
	BufferExecutions                    int        `json:"bufferExecutions"`
	BufferSourceCallExecutions          int        `json:"bufferSourceCallExecutions"`
	ArenaExecutions                     int        `json:"arenaExecutions"`
	ArenaSourceCallExecutions           int        `json:"arenaSourceCallExecutions"`
	TextViewExecutions                  int        `json:"textViewExecutions"`
	OwnedTextExecutions                 int        `json:"ownedTextExecutions"`
	OwnedTextSourceCallExecutions       int        `json:"ownedTextSourceCallExecutions"`
	LinuxFileExecutions                 int        `json:"linuxFileExecutions"`
	LinuxFileSourceCallExecutions       int        `json:"linuxFileSourceCallExecutions"`
	ForeignCallExecutions               int        `json:"foreignCallExecutions"`
	ForeignCIntCallExecutions           int        `json:"foreignCIntCallExecutions"`
	ForeignCLongCallExecutions          int        `json:"foreignCLongCallExecutions"`
	ForeignCScalarFamilyCallExecutions  int        `json:"foreignCScalarFamilyCallExecutions"`
	ForeignCSizeTCallExecutions         int        `json:"foreignCSizeTCallExecutions"`
	ForeignCSSizeTCallExecutions        int        `json:"foreignCSSizeTCallExecutions"`
	ForeignPointerCallExecutions        int        `json:"foreignPointerCallExecutions"`
	ForeignCStructResultCallExecutions  int        `json:"foreignCStructResultCallExecutions"`
	TextValidationSourceCallExecutions  int        `json:"textValidationSourceCallExecutions"`
	TextViewOutputSourceCallExecutions  int        `json:"textViewOutputSourceCallExecutions"`
	CStrValidationSourceCallExecutions  int        `json:"cStrValidationSourceCallExecutions"`
	ByteStringSourceCallExecutions      int        `json:"byteStringLiteralSourceCallExecutions"`
	UTF8StringSourceCallExecutions      int        `json:"utf8StringLiteralSourceCallExecutions"`
	CStringLiteralSourceCallExecutions  int        `json:"cStringLiteralSourceCallExecutions"`
	BuildSmokeExecutions                int        `json:"buildSmokeExecutions"`
	StandardLibraryBuildSmokeExecutions int        `json:"standardLibraryBuildSmokeExecutions"`
	ByteStringBuildSmokeExecutions      int        `json:"byteStringLiteralBuildSmokeExecutions"`
	UTF8StringBuildSmokeExecutions      int        `json:"utf8StringLiteralBuildSmokeExecutions"`
	CStringLiteralBuildSmokeExecutions  int        `json:"cStringLiteralBuildSmokeExecutions"`
}

// Run compiles every emission fixture with Clang and GCC at low and optimized profiles, compares
// the Linux process outcome and source-linked drop trace with the MIR-interpreter oracle, executes
// every production unit under the closed address/undefined sanitizer mode, and exercises the
// product build CLI.
func Run(manifestPath string) (Report, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Report{}, fmt.Errorf("native C differential requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	fixtureReport, err := cbackendfixture.Run(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if len(fixtureReport.Failures) != 0 || len(fixtureReport.Emitted) != fixtureReport.CasesRun {
		return Report{}, fmt.Errorf("C emission fixtures are not clean: %s", summarizeFailures(fixtureReport.Failures))
	}
	traceRuntimeSource, err := cruntime.DropTraceSource(trapExitBase)
	if err != nil {
		return Report{}, fmt.Errorf("bind scalar runtime: %w", err)
	}
	scalarRuntimeSource, err := cruntime.ScalarSource(trapExitBase)
	if err != nil {
		return Report{}, fmt.Errorf("bind uninstrumented scalar runtime: %w", err)
	}
	toolchains, err := resolveToolchains()
	if err != nil {
		return Report{}, err
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-native-c-")
	if err != nil {
		return Report{}, fmt.Errorf("create native fixture temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	report := Report{
		SchemaVersion: reportSchemaVersion,
		Target:        string(target.X8664LinuxGNU),
		Libc:          toolchains[0].Libc,
		Sanitizer:     cdriver.AddressUndefinedSanitizerMode,
	}
	for _, toolchain := range toolchains {
		if toolchain.Libc != report.Libc {
			return Report{}, fmt.Errorf("C toolchains disagree on libc: %q and %q", report.Libc, toolchain.Libc)
		}
		report.Compilers = append(report.Compilers, Compiler{Name: toolchain.Name, Version: toolchain.Version})
	}
	for _, profile := range nativeProfiles {
		report.Profiles = append(report.Profiles, profile.Name)
	}
	layoutManifest := filepath.Clean(filepath.Join(filepath.Dir(absoluteManifest), "..", "layout", "manifest-v0.json"))
	layoutTarget, layoutCases, err := layoutfixture.NativeProbeCases(layoutManifest)
	if err != nil {
		return Report{}, fmt.Errorf("load native layout probe cases: %w", err)
	}
	if layoutTarget != target.X8664LinuxGNU {
		return Report{}, fmt.Errorf("native layout probe target is %q, want %q", layoutTarget, target.X8664LinuxGNU)
	}
	layoutProbe, err := generateLayoutProbe(layoutTarget, layoutCases)
	if err != nil {
		return Report{}, fmt.Errorf("generate native layout probe: %w", err)
	}
	report.LayoutCases = slices.Clone(layoutProbe.Cases)
	layoutExecutions, err := runLayoutProbe(temporaryRoot, toolchains, layoutProbe)
	if err != nil {
		return Report{}, err
	}
	report.LayoutProbeExecutions = layoutExecutions
	abiManifest := filepath.Clean(filepath.Join(filepath.Dir(absoluteManifest), "..", "c-abi", "manifest-v0.json"))
	abiTarget, abiCases, err := cabifixture.NativeProbeCases(abiManifest)
	if err != nil {
		return Report{}, fmt.Errorf("load native C ABI probe cases: %w", err)
	}
	if abiTarget != target.X8664LinuxGNU {
		return Report{}, fmt.Errorf("native C ABI probe target is %q, want %q", abiTarget, target.X8664LinuxGNU)
	}
	abiProbes, err := generateABIProbes(abiTarget, abiCases)
	if err != nil {
		return Report{}, fmt.Errorf("generate native C ABI probes: %w", err)
	}
	for _, probe := range abiProbes {
		report.ABICases = append(report.ABICases, probe.ID)
	}
	report.ABIDirections = []string{"clang-caller-gcc-callee", "gcc-caller-clang-callee"}
	abiExecutions, err := runABIProbes(temporaryRoot, toolchains, abiProbes)
	if err != nil {
		return Report{}, err
	}
	report.ABIProbeExecutions = abiExecutions
	systemHeapSource, err := cstdlib.SystemHeapSource()
	if err != nil {
		return Report{}, fmt.Errorf("load SystemHeap implementation: %w", err)
	}
	systemHeapProbe, err := cstdlib.SystemHeapProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load SystemHeap probe: %w", err)
	}
	systemHeapExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "SystemHeap", "system-heap-probe", systemHeapProbe, systemHeapSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.SystemHeapExecutions = systemHeapExecutions
	systemHeapSourceCallExecutions, err := runSystemHeapSourceCallProbe(temporaryRoot, toolchains, systemHeapSource)
	if err != nil {
		return Report{}, err
	}
	report.SystemHeapSourceCallExecutions = systemHeapSourceCallExecutions
	storageSource, err := cstdlib.StorageSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Buffer storage implementation: %w", err)
	}
	bufferProbe, err := cstdlib.BufferProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Buffer probe: %w", err)
	}
	bufferExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Buffer", "buffer-probe", bufferProbe, storageSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.BufferExecutions = bufferExecutions
	bufferSourceCallExecutions, err := runBufferSourceCallProbe(temporaryRoot, toolchains, storageSource)
	if err != nil {
		return Report{}, err
	}
	report.BufferSourceCallExecutions = bufferSourceCallExecutions
	arenaSource, err := cstdlib.ArenaSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Arena implementation: %w", err)
	}
	arenaProbe, err := cstdlib.ArenaProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Arena probe: %w", err)
	}
	arenaExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Arena", "arena-probe", arenaProbe, arenaSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.ArenaExecutions = arenaExecutions
	arenaSourceCallExecutions, err := runArenaSourceCallProbe(temporaryRoot, toolchains, arenaSource)
	if err != nil {
		return Report{}, err
	}
	report.ArenaSourceCallExecutions = arenaSourceCallExecutions
	textViewSource, err := cstdlib.TextViewSource()
	if err != nil {
		return Report{}, fmt.Errorf("load text view implementation: %w", err)
	}
	textViewProbe, err := cstdlib.TextViewProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load text view probe: %w", err)
	}
	textViewExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "text view", "text-view-probe", textViewProbe, textViewSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.TextViewExecutions = textViewExecutions
	ownedTextSource, err := cstdlib.OwnedTextSource()
	if err != nil {
		return Report{}, fmt.Errorf("load owned text implementation: %w", err)
	}
	ownedTextProbe, err := cstdlib.OwnedTextProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load owned text probe: %w", err)
	}
	ownedTextExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "owned text", "owned-text-probe", ownedTextProbe, ownedTextSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.OwnedTextExecutions = ownedTextExecutions
	cStrValidationSourceCallExecutions, err := runCStrValidationSourceCallProbe(temporaryRoot, toolchains, ownedTextSource)
	if err != nil {
		return Report{}, err
	}
	report.CStrValidationSourceCallExecutions = cStrValidationSourceCallExecutions
	ownedTextSourceCallExecutions, err := runOwnedTextSourceCallProbe(temporaryRoot, toolchains, ownedTextSource)
	if err != nil {
		return Report{}, err
	}
	report.OwnedTextSourceCallExecutions = ownedTextSourceCallExecutions
	linuxFileSource, err := cstdlib.LinuxFileSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux File implementation: %w", err)
	}
	linuxFileProbe, err := cstdlib.LinuxFileProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux File probe: %w", err)
	}
	linuxFileExecutions, err := runStandardLibraryProbe(temporaryRoot, toolchains, "Linux File", "linux-file-probe", linuxFileProbe, linuxFileSource)
	if err != nil {
		return Report{}, err
	}
	report.LinuxFileExecutions = linuxFileExecutions
	linuxFileSourceCallExecutions, err := runLinuxFileSourceCallProbe(temporaryRoot, toolchains, linuxFileSource)
	if err != nil {
		return Report{}, err
	}
	report.LinuxFileSourceCallExecutions = linuxFileSourceCallExecutions
	foreignCallExecutions, err := runForeignCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCallExecutions = foreignCallExecutions
	foreignCIntCallExecutions, err := runForeignCIntCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCIntCallExecutions = foreignCIntCallExecutions
	foreignCLongCallExecutions, err := runForeignCLongCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCLongCallExecutions = foreignCLongCallExecutions
	foreignCScalarFamilyCallExecutions, err := runForeignCScalarFamilyCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCScalarFamilyCallExecutions = foreignCScalarFamilyCallExecutions
	foreignCSizeTCallExecutions, err := runForeignCSizeTCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCSizeTCallExecutions = foreignCSizeTCallExecutions
	foreignCSSizeTCallExecutions, err := runForeignCSSizeTCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCSSizeTCallExecutions = foreignCSSizeTCallExecutions
	foreignPointerCallExecutions, err := runForeignPointerCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignPointerCallExecutions = foreignPointerCallExecutions
	foreignCStructResultCallExecutions, err := runForeignCStructResultCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ForeignCStructResultCallExecutions = foreignCStructResultCallExecutions
	textValidationSourceCallExecutions, err := runTextValidationSourceCallProbe(temporaryRoot, toolchains, textViewSource)
	if err != nil {
		return Report{}, err
	}
	report.TextValidationSourceCallExecutions = textValidationSourceCallExecutions
	textViewOutputSourceCallExecutions, err := runTextViewOutputSourceCallProbe(temporaryRoot, toolchains, textViewSource)
	if err != nil {
		return Report{}, err
	}
	report.TextViewOutputSourceCallExecutions = textViewOutputSourceCallExecutions
	byteStringLiteralSourceCallExecutions, err := runByteStringLiteralSourceCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.ByteStringSourceCallExecutions = byteStringLiteralSourceCallExecutions
	utf8StringLiteralSourceCallExecutions, err := runUTF8StringLiteralSourceCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.UTF8StringSourceCallExecutions = utf8StringLiteralSourceCallExecutions
	cStringLiteralSourceCallExecutions, err := runCStringLiteralSourceCallProbe(temporaryRoot, toolchains)
	if err != nil {
		return Report{}, err
	}
	report.CStringLiteralSourceCallExecutions = cStringLiteralSourceCallExecutions
	for caseIndex, fixture := range fixtureReport.Emitted {
		if fixture.Unit.Target != string(target.X8664LinuxGNU) || fixture.Unit.Dialect != cbackend.DialectC11 {
			return Report{}, fmt.Errorf("case %s emitted unexpected profile %s/%s", fixture.ID, fixture.Unit.Target, fixture.Unit.Dialect)
		}
		report.Cases = append(report.Cases, fixture.ID)
		expectedExit, err := expectedExitCode(fixture)
		if err != nil {
			return Report{}, err
		}
		expectedDrops, err := expectedDropSiteIDs(fixture)
		if err != nil {
			return Report{}, err
		}
		for compilerIndex, toolchain := range toolchains {
			for profileIndex, profile := range nativeProfiles {
				artifact, err := cdriver.Build(toolchain, profile, fixture.TraceUnit.Source, traceRuntimeSource)
				if err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("program-%02d-%d-%d", caseIndex, compilerIndex, profileIndex))
				if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
					return Report{}, fmt.Errorf("materialize %s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				if err := execute(fixture.ID, executablePath, expectedExit, expectedDrops); err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				report.Executions++
				report.DropTraceExecutions++
			}
			sanitizedArtifact, err := cdriver.BuildSanitized(toolchain, fixture.Unit.Source, scalarRuntimeSource)
			if err != nil {
				return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			sanitizedPath := filepath.Join(temporaryRoot, fmt.Sprintf("sanitized-%02d-%d", caseIndex, compilerIndex))
			if err := os.WriteFile(sanitizedPath, sanitizedArtifact.Bytes, 0o700); err != nil {
				return Report{}, fmt.Errorf("materialize %s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			if err := executeSanitized(fixture.ID, sanitizedPath, expectedExit); err != nil {
				return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			report.SanitizerExecutions++
		}
	}
	buildSmoke, err := runBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.BuildSmokeExecutions = buildSmoke
	standardLibraryBuildSmoke, err := runStandardLibraryBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.StandardLibraryBuildSmokeExecutions = standardLibraryBuildSmoke
	byteStringLiteralBuildSmoke, err := runByteStringLiteralBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.ByteStringBuildSmokeExecutions = byteStringLiteralBuildSmoke
	utf8StringLiteralBuildSmoke, err := runUTF8StringLiteralBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.UTF8StringBuildSmokeExecutions = utf8StringLiteralBuildSmoke
	cStringLiteralBuildSmoke, err := runCStringLiteralBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.CStringLiteralBuildSmokeExecutions = cStringLiteralBuildSmoke
	return report, nil
}

func runByteStringLiteralSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"byte-string-literal-source-call",
		byteStringLiteralSourceCallSmokeSource,
		byteStringLiteralSourceCallCompanion,
		42,
		byteStringLiteralSourceCallExpectedCFragments,
	)
}

func runUTF8StringLiteralSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"utf8-string-literal-source-call",
		utf8StringLiteralSourceCallSmokeSource,
		utf8StringLiteralSourceCallCompanion,
		42,
		utf8StringLiteralSourceCallExpectedCFragments,
	)
}

func runCStringLiteralSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"c-string-literal-source-call",
		cStringLiteralSourceCallSmokeSource,
		cStringLiteralSourceCallCompanion,
		42,
		cStringLiteralSourceCallExpectedCFragments,
	)
}

func runSystemHeapSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, systemHeapSource string) (int, error) {
	if systemHeapSource == "" {
		return 0, fmt.Errorf("SystemHeap source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"system-heap-source-call",
		systemHeapSourceCallSmokeSource,
		systemHeapSource+systemHeapSourceCallBridge,
		42,
		systemHeapSourceCallExpectedCFragments,
	)
}

func runBufferSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, storageSource string) (int, error) {
	if storageSource == "" {
		return 0, fmt.Errorf("Buffer source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"buffer-source-call",
		bufferSourceCallSmokeSource,
		storageSource+bufferSourceCallBridge,
		42,
		bufferSourceCallExpectedCFragments,
	)
}

func runArenaSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, arenaSource string) (int, error) {
	if arenaSource == "" {
		return 0, fmt.Errorf("Arena source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"arena-source-call",
		arenaSourceCallSmokeSource,
		arenaSource+arenaSourceCallBridge,
		42,
		arenaSourceCallExpectedCFragments,
	)
}

func runForeignCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-call",
		foreignCallSmokeSource,
		foreignCallCompanionSource,
		42,
		[]string{
			"extern int32_t fairyc_fixture_add_one(int32_t fy_a_0);",
			"fairyc_fixture_add_one(INT32_C(41))",
		},
	)
}

func runForeignCIntCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-int-call",
		foreignCIntCallSmokeSource,
		foreignCIntCallCompanionSource,
		42,
		[]string{
			"_Static_assert(sizeof(int) == UINT64_C(4), \"fairyc target c.int size\");",
			"_Static_assert(_Alignof(int) == UINT64_C(4), \"fairyc target c.int alignment\");",
			"extern int fairyc_fixture_make_c_int(void);",
			"extern int32_t fairyc_fixture_consume_c_int(int fy_a_0);",
			"fairyc_fixture_make_c_int()",
			"fairyc_fixture_consume_c_int(fy_l_",
		},
	)
}

func runForeignCLongCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-long-call",
		foreignCLongCallSmokeSource,
		foreignCLongCallCompanionSource,
		42,
		[]string{
			"_Static_assert(sizeof(long) == UINT64_C(8), \"fairyc target c.long size\");",
			"_Static_assert(_Alignof(long) == UINT64_C(8), \"fairyc target c.long alignment\");",
			"_Static_assert(((long)-1) < (long)0, \"fairyc target c.long signed\");",
			"extern long fairyc_fixture_make_c_long(void);",
			"extern int32_t fairyc_fixture_consume_c_long(long fy_a_0);",
			"fairyc_fixture_make_c_long()",
			"fairyc_fixture_consume_c_long(fy_l_",
		},
	)
}

func runForeignCScalarFamilyCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-scalar-family-call",
		foreignCScalarFamilyCallSmokeSource,
		foreignCScalarFamilyCallCompanionSource,
		42,
		foreignCScalarFamilyExpectedCFragments,
	)
}

func runForeignCSizeTCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-size-t-call",
		foreignCSizeTCallSmokeSource,
		foreignCSizeTCallCompanionSource,
		42,
		[]string{
			"#include <stddef.h>",
			"_Static_assert(sizeof(size_t) == UINT64_C(8), \"fairyc target c.size_t size\");",
			"_Static_assert(_Alignof(size_t) == UINT64_C(8), \"fairyc target c.size_t alignment\");",
			"extern size_t fairyc_fixture_make_c_size_t(void);",
			"extern int32_t fairyc_fixture_consume_c_size_t(size_t fy_a_0);",
			"fairyc_fixture_make_c_size_t()",
			"fairyc_fixture_consume_c_size_t(fy_l_",
		},
	)
}

func runForeignCSSizeTCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-ssize-t-call",
		foreignCSSizeTCallSmokeSource,
		foreignCSSizeTCallCompanionSource,
		42,
		[]string{
			"#define _POSIX_C_SOURCE 200809L",
			"#include <sys/types.h>",
			"_Static_assert(sizeof(ssize_t) == UINT64_C(8), \"fairyc target c.ssize_t size\");",
			"_Static_assert(_Alignof(ssize_t) == UINT64_C(8), \"fairyc target c.ssize_t alignment\");",
			"extern ssize_t fairyc_fixture_make_c_ssize_t(void);",
			"extern int32_t fairyc_fixture_consume_c_ssize_t(ssize_t fy_a_0);",
			"fairyc_fixture_make_c_ssize_t()",
			"fairyc_fixture_consume_c_ssize_t(fy_l_",
		},
	)
}

func runForeignPointerCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-pointer-call",
		foreignPointerCallSmokeSource,
		foreignPointerCallCompanionSource,
		42,
		[]string{
			"extern uint32_t fairyc_fixture_pointer_scalars(const uint8_t * fy_a_0, uint32_t * fy_a_1, uintptr_t fy_a_2, uint8_t fy_a_3);",
			"fairyc_fixture_pointer_scalars(((void *)0), ((void *)0), ((uintptr_t)UINT64_C(0)), UINT8_C(7))",
		},
	)
}

func runForeignCStructResultCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-struct-result-call",
		foreignCStructResultCallSmokeSource,
		foreignCStructResultCallCompanionSource,
		42,
		foreignCStructResultCallExpectedCFragments,
	)
}

func runTextValidationSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, textViewSource string) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"text-validation-source-call",
		textValidationSourceCallSmokeSource,
		textViewSource,
		42,
		[]string{
			"extern uint32_t fy_std_text_utf8_valid_v0(const uint8_t * fy_a_0, uintptr_t fy_a_1);",
			"fy_std_text_utf8_valid_v0(((void *)0), ((uintptr_t)UINT64_C(0)))",
		},
	)
}

func runTextViewOutputSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, textViewSource string) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"text-view-output-source-call",
		textViewOutputSourceCallSmokeSource,
		textViewSource,
		42,
		[]string{
			"struct fy_std_text_view_v0 {",
			"extern uint32_t fy_std_text_validate_utf8_v0(const uint8_t * fy_a_0, uintptr_t fy_a_1, struct fy_std_text_view_v0 * fy_a_2);",
			" = &fy_l_",
			"fy_std_text_validate_utf8_v0(((void *)0), ((uintptr_t)UINT64_C(0)), fy_l_",
			".length",
		},
	)
}

func runCStrValidationSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, ownedTextSource string) (int, error) {
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"cstr-validation-source-call",
		cStrValidationSourceCallSmokeSource,
		ownedTextSource+cStrValidationSourceCallBridge,
		42,
		cStrValidationSourceCallExpectedCFragments,
	)
}

func runOwnedTextSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, ownedTextSource string) (int, error) {
	if ownedTextSource == "" {
		return 0, fmt.Errorf("owned-text source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"owned-text-source-call",
		ownedTextSourceCallSmokeSource,
		ownedTextSource+ownedTextSourceCallBridge,
		42,
		ownedTextSourceCallExpectedCFragments,
	)
}

func runLinuxFileSourceCallProbe(temporaryRoot string, toolchains []cdriver.Toolchain, linuxFileSource string) (int, error) {
	if linuxFileSource == "" {
		return 0, fmt.Errorf("Linux File source-call probe requires a non-empty implementation")
	}
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"linux-file-source-call",
		linuxFileSourceCallSmokeSource,
		linuxFileSource+linuxFileSourceCallBridge,
		42,
		linuxFileSourceCallExpectedCFragments,
	)
}

func runForeignSourceProbe(
	temporaryRoot string,
	toolchains []cdriver.Toolchain,
	probeID string,
	fairySource string,
	companionSource string,
	expectedExit int,
	requiredC []string,
) (int, error) {
	if len(toolchains) != 2 || toolchains[0].Name != "clang" || toolchains[1].Name != "gcc" {
		return 0, fmt.Errorf("%s probe requires resolved Clang then GCC toolchains", probeID)
	}
	if probeID == "" || fairySource == "" || companionSource == "" || len(requiredC) == 0 {
		return 0, fmt.Errorf("foreign source probe identity, sources, and C expectations must be non-empty")
	}
	sourcePath := "main.fy"
	parsed, err := frontend.AnalyzeSyntax(source.NewFile(sourcePath, []byte(fairySource)), source.DefaultLimits())
	if err != nil {
		return 0, fmt.Errorf("parse %s FY source: %w", probeID, err)
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return 0, fmt.Errorf("%s FY source is not parser-valid: %+v", probeID, parsed.Tree.Diagnostics)
	}
	semanticResult := semantic.Analyze([]semantic.File{{Path: sourcePath, Module: parsed.AST}})
	if semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return 0, fmt.Errorf("%s FY source did not complete Flow: phase=%q diagnostics=%+v unsupported=%+v", probeID, semanticResult.CompletedThrough, semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	program, err := mir.Lower(semanticResult.HIR)
	if err != nil {
		return 0, fmt.Errorf("lower %s FY source: %w", probeID, err)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return 0, fmt.Errorf("verify %s MIR: %+v", probeID, failures)
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0Profile())
	if err != nil {
		return 0, fmt.Errorf("emit %s C: %w", probeID, err)
	}
	for _, required := range requiredC {
		if !strings.Contains(unit.Source, required) {
			return 0, fmt.Errorf("%s C emission lost required fragment %q", probeID, required)
		}
	}
	directions := [][2]int{{0, 1}, {1, 0}}
	executions := 0
	for directionIndex, direction := range directions {
		caller := toolchains[direction[0]]
		callee := toolchains[direction[1]]
		for profileIndex, profile := range nativeProfiles {
			artifact, buildErr := cdriver.BuildCrossCompiler(caller, callee, profile, unit.Source, companionSource)
			if buildErr != nil {
				return 0, fmt.Errorf("%s probe with %s caller/%s callee/%s: %w", probeID, caller.Name, callee.Name, profile.Name, buildErr)
			}
			executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("%s-%d-%d", probeID, directionIndex, profileIndex))
			if writeErr := os.WriteFile(executablePath, artifact.Bytes, 0o700); writeErr != nil {
				return 0, fmt.Errorf("materialize %s probe with %s caller/%s callee/%s: %w", probeID, caller.Name, callee.Name, profile.Name, writeErr)
			}
			if executeErr := execute(probeID, executablePath, expectedExit, nil); executeErr != nil {
				return 0, fmt.Errorf("%s probe with %s caller/%s callee/%s: %w", probeID, caller.Name, callee.Name, profile.Name, executeErr)
			}
			executions++
		}
	}
	return executions, nil
}

func runStandardLibraryProbe(
	temporaryRoot string,
	toolchains []cdriver.Toolchain,
	name string,
	filePrefix string,
	probeSource string,
	implementationSource string,
) (int, error) {
	if len(toolchains) != 2 || toolchains[0].Name != "clang" || toolchains[1].Name != "gcc" {
		return 0, fmt.Errorf("%s probe requires resolved Clang then GCC toolchains", name)
	}
	if name == "" || filePrefix == "" || probeSource == "" || implementationSource == "" {
		return 0, fmt.Errorf("standard-library probe identity and sources must be non-empty")
	}
	executions := 0
	for compilerIndex, toolchain := range toolchains {
		for profileIndex, profile := range nativeProfiles {
			artifact, err := cdriver.Build(toolchain, profile, probeSource, implementationSource)
			if err != nil {
				return 0, fmt.Errorf("%s probe with %s/%s: %w", name, toolchain.Name, profile.Name, err)
			}
			executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("%s-%d-%d", filePrefix, compilerIndex, profileIndex))
			if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
				return 0, fmt.Errorf("materialize %s probe with %s/%s: %w", name, toolchain.Name, profile.Name, err)
			}
			if err := execute(name+" probe", executablePath, 0, nil); err != nil {
				return 0, fmt.Errorf("%s probe with %s/%s: %w", name, toolchain.Name, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
}

func runABIProbes(temporaryRoot string, toolchains []cdriver.Toolchain, probes []abiProbeCase) (int, error) {
	if len(toolchains) != 2 || toolchains[0].Name != "clang" || toolchains[1].Name != "gcc" {
		return 0, fmt.Errorf("native C ABI probes require resolved Clang then GCC toolchains")
	}
	directions := [][2]int{{0, 1}, {1, 0}}
	executions := 0
	for caseIndex, probe := range probes {
		for directionIndex, direction := range directions {
			caller := toolchains[direction[0]]
			callee := toolchains[direction[1]]
			for profileIndex, profile := range nativeProfiles {
				artifact, err := cdriver.BuildCrossCompiler(caller, callee, profile, probe.CallerSource, probe.CalleeSource)
				if err != nil {
					return 0, fmt.Errorf("C ABI probe %s with %s caller/%s callee/%s: %w", probe.ID, caller.Name, callee.Name, profile.Name, err)
				}
				executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("abi-probe-%02d-%d-%d", caseIndex, directionIndex, profileIndex))
				if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
					return 0, fmt.Errorf("materialize C ABI probe %s with %s caller/%s callee/%s: %w", probe.ID, caller.Name, callee.Name, profile.Name, err)
				}
				if err := execute("C ABI probe "+probe.ID, executablePath, 0, nil); err != nil {
					return 0, fmt.Errorf("C ABI probe %s with %s caller/%s callee/%s: %w", probe.ID, caller.Name, callee.Name, profile.Name, err)
				}
				executions++
			}
		}
	}
	return executions, nil
}

func runLayoutProbe(temporaryRoot string, toolchains []cdriver.Toolchain, probe layoutProbeUnit) (int, error) {
	executions := 0
	for compilerIndex, toolchain := range toolchains {
		for profileIndex, profile := range nativeProfiles {
			artifact, err := cdriver.Build(toolchain, profile, probe.Source, layoutProbeRuntimeSource)
			if err != nil {
				return 0, fmt.Errorf("layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("layout-probe-%d-%d", compilerIndex, profileIndex))
			if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
				return 0, fmt.Errorf("materialize layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			if err := execute("layout-probe", executablePath, 0, nil); err != nil {
				return 0, fmt.Errorf("layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
}

func runBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "product-build-smoke", buildSmokeSource)
}

func runStandardLibraryBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "stdlib-build-smoke", textValidationSourceCallSmokeSource)
}

func runByteStringLiteralBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "byte-string-literal-build-smoke", byteStringLiteralBuildSmokeSource)
}

func runUTF8StringLiteralBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "utf8-string-literal-build-smoke", utf8StringLiteralBuildSmokeSource)
}

func runCStringLiteralBuildSmoke(temporaryRoot string) (int, error) {
	return runBuildSourceSmoke(temporaryRoot, "c-string-literal-build-smoke", cStringLiteralBuildSmokeSource)
}

func runBuildSourceSmoke(temporaryRoot, smokeID, source string) (int, error) {
	if smokeID == "" || source == "" {
		return 0, fmt.Errorf("product build smoke identity and source must be non-empty")
	}
	sourceRoot := filepath.Join(temporaryRoot, smokeID)
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		return 0, fmt.Errorf("create %s root: %w", smokeID, err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "main.fy"), []byte(source), 0o600); err != nil {
		return 0, fmt.Errorf("write %s source: %w", smokeID, err)
	}
	executions := 0
	for _, compiler := range []string{"clang", "gcc"} {
		for _, profile := range nativeProfiles {
			output := fmt.Sprintf("build/smoke-%s-%s", compiler, profile.Name)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := cli.Run([]string{
				"experimental", "build", "main.fy",
				"--cc", compiler,
				"--profile", profile.Name,
				"--output", output,
			}, sourceRoot, &stdout, &stderr)
			if code != cli.ExitSuccess {
				return 0, fmt.Errorf("%s with %s/%s failed with code %d: %s", smokeID, compiler, profile.Name, code, strings.TrimSpace(stderr.String()))
			}
			expectedReceipt := fmt.Sprintf("built: %s (%s, %s, %s)\n", output, target.X8664LinuxGNU, compiler, profile.Name)
			if stdout.String() != expectedReceipt || stderr.Len() != 0 {
				return 0, fmt.Errorf("%s with %s/%s produced stdout %q and stderr %q", smokeID, compiler, profile.Name, stdout.String(), stderr.String())
			}
			if err := execute(smokeID, filepath.Join(sourceRoot, filepath.FromSlash(output)), 42, nil); err != nil {
				return 0, fmt.Errorf("%s with %s/%s: %w", smokeID, compiler, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
}

func resolveToolchains() ([]cdriver.Toolchain, error) {
	toolchains := make([]cdriver.Toolchain, 0, 2)
	for _, name := range []string{"clang", "gcc"} {
		toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, name)
		if err != nil {
			return nil, fmt.Errorf("native C differential requires %s: %w", name, err)
		}
		toolchains = append(toolchains, toolchain)
	}
	return toolchains, nil
}

func execute(caseID, executablePath string, expectedExit int, expectedDrops []uint32) error {
	result, err := executeProcess(executablePath, nil)
	if err != nil {
		return err
	}
	if len(result.stdout) != 0 {
		return fmt.Errorf("native executable produced unexpected stdout %q", string(result.stdout))
	}
	actualDrops, err := decodeDropTrace(result.stderr)
	if err != nil {
		return fmt.Errorf("native drop trace for %s: %w", caseID, err)
	}
	if !slices.Equal(actualDrops, expectedDrops) {
		return fmt.Errorf("native drop trace for %s: got %v, want %v", caseID, actualDrops, expectedDrops)
	}
	if result.exitCode != expectedExit {
		return fmt.Errorf("native exit for %s: got %d, want %d", caseID, result.exitCode, expectedExit)
	}
	return nil
}

func executeSanitized(caseID, executablePath string, expectedExit int) error {
	result, err := executeProcess(executablePath, sanitizerEnvironment(os.Environ()))
	if err != nil {
		return err
	}
	if len(result.stdout) != 0 {
		return fmt.Errorf("sanitized executable produced unexpected stdout %q", string(result.stdout))
	}
	if len(result.stderr) != 0 {
		diagnostic := strings.ReplaceAll(strings.TrimSpace(string(result.stderr)), filepath.Dir(executablePath), "<temporary>")
		return fmt.Errorf("sanitizer reported for %s: %s", caseID, diagnostic)
	}
	if result.exitCode != expectedExit {
		return fmt.Errorf("sanitized exit for %s: got %d, want %d", caseID, result.exitCode, expectedExit)
	}
	return nil
}

type processResult struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func executeProcess(executablePath string, environment []string) (processResult, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), executionTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath)
	command.Stdin = nil
	if environment != nil {
		command.Env = environment
	}
	stdout := boundedBuffer{limit: maxProcessOutput}
	stderr := boundedBuffer{limit: maxProcessOutput}
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return processResult{}, fmt.Errorf("native executable exceeded %s", executionTimeout)
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return processResult{}, fmt.Errorf("execute native program: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	if stdout.truncated || stderr.truncated || len(stdout.bytes)+len(stderr.bytes) > maxProcessOutput {
		return processResult{}, fmt.Errorf("native executable exceeded %d output bytes", maxProcessOutput)
	}
	return processResult{exitCode: exitCode, stdout: bytes.Clone(stdout.bytes), stderr: bytes.Clone(stderr.bytes)}, nil
}

func sanitizerEnvironment(base []string) []string {
	result := make([]string, 0, len(base)+3)
	for _, entry := range base {
		name, _, _ := strings.Cut(entry, "=")
		if name == "ASAN_OPTIONS" || name == "LSAN_OPTIONS" || name == "UBSAN_OPTIONS" {
			continue
		}
		result = append(result, entry)
	}
	return append(
		result,
		"ASAN_OPTIONS=abort_on_error=1:color=never:detect_leaks=1:halt_on_error=1:symbolize=0",
		"LSAN_OPTIONS=exitcode=23:report_objects=0",
		"UBSAN_OPTIONS=color=never:halt_on_error=1:print_stacktrace=0",
	)
}

func expectedDropSiteIDs(fixture cbackendfixture.EmittedCase) ([]uint32, error) {
	result := make([]uint32, 0, len(fixture.DropTrace))
	for _, event := range fixture.DropTrace {
		found := false
		for _, site := range fixture.TraceUnit.DropSites {
			if site.ResourceType != string(event.ResourceType) || site.Package != event.Binding.Package || site.Source != event.Binding.Source ||
				site.StartByte != event.Binding.StartByte || site.EndByte != event.Binding.EndByte {
				continue
			}
			result = append(result, site.ID)
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("drop trace for %s has no C site for %s at %s:%d-%d", fixture.ID, event.ResourceType, event.Binding.Source, event.Binding.StartByte, event.Binding.EndByte)
		}
	}
	return result, nil
}

func decodeDropTrace(contents []byte) ([]uint32, error) {
	if len(contents)%4 != 0 {
		return nil, fmt.Errorf("record stream has %d bytes, want a multiple of 4", len(contents))
	}
	result := make([]uint32, 0, len(contents)/4)
	for offset := 0; offset < len(contents); offset += 4 {
		result = append(result, binary.LittleEndian.Uint32(contents[offset:offset+4]))
	}
	return result, nil
}

func expectedExitCode(fixture cbackendfixture.EmittedCase) (int, error) {
	switch fixture.Outcome {
	case "return":
		if fixture.Value < 0 || fixture.Value >= trapExitBase {
			return 0, fmt.Errorf("native process-status oracle cannot represent return %d in %s", fixture.Value, fixture.ID)
		}
		return int(fixture.Value), nil
	case "trap":
		code, ok := cbackend.TrapCode(fixture.TrapCategory)
		if !ok {
			return 0, fmt.Errorf("unknown trap category %q in %s", fixture.TrapCategory, fixture.ID)
		}
		return trapExitBase + int(code), nil
	default:
		return 0, fmt.Errorf("unsupported native outcome %q in %s", fixture.Outcome, fixture.ID)
	}
}

func summarizeFailures(failures []cbackendfixture.Failure) string {
	if len(failures) == 0 {
		return "emitted case count mismatch"
	}
	values := make([]string, 0, len(failures))
	for _, failure := range failures {
		values = append(values, failure.CaseID+": "+failure.Detail)
	}
	slices.Sort(values)
	return strings.Join(values, "; ")
}

type boundedBuffer struct {
	limit     int
	bytes     []byte
	truncated bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	available := b.limit - len(b.bytes)
	if available > 0 {
		b.bytes = append(b.bytes, value[:min(available, len(value))]...)
	}
	if len(value) > available {
		b.truncated = true
	}
	return len(value), nil
}

func (b *boundedBuffer) String() string {
	return string(bytes.Clone(b.bytes))
}
