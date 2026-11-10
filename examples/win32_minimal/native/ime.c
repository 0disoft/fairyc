#define UNICODE
#define _UNICODE
#include <windows.h>
#include <imm.h>
#include "native/ime.h"
#include "native/profile.h"

static HWND edit;
static WNDPROC original;
#if FAIRY_WIN32_DIAGNOSTICS
static FairyIME observed;
#else
static struct { BOOL composing; } observed;
#endif
static BOOL boundary_pending, owns_key;

static LRESULT CALLBACK ime_proc(HWND window, UINT message, WPARAM wparam, LPARAM lparam) {
    WNDPROC previous = original;
    if (message == WM_IME_STARTCOMPOSITION) {
#if FAIRY_WIN32_DIAGNOSTICS
        observed.starts++;
#endif
        observed.composing = TRUE; }
#if FAIRY_WIN32_DIAGNOSTICS
    if (message == WM_IME_COMPOSITION) {
        observed.updates++;
        if (lparam & GCS_RESULTSTR) { observed.results++; }
    }
#endif
    if (message == WM_IME_ENDCOMPOSITION) {
#if FAIRY_WIN32_DIAGNOSTICS
        observed.ends++;
#endif
        observed.composing = FALSE; boundary_pending = TRUE;
    }
    if (message == WM_KILLFOCUS || message == WM_LBUTTONDOWN || message == WM_SETTEXT) { boundary_pending = FALSE; }
    LRESULT result = CallWindowProcW(previous, window, message, wparam, lparam);
    // The old edit procedure can end composition reentrantly during focus loss.
    if (message == WM_KILLFOCUS || message == WM_LBUTTONDOWN || message == WM_SETTEXT) { boundary_pending = FALSE; }
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

#if FAIRY_WIN32_DIAGNOSTICS
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

#endif

UINT fairy_ime_observe(const MSG *message) {
    owns_key = FALSE;
    if (message->hwnd != edit || message->message != WM_KEYDOWN) { return 0; }
#if FAIRY_WIN32_DIAGNOSTICS
    FairyIME state;
    fairy_ime_snapshot(&state);
    observed.keys++;
    observed.raw_key = (UINT)message->wParam;
    observed.key = message->wParam == VK_PROCESSKEY ? ImmGetVirtualKey(edit) : (UINT)message->wParam;
    observed.during_key = state.composing;
    UINT key = observed.key;
#else
    struct { BOOL composing; } state = {observed.composing};
    HIMC context = ImmGetContext(edit);
    if (context) {
        state.composing = state.composing || ImmGetCompositionStringW(context, GCS_COMPSTR, NULL, 0) > 0;
        ImmReleaseContext(edit, context);
    }
    UINT key = message->wParam == VK_PROCESSKEY ? ImmGetVirtualKey(edit) : (UINT)message->wParam;
#endif
    owns_key = state.composing || boundary_pending;
    boundary_pending = FALSE;
    return key;
}

BOOL fairy_ime_owns_key(void) { return owns_key; }
