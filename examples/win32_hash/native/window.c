#define UNICODE
#define _UNICODE
#define COBJMACROS
#include <windows.h>
#include <commdlg.h>
#include <commctrl.h>
#include <stdint.h>
#include <shellapi.h>
#include <oleacc.h>
#include "native/ui_core.h"

// Win32 operations only. Fairy owns file policy, SHA-256 and report bytes.
static struct {
    HDROP dropped;
    HWND window, tooltip;
    WCHAR *tooltip_text;
    HFONT font, heading, mono;
    HICON icon_large, icon_small;
    HBRUSH background, paper;
    BOOL high_contrast, cancel_requested, compare_pending, compare_live;
    uint32_t tone, compare_tone;
    int event;
    int minimum_width, minimum_height;
    RECT suggested;
    IAccPropServices *accessibility;
    BOOL ole_initialized;
    uint32_t named_controls;
} state;

size_t hash_drop_read(void *payload, uint8_t *output, size_t capacity);

static COLORREF ink(void) { return state.high_contrast ? GetSysColor(COLOR_WINDOWTEXT) : RGB(36, 35, 32); }
static COLORREF canvas(void) { return state.high_contrast ? GetSysColor(COLOR_WINDOW) : RGB(246, 243, 237); }
static COLORREF muted(void) { return state.high_contrast ? ink() : RGB(100, 96, 87); }

static void palette(void) {
    HIGHCONTRASTW contrast = {sizeof(contrast)};
    state.high_contrast = SystemParametersInfoW(SPI_GETHIGHCONTRAST, sizeof(contrast), &contrast, 0) &&
        (contrast.dwFlags & HCF_HIGHCONTRASTON);
    HBRUSH background = CreateSolidBrush(canvas());
    HBRUSH paper = CreateSolidBrush(state.high_contrast ? canvas() : RGB(255, 255, 253));
    if (state.background) { DeleteObject(state.background); }
    if (state.paper) { DeleteObject(state.paper); }
    state.background = background; state.paper = paper;
}

static void draw_control(const DRAWITEMSTRUCT *item) {
    HDC dc = item->hDC;
    RECT r = item->rcItem;
    if (item->CtlID == 211) {
        FillRect(dc, &r, state.background);
        int w = r.right, h = r.bottom, fold = w / 3;
        HPEN pen = CreatePen(PS_SOLID, MulDiv(2, GetDpiForWindow(state.window), 96), muted());
        HGDIOBJ previous = SelectObject(dc, pen);
        POINT outline[] = {{2, 2}, {w-fold-2, 2}, {w-2, fold+2}, {w-2, h-2}, {2, h-2}, {2, 2}};
        POINT corner[] = {{w-fold-2, 2}, {w-fold-2, fold+2}, {w-2, fold+2}};
        Polyline(dc, outline, ARRAYSIZE(outline)); Polyline(dc, corner, ARRAYSIZE(corner));
        for (int row = 0; row < 3; row++) {
            MoveToEx(dc, w/4, h/2 + row*h/7, NULL); LineTo(dc, w*3/4, h/2 + row*h/7);
        }
        SelectObject(dc, previous); DeleteObject(pen);
        return;
    }
    BOOL disabled = (item->itemState & ODS_DISABLED) != 0;
    COLORREF fill = state.high_contrast ? GetSysColor(disabled ? COLOR_BTNFACE : COLOR_HIGHLIGHT) :
        disabled ? RGB(223, 220, 214) : (item->itemState & ODS_SELECTED) ? RGB(73, 71, 66) : RGB(46, 45, 41);
    HBRUSH brush = CreateSolidBrush(fill);
    FillRect(dc, &r, brush); DeleteObject(brush);
    SetBkMode(dc, TRANSPARENT);
    SetTextColor(dc, state.high_contrast ? GetSysColor(disabled ? COLOR_GRAYTEXT : COLOR_HIGHLIGHTTEXT) :
        disabled ? muted() : RGB(255, 255, 253));
    HGDIOBJ previous = SelectObject(dc, state.font);
    WCHAR text[64]; GetWindowTextW(item->hwndItem, text, ARRAYSIZE(text));
    DrawTextW(dc, text, -1, &r, DT_SINGLELINE | DT_CENTER | DT_VCENTER |
        ((item->itemState & ODS_NOACCEL) ? DT_HIDEPREFIX : 0));
    if (item->itemState & ODS_FOCUS) { InflateRect(&r, -4, -4); DrawFocusRect(dc, &r); }
    SelectObject(dc, previous);
}

static void dispatch(MSG *message) {
    // Native multiline EDIT can consume Tab/Enter. Forward raw navigation keys
    // before IsDialogMessage; Fairy chooses focus movement and action mapping.
    if (message->message == WM_KEYDOWN && state.event != 3 &&
        (message->hwnd == state.window || IsChild(state.window, message->hwnd)) &&
        !(GetKeyState(VK_CONTROL) & 0x8000) && !(GetKeyState(VK_MENU) & 0x8000)) {
        if (message->wParam == VK_TAB) {
            state.event = GetKeyState(VK_SHIFT) & 0x8000 ? 9 : 8; return;
        }
        if (message->wParam == VK_RETURN) { state.event = 10; return; }
    }
    if (!IsDialogMessageW(state.window, message)) {
        TranslateMessage(message);
        DispatchMessageW(message);
    }
}

static LPCWSTR tooltip_text(HWND control) {
    int length = GetWindowTextLengthW(control);
    if (length <= 0 || length > 131071) { return L""; }
    WCHAR *text = HeapAlloc(GetProcessHeap(), 0, ((size_t)length + 1) * sizeof(WCHAR));
    if (!text) { return L""; }
    int count = GetWindowTextW(control, text, length + 1);
    HDC dc = GetDC(control);
    HFONT font = (HFONT)SendMessageW(control, WM_GETFONT, 0, 0);
    HGDIOBJ previous = dc && font ? SelectObject(dc, font) : NULL;
    SIZE extent = {0}; RECT area = {0};
    BOOL measured = dc && GetClientRect(control, &area) && GetTextExtentPoint32W(dc, text, count, &extent);
    if (previous) { SelectObject(dc, previous); }
    if (dc) { ReleaseDC(control, dc); }
    int margins = GetDlgCtrlID(control) == 201 ? (int)SendMessageW(control, EM_GETMARGINS, 0, 0) : 0;
    if (!measured || extent.cx <= area.right - LOWORD(margins) - HIWORD(margins)) {
        HeapFree(GetProcessHeap(), 0, text);
        return L"";
    }
    if (state.tooltip_text) { HeapFree(GetProcessHeap(), 0, state.tooltip_text); }
    state.tooltip_text = text;
    return text;
}

static BOOL open_tooltip(HWND window) {
    state.tooltip = CreateWindowExW(WS_EX_TOPMOST, TOOLTIPS_CLASSW, NULL, WS_POPUP | TTS_ALWAYSTIP | TTS_NOPREFIX,
        CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, CW_USEDEFAULT, window, NULL, GetModuleHandleW(NULL), NULL);
    if (!state.tooltip) { return FALSE; }
    for (int index = 0; index < 2; index++) {
        TOOLINFOW tool = {sizeof(tool)};
        tool.uFlags = TTF_IDISHWND | TTF_SUBCLASS | TTF_TRANSPARENT;
        tool.hwnd = window;
        tool.uId = (UINT_PTR)GetDlgItem(window, index == 0 ? 206 : 201);
        tool.lpszText = LPSTR_TEXTCALLBACKW;
        if (!SendMessageW(state.tooltip, TTM_ADDTOOLW, 0, (LPARAM)&tool)) { return FALSE; }
    }
    SendMessageW(state.tooltip, TTM_SETDELAYTIME, TTDT_INITIAL, 400);
    SendMessageW(state.tooltip, TTM_SETDELAYTIME, TTDT_RESHOW, 100);
    SendMessageW(state.tooltip, TTM_SETDELAYTIME, TTDT_AUTOPOP, 20000);
    return TRUE;
}

static LRESULT CALLBACK procedure(HWND window, UINT message, WPARAM wp, LPARAM lp) {
    switch (message) {
    case WM_DROPFILES:
        if (state.event != 3 && !state.dropped && IsWindowEnabled(GetDlgItem(window,202))) {
            state.dropped = (HDROP)wp;
        } else { DragFinish((HDROP)wp); }
        return 0;
    case WM_NOTIFY: {
        NMTTDISPINFOW *info = (NMTTDISPINFOW *)lp;
        if (info && info->hdr.hwndFrom == state.tooltip && info->hdr.code == TTN_GETDISPINFOW &&
            (info->hdr.idFrom == (UINT_PTR)GetDlgItem(window, 206) || info->hdr.idFrom == (UINT_PTR)GetDlgItem(window, 201))) {
            // Callback storage survives this message and is not limited to the
            // notification's inline 80-character buffer. Never cache DI_SETITEM:
            // a new file or a new width must re-evaluate text and clipping.
            info->lpszText = (LPWSTR)tooltip_text((HWND)info->hdr.idFrom);
            return 0;
        }
        break;
    }
    case WM_SETTINGCHANGE:
    case WM_SYSCOLORCHANGE:
        palette(); InvalidateRect(window, NULL, TRUE);
        if (state.event == 0) { state.event = 4; }
        break;
    case WM_ERASEBKGND: {
        RECT r; GetClientRect(window, &r); FillRect((HDC)wp, &r, state.background);
        return 1;
    }
    case WM_CTLCOLORSTATIC:
    case WM_CTLCOLOREDIT: {
        int id = GetDlgCtrlID((HWND)lp);
        COLORREF color = id == 201 || id == 209 ? muted() : ink();
        if ((id == 205 || id == 216) && !state.high_contrast) {
            uint32_t tone = id == 205 ? state.tone : state.compare_tone;
            color = tone == 1 ? RGB(68, 113, 47) : tone == 2 ? RGB(167, 45, 35) : muted();
        }
        SetTextColor((HDC)wp, color);
        SetBkColor((HDC)wp, (id == 203 || id == 214) && !state.high_contrast ? RGB(255, 255, 253) : canvas());
        return (LRESULT)(id == 203 || id == 214 ? state.paper : state.background);
    }
    case WM_DRAWITEM:
        if (wp == 204 || wp == 211) { draw_control((const DRAWITEMSTRUCT *)lp); return TRUE; }
        break;
    case WM_COMMAND:
        if (HIWORD(wp) == EN_CHANGE && LOWORD(wp) == 214 && (HWND)lp == GetDlgItem(window,214)) {
            state.compare_pending = TRUE;
            return 0;
        }
        if (HIWORD(wp) == BN_CLICKED && (LOWORD(wp) == 202 || LOWORD(wp) == 204 || LOWORD(wp) == 208 || LOWORD(wp) == 212 || LOWORD(wp) == 217) &&
            (HWND)lp == GetDlgItem(window, LOWORD(wp)) && IsWindowEnabled((HWND)lp)) {
            if (LOWORD(wp) == 212) { state.cancel_requested = TRUE; }
            else if (state.event != 3) { state.event = LOWORD(wp) == 202 ? 1 : LOWORD(wp) == 204 ? 2 : LOWORD(wp) == 217 ? 14 : 6; }
            return 0;
        }
        break;
    case WM_SIZE:
        if (state.event == 0) { state.event = 4; }
        return 0;
    case WM_DPICHANGED:
        if (fairy_ui_dpi_suggestion(wp,lp,&state.suggested)) {
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
    return fairy_ui_system_dpi();
}

size_t hash_ui_ticks(void) { return (size_t)GetTickCount64(); }

int32_t hash_ui_progress(void *window, size_t value) {
    if ((HWND)window != state.window || value > 100) { return 1; }
    SendDlgItemMessageW((HWND)window, 213, PBM_SETPOS, value, 0);
    return 0;
}

static BOOL child(HWND window, LPCWSTR kind, DWORD style, int id) {
    return fairy_ui_create_child(window,kind,style,(uint32_t)id)!=NULL;
}

static BOOL window_icons(HWND window) {
    return fairy_ui_icons(window,&state.icon_large,&state.icon_small);
}

static void close_icons(void) {
    fairy_ui_close_icons(&state.icon_large,&state.icon_small);
}

void *hash_ui_open(int32_t width, int32_t height) {
    INITCOMMONCONTROLSEX controls = {sizeof(controls), ICC_STANDARD_CLASSES | ICC_BAR_CLASSES | ICC_PROGRESS_CLASS};
    if (!InitCommonControlsEx(&controls)) { return NULL; }
    palette();
    HWND window=fairy_ui_open(L"FairyHashWindow",procedure,width,height);
    if (window && child(window, L"STATIC", SS_NOPREFIX | SS_ENDELLIPSIS | SS_NOTIFY, 206) &&
        child(window, L"BUTTON", WS_TABSTOP | BS_DEFPUSHBUTTON, 202) &&
        child(window, L"EDIT", WS_TABSTOP | ES_READONLY | ES_AUTOHSCROLL, 201) &&
        child(window, L"STATIC", SS_NOPREFIX, 207) &&
        child(window, L"EDIT", WS_TABSTOP | WS_BORDER | ES_READONLY | ES_MULTILINE | ES_NOHIDESEL, 203) &&
        child(window, L"STATIC", 0, 215) &&
        child(window, L"EDIT", WS_TABSTOP | WS_BORDER | ES_AUTOHSCROLL, 214) &&
        child(window, L"BUTTON", WS_TABSTOP, 217) &&
        child(window, L"STATIC", SS_NOPREFIX, 216) &&
        child(window, L"BUTTON", WS_TABSTOP, 208) &&
        child(window, L"BUTTON", WS_TABSTOP | BS_OWNERDRAW, 204) &&
        child(window, L"STATIC", SS_NOPREFIX, 205) && child(window, L"STATIC", SS_NOPREFIX, 209) &&
        child(window, L"STATIC", SS_ETCHEDHORZ, 210) && child(window, L"STATIC", SS_OWNERDRAW, 211) &&
        child(window, PROGRESS_CLASSW, PBS_SMOOTH, 213) && child(window, L"BUTTON", WS_TABSTOP, 212)) {
        state.window = window;
        SendDlgItemMessageW(window, 213, PBM_SETRANGE32, 0, 100);
        SendDlgItemMessageW(window, 214, EM_SETLIMITTEXT, 256, 0);
        if (open_tooltip(window) && window_icons(window)) { DragAcceptFiles(window,TRUE); return window; }
    }
    if (window) { DestroyWindow(window); }
    close_icons();
    UnregisterClassW(L"FairyHashWindow",GetModuleHandleW(NULL));
    return NULL;
}

int32_t hash_ui_poll(void *window, int32_t wait) {
    if ((HWND)window != state.window) { return 3; }
    MSG message;
    // A deferred Close must still dispatch paint/input/sent messages while
    // Fairy waits for a save commit or owned cleanup. Never block on GetMessage.
    while (fairy_ui_should_pump(wait,state.event==3,state.event || state.cancel_requested || state.compare_pending || state.dropped)) {
        if (wait) {
            int code = (int)GetMessageW(&message, NULL, 0, 0);
            if (code <= 0) { state.event = 3; break; }
        } else {
            if (!PeekMessageW(&message, NULL, 0, 0, PM_REMOVE)) { break; }
            if (message.message == WM_QUIT) { state.event = 3; break; }
        }
        dispatch(&message);
    }
    // Closing stays sticky and takes priority. Cancelling consumes only its
    // own latch, retaining a pending size/DPI event for the next poll.
    int event = state.event == 3 ? 3 : state.cancel_requested ? 7 : state.event ? state.event : state.compare_pending ? 11 : state.dropped ? 12 : 0;
    if (event == 7) { state.cancel_requested = FALSE; }
    else if (event == 11) { state.compare_pending = FALSE; }
    else if (event != 3) { state.event = 0; }
    return event;
}

size_t hash_ui_dropped(void *window, uint8_t *output, size_t capacity) {
    if ((HWND)window != state.window) { return 131073; }
    HDROP drop = state.dropped; state.dropped = NULL;
    return hash_drop_read(drop,output,capacity);
}

int32_t hash_ui_focus_id(void *window) {
    HWND focus = GetFocus();
    return (HWND)window == state.window && focus && IsChild(state.window, focus) ? GetDlgCtrlID(focus) : 0;
}
int32_t hash_ui_next_focus(void *window, uint32_t reverse) {
    return (HWND)window==state.window?fairy_ui_next_focus((HWND)window,reverse):1;
}

int32_t hash_ui_text(void *window, uint32_t id, const uint8_t *text, size_t length) {
    if ((HWND)window!=state.window || (id!=0 && (id<201 || id>217)) || length>131071) { return 1; }
    if (id==201 || id==206) { SendMessageW(state.tooltip,TTM_POP,0,0); }
    return fairy_ui_text(id?GetDlgItem((HWND)window,(int)id):(HWND)window,text,length);
}

int32_t hash_ui_tone(void *window, uint32_t tone) {
    if ((HWND)window != state.window || tone > 2) { return 1; }
    state.tone = tone;
    InvalidateRect(GetDlgItem((HWND)window, 205), NULL, TRUE);
    return 0;
}

size_t hash_ui_expected(void *window, uint8_t *output, size_t capacity) {
    if ((HWND)window != state.window || !output || !capacity || capacity > 4096) { return 4097; }
    WCHAR text[257];
    int count = GetWindowTextW(GetDlgItem((HWND)window,214),text,ARRAYSIZE(text));
    if (!count) { return 0; }
    int written = WideCharToMultiByte(CP_UTF8,0,text,count,(char *)output,(int)capacity,NULL,NULL);
    return written ? (size_t)written : 4097;
}

int32_t hash_ui_compare_tone(void *window, uint32_t tone) {
    if ((HWND)window != state.window || tone > 2 || !state.accessibility) { return 1; }
    HWND label = GetDlgItem((HWND)window,216);
    if (!state.compare_live) {
        if (fairy_ui_live(state.accessibility,label,1,0)) { return 1; }
        state.compare_live = TRUE;
    }
    BOOL announce = tone != 0 && tone != state.compare_tone;
    state.compare_tone = tone;
    InvalidateRect(label,NULL,TRUE);
    if (announce) { NotifyWinEvent(EVENT_OBJECT_LIVEREGIONCHANGED,label,OBJID_CLIENT,CHILDID_SELF); }
    return 0;
}

int32_t hash_ui_live(void *window, uint32_t setting, uint32_t notify) {
    if ((HWND)window!=state.window) { return 1; }
    return fairy_ui_live(state.accessibility,GetDlgItem((HWND)window,205),setting,notify);
}

int32_t hash_ui_name(void *window, uint32_t id, const uint8_t *text, size_t length) {
    if ((HWND)window!=state.window || id<201 || id>217 || !length || length>131071) { return 1; }
    int32_t code=fairy_ui_name(&state.accessibility,&state.ole_initialized,GetDlgItem((HWND)window,(int)id),text,length);
    if (!code) { state.named_controls|=1u<<(id-201); }return code;
}

int32_t hash_ui_copy(void *window, const uint8_t *text, size_t length) {
    if ((HWND)window != state.window || !text || length > 131071) { return 1; }
    HGLOBAL data = GlobalAlloc(GMEM_MOVEABLE, (length + 1) * sizeof(WCHAR));
    if (!data) { return 1; }
    WCHAR *wide = GlobalLock(data);
    if (!wide) { GlobalFree(data); return 1; }
    int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)text, (int)length, wide, (int)length);
    wide[count] = 0; GlobalUnlock(data);
    if (!count || !OpenClipboard((HWND)window)) { GlobalFree(data); return 1; }
    BOOL success = EmptyClipboard() && SetClipboardData(CF_UNICODETEXT, data) != NULL;
    CloseClipboard();
    if (!success) { GlobalFree(data); }
    return success ? 0 : 1;
}

int32_t hash_ui_enable(void *window, uint32_t id, uint32_t enabled) {
    HWND control = GetDlgItem((HWND)window, (int)id);
    if ((HWND)window != state.window || !control || enabled > 1) { return 1; }
    if (fairy_ui_enable(control,enabled)) { return 1; }
    if (id == 202) {
        DragAcceptFiles((HWND)window,enabled != 0);
        if (!enabled && state.dropped) { DragFinish(state.dropped); state.dropped = NULL; }
    }
    return IsWindowEnabled(control) == (enabled != 0) ? 0 : 1;
}

int32_t hash_ui_visible(void *window, uint32_t id, uint32_t shown) {
    HWND control = GetDlgItem((HWND)window, (int)id);
    if ((HWND)window != state.window || !control || shown > 1) { return 1; }
    ShowWindow(control, shown ? SW_SHOWNA : SW_HIDE);
    // Check the child style even before Fairy shows the parent window.
    return !!(GetWindowLongW(control, GWL_STYLE) & WS_VISIBLE) == !!shown ? 0 : 1;
}

int32_t hash_ui_focus(void *window, uint32_t id) {
    return (HWND)window==state.window?fairy_ui_focus(GetDlgItem((HWND)window,(int)id)):1;
}

int32_t hash_ui_extent(void *window, uint32_t axis) {
    return (HWND)window==state.window?fairy_ui_extent((HWND)window,axis):-1;
}

int32_t hash_ui_move(void *window, uint32_t id, int32_t x, int32_t y, int32_t width, int32_t height) {
    if ((HWND)window != state.window || width < 0 || height < 0 || id < 201 || id > 217) { return 1; }
    HWND control = GetDlgItem((HWND)window, (int)id);
    // Fairy submits the entire layout before hash_ui_redraw erases and paints
    // all children. Do not paint intermediate, partially moved arrangements.
    if (fairy_ui_move(control,x,y,width,height)) { return 1; }
    if (id == 203) {
        int inset = MulDiv(10, GetDpiForWindow((HWND)window), 96);
        RECT area = {inset, inset, width-inset, height-inset};
        SendMessageW(control, EM_SETRECT, 0, (LPARAM)&area);
    }
    return 0;
}

int32_t hash_ui_redraw(void *window) {
    return (HWND)window==state.window?fairy_ui_redraw((HWND)window):1;
}

int32_t hash_ui_layout(void *window, int32_t minimum_width, int32_t minimum_height, int32_t font_height) {
    if ((HWND)window != state.window || minimum_width <= 0 || minimum_height <= 0 || font_height <= 0) { return 1; }
    RECT bounds = {0, 0, minimum_width, minimum_height};
    if (!AdjustWindowRectExForDpi(&bounds, WS_OVERLAPPEDWINDOW, FALSE, WS_EX_CONTROLPARENT, GetDpiForWindow((HWND)window))) { return 1; }
    state.minimum_width = bounds.right-bounds.left; state.minimum_height = bounds.bottom-bounds.top;
    SendMessageW(state.tooltip, TTM_POP, 0, 0);
    SendMessageW(state.tooltip, TTM_SETMAXTIPWIDTH, 0, MulDiv(600, GetDpiForWindow((HWND)window), 96));
    LOGFONTW current = {0};
    if (state.font && GetObjectW(state.font, sizeof(current), &current) && current.lfHeight == -font_height) { return 0; }
    HFONT font = CreateFontW(-font_height, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE, DEFAULT_CHARSET,
        OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, DEFAULT_PITCH, L"Segoe UI");
    HFONT heading = CreateFontW(-MulDiv(font_height, 18, 14), 0, 0, 0, FW_SEMIBOLD, FALSE, FALSE, FALSE, DEFAULT_CHARSET,
        OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, DEFAULT_PITCH, L"Segoe UI");
    HFONT mono = CreateFontW(-MulDiv(font_height, 16, 14), 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE, DEFAULT_CHARSET,
        OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS, CLEARTYPE_QUALITY, FIXED_PITCH, L"Consolas");
    if (!font || !heading || !mono) {
        if (font) { DeleteObject(font); } if (heading) { DeleteObject(heading); } if (mono) { DeleteObject(mono); }
        return 1;
    }
    for (int id = 201; id <= 217; id++) {
        HFONT selected = id == 203 || id == 214 ? mono : id == 206 ? heading : font;
        SendMessageW(GetDlgItem((HWND)window, id), WM_SETFONT, (WPARAM)selected, TRUE);
    }
    SendMessageW(state.tooltip, WM_SETFONT, (WPARAM)font, TRUE);
    if (state.font) { DeleteObject(state.font); }
    if (state.heading) { DeleteObject(state.heading); }
    if (state.mono) { DeleteObject(state.mono); }
    state.font = font; state.heading = heading; state.mono = mono;
    return 0;
}

int32_t hash_ui_accept_dpi(void *window) {
    return (HWND)window==state.window && fairy_ui_apply_dpi((HWND)window,&state.suggested) && window_icons((HWND)window)?0:1;
}

size_t hash_ui_dialog(void *window, uint32_t save, const uint8_t *initial, size_t initial_length, uint8_t *output, size_t capacity) {
    WCHAR path[32768] = {0};
    if ((HWND)window != state.window || save > 2 || !output || capacity > 131072 || capacity == 0 || initial_length > 32767) { return 131073; }
    if (initial_length) {
        int count = MultiByteToWideChar(CP_UTF8, MB_ERR_INVALID_CHARS, (const char *)initial, (int)initial_length, path, 32767);
        if (!count) { return 131073; }
        path[count] = 0;
    }
    OPENFILENAMEW dialog = {0};
    dialog.lStructSize = sizeof(dialog); dialog.hwndOwner = (HWND)window;
    dialog.lpstrFile = path; dialog.nMaxFile = ARRAYSIZE(path);
    dialog.lpstrFilter = save == 1 ? L"JSON / SHA-256 (*.json;*.sha256)\0*.json;*.sha256\0All files\0*.*\0" :
        save == 2 ? L"SHA-256 checksums (*.sha256)\0*.sha256\0All files\0*.*\0" : L"All files\0*.*\0";
    dialog.lpstrDefExt = save == 1 ? L"json" : NULL;
    dialog.Flags = OFN_EXPLORER | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR | OFN_DONTADDTORECENT |
        (save == 1 ? OFN_OVERWRITEPROMPT : OFN_FILEMUSTEXIST);
    BOOL selected = save == 1 ? GetSaveFileNameW(&dialog) : GetOpenFileNameW(&dialog);
    DWORD error = selected ? 0 : CommDlgExtendedError();
    // Native modal dialogs can leave the owner exposed at new bounds. Repair
    // its full child surface on accept and cancellation, before returning.
    if (hash_ui_redraw(window) != 0) { return 131073; }
    if (!selected) { return error == 0 ? 0 : 131073; }
    int count = WideCharToMultiByte(CP_UTF8, WC_ERR_INVALID_CHARS, path, -1, (char *)output, (int)capacity, NULL, NULL);
    return count > 0 ? count - 1 : 131073;
}

void hash_ui_show(void *window) {
    ShowWindow((HWND)window, SW_SHOWNORMAL);
    UpdateWindow((HWND)window);
    SetFocus(GetDlgItem((HWND)window, 202));
}

void hash_ui_destroy(void *window) {
    if (state.dropped) { DragFinish(state.dropped); state.dropped = NULL; }
    if (state.accessibility) {
        fairy_ui_clear(state.accessibility,GetDlgItem((HWND)window,216),FALSE,TRUE);
        state.compare_live = FALSE;
        fairy_ui_clear(state.accessibility,GetDlgItem((HWND)window,205),FALSE,TRUE);
        for (int id = 201; id <= 217; id++) {
            if (state.named_controls & (1u << (id-201))) {
                fairy_ui_clear(state.accessibility,GetDlgItem((HWND)window,id),TRUE,FALSE);
            }
        }
        IAccPropServices_Release(state.accessibility); state.accessibility = NULL;
        state.named_controls = 0;
    }
    if (IsWindow(state.tooltip)) { DestroyWindow(state.tooltip); }
    state.tooltip = NULL;
    if (state.tooltip_text) { HeapFree(GetProcessHeap(), 0, state.tooltip_text); state.tooltip_text = NULL; }
    if (IsWindow((HWND)window)) { DestroyWindow((HWND)window); }
    close_icons();
    if (state.font) { DeleteObject(state.font); state.font = NULL; }
    if (state.heading) { DeleteObject(state.heading); state.heading = NULL; }
    if (state.mono) { DeleteObject(state.mono); state.mono = NULL; }
    if (state.background) { DeleteObject(state.background); state.background = NULL; }
    if (state.paper) { DeleteObject(state.paper); state.paper = NULL; }
    UnregisterClassW(L"FairyHashWindow", GetModuleHandleW(NULL));
    state.window = NULL;
    if (state.ole_initialized) { CoUninitialize(); state.ole_initialized = FALSE; }
}
