#define UNICODE
#define _UNICODE
#define COBJMACROS
#include <windows.h>
#include <commctrl.h>
#include <commdlg.h>
#include <stdint.h>
#include <oleacc.h>
#include "native/ime.h"
#include "native/ui_core.h"

// Private starter transport. Text, actions, key policy and layout belong to Fairy.
static struct {
    HWND window;
    HFONT font;
    HICON icon_large, icon_small;
    HBRUSH background, paper;
    BOOL contrast, closed, resized, dpi_pending, com_owned;
    int clicked, changed, key, flags, last_id, last_key, last_flags;
    int min_width, min_height, font_height;
    RECT suggested;
    IAccPropServices *accessibility;
    uint32_t named;
} state;

static void palette(void) {
    HIGHCONTRASTW contrast = {sizeof(contrast)};
    state.contrast = SystemParametersInfoW(SPI_GETHIGHCONTRAST,sizeof(contrast),&contrast,0) && (contrast.dwFlags&HCF_HIGHCONTRASTON);
    HBRUSH background=CreateSolidBrush(state.contrast?GetSysColor(COLOR_WINDOW):RGB(247,245,239));
    HBRUSH paper=CreateSolidBrush(GetSysColor(COLOR_WINDOW));
    if (state.background) { DeleteObject(state.background); }
    if (state.paper) { DeleteObject(state.paper); }
    state.background=background; state.paper=paper;
}
static BOOL icons(HWND window) {
    return fairy_ui_icons(window,&state.icon_large,&state.icon_small);
}
static LRESULT CALLBACK procedure(HWND window,UINT message,WPARAM wp,LPARAM lp) {
    switch (message) {
    case WM_COMMAND:
        if ((HWND)lp && (HWND)lp==GetDlgItem(window,LOWORD(wp))) {
            if (HIWORD(wp)==BN_CLICKED) { state.clicked=LOWORD(wp); }
            if (HIWORD(wp)==EN_CHANGE) { state.changed=LOWORD(wp); }
        }
        return 0;
    case WM_SIZE: state.resized=TRUE; return 0;
    case WM_DPICHANGED:
        if (fairy_ui_dpi_suggestion(wp,lp,&state.suggested)) { state.dpi_pending=TRUE; }
        return 0;
    case WM_GETMINMAXINFO:
        ((MINMAXINFO *)lp)->ptMinTrackSize.x=state.min_width;
        ((MINMAXINFO *)lp)->ptMinTrackSize.y=state.min_height; return 0;
    case WM_SETTINGCHANGE: case WM_SYSCOLORCHANGE: palette(); state.resized=TRUE; return 0;
    case WM_ERASEBKGND: { RECT r; GetClientRect(window,&r); FillRect((HDC)wp,&r,state.background); return 1; }
    case WM_CTLCOLORSTATIC: case WM_CTLCOLOREDIT: {
        HWND child=(HWND)lp; WCHAR kind[16]; GetClassNameW(child,kind,16);
        BOOL edit=lstrcmpW(kind,L"Edit")==0;
        SetTextColor((HDC)wp,GetSysColor(COLOR_WINDOWTEXT));
        SetBkColor((HDC)wp,edit||state.contrast?GetSysColor(COLOR_WINDOW):RGB(247,245,239));
        return (LRESULT)(edit?state.paper:state.background);
    }
    case WM_CLOSE: state.closed=TRUE; return 0;
    case WM_DESTROY: PostQuitMessage(0); return 0;
    }
    return DefWindowProcW(window,message,wp,lp);
}
int32_t desktop_system_dpi(void) {
    return fairy_ui_system_dpi();
}
void *desktop_open(int32_t width,int32_t height) {
    INITCOMMONCONTROLSEX controls={sizeof(controls),ICC_STANDARD_CLASSES|ICC_PROGRESS_CLASS};
    if (!InitCommonControlsEx(&controls)) { return NULL; } palette();
    state.window=fairy_ui_open(L"FairyDesktopStarter",procedure,width,height);
    if (state.window && !icons(state.window)) { DestroyWindow(state.window);state.window=NULL;UnregisterClassW(L"FairyDesktopStarter",GetModuleHandleW(NULL)); }
    return state.window;
}
int32_t desktop_control(void *window,uint32_t kind,uint32_t id) {
    if ((HWND)window!=state.window || id<101 || id>132 || kind<1 || kind>5) { return 1; }
    DWORD style=WS_CHILD|WS_VISIBLE;
    if (kind==1 || kind==4) { style|=WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL; }
    if (kind==2) { style|=WS_TABSTOP; if (id==103) { style|=BS_DEFPUSHBUTTON; } }
    if (kind==4) { style|=ES_READONLY|ES_MULTILINE; }
    HWND control=fairy_ui_create_child((HWND)window,kind==5?PROGRESS_CLASSW:kind==1||kind==4?L"EDIT":kind==2?L"BUTTON":L"STATIC",style,id);
    if (!control) { return 1; }
    if (kind==5) { SendMessageW(control,PBM_SETRANGE32,0,100); }
    if (kind==1) { SendMessageW(control,EM_SETLIMITTEXT,511,0); return fairy_ime_attach(control)?0:1; }
    return 0;
}
int32_t desktop_text(void *window,uint32_t id,const uint8_t *text,size_t length) {
    if ((HWND)window!=state.window || id<101 || id>132 || length>4095) { return 1; }
    return fairy_ui_text(GetDlgItem((HWND)window,id),text,length);
}
size_t desktop_read(void *window,uint32_t id,uint8_t *output,size_t capacity) {
    if ((HWND)window!=state.window || id<101 || id>132 || !capacity || capacity>4096) { return 4097; }
    size_t count=fairy_ui_read(GetDlgItem((HWND)window,id),output,capacity);return count==SIZE_MAX?4097:count;
}
int32_t desktop_name(void *window,uint32_t id,const uint8_t *text,size_t length) {
    if ((HWND)window!=state.window || id<101 || id>132 || !length || length>4095) { return 1; }
    int32_t code=fairy_ui_name(&state.accessibility,&state.com_owned,GetDlgItem((HWND)window,id),text,length);
    if (!code) { state.named|=1u<<(id-101); }return code;
}
int32_t desktop_live(void *window,uint32_t id,uint32_t setting,uint32_t notify) {
    if ((HWND)window!=state.window || id<101 || id>132) { return 1; }
    return fairy_ui_live(state.accessibility,GetDlgItem((HWND)window,id),setting,notify);
}
int32_t desktop_announce(void *window,uint32_t id) { return desktop_live(window,id,1,1); }
static int32_t poll_messages(void *window,BOOL wait) {
    if ((HWND)window!=state.window) { return 3; }
    // A sticky Close must keep draining messages during save commit/cleanup.
    while (fairy_ui_should_pump(wait,state.closed,state.closed || state.resized || state.dpi_pending || state.key || state.clicked || state.changed)) {
        MSG message;
        if (wait) { if (GetMessageW(&message,NULL,0,0)<=0) { state.closed=TRUE; break; } }
        else { if (!PeekMessageW(&message,NULL,0,0,PM_REMOVE)) { break; } if (message.message==WM_QUIT) { state.closed=TRUE; break; } }
        UINT key=fairy_ime_observe(&message);
        if (message.message==WM_KEYDOWN && (message.hwnd==state.window || IsChild(state.window,message.hwnd)) &&
            !fairy_ime_owns_key() && !(GetKeyState(VK_CONTROL)&0x8000) && !(GetKeyState(VK_MENU)&0x8000)) {
            if (!key) { key=(UINT)message.wParam; }
            if (key==VK_TAB || key==VK_RETURN || key==VK_ESCAPE) { state.key=(int)key; state.flags=(GetKeyState(VK_SHIFT)&0x8000)?1:0; continue; }
        }
        if (fairy_ime_owns_key() || !IsDialogMessageW(state.window,&message)) { TranslateMessage(&message); DispatchMessageW(&message); }
    }
    if (state.closed) { return 3; }
    if (state.dpi_pending) { state.dpi_pending=FALSE; return 5; }
    if (state.resized) { state.resized=FALSE; return 4; }
    if (state.key) { state.last_key=state.key; state.last_flags=state.flags; state.key=0; return 6; }
    if (state.clicked) { state.last_id=state.clicked; state.clicked=0; return 1; }
    if (state.changed) { state.last_id=state.changed; state.changed=0; return 2; }
    return 0;
}
int32_t desktop_poll(void *window) { return poll_messages(window,TRUE); }
int32_t desktop_poll_now(void *window) { return poll_messages(window,FALSE); }
int32_t desktop_field(void *window,uint32_t axis) {
    if ((HWND)window!=state.window) { return -1; }
    return axis==0?state.last_id:axis==1?state.last_key:axis==2?state.last_flags:axis==3?GetDlgCtrlID(GetFocus()):-1;
}
int32_t desktop_enable(void *window,uint32_t id,uint32_t enabled) {
    if ((HWND)window!=state.window || id<101 || id>132) { return 1; }
    return fairy_ui_enable(GetDlgItem((HWND)window,id),enabled);
}
int32_t desktop_focus_to(void *window,uint32_t id) {
    if ((HWND)window!=state.window || id<101 || id>132) { return 1; }
    return fairy_ui_focus(GetDlgItem((HWND)window,id));
}
int32_t desktop_progress(void *window,size_t percent) {
    if ((HWND)window!=state.window || percent>100) { return 1; }
    SendDlgItemMessageW((HWND)window,108,PBM_SETPOS,percent,0);return 0;
}
int32_t desktop_extent(void *window,uint32_t axis) {
    return (HWND)window==state.window?fairy_ui_extent((HWND)window,axis):-1;
}
int32_t desktop_apply_dpi(void *window) {
    return (HWND)window==state.window && fairy_ui_apply_dpi((HWND)window,&state.suggested) && icons((HWND)window)?0:1;
}
int32_t desktop_frame(void *window,int32_t width,int32_t height,int32_t font_height) {
    if ((HWND)window!=state.window || width<=0 || height<=0 || font_height<=0) { return 1; }
    RECT r={0,0,width,height};
    if (!AdjustWindowRectExForDpi(&r,WS_OVERLAPPEDWINDOW,FALSE,WS_EX_CONTROLPARENT,GetDpiForWindow((HWND)window))) { return 1; }
    state.min_width=r.right-r.left; state.min_height=r.bottom-r.top;
    if (state.font_height!=font_height) {
        HFONT font=CreateFontW(-font_height,0,0,0,FW_NORMAL,FALSE,FALSE,FALSE,DEFAULT_CHARSET,
            OUT_DEFAULT_PRECIS,CLIP_DEFAULT_PRECIS,CLEARTYPE_QUALITY,DEFAULT_PITCH,L"Segoe UI");
        if (!font) { return 1; }
        for (int id=101;id<=132;id++) { HWND child=GetDlgItem((HWND)window,id); if (child) { SendMessageW(child,WM_SETFONT,(WPARAM)font,FALSE); } }
        if (state.font) { DeleteObject(state.font); } state.font=font; state.font_height=font_height;
    }
    return 0;
}
int32_t desktop_move(void *window,uint32_t id,int32_t x,int32_t y,int32_t width,int32_t height) {
    if ((HWND)window!=state.window || id<101 || id>132) { return 1; }
    return fairy_ui_move(GetDlgItem((HWND)window,id),x,y,width,height);
}
int32_t desktop_redraw(void *window) {
    return (HWND)window==state.window?fairy_ui_redraw((HWND)window):1;
}
static size_t choose_file(void *window,uint8_t *output,size_t capacity,BOOL saving) {
    if ((HWND)window!=state.window || !output || !capacity || capacity>131072) { return 131073; }
    WCHAR path[32768]={0};OPENFILENAMEW dialog={0};dialog.lStructSize=sizeof(dialog);dialog.hwndOwner=(HWND)window;
    dialog.lpstrFile=path;dialog.nMaxFile=ARRAYSIZE(path);dialog.lpstrFilter=L"All files\0*.*\0";
    dialog.Flags=OFN_EXPLORER|OFN_PATHMUSTEXIST|OFN_NOCHANGEDIR|OFN_DONTADDTORECENT|(saving?OFN_OVERWRITEPROMPT:OFN_FILEMUSTEXIST);
    BOOL selected=saving?GetSaveFileNameW(&dialog):GetOpenFileNameW(&dialog);DWORD error=selected?0:CommDlgExtendedError();
    if (desktop_redraw(window)) { return 131073; } if (!selected) { return error?131073:0; }
    int count=WideCharToMultiByte(CP_UTF8,WC_ERR_INVALID_CHARS,path,-1,(char *)output,(int)capacity,NULL,NULL);
    return count>0?(size_t)count-1:131073;
}
size_t desktop_choose(void *window,uint8_t *output,size_t capacity) { return choose_file(window,output,capacity,FALSE); }
size_t desktop_choose_save(void *window,uint8_t *output,size_t capacity) { return choose_file(window,output,capacity,TRUE); }
int32_t desktop_focus_next(void *window,uint32_t reverse) {
    return (HWND)window==state.window?fairy_ui_next_focus((HWND)window,reverse):1;
}
void desktop_show(void *window) {
    ShowWindow((HWND)window,SW_SHOWNORMAL); UpdateWindow((HWND)window); SetFocus(GetDlgItem((HWND)window,102));
}
void desktop_destroy(void *window) {
    if (state.accessibility) {
        for (int id=101;id<=132;id++) {
            HWND child=GetDlgItem((HWND)window,id); if (!child) { continue; }
            fairy_ui_clear(state.accessibility,child,(state.named&(1u<<(id-101)))!=0,TRUE);
        }
        IAccPropServices_Release(state.accessibility); state.accessibility=NULL;
    }
    if (window) { DestroyWindow((HWND)window); } if (state.font) { DeleteObject(state.font); }
    fairy_ui_close_icons(&state.icon_large,&state.icon_small);
    if (state.background) { DeleteObject(state.background); } if (state.paper) { DeleteObject(state.paper); }
    if (state.com_owned) { CoUninitialize(); }
    UnregisterClassW(L"FairyDesktopStarter",GetModuleHandleW(NULL)); state.window=NULL;
}
