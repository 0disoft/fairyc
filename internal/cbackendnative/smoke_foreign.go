package cbackendnative

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
