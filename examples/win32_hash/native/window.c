#define UNICODE
#define _UNICODE
#include <windows.h>
#include <commdlg.h>
#include <stdint.h>

// Win32 operations only. Fairy owns file I/O, SHA-256, report bytes and policy.
static struct {
    HWND window;
    HFONT font;
    int event;
    int minimum_width, minimum_height;
    RECT suggested;
} state;

static void dispatch(MSG *message) {
    if (!IsDialogMessageW(state.window, message)) {
        TranslateMessage(message);
        DispatchMessageW(message);
    }
}

static LRESULT CALLBACK procedure(HWND window, UINT message, WPARAM wp, LPARAM lp) {
    switch (message) {
    case WM_COMMAND:
        if (HIWORD(wp) == BN_CLICKED && (LOWORD(wp) == 202 || LOWORD(wp) == 204) &&
            (HWND)lp == GetDlgItem(window, LOWORD(wp)) && IsWindowEnabled((HWND)lp)) {
            if (state.event != 3) { state.event = LOWORD(wp) == 202 ? 1 : 2; }
            return 0;
        }
        break;
    case WM_SIZE:
        if (state.event == 0) { state.event = 4; }
        return 0;
    case WM_DPICHANGED:
        if (lp && LOWORD(wp) == HIWORD(wp) && LOWORD(wp) > 0) {
            state.suggested = *(RECT *)lp;
            if (state.event != 3) { state.event = 5; }
        }
        return 0;
    case WM_GETMINMAXINFO:
        ((MINMAXINFO *)lp)->ptMinTrackSize.x = state.minimum_width;
        ((MINMAXINFO *)lp)->ptMinTrackSize.y = state.minimum_height;
        return 0;
    case WM_CLOSE:
        state.event = 3; // Sticky until Fairy finishes/cancels work and destroys.
        return 0;
    case WM_DESTROY:
        PostQuitMessage(0);
        return 0;
    }
    return DefWindowProcW(window, message, wp, lp);
}

int32_t hash_ui_system_dpi(void) {
    if (!SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2) &&
        !AreDpiAwarenessContextsEqual(GetThreadDpiAwarenessContext(), DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)) { return -1; }
    return (int32_t)GetDpiForSystem();
}

static BOOL child(HWND window, LPCWSTR kind, DWORD style, int id) {
    return CreateWindowExW(id == 201 || id == 203 ? WS_EX_CLIENTEDGE : 0, kind, L"",
        WS_CHILD | WS_VISIBLE | style, 0, 0, 0, 0, window, (HMENU)(INT_PTR)id, GetModuleHandleW(NULL), NULL) != NULL;
}

void *hash_ui_open(int32_t width, int32_t height) {
    WNDCLASSW type = {0};
    type.lpfnWndProc = procedure;
    type.hInstance = GetModuleHandleW(NULL);
    type.hCursor = LoadCursorW(NULL, IDC_ARROW);
    type.hbrBackground = (HBRUSH)(COLOR_WINDOW + 1);
    type.lpszClassName = L"FairyHashWindow";
    if (!RegisterClassW(&type)) { return NULL; }
    HWND window = CreateWindowExW(WS_EX_CONTROLPARENT, type.lpszClassName, L"Fairy Hash",
        WS_OVERLAPPEDWINDOW, CW_USEDEFAULT, CW_USEDEFAULT, width, height, NULL, NULL, type.hInstance, NULL);
    if (window && child(window, L"STATIC", 0, 206) &&
        child(window, L"EDIT", WS_TABSTOP | ES_READONLY | ES_AUTOHSCROLL, 201) &&
        child(window, L"BUTTON", WS_TABSTOP | BS_DEFPUSHBUTTON, 202) &&
        child(window, L"STATIC", 0, 207) &&
        child(window, L"EDIT", WS_TABSTOP | ES_READONLY | ES_AUTOHSCROLL, 203) &&
        child(window, L"BUTTON", WS_TABSTOP, 204) && child(window, L"STATIC", SS_NOPREFIX, 205)) {
        state.window = window;
        return window;
    }
    if (window) { DestroyWindow(window); }
    UnregisterClassW(type.lpszClassName, type.hInstance);
    return NULL;
}

int32_t hash_ui_poll(void *window, int32_t wait) {
    if ((HWND)window != state.window) { return 3; }
    MSG message;
    while (!state.event) {
        if (wait) {
            int code = (int)GetMessageW(&message, NULL, 0, 0);
            if (code <= 0) { state.event = 3; break; }
        } else {
            if (!PeekMessageW(&message, NULL, 0, 0, PM_REMOVE)) { break; }
            if (message.message == WM_QUIT) { state.event = 3; break; }
        }
        dispatch(&message);
    }
    int event = state.event;
    if (event != 3) { state.event = 0; }
    return event;
}

int32_t hash_ui_text(void *window, uint32_t id, const uint8_t *text, size_t length) {
    if ((HWND)window != state.window || (id != 0 && (id < 201 || id > 207)) || length > 131071) { return 1; }
    WCHAR *wide = HeapAlloc(GetProcessHeap(), 0, (length + 1) * sizeof(WCHAR));
    if (!wide) { return 1; }
    int count = length ? MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)text, (int)length, wide, (int)length) : 0;
    BOOL valid = length == 0 || count > 0;
    wide[count] = 0;
    BOOL success = valid && SetWindowTextW(id ? GetDlgItem((HWND)window, (int)id) : (HWND)window, wide);
    HeapFree(GetProcessHeap(), 0, wide);
    return success ? 0 : 1;
}

int32_t hash_ui_enable(void *window, uint32_t id, uint32_t enabled) {
    HWND control = GetDlgItem((HWND)window, (int)id);
    if ((HWND)window != state.window || !control || enabled > 1) { return 1; }
    EnableWindow(control, enabled != 0);
    return IsWindowEnabled(control) == (enabled != 0) ? 0 : 1;
}

int32_t hash_ui_extent(void *window, uint32_t axis) {
    RECT client;
    if (!GetClientRect((HWND)window, &client) || axis > 2) { return -1; }
    return axis == 2 ? (int32_t)GetDpiForWindow((HWND)window) : axis == 0 ? client.right : client.bottom;
}

int32_t hash_ui_move(void *window, uint32_t id, int32_t x, int32_t y, int32_t width, int32_t height) {
    if (width < 0 || height < 0 || id < 201 || id > 207) { return 1; }
    return MoveWindow(GetDlgItem((HWND)window, (int)id), x, y, width, height, TRUE) ? 0 : 1;
}

int32_t hash_ui_layout(void *window, int32_t minimum_width, int32_t minimum_height, int32_t font_height) {
    if ((HWND)window != state.window || minimum_width <= 0 || minimum_height <= 0 || font_height <= 0) { return 1; }
    state.minimum_width = minimum_width; state.minimum_height = minimum_height;
    LOGFONTW current = {0};
    if (state.font && GetObjectW(state.font, sizeof(current), &current) && current.lfHeight == -font_height) { return 0; }
    HFONT font = CreateFontW(-font_height, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE, DEFAULT_CHARSET,
        OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, DEFAULT_PITCH, L"Segoe UI");
    if (!font) { return 1; }
    for (int id = 201; id <= 207; id++) { SendMessageW(GetDlgItem((HWND)window, id), WM_SETFONT, (WPARAM)font, TRUE); }
    if (state.font) { DeleteObject(state.font); }
    state.font = font;
    return 0;
}

int32_t hash_ui_accept_dpi(void *window) {
    RECT r = state.suggested;
    return r.right > r.left && r.bottom > r.top &&
        SetWindowPos((HWND)window, NULL, r.left, r.top, r.right-r.left, r.bottom-r.top, SWP_NOZORDER | SWP_NOACTIVATE) ? 0 : 1;
}

size_t hash_ui_dialog(void *window, uint32_t save, const uint8_t *initial, size_t initial_length, uint8_t *output, size_t capacity) {
    WCHAR path[32768] = {0};
    if ((HWND)window != state.window || save > 1 || !output || capacity > 131072 || capacity == 0 || initial_length > 32767) { return 131073; }
    if (initial_length) {
        int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)initial, (int)initial_length, path, 32767);
        if (!count) { return 131073; }
        path[count] = 0;
    }
    OPENFILENAMEW dialog = {0};
    dialog.lStructSize = sizeof(dialog); dialog.hwndOwner = (HWND)window;
    dialog.lpstrFile = path; dialog.nMaxFile = ARRAYSIZE(path);
    dialog.lpstrFilter = save ? L"JSON report (*.json)\0*.json\0All files\0*.*\0" : L"All files\0*.*\0";
    dialog.lpstrDefExt = save ? L"json" : NULL;
    dialog.Flags = OFN_EXPLORER | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR | OFN_DONTADDTORECENT |
        (save ? OFN_OVERWRITEPROMPT : OFN_FILEMUSTEXIST);
    BOOL selected = save ? GetSaveFileNameW(&dialog) : GetOpenFileNameW(&dialog);
    if (!selected) { return CommDlgExtendedError() == 0 ? 0 : 131073; }
    int count = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, path, -1, (char *)output, (int)capacity, NULL, NULL);
    return count > 0 ? count - 1 : 131073;
}

void hash_ui_show(void *window) {
    ShowWindow((HWND)window, SW_SHOWNORMAL);
    UpdateWindow((HWND)window);
    SetFocus(GetDlgItem((HWND)window, 202));
}

void hash_ui_destroy(void *window) {
    if (IsWindow((HWND)window)) { DestroyWindow((HWND)window); }
    if (state.font) { DeleteObject(state.font); state.font = NULL; }
    UnregisterClassW(L"FairyHashWindow", GetModuleHandleW(NULL));
    state.window = NULL;
}
