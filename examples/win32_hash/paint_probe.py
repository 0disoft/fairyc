"""Read already-painted owned controls; never request a repair before sampling."""
import ctypes as c
from ctypes import wintypes as w
import time


def background_pixels(window, controls):
    user, gdi = c.WinDLL('user32'), c.WinDLL('gdi32')
    def api(dll, name, result, *types):
        function = getattr(dll, name)
        function.restype, function.argtypes = result, types
        return function
    bounds = api(user, 'GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
    rect = api(user, 'GetClientRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
    to_screen = api(user, 'ClientToScreen', w.BOOL, w.HWND, c.POINTER(w.POINT))
    hit = api(user, 'WindowFromPoint', w.HWND, w.POINT)
    root = api(user, 'GetAncestor', w.HWND, w.HWND, w.UINT)
    get_dc = api(user, 'GetDC', w.HDC, w.HWND)
    release = api(user, 'ReleaseDC', c.c_int, w.HWND, w.HDC)
    pixel = api(gdi, 'GetPixel', w.DWORD, w.HDC, c.c_int, c.c_int)
    # Let queued WM_PAINT and DWM composition finish without requesting new
    # paint. PrintWindow/RedrawWindow here would conceal the regression.
    time.sleep(0.1)
    outer = w.RECT()
    if not bounds(window, c.byref(outer)):
        raise RuntimeError('paint observation: missing owned window')
    points = [('canvas', window, 2, 2)]
    for identifier, control in controls:
        area = w.RECT()
        if not rect(control, c.byref(area)) or min(area.right, area.bottom) < 4:
            raise RuntimeError('paint observation: invalid control extent')
        # Top-left of document decoration and bottom-right of byte label
        # contain background, away from icon lines and label glyphs.
        points.append((str(identifier), control,
                       0 if identifier == 211 else area.right-2,
                       0 if identifier == 211 else area.bottom-2))
    observed = {}
    # Read the composed desktop, not a parent's partially painted backing DC.
    dc = get_dc(None)
    if not dc:
        raise RuntimeError('paint observation: missing visible DC')
    try:
        for name, control, x, y in points:
            point = w.POINT(x, y)
            if not to_screen(control, c.byref(point)):
                raise RuntimeError(f'paint observation: cannot map {name}')
            covering = hit(point)
            if root(covering, 2) != window:
                pid, label = w.DWORD(), c.create_unicode_buffer(128)
                api(user, 'GetWindowThreadProcessId', w.DWORD, w.HWND, c.POINTER(w.DWORD))(covering, c.byref(pid))
                api(user, 'GetClassNameW', c.c_int, w.HWND, w.LPWSTR, c.c_int)(covering, label, len(label))
                raise RuntimeError(f'paint observation occluded: {name}, point={(point.x, point.y)}, class={label.value}, pid={pid.value}, bounds={(outer.left, outer.top, outer.right, outer.bottom)}')
            color = pixel(dc, point.x, point.y)
            if color == 0xFFFFFFFF:
                raise RuntimeError(f'paint observation unreadable: {name}')
            observed[name] = color
    finally:
        release(None, dc)
    return observed
