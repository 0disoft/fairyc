#define UNICODE
#define _UNICODE
#include <windows.h>
#include <stdint.h>
#include <wchar.h>

static LRESULT CALLBACK window_proc(HWND window, UINT message, WPARAM wparam, LPARAM lparam) {
    switch (message) {
    case WM_PAINT: {
        PAINTSTRUCT paint;
        WCHAR text[128];
        HDC dc = BeginPaint(window, &paint);
        RECT area;
        GetClientRect(window, &area);
        if (LoadStringW(GetModuleHandleW(NULL), 102, text, 128) != 0) {
            DrawTextW(dc, text, -1, &area, DT_CENTER | DT_VCENTER | DT_SINGLELINE);
        }
        EndPaint(window, &paint);
        return 0;
    }
    case WM_TIMER:
        DestroyWindow(window);
        return 0;
    case WM_DESTROY:
        PostQuitMessage(0);
        return 0;
    default:
        return DefWindowProcW(window, message, wparam, lparam);
    }
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
    HWND window = CreateWindowExW(0, kind.lpszClassName, title, WS_OVERLAPPEDWINDOW,
                                 CW_USEDEFAULT, CW_USEDEFAULT, 480, 240,
                                 NULL, NULL, instance, NULL);
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
        TranslateMessage(&message);
        DispatchMessageW(&message);
    }
    return result < 0 ? 4 : (int32_t)message.wParam;
}
