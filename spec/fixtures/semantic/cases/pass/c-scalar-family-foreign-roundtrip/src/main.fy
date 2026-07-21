module main;

unsafe extern C fn c::bool make_bool() ;
unsafe extern C fn i32 consume_bool(c::bool value) ;
unsafe extern C fn c::char make_char() ;
unsafe extern C fn i32 consume_char(c::char value) ;
unsafe extern C fn c::double make_double() ;
unsafe extern C fn i32 consume_double(c::double value) ;
unsafe extern C fn c::float make_float() ;
unsafe extern C fn i32 consume_float(c::float value) ;
unsafe extern C fn c::longlong make_longlong() ;
unsafe extern C fn i32 consume_longlong(c::longlong value) ;
unsafe extern C fn c::schar make_schar() ;
unsafe extern C fn i32 consume_schar(c::schar value) ;
unsafe extern C fn c::short make_short() ;
unsafe extern C fn i32 consume_short(c::short value) ;
unsafe extern C fn c::uchar make_uchar() ;
unsafe extern C fn i32 consume_uchar(c::uchar value) ;
unsafe extern C fn c::uint make_uint() ;
unsafe extern C fn i32 consume_uint(c::uint value) ;
unsafe extern C fn c::ulong make_ulong() ;
unsafe extern C fn i32 consume_ulong(c::ulong value) ;
unsafe extern C fn c::ulonglong make_ulonglong() ;
unsafe extern C fn i32 consume_ulonglong(c::ulonglong value) ;
unsafe extern C fn c::ushort make_ushort() ;
unsafe extern C fn i32 consume_ushort(c::ushort value) ;

fn i32 check_bool() { return unsafe { consume_bool(make_bool()) }; }
fn i32 check_char() { return unsafe { consume_char(make_char()) }; }
fn i32 check_double() { return unsafe { consume_double(make_double()) }; }
fn i32 check_float() { return unsafe { consume_float(make_float()) }; }
fn i32 check_longlong() { return unsafe { consume_longlong(make_longlong()) }; }
fn i32 check_schar() { return unsafe { consume_schar(make_schar()) }; }
fn i32 check_short() { return unsafe { consume_short(make_short()) }; }
fn i32 check_uchar() { return unsafe { consume_uchar(make_uchar()) }; }
fn i32 check_uint() { return unsafe { consume_uint(make_uint()) }; }
fn i32 check_ulong() { return unsafe { consume_ulong(make_ulong()) }; }
fn i32 check_ulonglong() { return unsafe { consume_ulonglong(make_ulonglong()) }; }
fn i32 check_ushort() { return unsafe { consume_ushort(make_ushort()) }; }

fn i32 main() { return 0; }
