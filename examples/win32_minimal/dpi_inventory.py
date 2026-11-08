"""Read display bounds/scales without modifying monitor settings."""
import ctypes as c
from ctypes import wintypes as w


def monitors():
    user, shcore = c.WinDLL('user32'), c.WinDLL('shcore')
    callback = c.WINFUNCTYPE(w.BOOL, w.HMONITOR, w.HDC, c.POINTER(w.RECT), w.LPARAM)
    user.SetThreadDpiAwarenessContext.argtypes = [c.c_void_p]
    user.SetThreadDpiAwarenessContext.restype = c.c_void_p
    user.EnumDisplayMonitors.argtypes = [w.HDC, c.POINTER(w.RECT), callback, w.LPARAM]
    user.EnumDisplayMonitors.restype = w.BOOL
    shcore.GetScaleFactorForMonitor.argtypes = [w.HMONITOR, c.POINTER(c.c_int)]
    shcore.GetScaleFactorForMonitor.restype = w.LONG
    previous = user.SetThreadDpiAwarenessContext(c.c_void_p(-4))
    if not previous:
        raise RuntimeError('cannot read monitor inventory in physical coordinates')
    rows = []

    @callback
    def visit(monitor, unused_dc, rect, unused):
        scale = c.c_int()
        status = shcore.GetScaleFactorForMonitor(monitor, c.byref(scale))
        box = rect.contents
        rows.append(dict(bounds=[box.left, box.top, box.right, box.bottom],
                         scalePercent=scale.value if status == 0 else None))
        return True

    try:
        if not user.EnumDisplayMonitors(None, None, visit, 0):
            raise RuntimeError('cannot enumerate display monitors')
        return rows
    finally:
        user.SetThreadDpiAwarenessContext(previous)
