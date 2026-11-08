#define UNICODE
#define _UNICODE
#include <windows.h>
#include <stdint.h>
#include <wchar.h>
#include <string.h>

enum { INPUT_ID = 201, APPLY_ID = 202, RESULT_ID = 203, STATUS_ID = 204, INPUT_LIMIT = 127 };

typedef struct {
    int32_t pending_event;
    BOOL reply_pending;
    BOOL close_reply_pending;
    unsigned smoke_step;
    unsigned smoke_edits;
    int32_t exit_code;
    HWND input;
    HWND apply;
    HWND result;
    HWND status;
    unsigned edits;
    unsigned clicks;
    HANDLE command_input;
    HANDLE command_output;
    char command[1024];
    size_t command_size;
    BOOL command_overflow;
} WindowState;

static WindowState bridge_state;

static void resource_text(HWND control, UINT id) {
    WCHAR text[160];
    if (LoadStringW(GetModuleHandleW(NULL), id, text, (int)ARRAYSIZE(text)) != 0) {
        SetWindowTextW(control, text);
    }
}

static HWND control(HWND parent, LPCWSTR kind, LPCWSTR text, DWORD style, int id) {
    HWND child = CreateWindowExW(wcscmp(kind, L"EDIT") == 0 ? WS_EX_CLIENTEDGE : 0,
                                 kind, text, WS_CHILD | WS_VISIBLE | style,
                                 0, 0, 0, 0, parent, (HMENU)(INT_PTR)id,
                                 GetModuleHandleW(NULL), NULL);
    if (child != NULL) {
        SendMessageW(child, WM_SETFONT, (WPARAM)GetStockObject(DEFAULT_GUI_FONT), TRUE);
    }
    return child;
}

static void layout(HWND window, WindowState *state) {
    RECT area;
    GetClientRect(window, &area);
    int width = area.right > 320 ? area.right : 320;
    MoveWindow(GetDlgItem(window, 200), 16, 16, width - 32, 20, TRUE);
    MoveWindow(state->input, 16, 44, width - 132, 28, TRUE);
    MoveWindow(state->apply, width - 100, 44, 84, 28, TRUE);
    MoveWindow(state->result, 16, 94, width - 32, 48, TRUE);
    MoveWindow(state->status, 16, 154, width - 32, 24, TRUE);
}

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

static void json_text(Json *json, HWND control) {
    WCHAR text[160];
    char utf8[640];
    const char hex[] = "0123456789abcdef";
    text[0] = L'\0';
    GetWindowTextW(control, text, (int)ARRAYSIZE(text));
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
    json_bytes(&json, "}\n");
    return json.valid && automation_write(state, json.bytes, json.size);
}

static BOOL automation_error(WindowState *state) {
    const char error[] = "{\"ok\":false,\"error\":\"invalid_command_or_input\"}\n";
    return automation_write(state, error, sizeof(error) - 1);
}

static BOOL automation_command(HWND window, WindowState *state) {
    if (state->command_overflow || memchr(state->command, '\0', state->command_size) != NULL) {
        return automation_error(state);
    }
    state->command[state->command_size] = '\0';
    if (strcmp(state->command, "state") == 0) {
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
            if (state->pending_event || !IsWindow(window)) {
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

static int32_t acknowledge(HWND window) {
    if (bridge_state.reply_pending) {
        bridge_state.reply_pending = FALSE;
        if (!automation_state(window, &bridge_state)) { return 1; }
    }
    return 0;
}

int32_t fairy_win32_changed(void *native_window, uint32_t edits, uint32_t status_id) {
    HWND window = (HWND)native_window;
    HWND status = GetDlgItem(window, STATUS_ID);
    if (status == NULL) { return 1; }
    resource_text(status, status_id);
    bridge_state.edits = edits;
    return acknowledge(window);
}

int32_t fairy_win32_resolve_close(void *native_window, uint32_t allow) {
    HWND window = (HWND)native_window;
    if (!IsWindow(window) || allow > 1) { return 1; }
    if (allow != 0 && !DestroyWindow(window)) { return 1; }
    if (bridge_state.close_reply_pending) {
        const char *reply = allow != 0 ? "{\"ok\":true,\"closed\":true}\n" :
                                        "{\"ok\":true,\"closed\":false}\n";
        bridge_state.close_reply_pending = FALSE;
        if (!automation_write(&bridge_state, reply, strlen(reply))) { return 1; }
    }
    return 0;
}

size_t fairy_win32_read_input(void *native_window, uint8_t *output, size_t capacity) {
    WCHAR text[INPUT_LIMIT + 1];
    HWND input = GetDlgItem((HWND)native_window, INPUT_ID);
    if (input == NULL || output == NULL || capacity == 0 || capacity > 512) {
        return (size_t)-1;
    }
    int length = GetWindowTextW(input, text, (int)ARRAYSIZE(text));
    if (length == 0) {
        return 0;
    }
    int bytes = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, text, length,
                                   (char *)output, (int)capacity, NULL, NULL);
    return bytes == 0 ? (size_t)-1 : (size_t)bytes;
}

int32_t fairy_win32_present(void *native_window, const uint8_t *text, size_t length, uint32_t clicks, uint32_t status_id) {
    WCHAR wide[160];
    HWND window = (HWND)native_window;
    HWND result = GetDlgItem(window, RESULT_ID);
    if (result == NULL || text == NULL || length == 0 || length > 512) {
        return 1;
    }
    int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)text,
                                   (int)length, wide, (int)ARRAYSIZE(wide) - 1);
    if (count == 0) {
        return 1;
    }
    wide[count] = L'\0';
    if (!SetWindowTextW(result, wide)) {
        return 1;
    }
    resource_text(GetDlgItem(window, STATUS_ID), status_id);
    bridge_state.clicks = clicks;
    return acknowledge(window);
}

static LRESULT CALLBACK window_proc(HWND window, UINT message, WPARAM wparam, LPARAM lparam) {
    WindowState *state = (WindowState *)GetWindowLongPtrW(window, GWLP_USERDATA);
    switch (message) {
    case WM_NCCREATE:
        state = (WindowState *)((CREATESTRUCTW *)lparam)->lpCreateParams;
        SetWindowLongPtrW(window, GWLP_USERDATA, (LONG_PTR)state);
        return DefWindowProcW(window, message, wparam, lparam);
    case WM_CREATE:
        if (control(window, L"STATIC", L"&Message:", 0, 200) == NULL) {
            return -1;
        }
        state->input = control(window, L"EDIT", L"", WS_TABSTOP | ES_AUTOHSCROLL, INPUT_ID);
        state->apply = control(window, L"BUTTON", L"", WS_TABSTOP | BS_DEFPUSHBUTTON, APPLY_ID);
        state->result = control(window, L"STATIC", L"", SS_NOPREFIX, RESULT_ID);
        state->status = control(window, L"STATIC", L"", SS_NOPREFIX, STATUS_ID);
        if (!state->input || !state->apply || !state->result || !state->status) {
            return -1;
        }
        SendMessageW(state->input, EM_SETLIMITTEXT, INPUT_LIMIT, 0);
        resource_text(state->apply, 103);
        resource_text(state->result, 102);
        resource_text(state->status, 104);
        layout(window, state);
        return 0;
    case WM_COMMAND:
        if (LOWORD(wparam) == INPUT_ID && HIWORD(wparam) == EN_CHANGE &&
            (HWND)lparam == state->input) {
            state->pending_event = 2;
            return 0;
        }
        if (LOWORD(wparam) == APPLY_ID && HIWORD(wparam) == BN_CLICKED &&
            (HWND)lparam == state->apply) {
            state->pending_event = 1;
            return 0;
        }
        break;
    case WM_SIZE:
        if (state && state->status) {
            layout(window, state);
        }
        return 0;
    case WM_GETMINMAXINFO:
        ((MINMAXINFO *)lparam)->ptMinTrackSize.x = 400;
        ((MINMAXINFO *)lparam)->ptMinTrackSize.y = 240;
        return 0;
    case WM_SETFOCUS:
        if (state && state->input) {
            SetFocus(state->input);
        }
        return 0;
    case WM_CLOSE:
        state->pending_event = 3;
        return 0;
    case WM_TIMER:
        if (wparam == 2 && state->command_input != NULL) {
            automation_poll(window, state);
            return 0;
        }
        if (wparam == 1) {
            smoke_events(window, state);
            return 0;
        }
        break;
    case WM_DESTROY:
        PostQuitMessage(state->exit_code);
        return 0;
    case WM_NCDESTROY:
        SetWindowLongPtrW(window, GWLP_USERDATA, 0);
        break;
    }
    return DefWindowProcW(window, message, wparam, lparam);
}

void *fairy_win32_open(void) {
    HINSTANCE instance = GetModuleHandleW(NULL);
    WCHAR title[64];
    WNDCLASSW kind = {0};
    kind.lpfnWndProc = window_proc;
    kind.hInstance = instance;
    kind.hCursor = LoadCursorW(NULL, IDC_ARROW);
    kind.hbrBackground = (HBRUSH)(COLOR_WINDOW + 1);
    kind.lpszClassName = L"FairyMinimalWindow";
    if (GetConsoleWindow() != NULL || LoadStringW(instance, 101, title, 64) == 0 ||
        RegisterClassW(&kind) == 0) {
        return NULL;
    }
    WindowState *state = &bridge_state;
    memset(state, 0, sizeof(*state));
    HWND window = CreateWindowExW(WS_EX_CONTROLPARENT, kind.lpszClassName, title, WS_OVERLAPPEDWINDOW,
                                 CW_USEDEFAULT, CW_USEDEFAULT, 480, 240,
                                 NULL, NULL, instance, state);
    if (window == NULL) {
        return NULL;
    }
    if (argument(L"--smoke") && SetTimer(window, 1, 100, NULL) == 0) {
        DestroyWindow(window);
        return NULL;
    }
    if (argument(L"--automation-stdio")) {
        state->command_input = GetStdHandle(STD_INPUT_HANDLE);
        state->command_output = GetStdHandle(STD_OUTPUT_HANDLE);
        if (GetFileType(state->command_input) != FILE_TYPE_PIPE ||
            GetFileType(state->command_output) != FILE_TYPE_PIPE ||
            SetTimer(window, 2, 20, NULL) == 0) {
            DestroyWindow(window);
            return NULL;
        }
    }
    ShowWindow(window, SW_SHOWNORMAL);
    UpdateWindow(window);
    if (state->command_input != NULL && !automation_state(window, state)) {
        DestroyWindow(window);
        return NULL;
    }
    return window;
}

int32_t fairy_win32_next(void *native_window) {
    HWND window = (HWND)native_window;
    MSG message;
    while (TRUE) {
        if (bridge_state.pending_event != 0) {
            int32_t event = bridge_state.pending_event;
            bridge_state.pending_event = 0;
            return event;
        }
        int result = (int)GetMessageW(&message, NULL, 0, 0);
        if (result <= 0) {
            bridge_state.exit_code = result < 0 ? 4 : (int32_t)message.wParam;
            return 0;
        }
        if (!IsDialogMessageW(window, &message)) {
            TranslateMessage(&message);
            DispatchMessageW(&message);
        }
    }
}

int32_t fairy_win32_close(void *native_window) {
    HWND window = (HWND)native_window;
    if (IsWindow(window)) { DestroyWindow(window); }
    UnregisterClassW(L"FairyMinimalWindow", GetModuleHandleW(NULL));
    return bridge_state.exit_code;
}
