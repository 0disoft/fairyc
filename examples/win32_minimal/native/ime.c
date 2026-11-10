#define UNICODE
#define _UNICODE
#include <windows.h>
#include <imm.h>
#include "native/ime.h"

static HWND edit;
static WNDPROC original;
static FairyIME observed;

static LRESULT CALLBACK ime_proc(HWND window, UINT message, WPARAM wparam, LPARAM lparam) {
    WNDPROC previous = original;
    if (message == WM_IME_STARTCOMPOSITION) { observed.starts++; observed.composing = TRUE; }
    if (message == WM_IME_COMPOSITION) {
        observed.updates++;
        if (lparam & GCS_RESULTSTR) { observed.results++; }
    }
    if (message == WM_IME_ENDCOMPOSITION) { observed.ends++; observed.composing = FALSE; }
    LRESULT result = CallWindowProcW(previous, window, message, wparam, lparam);
    if (message == WM_NCDESTROY) { edit = NULL; original = NULL; }
    return result;
}

BOOL fairy_ime_attach(HWND input) {
    if (edit != NULL || input == NULL) { return FALSE; }
    SetLastError(0);
    WNDPROC previous = (WNDPROC)SetWindowLongPtrW(input, GWLP_WNDPROC, (LONG_PTR)ime_proc);
    if (previous == NULL) { return FALSE; }
    edit = input;
    original = previous;
    return TRUE;
}

void fairy_ime_snapshot(FairyIME *state) {
    *state = observed;
    state->language = LOWORD((UINT_PTR)GetKeyboardLayout(0));
    state->composition[0] = L'\0';
    HIMC context = edit ? ImmGetContext(edit) : NULL;
    if (!context) { return; }
    DWORD sentence = 0;
    state->open = ImmGetOpenStatus(context);
    ImmGetConversionStatus(context, &state->conversion, &sentence);
    LONG bytes = ImmGetCompositionStringW(context, GCS_COMPSTR, NULL, 0);
    if (bytes > 0 && bytes < (LONG)sizeof(state->composition) && bytes % sizeof(WCHAR) == 0) {
        if (ImmGetCompositionStringW(context, GCS_COMPSTR, state->composition, (DWORD)bytes) == bytes) {
            state->composition[bytes / sizeof(WCHAR)] = L'\0';
            state->composing = TRUE;
        }
    }
    ImmReleaseContext(edit, context);
}

UINT fairy_ime_observe(const MSG *message) {
    if (message->hwnd != edit || message->message != WM_KEYDOWN) { return 0; }
    FairyIME state;
    fairy_ime_snapshot(&state);
    observed.keys++;
    observed.raw_key = (UINT)message->wParam;
    observed.key = message->wParam == VK_PROCESSKEY ? ImmGetVirtualKey(edit) : (UINT)message->wParam;
    observed.during_key = state.composing;
    return observed.key;
}
