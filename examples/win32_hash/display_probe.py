"""Actual monitor movement and temporary OS high contrast, with restoration."""
import ctypes as c
from ctypes import wintypes as w
import json
import os
from pathlib import Path
import subprocess
import sys
import time


class Contrast(c.Structure):
    _fields_ = [('size', w.UINT), ('flags', w.DWORD), ('scheme', w.LPWSTR)]


def api(dll, name, result, *types):
    fn = getattr(dll, name)
    fn.restype, fn.argtypes = result, types
    return fn


def contrast_snapshot():
    user = c.WinDLL('user32', use_last_error=True)
    value = Contrast()
    value.size = c.sizeof(value)
    if not api(user, 'SystemParametersInfoW', w.BOOL, w.UINT, w.UINT, c.c_void_p, w.UINT)(0x42, value.size, c.byref(value), 0):
        raise RuntimeError('cannot read OS high-contrast settings')
    color = api(user, 'GetSysColor', w.DWORD, c.c_int)
    indices = list(range(25))+list(range(26, 31))
    return dict(flags=value.flags, scheme=value.scheme, indices=indices, colors=[color(n) for n in indices])


def set_contrast(snapshot, flags=None):
    user = c.WinDLL('user32', use_last_error=True)
    value = Contrast(c.sizeof(Contrast), snapshot['flags'] if flags is None else flags, snapshot['scheme'])
    # Broadcast current-session changes without persisting a new user preference.
    if not api(user, 'SystemParametersInfoW', w.BOOL, w.UINT, w.UINT, c.c_void_p, w.UINT)(0x43, value.size, c.byref(value), 2):
        raise RuntimeError(f'cannot change/restore OS high contrast: {c.get_last_error()}')


def restore(snapshot):
    set_contrast(snapshot)
    # Keep exact pre-test system colors if the theme engine did not restore them.
    user = c.WinDLL('user32')
    color = api(user, 'GetSysColor', w.DWORD, c.c_int)
    if [color(n) for n in snapshot['indices']] != snapshot['colors']:
        count = len(snapshot['indices'])
        indices, values = (c.c_int*count)(*snapshot['indices']), (w.DWORD*count)(*snapshot['colors'])
        if not api(user, 'SetSysColors', w.BOOL, c.c_int, c.POINTER(c.c_int), c.POINTER(w.DWORD))(count, indices, values):
            raise RuntimeError('cannot restore original system colors')


def painted_canvas(hwnd, label=None, ink=None):
    user, gdi = c.WinDLL('user32'), c.WinDLL('gdi32')
    rect, point = w.RECT(), w.POINT(5, 5)
    if not api(user, 'GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))(hwnd, c.byref(rect)):
        raise RuntimeError('missing contrast target')
    api(user, 'ClientToScreen', w.BOOL, w.HWND, c.POINTER(w.POINT))(hwnd, c.byref(point))
    dc = api(gdi, 'CreateCompatibleDC', w.HDC, w.HDC)(None)
    screen = api(user, 'GetDC', w.HDC, w.HWND)(None)
    bitmap = api(gdi, 'CreateCompatibleBitmap', w.HBITMAP, w.HDC, c.c_int, c.c_int)(screen, rect.right-rect.left, rect.bottom-rect.top)
    select = api(gdi, 'SelectObject', w.HANDLE, w.HDC, w.HANDLE)
    previous = None
    try:
        if not dc or not bitmap:
            raise RuntimeError('cannot allocate contrast observation')
        previous = select(dc, bitmap)
        if not api(user, 'PrintWindow', w.BOOL, w.HWND, w.HDC, w.UINT)(hwnd, dc, 2):
            raise RuntimeError('cannot render owned window for contrast observation')
        pixel = api(gdi, 'GetPixel', w.DWORD, w.HDC, c.c_int, c.c_int)
        color = pixel(dc, point.x-rect.left, point.y-rect.top)
        if color == 0xffffffff:
            raise RuntimeError('contrast canvas unreadable')
        if label:
            area = w.RECT()
            api(user, 'GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))(label, c.byref(area))
            matches = sum(pixel(dc, x-rect.left, y-rect.top) == ink
                          for y in range(area.top, area.bottom)
                          for x in range(area.left, min(area.right, area.left+160)))
            return dict(canvas=color, textPixels=matches)
        return color
    finally:
        if previous:
            select(dc, previous)
        if bitmap:
            api(gdi, 'DeleteObject', w.BOOL, w.HANDLE)(bitmap)
        if dc:
            api(gdi, 'DeleteDC', w.BOOL, w.HDC)(dc)
        if screen:
            api(user, 'ReleaseDC', c.c_int, w.HWND, w.HDC)(None, screen)


def inventory():
    user, shcore = c.WinDLL('user32'), c.WinDLL('shcore')
    callback = c.WINFUNCTYPE(w.BOOL, w.HMONITOR, w.HDC, c.POINTER(w.RECT), w.LPARAM)
    rows = []
    @callback
    def visit(monitor, dc, rect, unused):
        scale = c.c_int()
        status = api(shcore, 'GetScaleFactorForMonitor', w.LONG, w.HMONITOR, c.POINTER(c.c_int))(monitor, c.byref(scale))
        box = rect.contents
        rows.append(dict(bounds=[box.left, box.top, box.right, box.bottom], scalePercent=scale.value if status == 0 else None))
        return True
    if not api(user, 'EnumDisplayMonitors', w.BOOL, w.HDC, c.c_void_p, callback, w.LPARAM)(None, None, visit, 0):
        raise RuntimeError('monitor enumeration failed')
    return rows


def verify(driver, high_contrast):
    d, user = driver, c.WinDLL('user32')
    monitors, observations = inventory(), []
    original = w.RECT()
    d.window_rect(d.main, c.byref(original))
    context = api(user, 'GetWindowDpiAwarenessContext', c.c_void_p, w.HWND)(d.main)
    if not api(user, 'AreDpiAwarenessContextsEqual', w.BOOL, c.c_void_p, c.c_void_p)(context, c.c_void_p(-4)):
        raise RuntimeError('Hash window is not per-monitor v2 aware')
    try:
        # Move only this process's window. The OS, not a fabricated message,
        # decides its real monitor DPI and sends WM_DPICHANGED when applicable.
        for monitor in monitors:
            left, top, right, bottom = monitor['bounds']
            if not d.resize(d.main, None, left+20, top+40, 0, 0, 0x15):
                raise RuntimeError('cannot move owned window to monitor')
            expected = round(monitor['scalePercent']*96/100) if monitor['scalePercent'] else None
            observed = d.wait(lambda: d.window_dpi(d.main), lambda v: expected is None or v == expected, 'actual monitor DPI')
            d.wait(d.layout, bool, 'actual-monitor layout invalid')
            digest = w.RECT()
            d.window_rect(d.child(d.main, 203), c.byref(digest))
            if digest.bottom-digest.top != (70*observed+48)//96:
                raise RuntimeError('digest geometry did not track actual DPI')
            font = d.message(d.child(d.main, 201), 0x31)
            logical = c.create_string_buffer(92)  # Windows SDK LOGFONTW.
            if not api(c.WinDLL('gdi32'), 'GetObjectW', c.c_int, w.HANDLE, c.c_int, c.c_void_p)(font, 92, logical):
                raise RuntimeError('cannot inspect actual control font')
            font_height = c.cast(logical, c.POINTER(w.LONG)).contents.value
            if font_height != -((14*observed+48)//96):
                raise RuntimeError('font did not track actual DPI')
            observations.append(dict(dpi=observed, monitor=monitor, digestHeight=digest.bottom-digest.top, fontHeight=font_height))
    finally:
        d.resize(d.main, None, original.left, original.top, original.right-original.left, original.bottom-original.top, 0x14)
        d.wait(d.layout, bool, 'display test did not restore layout')
    result = dict(monitors=monitors, observations=observations, perMonitorV2=True,
                  physicalDifferentDpiTransition=len({row['dpi'] for row in observations}) > 1,
                  transitionSkipReason=None if len({row['dpi'] for row in observations}) > 1 else 'no connected monitors with different DPI')
    if high_contrast:
        snapshot = contrast_snapshot()
        path = d.root/'contrast-restore.json'
        path.write_text(json.dumps(snapshot), encoding='utf8')
        watchdog = subprocess.Popen([sys.executable, '-B', str(Path(__file__).resolve()), '--watchdog', str(path), str(os.getpid())],
                                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, creationflags=0x08000000)
        try:
            d.wait(lambda: path.with_suffix('.ready').exists(), bool, 'contrast restoration watchdog did not start', seconds=3)
            set_contrast(snapshot, snapshot['flags'] | 1)
            d.wait(lambda: contrast_snapshot()['flags'] & 1, bool, 'OS high contrast did not activate')
            def observe_contrast():
                system = api(user, 'GetSysColor', w.DWORD, c.c_int)
                ink = system(8)
                return dict(enabled=bool(contrast_snapshot()['flags'] & 1),
                            systemColor=system(5), systemTextColor=ink,
                            **painted_canvas(d.main, d.child(d.main, 207), ink))
            # Theme colors can settle after the HC flag changes. Compare live
            # system and app colors, not the pre-transition GetSysColor value.
            observed = d.wait(observe_contrast,
                lambda v: v['enabled'] and v['systemColor'] == v['canvas'] and v['systemTextColor'] != v['canvas'] and v['textPixels'] > 0,
                'application did not adopt system high-contrast canvas', seconds=8)
            d.wait(d.layout, bool, 'high-contrast layout invalid')
            result['highContrast'] = dict(osEnabled=True, systemWindowColor=observed['systemColor'], applicationCanvas=observed['canvas'],
                                         systemTextColor=observed['systemTextColor'], matchingTextPixels=observed['textPixels'])
        finally:
            restore(snapshot)
            restored = contrast_snapshot()
            if restored['flags'] != snapshot['flags'] or restored['colors'] != snapshot['colors']:
                raise RuntimeError('OS high-contrast/color restoration not verified; watchdog remains armed')
            watchdog.terminate()
            watchdog.wait(timeout=5)
        expected = snapshot['colors'][5] if snapshot['flags'] & 1 else 246 | (243 << 8) | (237 << 16)
        d.wait(lambda: painted_canvas(d.main), lambda v: v == expected, 'application did not restore its original palette')
        result['highContrast']['originalFlagsAndColorsRestored'] = True
        result['highContrast']['originalDefaultScheme'] = snapshot['scheme']
        result['highContrast']['returnedDefaultScheme'] = restored['scheme']
        result['highContrast']['schemeCanonicalized'] = snapshot['scheme'] != restored['scheme']
    return result


if __name__ == '__main__' and len(sys.argv) == 4 and sys.argv[1] == '--watchdog':
    snapshot = json.loads(Path(sys.argv[2]).read_text(encoding='utf8'))
    kernel = c.WinDLL('kernel32')
    process = api(kernel, 'OpenProcess', w.HANDLE, w.DWORD, w.BOOL, w.DWORD)(0x100000, False, int(sys.argv[3]))
    try:
        Path(sys.argv[2]).with_suffix('.ready').write_text('ready', encoding='ascii')
        if process:
            api(kernel, 'WaitForSingleObject', w.DWORD, w.HANDLE, w.DWORD)(process, 45000)
        restore(snapshot)
    finally:
        if process:
            api(kernel, 'CloseHandle', w.BOOL, w.HANDLE)(process)
