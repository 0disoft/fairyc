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
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/foreignstructprobe"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	reportSchemaVersion = "fairyc.native-c-report/v36"
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
    i32 value;
    cleanup {}
}
fn Option<Token> choose() { return .some(Token { .value = 41 }); }
fn Result<Token, i32> increment(Result<Token, i32> input) {
	let Token token = input?;
	return .ok(Token { .value = token.value + 1 });
}
fn i32 main() {
	let Token token = switch (choose()) {
		case .some(token) : { token }
		case .none : { Token { .value = 0 } }
	};
	return switch (increment(.ok(token))) {
		case .ok(token) : { token.value }
		case .err(error) : { -error }
    };
}
`

const publicFileBuildSmokeSource = `module main;

import std::fs as fs;

@effects(filesystem)
fn i32 write_then_close(fs::File input) {
    var fs::File file = input;
    let ref u8[] payload = b"Fairy C";
    let i32 write_code = switch (fs::write_all(ref mut file, payload)) {
        case .ok(written) : { if (written == 7) { 0 } else { 1 } }
        case .err(_) : { 2 }
    };
    if (write_code != 0) {
        return write_code;
    }
    let i32 position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 7) { 0 } else { 10 } }
        case .err(_) : { 11 }
    };
    if (position_code != 0) {
        return position_code;
    }
    let i32 size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 7) { 0 } else { 22 } }
        case .err(_) : { 23 }
    };
    if (size_code != 0) {
        return size_code;
    }
    let i32 seek_code = switch (fs::seek_start(ref mut file, 2)) {
        case .ok(position) : { if (position == 2) { 0 } else { 12 } }
        case .err(_) : { 13 }
    };
    if (seek_code != 0) {
        return seek_code;
    }
    let i32 observed_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 2) { 0 } else { 14 } }
        case .err(_) : { 15 }
    };
    if (observed_code != 0) {
        return observed_code;
    }
    let i32 preserved_size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 7) { 0 } else { 24 } }
        case .err(_) : { 25 }
    };
    if (preserved_size_code != 0) {
        return preserved_size_code;
    }
    let i32 size_position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 2) { 0 } else { 26 } }
        case .err(_) : { 27 }
    };
    if (size_position_code != 0) {
        return size_position_code;
    }
    let i32 current_code = switch (fs::seek_current(ref mut file, -1)) {
        case .ok(position) : { if (position == 1) { 0 } else { 28 } }
        case .err(_) : { 29 }
    };
    if (current_code != 0) {
        return current_code;
    }
    let i32 end_code = switch (fs::seek_end(ref mut file, -2)) {
        case .ok(position) : { if (position == 5) { 0 } else { 30 } }
        case .err(_) : { 31 }
    };
    if (end_code != 0) {
        return end_code;
    }
    let i32 beyond_end_code = switch (fs::seek_end(ref mut file, 3)) {
        case .ok(position) : { if (position == 10) { 0 } else { 32 } }
        case .err(_) : { 33 }
    };
    if (beyond_end_code != 0) {
        return beyond_end_code;
    }
    let i32 beyond_end_size_code = switch (fs::size(ref file)) {
        case .ok(size) : { if (size == 7) { 0 } else { 34 } }
        case .err(_) : { 35 }
    };
    if (beyond_end_size_code != 0) {
        return beyond_end_size_code;
    }
    let i32 beyond_end_position_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 10) { 0 } else { 36 } }
        case .err(_) : { 37 }
    };
    if (beyond_end_position_code != 0) {
        return beyond_end_position_code;
    }
    let i32 reset_code = switch (fs::seek_start(ref mut file, 2)) {
        case .ok(position) : { if (position == 2) { 0 } else { 38 } }
        case .err(_) : { 39 }
    };
    if (reset_code != 0) {
        return reset_code;
    }
    let usize invalid_offset = 18446744073709551615;
    let i32 rejected_code = switch (fs::seek_start(ref mut file, invalid_offset)) {
        case .ok(_) : { 18 }
        case .err(error) : { if (fs::error_status(error) == 2) { 0 } else { 19 } }
    };
    if (rejected_code != 0) {
        return rejected_code;
    }
    let i32 preserved_code = switch (fs::position(ref mut file)) {
        case .ok(position) : { if (position == 2) { 0 } else { 20 } }
        case .err(_) : { 21 }
    };
    if (preserved_code != 0) {
        return preserved_code;
    }
    let i32 restore_code = switch (fs::seek_start(ref mut file, 7)) {
        case .ok(position) : { if (position == 7) { 0 } else { 16 } }
        case .err(_) : { 17 }
    };
    if (restore_code != 0) {
        return restore_code;
    }
    let i32 sync_code = switch (fs::sync(ref mut file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 3 }
    };
    if (sync_code != 0) {
        return sync_code;
    }
    return switch (fs::close(file)) {
        case .ok(_) : { 0 }
        case .err(_) : { 4 }
    };
}

@effects(filesystem)
fn i32 read_then_close(fs::File input) {
    var fs::File file = input;
    var u8[7] buffer = [0, 0, 0, 0, 0, 0, 0];
    let i32 read_code = switch (fs::read(ref mut file, ref mut buffer)) {
        case .ok(read) : {
            if (read == 7 &&
                buffer[0] == 70 && buffer[1] == 97 && buffer[2] == 105 &&
                buffer[3] == 114 && buffer[4] == 121 && buffer[5] == 32 &&
                buffer[6] == 67) { 0 } else { 5 }
        }
        case .err(_) : { 6 }
    };
    if (read_code != 0) {
        return read_code;
    }
    return switch (fs::close(file)) {
        case .ok(_) : { 42 }
        case .err(_) : { 7 }
    };
}

@effects(filesystem)
fn i32 main() {
    let ref CStr path = c"fairyc-public-file-smoke.tmp";
    let i32 write_code = switch (fs::create_truncate(path)) {
        case .ok(file) : { write_then_close(file) }
        case .err(_) : { 8 }
    };
    if (write_code != 0) {
        return write_code;
    }
    return switch (fs::open_read(path)) {
        case .ok(file) : { read_then_close(file) }
        case .err(_) : { 9 }
    };
}
`

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

const foreignCallSmokeSource = `module main;
@link_name("fairyc_fixture_add_one")
unsafe extern C fn i32 add_one(i32 value) ;
fn i32 main() { return unsafe { add_one(41) }; }
`

const foreignCallCompanionSource = `#include <stdint.h>

int32_t fairyc_fixture_add_one(int32_t value) {
    return value + INT32_C(1);
}
`

const foreignCIntCallSmokeSource = `module main;
@link_name("fairyc_fixture_make_c_int")
unsafe extern C fn c::int make_c_int() ;
@link_name("fairyc_fixture_consume_c_int")
unsafe extern C fn i32 consume_c_int(c::int value) ;
fn i32 main() { return unsafe { consume_c_int(make_c_int()) }; }
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
unsafe extern C fn c::long make_c_long() ;
@link_name("fairyc_fixture_consume_c_long")
unsafe extern C fn i32 consume_c_long(c::long value) ;
fn i32 main() { return unsafe { consume_c_long(make_c_long()) }; }
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
unsafe extern C fn c::size_t make_c_size_t() ;
@link_name("fairyc_fixture_consume_c_size_t")
unsafe extern C fn i32 consume_c_size_t(c::size_t value) ;
fn i32 main() { return unsafe { consume_c_size_t(make_c_size_t()) }; }
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
unsafe extern C fn c::ssize_t make_c_ssize_t() ;
@link_name("fairyc_fixture_consume_c_ssize_t")
unsafe extern C fn i32 consume_c_ssize_t(c::ssize_t value) ;
fn i32 main() { return unsafe { consume_c_ssize_t(make_c_ssize_t()) }; }
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
unsafe extern C fn c::bool fairyc_fixture_make_c_bool() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_bool(c::bool value) ;
unsafe extern C fn c::char fairyc_fixture_make_c_char() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_char(c::char value) ;
unsafe extern C fn c::double fairyc_fixture_make_c_double() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_double(c::double value) ;
unsafe extern C fn c::float fairyc_fixture_make_c_float() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_float(c::float value) ;
unsafe extern C fn c::longlong fairyc_fixture_make_c_longlong() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_longlong(c::longlong value) ;
unsafe extern C fn c::schar fairyc_fixture_make_c_schar() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_schar(c::schar value) ;
unsafe extern C fn c::short fairyc_fixture_make_c_short() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_short(c::short value) ;
unsafe extern C fn c::uchar fairyc_fixture_make_c_uchar() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_uchar(c::uchar value) ;
unsafe extern C fn c::uint fairyc_fixture_make_c_uint() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_uint(c::uint value) ;
unsafe extern C fn c::ulong fairyc_fixture_make_c_ulong() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_ulong(c::ulong value) ;
unsafe extern C fn c::ulonglong fairyc_fixture_make_c_ulonglong() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_ulonglong(c::ulonglong value) ;
unsafe extern C fn c::ushort fairyc_fixture_make_c_ushort() ;
unsafe extern C fn i32 fairyc_fixture_consume_c_ushort(c::ushort value) ;

fn i32 check_bool() { return unsafe { fairyc_fixture_consume_c_bool(fairyc_fixture_make_c_bool()) }; }
fn i32 check_char() { return unsafe { fairyc_fixture_consume_c_char(fairyc_fixture_make_c_char()) }; }
fn i32 check_double() { return unsafe { fairyc_fixture_consume_c_double(fairyc_fixture_make_c_double()) }; }
fn i32 check_float() { return unsafe { fairyc_fixture_consume_c_float(fairyc_fixture_make_c_float()) }; }
fn i32 check_longlong() { return unsafe { fairyc_fixture_consume_c_longlong(fairyc_fixture_make_c_longlong()) }; }
fn i32 check_schar() { return unsafe { fairyc_fixture_consume_c_schar(fairyc_fixture_make_c_schar()) }; }
fn i32 check_short() { return unsafe { fairyc_fixture_consume_c_short(fairyc_fixture_make_c_short()) }; }
fn i32 check_uchar() { return unsafe { fairyc_fixture_consume_c_uchar(fairyc_fixture_make_c_uchar()) }; }
fn i32 check_uint() { return unsafe { fairyc_fixture_consume_c_uint(fairyc_fixture_make_c_uint()) }; }
fn i32 check_ulong() { return unsafe { fairyc_fixture_consume_c_ulong(fairyc_fixture_make_c_ulong()) }; }
fn i32 check_ulonglong() { return unsafe { fairyc_fixture_consume_c_ulonglong(fairyc_fixture_make_c_ulonglong()) }; }
fn i32 check_ushort() { return unsafe { fairyc_fixture_consume_c_ushort(fairyc_fixture_make_c_ushort()) }; }

fn i32 main() {
    let i32 failures = check_bool() + check_char() + check_double() + check_float() +
        check_longlong() + check_schar() + check_short() + check_uchar() + check_uint() +
        check_ulong() + check_ulonglong() + check_ushort();
    return if (failures == 0) { 42 } else { 1 };
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
unsafe extern C fn u32 fairyc_fixture_pointer_scalars(ptr u8 data, ptr mut u32 output, usize length, u8 marker) ;
fn i32 main() {
    return if (unsafe { fairyc_fixture_pointer_scalars(null, null, 0, 7) } == 42) { 42 } else { 1 };
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

const textValidationSourceCallSmokeSource = `module main;
@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn u32 utf8_valid(ptr u8 data, usize length) ;
fn i32 main() {
    return if (unsafe { utf8_valid(null, 0) } == 1) { 42 } else { 1 };
}
`

const textViewOutputSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_text_view_v0 {
    ptr u8 data;
    usize length;
}
@link_name("fy_std_text_validate_utf8_v0")
unsafe extern C fn u32 validate_utf8(ptr u8 data, usize length, ptr mut fy_std_text_view_v0 output) ;
fn i32 main() {
    var fy_std_text_view_v0 output = fy_std_text_view_v0 { .data = null, .length = 99 };
    let u32 status = unsafe {
        validate_utf8(
            null,
            0,
            ref mut output as ptr mut fy_std_text_view_v0
        )
    };
    return if (status == 0 && output.length == 0) { 42 } else { 1 };
}
`

const systemHeapSourceCallSmokeSource = `module main;
@repr(C)
struct fy_std_system_heap_block_v0 {
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}

@link_name("fairyc_fixture_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn c::size_t eight_size_t() ;
@link_name("fy_std_system_heap_allocate_v0")
unsafe extern C fn u32 allocate(c::size_t size, c::size_t alignment, ptr mut fy_std_system_heap_block_v0 output) ;
@link_name("fairyc_fixture_block_is_allocated")
unsafe extern C fn u32 block_is_allocated(ptr fy_std_system_heap_block_v0 block) ;
@link_name("fy_std_system_heap_deallocate_v0")
unsafe extern C fn u32 deallocate(ptr mut fy_std_system_heap_block_v0 block) ;
@link_name("fairyc_fixture_block_is_empty")
unsafe extern C fn u32 block_is_empty(ptr fy_std_system_heap_block_v0 block) ;

fn i32 main() {
    let c::size_t zero = unsafe { zero_size_t() };
    var fy_std_system_heap_block_v0 block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    let c::size_t size = unsafe { eight_size_t() };
    let c::size_t alignment = unsafe { eight_size_t() };
    let u32 allocated = unsafe {
        allocate(size, alignment, ref mut block as ptr mut fy_std_system_heap_block_v0)
    };
    let u32 observed = unsafe {
        block_is_allocated(ref block as ptr fy_std_system_heap_block_v0)
    };
    let u32 deallocated = unsafe {
        deallocate(ref mut block as ptr mut fy_std_system_heap_block_v0)
    };
    let u32 cleared = unsafe {
        block_is_empty(ref block as ptr fy_std_system_heap_block_v0)
    };
    return if (allocated == 0 && observed == 1 && deallocated == 0 && cleared == 1) { 42 } else { 1 };
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
    fy_std_system_heap_block_v0 block;
    c::size_t length;
}
@repr(C)
struct fy_std_system_heap_block_v0 {
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}

@link_name("fairyc_fixture_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_four_size_t")
unsafe extern C fn c::size_t four_size_t() ;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn c::size_t eight_size_t() ;
@link_name("fy_std_buffer_initialize_v0")
unsafe extern C fn u32 initialize(c::size_t capacity, ptr mut fy_std_buffer_v0 output) ;
@link_name("fy_std_buffer_resize_zeroed_v0")
unsafe extern C fn u32 resize_zeroed(ptr mut fy_std_buffer_v0 buffer, c::size_t new_length) ;
@link_name("fairyc_fixture_buffer_is_zeroed")
unsafe extern C fn u32 buffer_is_zeroed(ptr fy_std_buffer_v0 buffer) ;
@link_name("fy_std_buffer_drop_v0")
unsafe extern C fn u32 drop_buffer(ptr mut fy_std_buffer_v0 buffer) ;
@link_name("fairyc_fixture_buffer_is_empty")
unsafe extern C fn u32 buffer_is_empty(ptr fy_std_buffer_v0 buffer) ;

fn i32 main() {
    let c::size_t zero = unsafe { zero_size_t() };
    let fy_std_system_heap_block_v0 empty_block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    var fy_std_buffer_v0 buffer = fy_std_buffer_v0 {
        .block = empty_block,
        .length = zero,
    };
    let c::size_t capacity = unsafe { eight_size_t() };
    let u32 initialized = unsafe {
        initialize(capacity, ref mut buffer as ptr mut fy_std_buffer_v0)
    };
    let c::size_t length = unsafe { four_size_t() };
    let u32 resized = unsafe {
        resize_zeroed(ref mut buffer as ptr mut fy_std_buffer_v0, length)
    };
    let u32 observed = unsafe {
        buffer_is_zeroed(ref buffer as ptr fy_std_buffer_v0)
    };
    let u32 dropped = unsafe {
        drop_buffer(ref mut buffer as ptr mut fy_std_buffer_v0)
    };
    let u32 cleared = unsafe {
        buffer_is_empty(ref buffer as ptr fy_std_buffer_v0)
    };
    return if (initialized == 0 && resized == 0 && observed == 1 && dropped == 0 && cleared == 1) {
        42
    } else {
        1
    };
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
    ptr mut fy_std_arena_node_v0 head;
    c::ulong generation;
    u8 active_region;
}
@repr(C)
struct fy_std_region_v0 {
    ptr mut fy_std_arena_v0 arena;
    ptr mut fy_std_arena_node_v0 mark;
    c::ulong generation;
    u8 active;
}
@repr(C)
struct fy_std_arena_node_v0 {
    fy_std_system_heap_block_v0 storage;
    ptr mut fy_std_arena_node_v0 next;
}
@repr(C)
struct fy_std_system_heap_block_v0 {
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}

@link_name("fairyc_fixture_zero_c_ulong")
unsafe extern C fn c::ulong zero_c_ulong() ;
@link_name("fairyc_fixture_eight_size_t")
unsafe extern C fn c::size_t eight_size_t() ;
@link_name("fy_std_arena_initialize_v0")
unsafe extern C fn u32 initialize_arena(ptr mut fy_std_arena_v0 output) ;
@link_name("fy_std_region_begin_v0")
unsafe extern C fn u32 begin_region(ptr mut fy_std_arena_v0 arena, ptr mut fy_std_region_v0 output) ;
@link_name("fy_std_region_allocate_zeroed_v0")
unsafe extern C fn u32 allocate_zeroed(ptr mut fy_std_region_v0 scope, c::size_t size, c::size_t alignment, ptr mut ptr mut void output) ;
@link_name("fairyc_fixture_arena_allocation_is_zeroed")
unsafe extern C fn u32 allocation_is_zeroed(ptr mut fy_std_arena_v0 arena, ptr mut fy_std_region_v0 scope, ptr mut void data) ;
@link_name("fy_std_region_reset_v0")
unsafe extern C fn u32 reset_region(ptr mut fy_std_region_v0 scope) ;
@link_name("fairyc_fixture_arena_region_is_reset")
unsafe extern C fn u32 region_is_reset(ptr mut fy_std_arena_v0 arena, ptr mut fy_std_region_v0 scope) ;
@link_name("fy_std_arena_drop_v0")
unsafe extern C fn u32 drop_arena(ptr mut fy_std_arena_v0 arena) ;
@link_name("fairyc_fixture_arena_is_dropped")
unsafe extern C fn u32 arena_is_dropped(ptr mut fy_std_arena_v0 arena) ;

fn i32 main() {
    let c::ulong zero_generation = unsafe { zero_c_ulong() };
    var fy_std_arena_v0 arena = fy_std_arena_v0 {
        .head = null,
        .generation = zero_generation,
        .active_region = 0,
    };
    var fy_std_region_v0 scope = fy_std_region_v0 {
        .arena = null,
        .mark = null,
        .generation = zero_generation,
        .active = 0,
    };
    let u32 initialized = unsafe {
        initialize_arena(ref mut arena as ptr mut fy_std_arena_v0)
    };
    let u32 began = unsafe {
        begin_region(
            ref mut arena as ptr mut fy_std_arena_v0,
            ref mut scope as ptr mut fy_std_region_v0
        )
    };
    let c::size_t size = unsafe { eight_size_t() };
    let c::size_t alignment = unsafe { eight_size_t() };
    var ptr mut void data = null;
    let u32 allocated = unsafe {
        allocate_zeroed(
            ref mut scope as ptr mut fy_std_region_v0,
            size,
            alignment,
            ref mut data as ptr mut ptr mut void
        )
    };
    let u32 observed = unsafe {
        allocation_is_zeroed(
            ref mut arena as ptr mut fy_std_arena_v0,
            ref mut scope as ptr mut fy_std_region_v0,
            data
        )
    };
    let u32 reset = unsafe {
        reset_region(ref mut scope as ptr mut fy_std_region_v0)
    };
    let u32 cleared = unsafe {
        region_is_reset(
            ref mut arena as ptr mut fy_std_arena_v0,
            ref mut scope as ptr mut fy_std_region_v0
        )
    };
    let u32 dropped = unsafe {
        drop_arena(ref mut arena as ptr mut fy_std_arena_v0)
    };
    let u32 finalized = unsafe {
        arena_is_dropped(ref mut arena as ptr mut fy_std_arena_v0)
    };
    return if (initialized == 0 && began == 0 && allocated == 0 && observed == 1 &&
        reset == 0 && cleared == 1 && dropped == 0 && finalized == 1) {
        42
    } else {
        1
    };
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
    ptr u8 data;
    c::size_t length;
}

@link_name("fairyc_fixture_cstr_valid_storage")
unsafe extern C fn ptr u8 valid_storage() ;
@link_name("fairyc_fixture_cstr_valid_storage_length")
unsafe extern C fn c::size_t valid_storage_length() ;
@link_name("fairyc_fixture_cstr_interior_nul_storage")
unsafe extern C fn ptr u8 interior_nul_storage() ;
@link_name("fairyc_fixture_cstr_interior_nul_storage_length")
unsafe extern C fn c::size_t interior_nul_storage_length() ;
@link_name("fairyc_fixture_cstr_unterminated_storage")
unsafe extern C fn ptr u8 unterminated_storage() ;
@link_name("fairyc_fixture_cstr_unterminated_storage_length")
unsafe extern C fn c::size_t unterminated_storage_length() ;
@link_name("fy_std_cstr_validate_v0")
unsafe extern C fn u32 validate_cstr(ptr u8 storage, c::size_t storage_length, ptr mut fy_std_cstr_view_v0 output) ;
@link_name("fairyc_fixture_cstr_view_is_valid")
unsafe extern C fn u32 view_is_valid(ptr fy_std_cstr_view_v0 view) ;

fn i32 main() {
    let ptr u8 valid = unsafe { valid_storage() };
    let c::size_t valid_length = unsafe { valid_storage_length() };
    var fy_std_cstr_view_v0 output = fy_std_cstr_view_v0 {
        .data = null,
        .length = valid_length,
    };
    let u32 accepted = unsafe {
        validate_cstr(valid, valid_length, ref mut output as ptr mut fy_std_cstr_view_v0)
    };
    let u32 valid_observed = unsafe {
        view_is_valid(ref output as ptr fy_std_cstr_view_v0)
    };

    let ptr u8 interior_nul = unsafe { interior_nul_storage() };
    let c::size_t interior_nul_length = unsafe { interior_nul_storage_length() };
    let u32 interior_nul_status = unsafe {
        validate_cstr(
            interior_nul,
            interior_nul_length,
            ref mut output as ptr mut fy_std_cstr_view_v0
        )
    };
    let u32 interior_failure_preserved_output = unsafe {
        view_is_valid(ref output as ptr fy_std_cstr_view_v0)
    };

    let ptr u8 unterminated = unsafe { unterminated_storage() };
    let c::size_t unterminated_length = unsafe { unterminated_storage_length() };
    let u32 unterminated_status = unsafe {
        validate_cstr(
            unterminated,
            unterminated_length,
            ref mut output as ptr mut fy_std_cstr_view_v0
        )
    };
    let u32 unterminated_failure_preserved_output = unsafe {
        view_is_valid(ref output as ptr fy_std_cstr_view_v0)
    };

    return if (accepted == 0 && valid_observed == 1 &&
        interior_nul_status == 4 && interior_failure_preserved_output == 1 &&
        unterminated_status == 3 && unterminated_failure_preserved_output == 1) {
        42
    } else {
        1
    };
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
    ptr mut void data;
    c::size_t size;
    c::size_t alignment;
}
@repr(C)
struct fy_std_buffer_v0 {
    fy_std_system_heap_block_v0 block;
    c::size_t length;
}
@repr(C)
struct fy_std_text_view_v0 {
    ptr u8 data;
    c::size_t length;
}
@repr(C)
struct fy_std_string_v0 {
    fy_std_buffer_v0 bytes;
}
@repr(C)
struct fy_std_cstr_view_v0 {
    ptr u8 data;
    c::size_t length;
}
@repr(C)
struct fy_std_cstring_v0 {
    fy_std_buffer_v0 bytes;
}
@repr(C)
struct fy_std_owned_text_result_v0 {
    u32 status;
    u32 allocation_status;
}

@link_name("fairyc_fixture_owned_text_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_owned_text_storage")
unsafe extern C fn ptr u8 text_storage() ;
@link_name("fairyc_fixture_owned_text_storage_length")
unsafe extern C fn c::size_t text_storage_length() ;

@link_name("fy_std_string_copy_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_copy(ptr fy_std_text_view_v0 input, ptr mut fy_std_string_v0 output) ;
@link_name("fy_std_string_as_view_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_as_view(ptr fy_std_string_v0 input, ptr mut fy_std_text_view_v0 output) ;
@link_name("fy_std_string_append_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_append(ptr mut fy_std_string_v0 value, ptr fy_std_text_view_v0 suffix) ;
@link_name("fy_std_string_drop_v0")
unsafe extern C fn fy_std_owned_text_result_v0 string_drop(ptr mut fy_std_string_v0 value) ;

@link_name("fy_std_cstring_from_text_v0")
unsafe extern C fn fy_std_owned_text_result_v0 cstring_from_text(ptr fy_std_text_view_v0 input, ptr mut fy_std_cstring_v0 output) ;
@link_name("fy_std_cstring_as_view_v0")
unsafe extern C fn fy_std_owned_text_result_v0 cstring_as_view(ptr fy_std_cstring_v0 input, ptr mut fy_std_cstr_view_v0 output) ;
@link_name("fy_std_cstring_drop_v0")
unsafe extern C fn fy_std_owned_text_result_v0 cstring_drop(ptr mut fy_std_cstring_v0 value) ;

@link_name("fairyc_fixture_string_view_is_fairy")
unsafe extern C fn u32 string_view_is_fairy(ptr fy_std_text_view_v0 view) ;
@link_name("fairyc_fixture_string_view_is_fairy_twice")
unsafe extern C fn u32 string_view_is_fairy_twice(ptr fy_std_text_view_v0 view) ;
@link_name("fairyc_fixture_string_is_empty")
unsafe extern C fn u32 string_is_empty(ptr fy_std_string_v0 value) ;
@link_name("fairyc_fixture_cstr_view_is_fairy")
unsafe extern C fn u32 cstr_view_is_fairy(ptr fy_std_cstr_view_v0 view) ;
@link_name("fairyc_fixture_cstring_is_empty")
unsafe extern C fn u32 cstring_is_empty(ptr fy_std_cstring_v0 value) ;

fn i32 main() {
    let c::size_t zero = unsafe { zero_size_t() };
    let ptr u8 storage = unsafe { text_storage() };
    let c::size_t storage_length = unsafe { text_storage_length() };
    let fy_std_text_view_v0 input = fy_std_text_view_v0 {
        .data = storage,
        .length = storage_length,
    };

    let fy_std_system_heap_block_v0 string_block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    let fy_std_buffer_v0 string_buffer = fy_std_buffer_v0 {
        .block = string_block,
        .length = zero,
    };
    var fy_std_string_v0 owned_string = fy_std_string_v0 { .bytes = string_buffer };
    var fy_std_text_view_v0 string_view = fy_std_text_view_v0 {
        .data = null,
        .length = zero,
    };
    let fy_std_owned_text_result_v0 copied = unsafe {
        string_copy(
            ref input as ptr fy_std_text_view_v0,
            ref mut owned_string as ptr mut fy_std_string_v0
        )
    };
    let fy_std_owned_text_result_v0 viewed_once = unsafe {
        string_as_view(
            ref owned_string as ptr fy_std_string_v0,
            ref mut string_view as ptr mut fy_std_text_view_v0
        )
    };
    let u32 observed_once = unsafe {
        string_view_is_fairy(ref string_view as ptr fy_std_text_view_v0)
    };
    let fy_std_owned_text_result_v0 appended = unsafe {
        string_append(
            ref mut owned_string as ptr mut fy_std_string_v0,
            ref input as ptr fy_std_text_view_v0
        )
    };
    let fy_std_owned_text_result_v0 viewed_twice = unsafe {
        string_as_view(
            ref owned_string as ptr fy_std_string_v0,
            ref mut string_view as ptr mut fy_std_text_view_v0
        )
    };
    let u32 observed_twice = unsafe {
        string_view_is_fairy_twice(ref string_view as ptr fy_std_text_view_v0)
    };
    let fy_std_owned_text_result_v0 dropped_string = unsafe {
        string_drop(ref mut owned_string as ptr mut fy_std_string_v0)
    };
    let u32 string_empty = unsafe {
        string_is_empty(ref owned_string as ptr fy_std_string_v0)
    };

    let fy_std_system_heap_block_v0 cstring_block = fy_std_system_heap_block_v0 {
        .data = null,
        .size = zero,
        .alignment = zero,
    };
    let fy_std_buffer_v0 cstring_buffer = fy_std_buffer_v0 {
        .block = cstring_block,
        .length = zero,
    };
    var fy_std_cstring_v0 owned_cstring = fy_std_cstring_v0 { .bytes = cstring_buffer };
    var fy_std_cstr_view_v0 cstr_view = fy_std_cstr_view_v0 {
        .data = null,
        .length = zero,
    };
    let fy_std_owned_text_result_v0 created_cstring = unsafe {
        cstring_from_text(
            ref input as ptr fy_std_text_view_v0,
            ref mut owned_cstring as ptr mut fy_std_cstring_v0
        )
    };
    let fy_std_owned_text_result_v0 viewed_cstring = unsafe {
        cstring_as_view(
            ref owned_cstring as ptr fy_std_cstring_v0,
            ref mut cstr_view as ptr mut fy_std_cstr_view_v0
        )
    };
    let u32 observed_cstring = unsafe {
        cstr_view_is_fairy(ref cstr_view as ptr fy_std_cstr_view_v0)
    };
    let fy_std_owned_text_result_v0 dropped_cstring = unsafe {
        cstring_drop(ref mut owned_cstring as ptr mut fy_std_cstring_v0)
    };
    let u32 cstring_empty = unsafe {
        cstring_is_empty(ref owned_cstring as ptr fy_std_cstring_v0)
    };

    return if (copied.status == 0 && copied.allocation_status == 0 &&
        viewed_once.status == 0 && viewed_once.allocation_status == 0 && observed_once == 1 &&
        appended.status == 0 && appended.allocation_status == 0 &&
        viewed_twice.status == 0 && viewed_twice.allocation_status == 0 && observed_twice == 1 &&
        dropped_string.status == 0 && dropped_string.allocation_status == 0 && string_empty == 1 &&
        created_cstring.status == 0 && created_cstring.allocation_status == 0 &&
        viewed_cstring.status == 0 && viewed_cstring.allocation_status == 0 && observed_cstring == 1 &&
        dropped_cstring.status == 0 && dropped_cstring.allocation_status == 0 && cstring_empty == 1) {
        42
    } else {
        0
    };
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
    u32 status;
    i32 error_number;
    c::size_t count;
}
@repr(C)
struct fy_std_linux_file_v0 {
    c::int descriptor;
    u8 open;
}
@repr(C)
struct fy_std_linux_open_options_v0 {
    u8 read;
    u8 write;
    u8 create;
    u8 truncate;
    u32 mode;
}

@link_name("fairyc_fixture_file_prepare_path")
unsafe extern C fn u32 prepare_path() ;
@link_name("fairyc_fixture_file_path")
unsafe extern C fn ptr u8 file_path() ;
@link_name("fairyc_fixture_file_path_storage_length")
unsafe extern C fn c::size_t file_path_storage_length() ;
@link_name("fairyc_fixture_file_payload")
unsafe extern C fn ptr u8 file_payload() ;
@link_name("fairyc_fixture_file_payload_length")
unsafe extern C fn c::size_t file_payload_length() ;
@link_name("fairyc_fixture_file_three_size_t")
unsafe extern C fn c::size_t three_size_t() ;
@link_name("fairyc_fixture_file_zero_size_t")
unsafe extern C fn c::size_t zero_size_t() ;
@link_name("fairyc_fixture_file_closed_descriptor")
unsafe extern C fn c::int closed_descriptor() ;
@link_name("fairyc_fixture_file_read_segment_one")
unsafe extern C fn ptr mut u8 read_segment_one() ;
@link_name("fairyc_fixture_file_read_segment_two")
unsafe extern C fn ptr mut u8 read_segment_two() ;
@link_name("fairyc_fixture_file_read_segment_three")
unsafe extern C fn ptr mut u8 read_segment_three() ;

@link_name("fy_std_linux_file_open_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_open(ptr u8 path, c::size_t path_storage_length, ptr fy_std_linux_open_options_v0 options, ptr mut fy_std_linux_file_v0 output) ;
@link_name("fy_std_linux_file_read_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_read(ptr mut fy_std_linux_file_v0 file, ptr mut u8 buffer, c::size_t length) ;
@link_name("fy_std_linux_file_write_all_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_write_all(ptr mut fy_std_linux_file_v0 file, ptr u8 buffer, c::size_t length) ;
@link_name("fy_std_linux_file_sync_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_sync(ptr mut fy_std_linux_file_v0 file) ;
@link_name("fy_std_linux_file_close_v0")
unsafe extern C fn fy_std_file_result_v0 linux_file_close(ptr mut fy_std_linux_file_v0 file) ;

@link_name("fairyc_fixture_file_result_matches")
unsafe extern C fn u32 result_matches(ptr fy_std_file_result_v0 result, u32 status, i32 error_number, c::size_t count) ;
@link_name("fairyc_fixture_file_is_closed")
unsafe extern C fn u32 file_is_closed(ptr fy_std_linux_file_v0 file) ;
@link_name("fairyc_fixture_file_buffers_match_and_cleanup")
unsafe extern C fn u32 buffers_match_and_cleanup() ;

fn i32 main() {
    let u32 prepared = unsafe { prepare_path() };
    let ptr u8 path = unsafe { file_path() };
    let c::size_t path_length = unsafe { file_path_storage_length() };
    let ptr u8 payload = unsafe { file_payload() };
    let c::size_t payload_length = unsafe { file_payload_length() };
    let c::size_t three = unsafe { three_size_t() };
    let c::size_t zero = unsafe { zero_size_t() };
    let c::int closed = unsafe { closed_descriptor() };

    var fy_std_linux_file_v0 file = fy_std_linux_file_v0 {
        .descriptor = closed,
        .open = 0,
    };
    let fy_std_linux_open_options_v0 write_options = fy_std_linux_open_options_v0 {
        .read = 0,
        .write = 1,
        .create = 0,
        .truncate = 1,
        .mode = 384,
    };
    let fy_std_file_result_v0 opened_for_write = unsafe {
        linux_file_open(
            path,
            path_length,
            ref write_options as ptr fy_std_linux_open_options_v0,
            ref mut file as ptr mut fy_std_linux_file_v0
        )
    };
    let u32 opened_for_write_ok = unsafe {
        result_matches(ref opened_for_write as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let fy_std_file_result_v0 written = unsafe {
        linux_file_write_all(ref mut file as ptr mut fy_std_linux_file_v0, payload, payload_length)
    };
    let u32 written_ok = unsafe {
        result_matches(ref written as ptr fy_std_file_result_v0, 0, 0, payload_length)
    };
    let fy_std_file_result_v0 synced = unsafe {
        linux_file_sync(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 synced_ok = unsafe {
        result_matches(ref synced as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let fy_std_file_result_v0 closed_after_write = unsafe {
        linux_file_close(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 closed_after_write_ok = unsafe {
        result_matches(ref closed_after_write as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let u32 closed_state_after_write = unsafe {
        file_is_closed(ref file as ptr fy_std_linux_file_v0)
    };
    let fy_std_file_result_v0 closed_twice = unsafe {
        linux_file_close(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 closed_twice_ok = unsafe {
        result_matches(ref closed_twice as ptr fy_std_file_result_v0, 3, 0, zero)
    };

    let fy_std_linux_open_options_v0 read_options = fy_std_linux_open_options_v0 {
        .read = 1,
        .write = 0,
        .create = 0,
        .truncate = 0,
        .mode = 0,
    };
    let fy_std_file_result_v0 opened_for_read = unsafe {
        linux_file_open(
            path,
            path_length,
            ref read_options as ptr fy_std_linux_open_options_v0,
            ref mut file as ptr mut fy_std_linux_file_v0
        )
    };
    let u32 opened_for_read_ok = unsafe {
        result_matches(ref opened_for_read as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let ptr mut u8 first_buffer = unsafe { read_segment_one() };
    let ptr mut u8 second_buffer = unsafe { read_segment_two() };
    let ptr mut u8 third_buffer = unsafe { read_segment_three() };
    let fy_std_file_result_v0 read_first = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, first_buffer, three)
    };
    let u32 read_first_ok = unsafe {
        result_matches(ref read_first as ptr fy_std_file_result_v0, 0, 0, three)
    };
    let fy_std_file_result_v0 read_second = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, second_buffer, three)
    };
    let u32 read_second_ok = unsafe {
        result_matches(ref read_second as ptr fy_std_file_result_v0, 0, 0, three)
    };
    let fy_std_file_result_v0 read_third = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, third_buffer, three)
    };
    let u32 read_third_ok = unsafe {
        result_matches(ref read_third as ptr fy_std_file_result_v0, 0, 0, three)
    };
    let fy_std_file_result_v0 read_eof = unsafe {
        linux_file_read(ref mut file as ptr mut fy_std_linux_file_v0, first_buffer, three)
    };
    let u32 read_eof_ok = unsafe {
        result_matches(ref read_eof as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let fy_std_file_result_v0 closed_after_read = unsafe {
        linux_file_close(ref mut file as ptr mut fy_std_linux_file_v0)
    };
    let u32 closed_after_read_ok = unsafe {
        result_matches(ref closed_after_read as ptr fy_std_file_result_v0, 0, 0, zero)
    };
    let u32 closed_state_after_read = unsafe {
        file_is_closed(ref file as ptr fy_std_linux_file_v0)
    };
    let u32 observed = unsafe { buffers_match_and_cleanup() };

    return if (prepared == 1 && opened_for_write_ok == 1 && written_ok == 1 && synced_ok == 1 &&
        closed_after_write_ok == 1 && closed_state_after_write == 1 && closed_twice_ok == 1 &&
        opened_for_read_ok == 1 && read_first_ok == 1 && read_second_ok == 1 &&
        read_third_ok == 1 && read_eof_ok == 1 && closed_after_read_ok == 1 &&
        closed_state_after_read == 1 && observed == 1) {
        42
    } else {
        0
    };
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
	LinuxConsoleExecutions              int        `json:"linuxConsoleExecutions"`
	LinuxFileExecutions                 int        `json:"linuxFileExecutions"`
	LinuxFileSourceCallExecutions       int        `json:"linuxFileSourceCallExecutions"`
	LinuxPublicFileExecutions           int        `json:"linuxPublicFileExecutions"`
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
	bufferProbeImplementation, err := cstdlib.BufferProbeImplementationSource()
	if err != nil {
		return Report{}, fmt.Errorf("load instrumented Buffer implementation: %w", err)
	}
	bufferExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Buffer", "buffer-probe", bufferProbe, bufferProbeImplementation,
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
	arenaProbeImplementation, err := cstdlib.ArenaProbeImplementationSource()
	if err != nil {
		return Report{}, fmt.Errorf("load instrumented Arena implementation: %w", err)
	}
	arenaExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Arena", "arena-probe", arenaProbe, arenaProbeImplementation,
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
	ownedTextProbeImplementation, err := cstdlib.OwnedTextProbeImplementationSource()
	if err != nil {
		return Report{}, fmt.Errorf("load instrumented owned text implementation: %w", err)
	}
	ownedTextExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "owned text", "owned-text-probe", ownedTextProbe, ownedTextProbeImplementation,
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
	linuxConsoleSource, err := cstdlib.LinuxConsoleSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux console implementation: %w", err)
	}
	linuxConsoleProbe, err := cstdlib.LinuxConsoleProbeSource()
	if err != nil {
		return Report{}, fmt.Errorf("load Linux console probe: %w", err)
	}
	linuxConsoleExecutions, err := runStandardLibraryProbe(
		temporaryRoot, toolchains, "Linux console", "linux-console-probe", linuxConsoleProbe, linuxConsoleSource,
	)
	if err != nil {
		return Report{}, err
	}
	report.LinuxConsoleExecutions = linuxConsoleExecutions
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
	publicFileBuildSmoke, err := runPublicFileBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.LinuxPublicFileExecutions = publicFileBuildSmoke
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
	unit := foreignstructprobe.Generate()
	return runForeignSourceProbe(
		temporaryRoot,
		toolchains,
		"foreign-c-struct-result-call",
		unit.FairySource,
		unit.CompanionC,
		unit.ExpectedExit,
		unit.RequiredCFragments,
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
	checked, err := compiler.Check([]semantic.File{{Path: sourcePath, Module: parsed.AST}})
	if err != nil {
		return 0, fmt.Errorf("compile %s FY source: %w", probeID, err)
	}
	semanticResult := checked.Semantic
	if semanticResult.CompletedThrough != semantic.PhaseFlow || len(semanticResult.Diagnostics) != 0 || len(semanticResult.Unsupported) != 0 {
		return 0, fmt.Errorf("%s FY source did not complete Flow: phase=%q diagnostics=%+v unsupported=%+v", probeID, semanticResult.CompletedThrough, semanticResult.Diagnostics, semanticResult.Unsupported)
	}
	if !checked.Verified {
		return 0, fmt.Errorf("%s FY source did not produce verified MIR", probeID)
	}
	unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0Profile())
	if err != nil {
		return 0, fmt.Errorf("emit %s C: %w", probeID, err)
	}
	for _, required := range requiredC {
		if !strings.Contains(unit.Source, required) {
			return 0, fmt.Errorf("%s C emission lost required fragment %q", probeID, required)
		}
	}
	runtimeSource, err := cruntime.ScalarSource(trapExitBase)
	if err != nil {
		return 0, fmt.Errorf("bind %s scalar runtime: %w", probeID, err)
	}
	directions := [][2]int{{0, 1}, {1, 0}}
	executions := 0
	for directionIndex, direction := range directions {
		caller := toolchains[direction[0]]
		callee := toolchains[direction[1]]
		for profileIndex, profile := range nativeProfiles {
			artifact, buildErr := cdriver.BuildCrossCompilerWithRuntime(caller, callee, profile, unit.Source, companionSource, runtimeSource)
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

func runPublicFileBuildSmoke(temporaryRoot string) (int, error) {
	definition, err := sdkstdlib.Load()
	if err != nil {
		return 0, fmt.Errorf("load compiler-bundled standard library: %w", err)
	}
	return runBuildSourceSmokeWithAdditionalSources(
		temporaryRoot,
		"public-file-build-smoke",
		publicFileBuildSmokeSource,
		definition.Sources,
	)
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
	return runBuildSourceSmokeWithAdditionalSources(temporaryRoot, smokeID, source, nil)
}

func runBuildSourceSmokeWithAdditionalSources(
	temporaryRoot,
	smokeID,
	source string,
	additionalSources []sdkstdlib.Source,
) (int, error) {
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
	for _, additional := range additionalSources {
		path := filepath.Join(sourceRoot, filepath.FromSlash(additional.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return 0, fmt.Errorf("create %s additional-source directory: %w", smokeID, err)
		}
		if err := os.WriteFile(path, additional.Bytes, 0o600); err != nil {
			return 0, fmt.Errorf("write %s additional source %s: %w", smokeID, additional.Path, err)
		}
	}
	executions := 0
	for _, compiler := range []string{"clang", "gcc"} {
		for _, profile := range nativeProfiles {
			output := fmt.Sprintf("build/smoke-%s-%s", compiler, profile.Name)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			arguments := []string{"experimental", "build", "main.fy"}
			for _, additional := range additionalSources {
				arguments = append(arguments, "--source", additional.Path)
			}
			arguments = append(arguments,
				"--cc", compiler,
				"--profile", profile.Name,
				"--output", output,
			)
			code := cli.Run(arguments, sourceRoot, &stdout, &stderr)
			if code != cli.ExitSuccess {
				return 0, fmt.Errorf("%s with %s/%s failed with code %d: %s", smokeID, compiler, profile.Name, code, strings.TrimSpace(stderr.String()))
			}
			expectedReceipt := fmt.Sprintf("built: %s (%s, %s, %s)\n", output, target.X8664LinuxGNU, compiler, profile.Name)
			if stdout.String() != expectedReceipt || stderr.Len() != 0 {
				return 0, fmt.Errorf("%s with %s/%s produced stdout %q and stderr %q", smokeID, compiler, profile.Name, stdout.String(), stderr.String())
			}
			runRoot := filepath.Join(sourceRoot, fmt.Sprintf("run-%s-%s", compiler, profile.Name))
			if err := os.Mkdir(runRoot, 0o700); err != nil {
				return 0, fmt.Errorf("create %s execution root for %s/%s: %w", smokeID, compiler, profile.Name, err)
			}
			if err := executeInDirectory(smokeID, filepath.Join(sourceRoot, filepath.FromSlash(output)), runRoot, 42, nil); err != nil {
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
	return executeInDirectory(caseID, executablePath, "", expectedExit, expectedDrops)
}

func executeInDirectory(caseID, executablePath, workingDirectory string, expectedExit int, expectedDrops []uint32) error {
	result, err := executeProcessInDirectory(executablePath, workingDirectory, nil)
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
	return executeProcessInDirectory(executablePath, "", environment)
}

func executeProcessInDirectory(executablePath, workingDirectory string, environment []string) (processResult, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), executionTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath)
	command.Stdin = nil
	if workingDirectory != "" {
		command.Dir = workingDirectory
	}
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
