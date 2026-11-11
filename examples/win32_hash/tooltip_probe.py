"""Observe an owned app's real tooltip using the standard Common Controls API.

Temporarily enable tracking on an existing registered tool, then restore its
exact TOOLINFO. Text still comes from the app's production notification callback;
no text, path or hash is injected. No desktop cursor/foreground access is needed.
"""
import ctypes as c
from ctypes import wintypes as w


class Tool(c.Structure):
    _fields_ = [('size', w.UINT), ('flags', w.UINT), ('hwnd', w.HWND), ('id', c.c_size_t),
                ('rect', w.RECT), ('instance', w.HINSTANCE), ('text', c.c_void_p),
                ('data', c.c_ssize_t), ('reserved', c.c_void_p)]


def probe(pid, main, popup, control, message, wait, visible, text, expected, clipped):
    kernel = c.WinDLL('kernel32', use_last_error=True)
    def api(name, result, *args):
        fn = getattr(kernel, name)
        fn.restype, fn.argtypes = result, args
        return fn
    open_process = api('OpenProcess', w.HANDLE, w.DWORD, w.BOOL, w.DWORD)
    allocate = api('VirtualAllocEx', c.c_void_p, w.HANDLE, c.c_void_p, c.c_size_t, w.DWORD, w.DWORD)
    free = api('VirtualFreeEx', w.BOOL, w.HANDLE, c.c_void_p, c.c_size_t, w.DWORD)
    write = api('WriteProcessMemory', w.BOOL, w.HANDLE, c.c_void_p, c.c_void_p, c.c_size_t, c.POINTER(c.c_size_t))
    read = api('ReadProcessMemory', w.BOOL, w.HANDLE, c.c_void_p, c.c_void_p, c.c_size_t, c.POINTER(c.c_size_t))
    close = api('CloseHandle', w.BOOL, w.HANDLE)
    process = open_process(0x38, False, pid)  # Only this verifier's child process.
    if not process:
        raise RuntimeError('cannot inspect owned native tooltip')
    remote = None
    original = None
    try:
        remote = allocate(process, None, c.sizeof(Tool), 0x3000, 4)
        if not remote:
            raise RuntimeError('cannot allocate owned TOOLINFO')
        def store(value):
            count = c.c_size_t()
            data = c.create_string_buffer(value)
            if not write(process, remote, data, len(value), c.byref(count)) or count.value != len(value):
                raise RuntimeError('cannot transfer owned TOOLINFO')
        initial = Tool()
        initial.size, initial.hwnd, initial.id = c.sizeof(Tool), main, control
        store(bytes(initial))
        if not message(popup, 0x400+53, 0, remote):  # TTM_GETTOOLINFOW
            raise RuntimeError('tooltip is not registered for the actual control')
        info, count = Tool(), c.c_size_t()
        if not read(process, remote, c.byref(info), c.sizeof(info), c.byref(count)) or count.value != c.sizeof(info):
            raise RuntimeError('cannot read registered TOOLINFO')
        original = bytes(info)
        if info.flags & 0x11 != 0x11 or info.hwnd != main or info.id != control or info.text != c.c_void_p(-1).value:
            raise RuntimeError('expected production IDISHWND/SUBCLASS/callback registration')
        message(popup, 0x400+28)  # TTM_POP
        info.flags |= 0x20  # TTF_TRACK: exercise the same callback without OS input.
        store(bytes(info))
        message(popup, 0x400+54, 0, remote)  # TTM_SETTOOLINFOW
        message(popup, 0x400+17, 1, remote)  # TTM_TRACKACTIVATE
        if clipped:
            wait(lambda: (bool(visible(popup)), text(popup)), lambda v: v == (True, expected), 'native tooltip callback/popup')
        elif visible(popup):
            raise RuntimeError('unclipped text unexpectedly shows a native tooltip')
    finally:
        try:
            if remote and original is not None:
                message(popup, 0x400+17, 0, remote)
                store(original)
                message(popup, 0x400+54, 0, remote)
                message(popup, 0x400+28)
        finally:
            if remote:
                free(process, remote, 0, 0x8000)
            close(process)
