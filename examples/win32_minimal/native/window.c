#define UNICODE
#define _UNICODE
#include <windows.h>
#include <stdint.h>
#include <wchar.h>
#include <string.h>
#include "native/ime.h"
#include "native/profile.h"

enum { INPUT_ID = 201, APPLY_ID = 202, RESULT_ID = 203, STATUS_ID = 204, INPUT_LIMIT = 127 };

typedef struct {
    int32_t pending_event;
    BOOL size_pending;
    BOOL dpi_pending;
    BOOL dpi_reply_pending;
    UINT dpi;
    unsigned dpi_changes;
    unsigned dpi_injections;
    RECT dpi_suggested;
    RECT dpi_applied;
    HFONT font;
    int32_t font_height;
    UINT pointer_hit;
    UINT pointer_message;
    UINT pointer_press_hit;
    unsigned pointer_moves;
    unsigned pointer_presses;
    LPARAM pointer_press_point;
    ULONG_PTR pointer_press_extra;
    RECT pointer_press_window;
    int32_t key_code;
    int32_t key_flags;
    int32_t key_focus;
    int32_t key_action;
    unsigned keys;
    int key_expected;
    BOOL key_pending;
    BOOL initial_reply_pending;
    BOOL resize_reply_pending;
    int32_t min_width;
    int32_t min_height;
    unsigned resizes;
    void *app_fiber;
    void *pump_fiber;
    BOOL pump_finished;
    BOOL sizing;
    unsigned size_entries;
    unsigned size_exits;
    unsigned live_layouts;
    BOOL reply_pending;
    BOOL close_reply_pending;
    BOOL close_in_flight;
    BOOL desktop_close_armed;
    BOOL system_close_seen;
    unsigned smoke_step;
    unsigned smoke_edits;
    int32_t exit_code;
    HWND input;
    HWND apply;
    HWND result;
    HWND status;
    unsigned edits;
    unsigned clicks;
#if FAIRY_WIN32_DIAGNOSTICS
    HANDLE command_input;
    HANDLE command_output;
    char command[1024];
    size_t command_size;
    BOOL command_overflow;
#endif
} WindowState;

static WindowState bridge_state;
static const int layout_ids[] = {200, INPUT_ID, APPLY_ID, RESULT_ID, STATUS_ID};
#if FAIRY_WIN32_DIAGNOSTICS
static const char *layout_names[] = {"label", "input", "apply", "result", "status"};
#endif

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

#if FAIRY_WIN32_DIAGNOSTICS
#include "native/diagnostics.h"
#endif

static int32_t acknowledge(HWND window) {
#if FAIRY_WIN32_DIAGNOSTICS
    if (bridge_state.reply_pending) {
        bridge_state.reply_pending = FALSE;
        if (!automation_state(window, &bridge_state)) { return 1; }
    }
#else
    (void)window;
#endif
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

int32_t fairy_win32_client_extent(void *native_window, uint32_t axis) {
    RECT client;
    if (axis > 1 || !GetClientRect((HWND)native_window, &client)) { return -1; }
    return axis == 0 ? client.right : client.bottom;
}

int32_t fairy_win32_initialize_dpi(void) {
    if (!SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2) &&
        !AreDpiAwarenessContextsEqual(GetThreadDpiAwarenessContext(), DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)) { return -1; }
    return (int32_t)GetDpiForSystem();
}

int32_t fairy_win32_dpi(void *native_window) {
    return IsWindow((HWND)native_window) ? (int32_t)bridge_state.dpi : -1;
}

int32_t fairy_win32_font(void *native_window, int32_t height) {
    HWND window = (HWND)native_window;
    if (!IsWindow(window) || height <= 0 || height > 512) { return 1; }
    if (bridge_state.font != NULL && bridge_state.font_height == height) { return 0; }
    HFONT font = CreateFontW(-height, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE, DEFAULT_CHARSET,
                            OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY,
                            DEFAULT_PITCH | FF_DONTCARE, L"Segoe UI");
    if (font == NULL) { return 1; }
    for (size_t i = 0; i < ARRAYSIZE(layout_ids); i++) {
        SendMessageW(GetDlgItem(window, layout_ids[i]), WM_SETFONT, (WPARAM)font, TRUE);
    }
    if (bridge_state.font != NULL) { DeleteObject(bridge_state.font); }
    bridge_state.font = font;
    bridge_state.font_height = height;
    return 0;
}

int32_t fairy_win32_accept_dpi(void *native_window, uint32_t changes, int32_t min_width, int32_t min_height) {
    HWND window = (HWND)native_window;
    RECT suggested = bridge_state.dpi_suggested;
    if (min_width <= 0 || min_height <= 0) { return 1; }
    bridge_state.min_width = min_width;
    bridge_state.min_height = min_height;
    if (suggested.right <= suggested.left || suggested.bottom <= suggested.top ||
        !SetWindowPos(window, NULL, suggested.left, suggested.top,
                      suggested.right - suggested.left, suggested.bottom - suggested.top,
                      SWP_NOZORDER | SWP_NOACTIVATE)) { return 1; }
    bridge_state.dpi_changes = changes;
    bridge_state.dpi_applied = suggested;
    // Fairy's on_dpi immediately runs on_size for this client size.
    bridge_state.size_pending = FALSE;
    return 0;
}

int32_t fairy_win32_move(void *native_window, uint32_t control_id, int32_t x, int32_t y,
                        int32_t width, int32_t height) {
    if (control_id < 200 || control_id > STATUS_ID || width < 0 || height < 0) { return 1; }
    return MoveWindow(GetDlgItem((HWND)native_window, (int)control_id), x, y, width, height, TRUE) ? 0 : 1;
}

int32_t fairy_win32_layout_done(void *native_window, int32_t min_width, int32_t min_height, uint32_t resizes) {
    HWND window = (HWND)native_window;
    if (!IsWindow(window) || min_width <= 0 || min_height <= 0) { return 1; }
    bridge_state.min_width = min_width;
    bridge_state.min_height = min_height;
    bridge_state.resizes = resizes;
#if FAIRY_WIN32_DIAGNOSTICS
    if (bridge_state.sizing && (GetAsyncKeyState(VK_LBUTTON) & 0x8000)) {
        bridge_state.live_layouts++;
    }
    if (!bridge_state.dpi_pending &&
        (bridge_state.initial_reply_pending || bridge_state.resize_reply_pending || bridge_state.dpi_reply_pending)) {
        bridge_state.initial_reply_pending = FALSE;
        bridge_state.resize_reply_pending = FALSE;
        bridge_state.dpi_reply_pending = FALSE;
        if (!automation_state(window, &bridge_state)) { return 1; }
    }
#endif
    return 0;
}

int32_t fairy_win32_resolve_close(void *native_window, uint32_t allow) {
    HWND window = (HWND)native_window;
    if (!IsWindow(window) || allow > 1) { return 1; }
    if (allow != 0 && !DestroyWindow(window)) { return 1; }
#if FAIRY_WIN32_DIAGNOSTICS
    if (bridge_state.close_reply_pending) {
        Json reply = {{0}, 0, TRUE};
        json_bytes(&reply, allow != 0 ? "{\"ok\":true,\"closed\":true" :
                                       "{\"ok\":true,\"closed\":false");
        if (bridge_state.desktop_close_armed) {
            json_bytes(&reply, bridge_state.system_close_seen ? ",\"systemClose\":true" :
                                                               ",\"systemClose\":false");
            json_bytes(&reply, bridge_state.close_in_flight ? ",\"closeHandoff\":true" : ",\"closeHandoff\":false");
        }
        json_bytes(&reply, "}\n");
        bridge_state.close_reply_pending = FALSE;
        bridge_state.desktop_close_armed = FALSE;
        if (!reply.valid || !automation_write(&bridge_state, reply.bytes, reply.size)) { return 1; }
    }
#endif
    return 0;
}

int32_t fairy_win32_key(void *native_window, uint32_t axis) {
    if (native_window == NULL || axis > 2) { return -1; }
    if (axis == 2) { return bridge_state.key_focus; }
    return axis == 0 ? bridge_state.key_code : bridge_state.key_flags;
}

int32_t fairy_win32_focus(void *native_window, int32_t control_id) {
    HWND child = GetDlgItem((HWND)native_window, control_id);
    if (!child || !IsWindowVisible(child) || !IsWindowEnabled(child) ||
        !(GetWindowLongPtrW(child, GWL_STYLE) & WS_TABSTOP)) { return 1; }
    SetFocus(child);
    return GetFocus() == child ? 0 : 1;
}

int32_t fairy_win32_key_done(void *native_window, uint32_t keys, int32_t action) {
    if (action != 0 && action != 1 && action != 3 && action != 4) { return 1; }
    bridge_state.keys = keys;
    bridge_state.key_action = action;
#if FAIRY_WIN32_DIAGNOSTICS
    if (bridge_state.key_expected == 0) { return 0; }
    bridge_state.key_expected = 0;
    if (IsWindow((HWND)native_window)) { return automation_state((HWND)native_window, &bridge_state) ? 0 : 1; }
    Json reply = {{0}, 0, TRUE};
    json_bytes(&reply, "{\"ok\":true,\"closed\":true,\"keyCode\":"); json_signed(&reply, bridge_state.key_code);
    json_bytes(&reply, ",\"keyFlags\":"); json_signed(&reply, bridge_state.key_flags);
    json_bytes(&reply, ",\"keyFocus\":"); json_signed(&reply, bridge_state.key_focus);
    json_bytes(&reply, ",\"keyAction\":"); json_signed(&reply, action);
    json_bytes(&reply, ",\"keys\":"); json_number(&reply, keys);
    json_bytes(&reply, "}\n");
    return reply.valid && automation_write(&bridge_state, reply.bytes, reply.size) ? 0 : 1;
#else
    (void)native_window;
    return 0;
#endif

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
        if (!fairy_ime_attach(state->input)) { return -1; }
        SendMessageW(state->input, EM_SETLIMITTEXT, INPUT_LIMIT, 0);
        resource_text(state->apply, 103);
        resource_text(state->result, 102);
        resource_text(state->status, 104);
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
        if (state) {
            state->size_pending = TRUE;
            if (state->pump_fiber && GetCurrentFiber() == state->pump_fiber) {
                SwitchToFiber(state->app_fiber); /* live-size handoff */
            }
        }
        return 0;
#if FAIRY_WIN32_DIAGNOSTICS
    case WM_NCMOUSEMOVE:
        if (state) { state->pointer_hit = (UINT)wparam; state->pointer_moves++; }
        break;
    case WM_NCLBUTTONDOWN:
    case WM_NCLBUTTONDBLCLK:
        if (state) {
            state->pointer_message = message;
            state->pointer_press_hit = (UINT)wparam;
            state->pointer_press_point = lparam;
            state->pointer_press_extra = (ULONG_PTR)GetMessageExtraInfo();
            GetWindowRect(window, &state->pointer_press_window);
            state->pointer_presses++;
        }
        break;
    case WM_APP + 1:
        // Read-only delivery probe, scoped to the controller's owned HWND.
        if (wparam == 0) { return 0x46594931; }
        if (!state) { return 0; }
        if (wparam == 1) { return state->pointer_hit; }
        if (wparam == 2) { return state->pointer_presses; }
        if (wparam == 3) { return state->pointer_message; }
        if (wparam == 4) { return state->pointer_press_hit; }
        if (wparam == 5) { return state->pointer_moves; }
        if (wparam == 6) { return (LRESULT)GetCapture(); }
        if (wparam == 7) { return GetDlgCtrlID(GetFocus()); }
        if (wparam == 8) { return (DWORD)state->pointer_press_point; }
        if (wparam == 9) { return (LRESULT)state->pointer_press_extra; }
        if (wparam == 10) { return state->pointer_press_window.left; }
        if (wparam == 11) { return state->pointer_press_window.top; }
        if (wparam == 12) { return state->pointer_press_window.right - state->pointer_press_window.left; }
        if (wparam == 13) { return state->pointer_press_window.bottom - state->pointer_press_window.top; }
        return 0;
#endif
    case WM_DPICHANGED:
        if (LOWORD(wparam) > 0 && LOWORD(wparam) <= 960 && LOWORD(wparam) == HIWORD(wparam) && lparam != 0) {
            state->dpi = LOWORD(wparam);
            state->dpi_suggested = *(const RECT *)lparam;
            state->dpi_pending = TRUE;
            if (state->pump_fiber && GetCurrentFiber() == state->pump_fiber) {
                SwitchToFiber(state->app_fiber);
            }
        }
        return 0;
    case WM_ENTERSIZEMOVE:
        state->sizing = TRUE;
        state->size_entries++;
        return 0;
    case WM_EXITSIZEMOVE:
        state->sizing = FALSE;
        state->size_exits++;
        return 0;
    case WM_GETMINMAXINFO:
        if (state && state->min_width > 0 && state->min_height > 0) {
            ((MINMAXINFO *)lparam)->ptMinTrackSize.x = state->min_width;
            ((MINMAXINFO *)lparam)->ptMinTrackSize.y = state->min_height;
        }
        return 0;
    case WM_SETFOCUS:
        if (state && state->input) {
            SetFocus(state->input);
        }
        return 0;
    case WM_CLOSE:
        state->pending_event = 3;
        if (state->pump_fiber && GetCurrentFiber() == state->pump_fiber) {
            state->close_in_flight = TRUE;
            SwitchToFiber(state->app_fiber); /* close handoff */
            state->close_in_flight = FALSE;
        }
        return 0;
#if FAIRY_WIN32_DIAGNOSTICS
    case WM_SYSCOMMAND:
        if ((wparam & 0xFFF0) == SC_CLOSE) { state->system_close_seen = TRUE; }
        break;
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
#endif
    case WM_DESTROY:
        PostQuitMessage(state->exit_code);
        return 0;
    case WM_NCDESTROY:
        SetWindowLongPtrW(window, GWLP_USERDATA, 0);
        break;
    }
    return DefWindowProcW(window, message, wparam, lparam);
}

static VOID WINAPI message_fiber(void *native_window) {
    HWND window = (HWND)native_window;
    MSG message;
    while (TRUE) {
        int result = (int)GetMessageW(&message, NULL, 0, 0);
        if (result <= 0) {
            bridge_state.exit_code = result < 0 ? 4 : (int32_t)message.wParam;
            bridge_state.pump_finished = TRUE;
            break;
        }
        UINT original_key = fairy_ime_observe(&message);
        BOOL ime_owns_key = fairy_ime_owns_key();
        if (message.message == WM_KEYDOWN && (message.hwnd == window || IsChild(window, message.hwnd)) &&
            (message.wParam == VK_RETURN || message.wParam == VK_ESCAPE || message.wParam == VK_TAB ||
             (ime_owns_key && (original_key == VK_RETURN || original_key == VK_ESCAPE || original_key == VK_TAB)))) {
            // Capture native key data before IsDialogMessage can activate a default button.
            bridge_state.key_code = original_key ? (int32_t)original_key : (int32_t)message.wParam;
            bridge_state.key_focus = GetDlgCtrlID(message.hwnd);
            bridge_state.key_flags = ((GetKeyState(VK_SHIFT) & 0x8000) ? 1 : 0) |
                ((GetKeyState(VK_CONTROL) & 0x8000) ? 2 : 0) |
                ((GetKeyState(VK_MENU) & 0x8000) ? 4 : 0) |
                (((GetKeyState(VK_LWIN) | GetKeyState(VK_RWIN)) & 0x8000) ? 8 : 0) |
                ((message.lParam & ((LPARAM)1 << 30)) ? 16 : 0) |
                (ime_owns_key ? 32 : 0);
            bridge_state.key_pending = TRUE;
            if (ime_owns_key) {
                // Let the native edit/IME process composition, bypassing dialog shortcuts.
                TranslateMessage(&message);
                DispatchMessageW(&message);
            }
        } else if (!IsDialogMessageW(window, &message)) {
            TranslateMessage(&message);
            DispatchMessageW(&message);
        }
        SwitchToFiber(bridge_state.app_fiber);
    }
    // A fiber procedure must never return (that would terminate the UI thread).
    for (;;) { SwitchToFiber(bridge_state.app_fiber); }
}

void *fairy_win32_open(int32_t width, int32_t height) {
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
                                 CW_USEDEFAULT, CW_USEDEFAULT, width, height,
                                 NULL, NULL, instance, state);
    if (window == NULL) {
        return NULL;
    }
    state->dpi = GetDpiForWindow(window);
#if FAIRY_WIN32_DIAGNOSTICS
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
#endif
    ShowWindow(window, SW_SHOWNORMAL);
    UpdateWindow(window);
#if FAIRY_WIN32_DIAGNOSTICS
    state->initial_reply_pending = state->command_input != NULL;
#endif
    state->size_pending = TRUE;
    if (!IsThreadAFiber()) {
        state->app_fiber = ConvertThreadToFiberEx(NULL, FIBER_FLAG_FLOAT_SWITCH);
    }
    if (state->app_fiber != NULL) {
        state->pump_fiber = CreateFiberEx(0, 1024 * 1024, FIBER_FLAG_FLOAT_SWITCH, message_fiber, window);
    }
    if (state->pump_fiber == NULL) {
        DestroyWindow(window);
        UnregisterClassW(L"FairyMinimalWindow", instance);
        if (state->app_fiber != NULL) { ConvertFiberToThread(); }
        return NULL;
    }
    return window;
}

int32_t fairy_win32_next(void *native_window) {
    if (native_window == NULL || bridge_state.pump_fiber == NULL) { return 0; }
    while (TRUE) {
        if (bridge_state.dpi_pending) {
            bridge_state.dpi_pending = FALSE;
            return 5;
        }
        if (bridge_state.size_pending) {
            bridge_state.size_pending = FALSE;
            return 4;
        }
        if (bridge_state.pending_event != 0) {
            int32_t event = bridge_state.pending_event;
            bridge_state.pending_event = 0;
            return event;
        }
        if (bridge_state.key_pending) {
            bridge_state.key_pending = FALSE;
            return 6;
        }
        if (bridge_state.pump_finished) { return 0; }
        SwitchToFiber(bridge_state.pump_fiber);
    }
}

int32_t fairy_win32_close(void *native_window) {
    HWND window = (HWND)native_window;
    if (IsWindow(window)) { DestroyWindow(window); }
    if (bridge_state.pump_fiber != NULL) {
        // Resume any suspended WndProc/modal frames through WM_QUIT before deletion.
        while (!bridge_state.pump_finished) { SwitchToFiber(bridge_state.pump_fiber); }
        DeleteFiber(bridge_state.pump_fiber);
        bridge_state.pump_fiber = NULL;
        if (!ConvertFiberToThread() && bridge_state.exit_code == 0) { bridge_state.exit_code = 4; }
        bridge_state.app_fiber = NULL;
    }
    UnregisterClassW(L"FairyMinimalWindow", GetModuleHandleW(NULL));
    if (bridge_state.font != NULL) { DeleteObject(bridge_state.font); bridge_state.font = NULL; }
    return bridge_state.exit_code;
}
