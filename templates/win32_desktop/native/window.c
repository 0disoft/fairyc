#define UNICODE
#define _UNICODE
#define COBJMACROS
#include <windows.h>
#include <commctrl.h>
#include <stdint.h>
#include <initguid.h>
#include <oleacc.h>
#include <UIAutomationCore.h>
#include <UIAutomationCoreApi.h>
#include "native/ime.h"

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
    UINT dpi=GetDpiForWindow(window); HINSTANCE instance=GetModuleHandleW(NULL);
    HICON icon_large=LoadImageW(instance,MAKEINTRESOURCEW(1),IMAGE_ICON,
        GetSystemMetricsForDpi(SM_CXICON,dpi),GetSystemMetricsForDpi(SM_CYICON,dpi),0);
    HICON icon_small=LoadImageW(instance,MAKEINTRESOURCEW(1),IMAGE_ICON,
        GetSystemMetricsForDpi(SM_CXSMICON,dpi),GetSystemMetricsForDpi(SM_CYSMICON,dpi),0);
    if (!icon_large || !icon_small) { if (icon_large) { DestroyIcon(icon_large); } if (icon_small) { DestroyIcon(icon_small); } return FALSE; }
    SendMessageW(window,WM_SETICON,ICON_BIG,(LPARAM)icon_large); SendMessageW(window,WM_SETICON,ICON_SMALL,(LPARAM)icon_small);
    if (state.icon_large) { DestroyIcon(state.icon_large); } if (state.icon_small) { DestroyIcon(state.icon_small); }
    state.icon_large=icon_large; state.icon_small=icon_small; return TRUE;
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
        if (lp && LOWORD(wp)==HIWORD(wp)) { state.suggested=*(RECT *)lp; state.dpi_pending=TRUE; }
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
    if (!SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2) &&
        !AreDpiAwarenessContextsEqual(GetThreadDpiAwarenessContext(),DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)) { return -1; }
    return (int32_t)GetDpiForSystem();
}
void *desktop_open(int32_t width,int32_t height) {
    INITCOMMONCONTROLSEX controls={sizeof(controls),ICC_STANDARD_CLASSES};
    if (!InitCommonControlsEx(&controls)) { return NULL; } palette();
    WNDCLASSW type={0}; type.lpfnWndProc=procedure; type.hInstance=GetModuleHandleW(NULL);
    type.hCursor=LoadCursorW(NULL,IDC_ARROW); type.lpszClassName=L"FairyDesktopStarter";
    WCHAR title[128]; RECT bounds={0,0,width,height};
    if (!LoadStringW(type.hInstance,101,title,128) || !RegisterClassW(&type) ||
        !AdjustWindowRectExForDpi(&bounds,WS_OVERLAPPEDWINDOW,FALSE,WS_EX_CONTROLPARENT,GetDpiForSystem())) { return NULL; }
    state.window=CreateWindowExW(WS_EX_CONTROLPARENT,type.lpszClassName,title,WS_OVERLAPPEDWINDOW|WS_CLIPCHILDREN,
        CW_USEDEFAULT,CW_USEDEFAULT,bounds.right-bounds.left,bounds.bottom-bounds.top,NULL,NULL,type.hInstance,NULL);
    if (state.window && !icons(state.window)) { DestroyWindow(state.window); state.window=NULL; }
    return state.window;
}
int32_t desktop_control(void *window,uint32_t kind,uint32_t id) {
    if ((HWND)window!=state.window || id<101 || id>132 || kind<1 || kind>4) { return 1; }
    DWORD style=WS_CHILD|WS_VISIBLE;
    if (kind==1 || kind==4) { style|=WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL; }
    if (kind==2) { style|=WS_TABSTOP|BS_DEFPUSHBUTTON; }
    if (kind==4) { style|=ES_READONLY|ES_MULTILINE; }
    HWND control=CreateWindowExW(0,kind==1||kind==4?L"EDIT":kind==2?L"BUTTON":L"STATIC",L"",style,
        0,0,0,0,(HWND)window,(HMENU)(INT_PTR)id,GetModuleHandleW(NULL),NULL);
    if (!control) { return 1; }
    if (kind==1) { SendMessageW(control,EM_SETLIMITTEXT,511,0); return fairy_ime_attach(control)?0:1; }
    return 0;
}
int32_t desktop_text(void *window,uint32_t id,const uint8_t *text,size_t length) {
    if ((HWND)window!=state.window || id<101 || id>132 || length>4095) { return 1; }
    WCHAR wide[4096]; int count=length?MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,(const char *)text,(int)length,wide,4095):0;
    if (length && !count) { return 1; } wide[count]=0;
    return SetWindowTextW(GetDlgItem((HWND)window,id),wide)?0:1;
}
size_t desktop_read(void *window,uint32_t id,uint8_t *output,size_t capacity) {
    if ((HWND)window!=state.window || id<101 || id>132 || !output || capacity>4096 || !capacity) { return 4097; }
    WCHAR wide[1024]; HWND control=GetDlgItem((HWND)window,id);
    int length=GetWindowTextLengthW(control); if (length>=1024 || length<0) { return 4097; }
    if (!length) { return 0; }
    if (GetWindowTextW(control,wide,1024)!=length) { return 4097; }
    int count=WideCharToMultiByte(CP_UTF8,WC_ERR_INVALID_CHARS,wide,length,(char *)output,(int)capacity,NULL,NULL);
    return count?(size_t)count:4097;
}
int32_t desktop_name(void *window,uint32_t id,const uint8_t *text,size_t length) {
    if ((HWND)window!=state.window || id<101 || id>132 || !length || length>4095) { return 1; }
    if (!state.accessibility) {
        HRESULT initialized=CoInitializeEx(NULL,COINIT_APARTMENTTHREADED);
        if (FAILED(initialized) && initialized!=RPC_E_CHANGED_MODE) { return 1; }
        state.com_owned=SUCCEEDED(initialized);
        if (FAILED(CoCreateInstance(&CLSID_AccPropServices,NULL,CLSCTX_INPROC_SERVER,&IID_IAccPropServices,(void **)&state.accessibility))) { return 1; }
    }
    WCHAR wide[4096]; int count=MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,(const char *)text,(int)length,wide,4095);
    if (!count) { return 1; } wide[count]=0;
    if (FAILED(IAccPropServices_SetHwndPropStr(state.accessibility,GetDlgItem((HWND)window,id),(DWORD)OBJID_CLIENT,CHILDID_SELF,PROPID_ACC_NAME,wide))) { return 1; }
    state.named|=1u<<(id-101); return 0;
}
int32_t desktop_announce(void *window,uint32_t id) {
    if ((HWND)window!=state.window || !state.accessibility || id<101 || id>132) { return 1; }
    VARIANT value={0}; value.vt=VT_I4; value.lVal=1; HWND label=GetDlgItem((HWND)window,id);
    if (FAILED(IAccPropServices_SetHwndProp(state.accessibility,label,(DWORD)OBJID_CLIENT,CHILDID_SELF,LiveSetting_Property_GUID,value))) { return 1; }
    NotifyWinEvent(EVENT_OBJECT_LIVEREGIONCHANGED,label,OBJID_CLIENT,CHILDID_SELF); return 0;
}
int32_t desktop_poll(void *window) {
    if ((HWND)window!=state.window) { return 3; }
    while (!state.closed && !state.resized && !state.dpi_pending && !state.key && !state.clicked && !state.changed) {
        MSG message; int status=(int)GetMessageW(&message,NULL,0,0);
        if (status<=0) { state.closed=TRUE; break; }
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
int32_t desktop_field(void *window,uint32_t axis) {
    if ((HWND)window!=state.window) { return -1; }
    return axis==0?state.last_id:axis==1?state.last_key:axis==2?state.last_flags:-1;
}
int32_t desktop_extent(void *window,uint32_t axis) {
    RECT r; if ((HWND)window!=state.window || !GetClientRect((HWND)window,&r)) { return -1; }
    return axis==0?r.right:axis==1?r.bottom:axis==2?(int32_t)GetDpiForWindow((HWND)window):-1;
}
int32_t desktop_apply_dpi(void *window) {
    RECT r=state.suggested;
    return (HWND)window==state.window && r.right>r.left && r.bottom>r.top &&
        SetWindowPos((HWND)window,NULL,r.left,r.top,r.right-r.left,r.bottom-r.top,SWP_NOZORDER|SWP_NOACTIVATE) && icons((HWND)window)?0:1;
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
    if ((HWND)window!=state.window || id<101 || id>132 || width<0 || height<0) { return 1; }
    return SetWindowPos(GetDlgItem((HWND)window,id),NULL,x,y,width,height,SWP_NOZORDER|SWP_NOACTIVATE|SWP_NOREDRAW)?0:1;
}
int32_t desktop_redraw(void *window) {
    return (HWND)window==state.window && RedrawWindow((HWND)window,NULL,NULL,RDW_INVALIDATE|RDW_ERASE|RDW_ALLCHILDREN|RDW_UPDATENOW)?0:1;
}
int32_t desktop_focus_next(void *window,uint32_t reverse) {
    if ((HWND)window!=state.window || reverse>1) { return 1; }
    HWND next=GetNextDlgTabItem((HWND)window,GetFocus(),reverse!=0); if (!next) { return 1; }
    SetFocus(next); return GetFocus()==next?0:1;
}
void desktop_show(void *window) {
    ShowWindow((HWND)window,SW_SHOWNORMAL); UpdateWindow((HWND)window); SetFocus(GetDlgItem((HWND)window,102));
}
void desktop_destroy(void *window) {
    if (state.accessibility) {
        for (int id=101;id<=132;id++) {
            HWND child=GetDlgItem((HWND)window,id); if (!child) { continue; }
            if (state.named&(1u<<(id-101))) { IAccPropServices_ClearHwndProps(state.accessibility,child,(DWORD)OBJID_CLIENT,CHILDID_SELF,&PROPID_ACC_NAME,1); }
            IAccPropServices_ClearHwndProps(state.accessibility,child,(DWORD)OBJID_CLIENT,CHILDID_SELF,&LiveSetting_Property_GUID,1);
        }
        IAccPropServices_Release(state.accessibility); state.accessibility=NULL;
    }
    if (window) { DestroyWindow((HWND)window); } if (state.font) { DeleteObject(state.font); }
    if (state.icon_large) { DestroyIcon(state.icon_large); } if (state.icon_small) { DestroyIcon(state.icon_small); }
    if (state.background) { DeleteObject(state.background); } if (state.paper) { DeleteObject(state.paper); }
    if (state.com_owned) { CoUninitialize(); }
    UnregisterClassW(L"FairyDesktopStarter",GetModuleHandleW(NULL)); state.window=NULL;
}
