package cbackendnative

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
