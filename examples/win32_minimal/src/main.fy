module main;

unsafe extern C fn ptr mut void fairy_win32_open();
unsafe extern C fn i32 fairy_win32_next(ptr mut void window);
unsafe extern C fn i32 fairy_win32_changed(ptr mut void window, u32 edits, u32 status_id);
unsafe extern C fn i32 fairy_win32_resolve_close(ptr mut void window, u32 allow);
unsafe extern C fn i32 fairy_win32_close(ptr mut void window);
unsafe extern C fn usize fairy_win32_read_input(ptr mut void window, ptr mut u8 output, usize capacity);
unsafe extern C fn i32 fairy_win32_present(ptr mut void window, ptr u8 text, usize length, u32 clicks, u32 status_id);

unsafe fn i32 on_change(ptr mut void window, u32 edits) {
    return fairy_win32_changed(window, edits, 105);
}

unsafe fn i32 on_close(ptr mut void window) {
    return fairy_win32_resolve_close(window, 1);
}

unsafe fn i32 on_click(ptr mut void window, u32 clicks) {
    var u8[512] input = [0;512];
    let ref mut u8[] writable = ref mut input;
    let usize count = fairy_win32_read_input(window, writable as ptr mut u8, writable.len);
    if (count > 511) { return 1; }
    if (count == 0) {
        let message = b"Enter a message first.";
        return fairy_win32_present(window, message as ptr u8, message.len, clicks, 109);
    }
    let prefix = b"Result: ";
    if (count > 512 - prefix.len) { return 1; }
    var u8[512] output = [0;512];
    var usize i = 0;
    while (i < prefix.len) { output[i] = prefix[i]; i += 1; }
    let ref u8[] source = ref input;
    i = 0;
    while (i < count) { output[prefix.len + i] = source[i]; i += 1; }
    let ref u8[] result = ref output;
    return fairy_win32_present(window, result as ptr u8, prefix.len + count, clicks, 106);
}

fn i32 main() {
    let window = unsafe { fairy_win32_open() };
    if (window == null) { return 1; }
    var u32 clicks = 0;
    var u32 edits = 0;
    var i32 result = 0;
    while (true) {
        let i32 event = unsafe { fairy_win32_next(window) };
        if (event == 0) { break; }
        if (event == 1) {
            clicks += 1;
            result = unsafe { on_click(window, clicks) };
        } else if (event == 2) {
            edits += 1;
            result = unsafe { on_change(window, edits) };
        } else if (event == 3) {
            result = unsafe { on_close(window) };
        } else {
            result = 1;
        }
        if (result != 0) { break; }
    }
    let i32 native_result = unsafe { fairy_win32_close(window) };
    return if (result != 0) { result } else { native_result };
}
