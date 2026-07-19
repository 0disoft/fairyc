module main;

unsafe extern C fn make_bool() -> c::bool;
unsafe extern C fn consume_bool(value: c::bool) -> i32;
unsafe extern C fn make_char() -> c::char;
unsafe extern C fn consume_char(value: c::char) -> i32;
unsafe extern C fn make_double() -> c::double;
unsafe extern C fn consume_double(value: c::double) -> i32;
unsafe extern C fn make_float() -> c::float;
unsafe extern C fn consume_float(value: c::float) -> i32;
unsafe extern C fn make_longlong() -> c::longlong;
unsafe extern C fn consume_longlong(value: c::longlong) -> i32;
unsafe extern C fn make_schar() -> c::schar;
unsafe extern C fn consume_schar(value: c::schar) -> i32;
unsafe extern C fn make_short() -> c::short;
unsafe extern C fn consume_short(value: c::short) -> i32;
unsafe extern C fn make_uchar() -> c::uchar;
unsafe extern C fn consume_uchar(value: c::uchar) -> i32;
unsafe extern C fn make_uint() -> c::uint;
unsafe extern C fn consume_uint(value: c::uint) -> i32;
unsafe extern C fn make_ulong() -> c::ulong;
unsafe extern C fn consume_ulong(value: c::ulong) -> i32;
unsafe extern C fn make_ulonglong() -> c::ulonglong;
unsafe extern C fn consume_ulonglong(value: c::ulonglong) -> i32;
unsafe extern C fn make_ushort() -> c::ushort;
unsafe extern C fn consume_ushort(value: c::ushort) -> i32;

fn check_bool() -> i32 { unsafe { consume_bool(make_bool()) } }
fn check_char() -> i32 { unsafe { consume_char(make_char()) } }
fn check_double() -> i32 { unsafe { consume_double(make_double()) } }
fn check_float() -> i32 { unsafe { consume_float(make_float()) } }
fn check_longlong() -> i32 { unsafe { consume_longlong(make_longlong()) } }
fn check_schar() -> i32 { unsafe { consume_schar(make_schar()) } }
fn check_short() -> i32 { unsafe { consume_short(make_short()) } }
fn check_uchar() -> i32 { unsafe { consume_uchar(make_uchar()) } }
fn check_uint() -> i32 { unsafe { consume_uint(make_uint()) } }
fn check_ulong() -> i32 { unsafe { consume_ulong(make_ulong()) } }
fn check_ulonglong() -> i32 { unsafe { consume_ulonglong(make_ulonglong()) } }
fn check_ushort() -> i32 { unsafe { consume_ushort(make_ushort()) } }

fn main() -> i32 { 0 }
