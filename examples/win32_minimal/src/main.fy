module main;

unsafe extern C fn ptr mut void fairy_win32_open(i32 width, i32 height);
unsafe extern C fn i32 fairy_win32_initialize_dpi();
unsafe extern C fn i32 fairy_win32_dpi(ptr mut void window);
unsafe extern C fn i32 fairy_win32_font(ptr mut void window, i32 height);
unsafe extern C fn i32 fairy_win32_accept_dpi(ptr mut void window, u32 changes, i32 min_width, i32 min_height);
unsafe extern C fn i32 fairy_win32_next(ptr mut void window);
unsafe extern C fn i32 fairy_win32_key(ptr mut void window, u32 axis);
unsafe extern C fn i32 fairy_win32_key_done(ptr mut void window, u32 keys, i32 action);
unsafe extern C fn i32 fairy_win32_changed(ptr mut void window, u32 edits, u32 status_id);
unsafe extern C fn i32 fairy_win32_resolve_close(ptr mut void window, u32 allow);
unsafe extern C fn i32 fairy_win32_client_extent(ptr mut void window, u32 axis);
unsafe extern C fn i32 fairy_win32_move(ptr mut void window, u32 control_id, i32 x, i32 y, i32 width, i32 height);
unsafe extern C fn i32 fairy_win32_layout_done(ptr mut void window, i32 min_width, i32 min_height, u32 resizes);
unsafe extern C fn i32 fairy_win32_close(ptr mut void window);
unsafe extern C fn usize fairy_win32_read_input(ptr mut void window, ptr mut u8 output, usize capacity);
unsafe extern C fn i32 fairy_win32_present(ptr mut void window, ptr u8 text, usize length, u32 clicks, u32 status_id);

unsafe fn i32 on_change(ptr mut void window, u32 edits) {
    return fairy_win32_changed(window, edits, 105);
}

unsafe fn i32 on_close(ptr mut void window) {
    return fairy_win32_resolve_close(window, 1);
}

fn i32 on_key(i32 code, i32 flags) {
    if (flags != 0) { return 0; }
    if (code == 13) { return 1; }
    if (code == 27) { return 3; }
    return 0;
}

unsafe fn i32 on_size(ptr mut void window, u32 resizes) {
    let i32 actual_width = fairy_win32_client_extent(window, 0);
    let i32 actual_height = fairy_win32_client_extent(window, 1);
    if (actual_width < 0 || actual_height < 0) { return 1; }
    let i32 dpi = fairy_win32_dpi(window);
    if (dpi < 1 || dpi > 960 || fairy_win32_font(window, scale(12, dpi)) != 0) { return 1; }
    let i32 width = if (actual_width < scale(320, dpi)) { scale(320, dpi) } else { actual_width };
    let i32 inset = 16;
    let i32 left = scale(inset, dpi);
    if (fairy_win32_move(window, 200, left, scale(16, dpi), width - 2 * left, scale(20, dpi)) != 0) { return 1; }
    if (fairy_win32_move(window, 201, left, scale(44, dpi), width - scale(132, dpi), scale(28, dpi)) != 0) { return 1; }
    if (fairy_win32_move(window, 202, width - scale(100, dpi), scale(44, dpi), scale(84, dpi), scale(28, dpi)) != 0) { return 1; }
    if (fairy_win32_move(window, 203, left, scale(94, dpi), width - 2 * left, scale(48, dpi)) != 0) { return 1; }
    if (fairy_win32_move(window, 204, left, scale(154, dpi), width - 2 * left, scale(24, dpi)) != 0) { return 1; }
    return fairy_win32_layout_done(window, scale(400, dpi), scale(240, dpi), resizes);
}

fn i32 scale(i32 value, i32 dpi) { return (value * dpi + 48) / 96; }

unsafe fn i32 on_dpi(ptr mut void window, u32 resizes, u32 dpi_changes) {
    let i32 dpi = fairy_win32_dpi(window);
    if (dpi < 1 || dpi > 960) { return 1; }
    if (fairy_win32_accept_dpi(window, dpi_changes, scale(400, dpi), scale(240, dpi)) != 0) { return 1; }
    return on_size(window, resizes);
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
    let i32 initial_dpi = unsafe { fairy_win32_initialize_dpi() };
    if (initial_dpi < 1 || initial_dpi > 960) { return 1; }
    let window = unsafe { fairy_win32_open(scale(480, initial_dpi), scale(240, initial_dpi)) };
    if (window == null) { return 1; }
    var u32 clicks = 0;
    var u32 edits = 0;
    var u32 resizes = 0;
    var u32 dpi_changes = 0;
    var u32 keys = 0;
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
        } else if (event == 4) {
            resizes += 1;
            result = unsafe { on_size(window, resizes) };
        } else if (event == 5) {
            dpi_changes += 1;
            resizes += 1;
            result = unsafe { on_dpi(window, resizes, dpi_changes) };
        } else if (event == 6) {
            keys += 1;
            let i32 code = unsafe { fairy_win32_key(window, 0) };
            let i32 flags = unsafe { fairy_win32_key(window, 1) };
            let i32 action = on_key(code, flags);
            if (action == 1) {
                clicks += 1;
                result = unsafe { on_click(window, clicks) };
            } else if (action == 3) {
                result = unsafe { on_close(window) };
            }
            if (result == 0) {
                result = unsafe { fairy_win32_key_done(window, keys, action) };
            }
        } else {
            result = 1;
        }
        if (result != 0) { break; }
    }
    let i32 native_result = unsafe { fairy_win32_close(window) };
    return if (result != 0) { result } else { native_result };
}
