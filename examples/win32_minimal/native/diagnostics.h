// Private verification code; included only in the diagnostic translation unit.
static BOOL control_resource(HWND control, UINT id) {
    WCHAR actual[160];
    WCHAR expected[160];
    return GetWindowTextW(control, actual, (int)ARRAYSIZE(actual)) > 0 &&
           LoadStringW(GetModuleHandleW(NULL), id, expected, (int)ARRAYSIZE(expected)) > 0 &&
           wcscmp(actual, expected) == 0;
}

static void smoke_finish(HWND window, WindowState *state, BOOL passed) {
    if (passed) {
        SendMessageW(window, WM_CLOSE, 0, 0);
        return;
    }
    KillTimer(window, 1);
    state->exit_code = 5;
    DestroyWindow(window);
}

static void smoke_events(HWND window, WindowState *state) {
    WCHAR actual[160];
    WCHAR long_input[201];
    switch (state->smoke_step++) {
    case 0:
        {
            RECT input;
            if (!GetWindowRect(state->input, &input)) { break; }
            MapWindowPoints(NULL, window, (POINT *)&input, 2);
            if (input.left != MulDiv(16, (int)state->dpi, 96)) { break; }
        }
        if (!control_resource(window, 101) || state->clicks != 0 ||
            !control_resource(state->status, 104)) { break; }
        SendMessageW(window, WM_COMMAND, MAKEWPARAM(APPLY_ID, BN_CLICKED), 0);
        if (state->pending_event) { break; }
        SendMessageW(state->apply, BM_CLICK, 0, 0);
        return;
    case 1:
        if (state->clicks != 1 || !control_resource(state->result, 108) ||
            !control_resource(state->status, 109)) { break; }
        state->smoke_edits = state->edits;
        if (!SetWindowTextW(state->input, L"Fairy \uD55C\uAE00 \u03A9")) { break; }
        return;
    case 2:
        if (state->edits <= state->smoke_edits || !control_resource(state->status, 105) ||
            !control_resource(state->result, 108)) { break; }
        SendMessageW(state->apply, BM_CLICK, 0, 0);
        return;
    case 3:
        if (state->clicks != 2 || !control_resource(state->status, 106) ||
            GetWindowTextW(state->result, actual, (int)ARRAYSIZE(actual)) == 0 ||
            wcscmp(actual, L"Result: Fairy \uD55C\uAE00 \u03A9") != 0) { break; }
        state->smoke_edits = state->edits;
        if (!SetWindowTextW(state->input, L"")) { break; }
        return;
    case 4:
        if (state->edits <= state->smoke_edits) { break; }
        SendMessageW(state->apply, BM_CLICK, 0, 0);
        return;
    case 5:
        if (state->clicks != 3 || !control_resource(state->result, 108)) { break; }
        for (int i = 0; i < 200; i++) { long_input[i] = L'x'; }
        long_input[200] = L'\0';
        state->smoke_edits = state->edits;
        SendMessageW(state->input, EM_REPLACESEL, TRUE, (LPARAM)long_input);
        return;
    case 6:
        if (state->edits <= state->smoke_edits || GetWindowTextLengthW(state->input) != INPUT_LIMIT) { break; }
        SendMessageW(state->apply, BM_CLICK, 0, 0);
        return;
    case 7:
        smoke_finish(window, state, state->clicks == 4 && control_resource(state->status, 106) &&
                     GetWindowTextW(state->result, actual, (int)ARRAYSIZE(actual)) == 8 + INPUT_LIMIT);
        return;
    }
    smoke_finish(window, state, FALSE);
}

typedef struct {
    char bytes[4096];
    size_t size;
    BOOL valid;
} Json;

static void json_bytes(Json *json, const char *value) {
    size_t length = strlen(value);
    if (length > sizeof(json->bytes) - json->size) {
        json->valid = FALSE;
        return;
    }
    memcpy(json->bytes + json->size, value, length);
    json->size += length;
}

static void json_number(Json *json, unsigned value) {
    char digits[11];
    size_t at = sizeof(digits) - 1;
    digits[at] = '\0';
    do {
        digits[--at] = (char)('0' + value % 10);
        value /= 10;
    } while (value != 0);
    json_bytes(json, digits + at);
}

static void json_string(Json *json, const WCHAR *text) {
    char utf8[640];
    const char hex[] = "0123456789abcdef";
    int length = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, text, -1,
                                    utf8, (int)sizeof(utf8), NULL, NULL);
    if (length == 0) {
        json->valid = FALSE;
        return;
    }
    json_bytes(json, "\"");
    for (int i = 0; i < length - 1; i++) {
        unsigned char byte = (unsigned char)utf8[i];
        if (byte == '"' || byte == '\\') {
            char escaped[] = {'\\', (char)byte, '\0'};
            json_bytes(json, escaped);
        } else if (byte < 32) {
            char escaped[] = {'\\', 'u', '0', '0', hex[byte >> 4], hex[byte & 15], '\0'};
            json_bytes(json, escaped);
        } else {
            char raw[] = {(char)byte, '\0'};
            json_bytes(json, raw);
        }
    }
    json_bytes(json, "\"");
}

static void json_text(Json *json, HWND control) {
    WCHAR text[160] = {0};
    GetWindowTextW(control, text, (int)ARRAYSIZE(text));
    json_string(json, text);
}

static void json_signed(Json *json, LONG value) {
    if (value < 0) { json_bytes(json, "-"); }
    json_number(json, value < 0 ? (unsigned)(-(int64_t)value) : (unsigned)value);
}

static void json_rect(Json *json, const RECT *rect) {
    json_bytes(json, "{\"x\":"); json_signed(json, rect->left);
    json_bytes(json, ",\"y\":"); json_signed(json, rect->top);
    json_bytes(json, ",\"width\":"); json_signed(json, rect->right - rect->left);
    json_bytes(json, ",\"height\":"); json_signed(json, rect->bottom - rect->top);
    json_bytes(json, "}");
}

static BOOL json_geometry(Json *json, HWND window, WindowState *state) {
    RECT rect;
    if (!GetWindowRect(window, &rect)) { return FALSE; }
    json_bytes(json, ",\"window\":"); json_rect(json, &rect);
    if (!GetClientRect(window, &rect)) { return FALSE; }
    json_bytes(json, ",\"client\":"); json_rect(json, &rect);
    json_bytes(json, ",\"resizes\":"); json_number(json, state->resizes);
    json_bytes(json, ",\"dpi\":"); json_number(json, state->dpi);
    json_bytes(json, ",\"windowDpi\":"); json_number(json, GetDpiForWindow(window));
    json_bytes(json, AreDpiAwarenessContextsEqual(GetWindowDpiAwarenessContext(window),
                DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2) ? ",\"perMonitorV2\":true" : ",\"perMonitorV2\":false");
    json_bytes(json, ",\"dpiChanges\":"); json_number(json, state->dpi_changes);
    json_bytes(json, ",\"dpiInjections\":"); json_number(json, state->dpi_injections);
    json_bytes(json, ",\"dpiSuggested\":"); json_rect(json, &state->dpi_suggested);
    json_bytes(json, ",\"dpiApplied\":"); json_rect(json, &state->dpi_applied);
    json_bytes(json, state->sizing ? ",\"sizing\":true" : ",\"sizing\":false");
    json_bytes(json, (GetAsyncKeyState(VK_LBUTTON) & 0x8000) ? ",\"mouseDown\":true" : ",\"mouseDown\":false");
    json_bytes(json, ",\"sizeEntries\":"); json_number(json, state->size_entries);
    json_bytes(json, ",\"sizeExits\":"); json_number(json, state->size_exits);
    json_bytes(json, ",\"liveLayouts\":"); json_number(json, state->live_layouts);
    json_bytes(json, ",\"minimum\":{\"width\":"); json_signed(json, state->min_width);
    json_bytes(json, ",\"height\":"); json_signed(json, state->min_height);
    json_bytes(json, "},\"layout\":{");
    for (size_t i = 0; i < ARRAYSIZE(layout_ids); i++) {
        if (!GetWindowRect(GetDlgItem(window, layout_ids[i]), &rect)) { return FALSE; }
        SetLastError(0);
        if (MapWindowPoints(NULL, window, (POINT *)&rect, 2) == 0 && GetLastError() != 0) { return FALSE; }
        if (i != 0) { json_bytes(json, ","); }
        json_bytes(json, "\""); json_bytes(json, layout_names[i]); json_bytes(json, "\":");
        json_rect(json, &rect);
    }
    json_bytes(json, "},\"fonts\":{");
    for (size_t i = 0; i < ARRAYSIZE(layout_ids); i++) {
        LOGFONTW font;
        HFONT handle = (HFONT)SendMessageW(GetDlgItem(window, layout_ids[i]), WM_GETFONT, 0, 0);
        if (GetObjectW(handle, sizeof(font), &font) == 0) { return FALSE; }
        if (i != 0) { json_bytes(json, ","); }
        json_bytes(json, "\""); json_bytes(json, layout_names[i]); json_bytes(json, "\":");
        json_signed(json, -font.lfHeight);
    }
    json_bytes(json, "}");
    return TRUE;
}

static BOOL automation_write(WindowState *state, const char *bytes, size_t length) {
    while (length > 0) {
        DWORD written = 0;
        if (!WriteFile(state->command_output, bytes, (DWORD)length, &written, NULL) || written == 0) {
            return FALSE;
        }
        bytes += written;
        length -= written;
    }
    return TRUE;
}

static BOOL automation_state(HWND window, WindowState *state) {
    Json json = {{0}, 0, TRUE};
    json_bytes(&json, "{\"schema\":\"fairyc.win32-example-control/v1\",\"ok\":true,\"pid\":");
    json_number(&json, (unsigned)GetCurrentProcessId());
    json_bytes(&json, IsWindowVisible(window) ? ",\"visible\":true,\"title\":" : ",\"visible\":false,\"title\":");
    json_text(&json, window);
    json_bytes(&json, ",\"input\":");
    json_text(&json, state->input);
    json_bytes(&json, ",\"result\":");
    json_text(&json, state->result);
    json_bytes(&json, ",\"status\":");
    json_text(&json, state->status);
    json_bytes(&json, ",\"edits\":");
    json_number(&json, state->edits);
    json_bytes(&json, ",\"clicks\":");
    json_number(&json, state->clicks);
    json_bytes(&json, ",\"closed\":false,\"keyCode\":"); json_signed(&json, state->key_code);
    json_bytes(&json, ",\"keyFlags\":"); json_signed(&json, state->key_flags);
    json_bytes(&json, ",\"keyFocus\":"); json_signed(&json, state->key_focus);
    json_bytes(&json, ",\"keyAction\":"); json_signed(&json, state->key_action);
    json_bytes(&json, ",\"keys\":"); json_number(&json, state->keys);
    json_bytes(&json, ",\"awaitingKey\":"); json_signed(&json, state->key_expected);
    json_bytes(&json, ",\"focus\":"); json_signed(&json, GetDlgCtrlID(GetFocus()));
    FairyIME ime;
    fairy_ime_snapshot(&ime);
    json_bytes(&json, ",\"ime\":{\"language\":"); json_number(&json, ime.language);
    json_bytes(&json, ime.open ? ",\"open\":true" : ",\"open\":false");
    json_bytes(&json, ime.composing ? ",\"composing\":true" : ",\"composing\":false");
    json_bytes(&json, ",\"conversion\":"); json_number(&json, ime.conversion);
    json_bytes(&json, ",\"composition\":"); json_string(&json, ime.composition);
    json_bytes(&json, ",\"starts\":"); json_number(&json, ime.starts);
    json_bytes(&json, ",\"updates\":"); json_number(&json, ime.updates);
    json_bytes(&json, ",\"ends\":"); json_number(&json, ime.ends);
    json_bytes(&json, ",\"results\":"); json_number(&json, ime.results);
    json_bytes(&json, ",\"nativeKeys\":"); json_number(&json, ime.keys);
    json_bytes(&json, ",\"lastKey\":"); json_number(&json, ime.key);
    json_bytes(&json, ",\"lastRawKey\":"); json_number(&json, ime.raw_key);
    json_bytes(&json, ime.during_key ? ",\"keyDuringComposition\":true}" : ",\"keyDuringComposition\":false}");
    if (!json_geometry(&json, window, state)) { return FALSE; }
    json_bytes(&json, "}\n");
    return json.valid && automation_write(state, json.bytes, json.size);
}

static BOOL automation_error(WindowState *state) {
    const char error[] = "{\"ok\":false,\"error\":\"invalid_command_or_input\"}\n";
    return automation_write(state, error, sizeof(error) - 1);
}

static BOOL resize_dimension(const char **cursor, char delimiter, int *output) {
    int value = 0;
    if (**cursor < '1' || **cursor > '9') { return FALSE; }
    while (**cursor >= '0' && **cursor <= '9') {
        value = value * 10 + (*(*cursor)++ - '0');
        if (value > 4096) { return FALSE; }
    }
    if (**cursor != delimiter) { return FALSE; }
    if (delimiter != '\0') { (*cursor)++; }
    *output = value;
    return TRUE;
}

static BOOL automation_command(HWND window, WindowState *state) {
    if (state->command_overflow || memchr(state->command, '\0', state->command_size) != NULL) {
        return automation_error(state);
    }
    state->command[state->command_size] = '\0';
    if (strcmp(state->command, "state") == 0) {
        return automation_state(window, state);
    }
    if (strcmp(state->command, "ime-focus") == 0) {
        SetForegroundWindow(window);
        SetFocus(state->input);
        return automation_state(window, state);
    }
    if (strcmp(state->command, "click") == 0) {
        SendMessageW(state->apply, BM_CLICK, 0, 0);
        state->reply_pending = TRUE;
        return TRUE;
    }
    if (strcmp(state->command, "close") == 0) {
        state->close_reply_pending = TRUE;
        SendMessageW(window, WM_CLOSE, 0, 0);
        return TRUE;
    }
    if (strcmp(state->command, "arm-close") == 0) {
        state->close_reply_pending = TRUE;
        state->desktop_close_armed = TRUE;
        state->system_close_seen = FALSE;
        if (IsIconic(window) || IsZoomed(window)) { ShowWindow(window, SW_RESTORE); }
        SetForegroundWindow(window);
        return automation_state(window, state);
    }
    if (strncmp(state->command, "arm-key\t", 8) == 0) {
        const char *cursor = state->command + 8;
        int code, focus;
        if (!resize_dimension(&cursor, '\t', &code) || (code != VK_RETURN && code != VK_ESCAPE && code != VK_TAB) ||
            state->key_expected != 0) { return automation_error(state); }
        if (strcmp(cursor, "0") == 0) { focus = 0; }
        else if (!resize_dimension(&cursor, '\0', &focus) || (focus != INPUT_ID && focus != APPLY_ID)) {
            return automation_error(state);
        }
        SetForegroundWindow(window);
        if (focus != 0) {
            SetFocus(GetDlgItem(window, focus));
            if (GetFocus() != GetDlgItem(window, focus)) { return automation_error(state); }
        }
        state->key_expected = code;
        return automation_state(window, state);
    }
    if (strncmp(state->command, "resize\t", 7) == 0) {
        const char *cursor = state->command + 7;
        int width, height;
        if (!resize_dimension(&cursor, '\t', &width) || !resize_dimension(&cursor, '\0', &height) ||
            width < state->min_width || height < state->min_height) { return automation_error(state); }
        if (IsIconic(window) || IsZoomed(window)) { ShowWindow(window, SW_RESTORE); }
        if (!SetWindowPos(window, NULL, 0, 0, width, height,
                          SWP_NOMOVE | SWP_NOZORDER | SWP_NOACTIVATE)) { return automation_error(state); }
        if (state->size_pending) {
            state->resize_reply_pending = TRUE;
            return TRUE;
        }
        return automation_state(window, state);
    }
    if (strncmp(state->command, "dpi-test\t", 9) == 0) {
        const char *cursor = state->command + 9;
        int dpi, width, height;
        RECT suggested;
        if (!resize_dimension(&cursor, '\t', &dpi) || dpi < 96 || dpi > 384 ||
            !resize_dimension(&cursor, '\t', &width) || !resize_dimension(&cursor, '\0', &height) ||
            !GetWindowRect(window, &suggested)) { return automation_error(state); }
        suggested.right = suggested.left + width;
        suggested.bottom = suggested.top + height;
        state->dpi_injections++;
        state->dpi_reply_pending = TRUE;
        SendMessageW(window, WM_DPICHANGED, MAKEWPARAM(dpi, dpi), (LPARAM)&suggested);
        return TRUE;
    }
    if (strncmp(state->command, "input\t", 6) == 0) {
        WCHAR text[INPUT_LIMIT + 1];
        int length = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, state->command + 6,
                                        -1, text, (int)ARRAYSIZE(text));
        if (length == 0 || wcschr(text, L'\r') != NULL ||
            !SetWindowTextW(state->input, text)) {
            return automation_error(state);
        }
        if (state->pending_event != 0) {
            state->reply_pending = TRUE;
            return TRUE;
        }
        return automation_state(window, state);
    }
    return automation_error(state);
}

static void automation_poll(HWND window, WindowState *state) {
    DWORD available = 0;
    if (!PeekNamedPipe(state->command_input, NULL, 0, NULL, &available, NULL)) {
        DestroyWindow(window);
        return;
    }
    if (available > sizeof(state->command)) {
        available = (DWORD)sizeof(state->command);
    }
    while (available-- > 0) {
        char byte;
        DWORD count = 0;
        if (!ReadFile(state->command_input, &byte, 1, &count, NULL) || count != 1) {
            DestroyWindow(window);
            return;
        }
        if (byte == '\n') {
            BOOL sent = automation_command(window, state);
            state->command_size = 0;
            state->command_overflow = FALSE;
            if (state->pending_event || state->size_pending || state->dpi_pending || !IsWindow(window)) {
                return;
            }
            if (!sent) {
                DestroyWindow(window);
                return;
            }
        } else if (state->command_size < sizeof(state->command) - 1) {
            state->command[state->command_size++] = byte;
        } else {
            state->command_overflow = TRUE;
        }
    }
}

static BOOL argument(LPCWSTR flag) {
    LPCWSTR cursor = GetCommandLineW();
    if (*cursor == L'"') {
        cursor++;
        while (*cursor != L'\0' && *cursor != L'"') { cursor++; }
        if (*cursor != L'\0') { cursor++; }
    } else {
        while (*cursor != L'\0' && *cursor != L' ' && *cursor != L'\t') { cursor++; }
    }
    while (*cursor == L' ' || *cursor == L'\t') { cursor++; }
    return wcscmp(cursor, flag) == 0;
}

