"""Preserve materialized clipboard formats without a live OLE clipboard wrapper."""
import ctypes as c
from ctypes import wintypes as w
from contextlib import contextmanager
import time


@contextmanager
def preserve_clipboard():
    user, ole, gdi = (c.WinDLL(name, use_last_error=True) for name in ('user32', 'ole32', 'gdi32'))
    def api(dll, name, result, *types):
        fn = getattr(dll, name)
        fn.restype, fn.argtypes = result, types
        return fn
    open_clipboard = api(user, 'OpenClipboard', w.BOOL, w.HWND)
    close = api(user, 'CloseClipboard', w.BOOL)
    empty = api(user, 'EmptyClipboard', w.BOOL)
    enum = api(user, 'EnumClipboardFormats', w.UINT, w.UINT)
    get = api(user, 'GetClipboardData', w.HANDLE, w.UINT)
    put = api(user, 'SetClipboardData', w.HANDLE, w.UINT, w.HANDLE)
    name = api(user, 'GetClipboardFormatNameW', c.c_int, w.UINT, w.LPWSTR, c.c_int)
    duplicate = api(ole, 'OleDuplicateData', w.HANDLE, w.HANDLE, w.WORD, w.UINT)
    copy_emf = api(gdi, 'CopyEnhMetaFileW', w.HANDLE, w.HANDLE, w.LPCWSTR)
    owner = api(user, 'CreateWindowExW', w.HWND, w.DWORD, w.LPCWSTR, w.LPCWSTR, w.DWORD,
                c.c_int, c.c_int, c.c_int, c.c_int, w.HWND, w.HMENU, w.HINSTANCE, c.c_void_p)(
                    0, 'STATIC', 'Clipboard verification owner', 0x80000000, 0, 0, 0, 0, None, None, None, None)
    destroy = api(user, 'DestroyWindow', w.BOOL, w.HWND)
    if not owner:
        raise RuntimeError('cannot create hidden clipboard verification owner')
    class Medium(c.Structure):
        _fields_ = [('tymed', w.DWORD), ('handle', w.HANDLE), ('release', c.c_void_p)]
    release = api(ole, 'ReleaseStgMedium', None, c.POINTER(Medium))
    saved = []
    def acquire():
        deadline = time.monotonic() + 3
        while not open_clipboard(owner):
            if time.monotonic() >= deadline:
                raise RuntimeError('cannot snapshot/restore a locked clipboard')
            time.sleep(0.02)
    try:
        acquire()
        try:
            format_id = 0
            while True:
                c.set_last_error(0)
                format_id = enum(format_id)
                if not format_id:
                    if c.get_last_error():
                        raise RuntimeError('clipboard format enumeration failed')
                    break
                label = c.create_unicode_buffer(256)
                name(format_id, label, len(label))
                # These identify a live OLE owner, not user content. The actual
                # text/image/file/custom data is eagerly copied below.
                if label.value in ('DataObject', 'Ole Private Data'):
                    continue
                if format_id == 0x80:
                    raise RuntimeError('cannot snapshot an owner-display clipboard')
                original = get(format_id)
                if not original:
                    # Windows may advertise synthesized/delayed formats whose
                    # source no longer exists. Preserve the available handles;
                    # restored base formats can regenerate their conversions.
                    continue
                kind = 64 if format_id in (14, 0x8E) else 32 if format_id in (3, 0x83) else 16 if format_id in (2, 9, 0x82) else 1
                copied = copy_emf(original, None) if kind == 64 else duplicate(original, {0x82: 2, 0x83: 3}.get(format_id, format_id), 2)
                if not copied:
                    raise RuntimeError(f'cannot duplicate clipboard format {format_id}')
                saved.append([format_id, copied, kind])
        finally:
            close()
        try:
            yield
        finally:
            acquire()
            try:
                if not empty():
                    raise RuntimeError('cannot clear clipboard for restoration')
                for entry in saved:
                    format_id, handle, kind = entry
                    if not put(format_id, handle):
                        raise RuntimeError(f'cannot restore clipboard format {format_id}')
                    entry[1] = None  # Clipboard now owns this handle.
            finally:
                close()
    finally:
        for format_id, handle, kind in saved:
            if handle:
                release(c.byref(Medium(kind, handle, None)))
        destroy(owner)
