#ifndef FAIRY_UI_CORE_H
#define FAIRY_UI_CORE_H
#include <windows.h>
#include <stdint.h>
#include <stddef.h>
#include <oleacc.h>

WCHAR *fairy_ui_wide(const uint8_t *text, size_t length);
HWND fairy_ui_child(HWND window, uint32_t id);
HWND fairy_ui_open(LPCWSTR class_name, WNDPROC procedure, int32_t width, int32_t height);
HWND fairy_ui_create_child(HWND window, LPCWSTR class_name, DWORD style, uint32_t id);
int32_t fairy_ui_redraw(HWND window);
BOOL fairy_ui_should_pump(BOOL wait, BOOL closing, BOOL pending);
int32_t fairy_ui_text(HWND control, const uint8_t *text, size_t length);
size_t fairy_ui_read(HWND control, uint8_t *output, size_t capacity);
int32_t fairy_ui_enable(HWND control, uint32_t enabled);
int32_t fairy_ui_focus(HWND control);
int32_t fairy_ui_next_focus(HWND window, uint32_t reverse);
int32_t fairy_ui_extent(HWND window, uint32_t axis);
int32_t fairy_ui_move(HWND control, int32_t x, int32_t y, int32_t width, int32_t height);
int32_t fairy_ui_system_dpi(void);
BOOL fairy_ui_dpi_suggestion(WPARAM dpi, LPARAM rectangle, RECT *output);
BOOL fairy_ui_apply_dpi(HWND window, const RECT *rectangle);
BOOL fairy_ui_icons(HWND window, HICON *large_icon, HICON *small_icon);
void fairy_ui_close_icons(HICON *large_icon, HICON *small_icon);
int32_t fairy_ui_name(IAccPropServices **service, BOOL *owned, HWND control, const uint8_t *text, size_t length);
int32_t fairy_ui_live(IAccPropServices *service, HWND control, uint32_t setting, uint32_t notify);
void fairy_ui_clear(IAccPropServices *service, HWND control, BOOL name, BOOL live);
#endif
