#define UNICODE
#define _UNICODE
#include <windows.h>
#include <stdint.h>
#include <wchar.h>

enum { INPUT_ID = 201, APPLY_ID = 202, RESULT_ID = 203, STATUS_ID = 204, INPUT_LIMIT = 127 };

typedef struct {
    HWND input;
    HWND apply;
    HWND result;
    HWND status;
    unsigned edits;
    unsigned clicks;
} WindowState;

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

static LRESULT CALLBACK window_proc(HWND window, UINT message, WPARAM wparam, LPARAM lparam) {
    WindowState *state = (WindowState *)GetWindowLongPtrW(window, GWLP_USERDATA);
    switch (message) {
    case WM_NCCREATE:
        state = (WindowState *)((CREATESTRUCTW *)lparam)->lpCreateParams;
        SetWindowLongPtrW(window, GWLP_USERDATA, (LONG_PTR)state);
        return TRUE;
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
            state->edits++;
            resource_text(state->status, 105);
            return 0;
        }
        if (LOWORD(wparam) == APPLY_ID && HIWORD(wparam) == BN_CLICKED &&
            (HWND)lparam == state->apply) {
            WCHAR text[160];
            int prefix = LoadStringW(GetModuleHandleW(NULL), 107, text, (int)ARRAYSIZE(text));
            state->clicks++;
            if (prefix > 0 && GetWindowTextW(state->input, text + prefix,
                                            (int)ARRAYSIZE(text) - prefix) > 0) {
                SetWindowTextW(state->result, text);
                resource_text(state->status, 106);
            } else {
                resource_text(state->result, 108);
                resource_text(state->status, 109);
            }
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
    case WM_TIMER:
        DestroyWindow(window);
        return 0;
    case WM_DESTROY:
        PostQuitMessage(0);
        return 0;
    case WM_NCDESTROY:
        SetWindowLongPtrW(window, GWLP_USERDATA, 0);
        break;
    }
    return DefWindowProcW(window, message, wparam, lparam);
}

int32_t fairy_win32_run(void) {
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
        return 1;
    }
    WindowState state = {0};
    HWND window = CreateWindowExW(WS_EX_CONTROLPARENT, kind.lpszClassName, title, WS_OVERLAPPEDWINDOW,
                                 CW_USEDEFAULT, CW_USEDEFAULT, 480, 240,
                                 NULL, NULL, instance, &state);
    if (window == NULL) {
        return 2;
    }
    if (wcsstr(GetCommandLineW(), L"--smoke") != NULL && SetTimer(window, 1, 100, NULL) == 0) {
        DestroyWindow(window);
        return 3;
    }
    ShowWindow(window, SW_SHOWNORMAL);
    UpdateWindow(window);
    MSG message;
    int result;
    while ((result = (int)GetMessageW(&message, NULL, 0, 0)) > 0) {
        if (!IsDialogMessageW(window, &message)) {
            TranslateMessage(&message);
            DispatchMessageW(&message);
        }
    }
    return result < 0 ? 4 : (int32_t)message.wParam;
}
