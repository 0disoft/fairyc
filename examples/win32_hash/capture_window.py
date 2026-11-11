"""Capture one owned native window to PNG without third-party packages."""
import ctypes as c
from ctypes import wintypes as w
from pathlib import Path
import struct
import zlib


def capture(hwnd, target):
    user, gdi = c.WinDLL('user32', use_last_error=True), c.WinDLL('gdi32', use_last_error=True)
    def api(dll, name, result, *types):
        fn = getattr(dll, name)
        fn.restype, fn.argtypes = result, types
        return fn
    rect = w.RECT()
    if not api(user, 'GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))(hwnd, c.byref(rect)):
        raise RuntimeError('capture: missing window')
    width, height = rect.right - rect.left, rect.bottom - rect.top
    if not 0 < width <= 16384 or not 0 < height <= 16384:
        raise RuntimeError('capture: invalid dimensions')
    dc = api(gdi, 'CreateCompatibleDC', w.HDC, w.HDC)(None)
    bitmap, previous = None, None
    delete = api(gdi, 'DeleteObject', w.BOOL, w.HANDLE)
    select = api(gdi, 'SelectObject', w.HANDLE, w.HDC, w.HANDLE)
    try:
        header = c.create_string_buffer(struct.pack('<IiiHHIIiiII', 40, width, -height, 1, 32, 0, 0, 0, 0, 0, 0))
        pixels = c.c_void_p()
        bitmap = api(gdi, 'CreateDIBSection', w.HBITMAP, w.HDC, c.c_void_p, w.UINT,
                     c.POINTER(c.c_void_p), w.HANDLE, w.DWORD)(dc, header, 0, c.byref(pixels), None, 0)
        if not dc or not bitmap or not pixels:
            raise RuntimeError('capture: bitmap allocation failed')
        previous = select(dc, bitmap)
        if not api(user, 'PrintWindow', w.BOOL, w.HWND, w.HDC, w.UINT)(hwnd, dc, 2):
            raise RuntimeError('capture: PrintWindow failed')
        api(gdi, 'GdiFlush', w.BOOL)()
        bgra = c.string_at(pixels, width * height * 4)
        # DWM's invisible resize margins are not part of the visible window.
        frame = w.RECT()
        dwm = c.WinDLL('dwmapi')
        bounds = api(dwm, 'DwmGetWindowAttribute', c.c_long, w.HWND, w.DWORD, c.c_void_p, w.DWORD)
        left = top = 0
        output_width, output_height = width, height
        if bounds(hwnd, 9, c.byref(frame), c.sizeof(frame)) == 0 and (
                rect.left <= frame.left < frame.right <= rect.right and rect.top <= frame.top < frame.bottom <= rect.bottom):
            left, top = frame.left - rect.left, frame.top - rect.top
            output_width, output_height = frame.right - frame.left, frame.bottom - frame.top
        rows = bytearray()
        for y in range(top, top + output_height):
            start = (y*width + left)*4
            source = bgra[start:start + output_width*4]
            rgb = bytearray(output_width*3)
            rgb[0::3], rgb[1::3], rgb[2::3] = source[2::4], source[1::4], source[0::4]
            rows.append(0)
            rows.extend(rgb)
        def chunk(kind, data):
            return struct.pack('>I', len(data)) + kind + data + struct.pack('>I', zlib.crc32(kind + data))
        png = b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', output_width, output_height, 8, 2, 0, 0, 0))
        png += chunk(b'IDAT', zlib.compress(rows)) + chunk(b'IEND', b'')
        Path(target).parent.mkdir(parents=True, exist_ok=True)
        Path(target).write_bytes(png)
    finally:
        if previous:
            select(dc, previous)
        if bitmap:
            delete(bitmap)
        if dc:
            api(gdi, 'DeleteDC', w.BOOL, w.HDC)(dc)
