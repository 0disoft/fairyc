#define UNICODE
#define _UNICODE
#define COBJMACROS
#include <windows.h>
#include <initguid.h>
#include <UIAutomationCore.h>
#include <UIAutomationCoreApi.h>
#include <string.h>
#include "native/ui_core.h"

// HWND ownership and product/event policy stay with the calling application.
BOOL fairy_ui_should_pump(BOOL wait,BOOL closing,BOOL pending) { return !pending || (!wait && closing); }
HWND fairy_ui_child(HWND window,uint32_t id) { return window && id && id<=65535?GetDlgItem(window,(int)id):NULL; }
HWND fairy_ui_open(LPCWSTR class_name,WNDPROC procedure,int32_t width,int32_t height) {
    if (!class_name || !procedure || width<=0 || height<=0) { return NULL; }
    WNDCLASSW type={0};type.lpfnWndProc=procedure;type.hInstance=GetModuleHandleW(NULL);
    type.hCursor=LoadCursorW(NULL,IDC_ARROW);type.lpszClassName=class_name;
    WCHAR title[128];RECT bounds={0,0,width,height};
    if (!LoadStringW(type.hInstance,101,title,128) || !RegisterClassW(&type)) { return NULL; }
    HWND window=NULL;
    if (AdjustWindowRectExForDpi(&bounds,WS_OVERLAPPEDWINDOW,FALSE,WS_EX_CONTROLPARENT,GetDpiForSystem())) {
        window=CreateWindowExW(WS_EX_CONTROLPARENT,class_name,title,WS_OVERLAPPEDWINDOW|WS_CLIPCHILDREN,
            CW_USEDEFAULT,CW_USEDEFAULT,bounds.right-bounds.left,bounds.bottom-bounds.top,NULL,NULL,type.hInstance,NULL);
    }
    if (!window) { UnregisterClassW(class_name,type.hInstance); }return window;
}
HWND fairy_ui_create_child(HWND window,LPCWSTR class_name,DWORD style,uint32_t id) {
    if (!window || !class_name || !id || id>65535) { return NULL; }
    return CreateWindowExW(0,class_name,L"",WS_CHILD|WS_VISIBLE|style,0,0,0,0,window,(HMENU)(INT_PTR)id,GetModuleHandleW(NULL),NULL);
}
int32_t fairy_ui_redraw(HWND window) {
    return window && RedrawWindow(window,NULL,NULL,RDW_INVALIDATE|RDW_ERASE|RDW_ALLCHILDREN|RDW_UPDATENOW)?0:1;
}
WCHAR *fairy_ui_wide(const uint8_t *text, size_t length) {
    if (length>131071 || (!text && length) || (length && memchr(text,0,length))) { return NULL; }
    WCHAR *wide=HeapAlloc(GetProcessHeap(),0,(length+1)*sizeof(WCHAR));
    if (!wide) { return NULL; }
    int count=length?MultiByteToWideChar(CP_UTF8,MB_ERR_INVALID_CHARS,(const char *)text,(int)length,wide,(int)length):0;
    if (length && !count) { HeapFree(GetProcessHeap(),0,wide);return NULL; }
    wide[count]=0;return wide;
}
int32_t fairy_ui_text(HWND control,const uint8_t *text,size_t length) {
    if (!control) { return 1; } WCHAR *wide=fairy_ui_wide(text,length);if (!wide) { return 1; }
    BOOL ok=SetWindowTextW(control,wide);HeapFree(GetProcessHeap(),0,wide);return ok?0:1;
}
size_t fairy_ui_read(HWND control,uint8_t *output,size_t capacity) {
    if (!control || !output || !capacity || capacity>131072) { return SIZE_MAX; }
    int length=GetWindowTextLengthW(control);if (length<0 || length>131071) { return SIZE_MAX; }
    if (!length) { return 0; }
    WCHAR *wide=HeapAlloc(GetProcessHeap(),0,((size_t)length+1)*sizeof(WCHAR));if (!wide) { return SIZE_MAX; }
    int count=GetWindowTextW(control,wide,length+1);
    int written=count==length?WideCharToMultiByte(CP_UTF8,WC_ERR_INVALID_CHARS,wide,count,(char *)output,(int)capacity,NULL,NULL):0;
    HeapFree(GetProcessHeap(),0,wide);return written?(size_t)written:SIZE_MAX;
}
int32_t fairy_ui_enable(HWND control,uint32_t enabled) {
    if (!control || enabled>1) { return 1; } EnableWindow(control,enabled!=0);return IsWindowEnabled(control)==(enabled!=0)?0:1;
}
int32_t fairy_ui_focus(HWND control) {
    if (!control || !IsWindowEnabled(control)) { return 1; } SetFocus(control);return GetFocus()==control?0:1;
}
int32_t fairy_ui_next_focus(HWND window,uint32_t reverse) {
    if (!window || reverse>1) { return 1; } return fairy_ui_focus(GetNextDlgTabItem(window,GetFocus(),reverse!=0));
}
int32_t fairy_ui_extent(HWND window,uint32_t axis) {
    RECT r;if (!window || axis>2 || !GetClientRect(window,&r)) { return -1; }
    return axis==0?r.right:axis==1?r.bottom:(int32_t)GetDpiForWindow(window);
}
int32_t fairy_ui_move(HWND control,int32_t x,int32_t y,int32_t width,int32_t height) {
    if (!control || width<0 || height<0) { return 1; }
    return SetWindowPos(control,NULL,x,y,width,height,SWP_NOZORDER|SWP_NOACTIVATE|SWP_NOREDRAW)?0:1;
}
int32_t fairy_ui_system_dpi(void) {
    if (!SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2) &&
        !AreDpiAwarenessContextsEqual(GetThreadDpiAwarenessContext(),DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)) { return -1; }
    return (int32_t)GetDpiForSystem();
}
BOOL fairy_ui_dpi_suggestion(WPARAM dpi,LPARAM rectangle,RECT *output) {
    if (!rectangle || !output || !LOWORD(dpi) || LOWORD(dpi)!=HIWORD(dpi)) { return FALSE; }
    RECT r=*(const RECT *)rectangle;if (r.right<=r.left || r.bottom<=r.top) { return FALSE; }*output=r;return TRUE;
}
BOOL fairy_ui_apply_dpi(HWND window,const RECT *r) {
    return window && r && r->right>r->left && r->bottom>r->top &&
        SetWindowPos(window,NULL,r->left,r->top,r->right-r->left,r->bottom-r->top,SWP_NOZORDER|SWP_NOACTIVATE);
}
void fairy_ui_close_icons(HICON *large_icon,HICON *small_icon) {
    if (*large_icon) { DestroyIcon(*large_icon);*large_icon=NULL; } if (*small_icon) { DestroyIcon(*small_icon);*small_icon=NULL; }
}
BOOL fairy_ui_icons(HWND window,HICON *large_icon,HICON *small_icon) {
    if (!window || !large_icon || !small_icon) { return FALSE; } UINT dpi=GetDpiForWindow(window);HINSTANCE instance=GetModuleHandleW(NULL);
    HICON next_large=LoadImageW(instance,MAKEINTRESOURCEW(1),IMAGE_ICON,GetSystemMetricsForDpi(SM_CXICON,dpi),GetSystemMetricsForDpi(SM_CYICON,dpi),0);
    HICON next_small=LoadImageW(instance,MAKEINTRESOURCEW(1),IMAGE_ICON,GetSystemMetricsForDpi(SM_CXSMICON,dpi),GetSystemMetricsForDpi(SM_CYSMICON,dpi),0);
    if (!next_large || !next_small) { fairy_ui_close_icons(&next_large,&next_small);return FALSE; }
    SendMessageW(window,WM_SETICON,ICON_BIG,(LPARAM)next_large);SendMessageW(window,WM_SETICON,ICON_SMALL,(LPARAM)next_small);
    fairy_ui_close_icons(large_icon,small_icon);*large_icon=next_large;*small_icon=next_small;return TRUE;
}
int32_t fairy_ui_name(IAccPropServices **service,BOOL *owned,HWND control,const uint8_t *text,size_t length) {
    if (!service || !owned || !control || !length) { return 1; }
    WCHAR *wide=fairy_ui_wide(text,length);if (!wide) { return 1; }
    if (!*service) {
        HRESULT status=CoInitializeEx(NULL,COINIT_APARTMENTTHREADED);
        if (FAILED(status) && status!=RPC_E_CHANGED_MODE) { HeapFree(GetProcessHeap(),0,wide);return 1; }
        *owned=SUCCEEDED(status);
        if (FAILED(CoCreateInstance(&CLSID_AccPropServices,NULL,CLSCTX_INPROC_SERVER,&IID_IAccPropServices,(void **)service))) {
            if (*owned) { CoUninitialize();*owned=FALSE; }HeapFree(GetProcessHeap(),0,wide);return 1;
        }
    }
    HRESULT status=IAccPropServices_SetHwndPropStr(*service,control,(DWORD)OBJID_CLIENT,CHILDID_SELF,PROPID_ACC_NAME,wide);
    HeapFree(GetProcessHeap(),0,wide);return SUCCEEDED(status)?0:1;
}
int32_t fairy_ui_live(IAccPropServices *service,HWND control,uint32_t setting,uint32_t notify) {
    if (!service || !control || setting>2 || notify>1) { return 1; }
    VARIANT value={0};value.vt=VT_I4;value.lVal=(LONG)setting;
    if (FAILED(IAccPropServices_SetHwndProp(service,control,(DWORD)OBJID_CLIENT,CHILDID_SELF,LiveSetting_Property_GUID,value))) { return 1; }
    if (notify && setting) { NotifyWinEvent(EVENT_OBJECT_LIVEREGIONCHANGED,control,OBJID_CLIENT,CHILDID_SELF); }return 0;
}
void fairy_ui_clear(IAccPropServices *service,HWND control,BOOL name,BOOL live) {
    if (!service || !control) { return; }
    if (name) { IAccPropServices_ClearHwndProps(service,control,(DWORD)OBJID_CLIENT,CHILDID_SELF,&PROPID_ACC_NAME,1); }
    if (live) { IAccPropServices_ClearHwndProps(service,control,(DWORD)OBJID_CLIENT,CHILDID_SELF,&LiveSetting_Property_GUID,1); }
}
