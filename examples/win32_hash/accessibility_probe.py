"""Read native UI Automation properties from an owned process (no UIA actions)."""
import ctypes as c
from ctypes import wintypes as w
import uuid


class Data(c.Union):
    _fields_ = [('integer', c.c_longlong), ('pointer', c.c_void_p), ('real', c.c_double), ('raw', c.c_byte*16)]


class Variant(c.Structure):
    _fields_ = [('kind', w.WORD), ('reserved', w.WORD*3), ('data', Data)]


def inspect(hwnd, pid):
    ole, automation = c.WinDLL('ole32'), c.WinDLL('oleaut32')
    ole.CoInitializeEx.argtypes, ole.CoInitializeEx.restype = [c.c_void_p, w.DWORD], w.LONG
    ole.CoCreateInstance.argtypes = [c.c_void_p, c.c_void_p, w.DWORD, c.c_void_p, c.POINTER(c.c_void_p)]
    ole.CoCreateInstance.restype = w.LONG
    automation.VariantClear.argtypes = [c.POINTER(Variant)]
    def check(result):
        if result < 0:
            raise RuntimeError(f'UI Automation HRESULT {result & 0xffffffff:#x}')
    def call(pointer, index, types, *args):
        # Public Windows SDK IUIAutomation/IUIAutomationElement vtable order.
        table = c.cast(pointer, c.POINTER(c.POINTER(c.c_void_p))).contents
        result = c.WINFUNCTYPE(w.LONG, c.c_void_p, *types)(table[index])(pointer, *args)
        check(result)
    def release(pointer):
        if pointer:
            call(pointer, 2, [])
    def guid(value):
        return (c.c_byte*16).from_buffer_copy(uuid.UUID(value).bytes_le)
    def prop(element, identifier):
        value = Variant()
        try:
            call(element, 10, [c.c_int, c.POINTER(Variant)], identifier, c.byref(value))
            if value.kind == 8:
                return c.wstring_at(value.data.pointer) if value.data.pointer else ''
            if value.kind == 3:
                return c.c_int(value.data.integer).value
            if value.kind == 11:
                return bool(value.data.integer & 0xffff)
            if value.kind == 5:
                return value.data.real
            return None
        finally:
            automation.VariantClear(c.byref(value))
    check(ole.CoInitializeEx(None, 0))
    client, root, condition, elements = (c.c_void_p() for _ in range(4))
    try:
        clsid = guid('ff48dba4-60ef-4201-aa87-54103eef594e')
        iid = guid('30cbe57d-d9d0-452a-ab13-7ac5ac4825ee')
        check(ole.CoCreateInstance(c.byref(clsid), None, 1, c.byref(iid), c.byref(client)))
        call(client, 6, [w.HWND, c.POINTER(c.c_void_p)], hwnd, c.byref(root))
        if prop(root, 30002) != pid:
            raise RuntimeError('accessibility root is not the owned process')
        call(client, 21, [c.POINTER(c.c_void_p)], c.byref(condition))
        call(root, 6, [c.c_int, c.c_void_p, c.POINTER(c.c_void_p)], 4, condition, c.byref(elements))
        count = c.c_int()
        call(elements, 3, [c.POINTER(c.c_int)], c.byref(count))
        properties = dict(id=30011, name=30005, type=30003, enabled=30010, focusable=30009,
                          focused=30008, accessKey=30007, value=30045, readOnly=30046,
                          valuePattern=30043, rangePattern=30033,
                          rangeValue=30047, rangeMin=30049, rangeMax=30050, liveSetting=30135)
        rows = []
        for index in range(count.value):
            element = c.c_void_p()
            try:
                call(elements, 4, [c.c_int, c.POINTER(c.c_void_p)], index, c.byref(element))
                if prop(element, 30002) == pid:
                    rows.append({name: prop(element, identifier) for name, identifier in properties.items()})
            finally:
                release(element)
        return dict(ok=True, processId=pid, controls=rows, provider='native-IUIAutomation')
    finally:
        for pointer in (elements, condition, root, client):
            release(pointer)
        ole.CoUninitialize()


class StatusEvents:
    """Observe real, process-filtered live-region WinEvents without UI actions."""
    def __init__(self, driver):
        self.driver, self.events = driver, []
        self.user = c.WinDLL('user32', use_last_error=True)
        callback = c.WINFUNCTYPE(None, w.HANDLE, w.DWORD, w.HWND, w.LONG, w.LONG, w.DWORD, w.DWORD)
        def receive(hook, event, hwnd, object_id, child_id, thread, ticks):
            if hwnd == driver.child(driver.main, 205) and object_id == -4 and child_id == 0:
                self.events.append(dict(event=event, text=driver.text(hwnd)))
        self.callback = callback(receive)

    def __enter__(self):
        install = self.user.SetWinEventHook
        install.argtypes, install.restype = [w.DWORD, w.DWORD, w.HMODULE, c.c_void_p, w.DWORD, w.DWORD, w.DWORD], w.HANDLE
        self.hook = install(0x8019, 0x8019, None, self.callback, self.driver.process.pid, 0, 0)
        if not self.hook: raise OSError(c.get_last_error(), 'cannot observe status live events')
        return self

    def drain(self):
        message = w.MSG()
        peek = self.user.PeekMessageW
        peek.argtypes, peek.restype = [c.POINTER(w.MSG), w.HWND, w.UINT, w.UINT, w.UINT], w.BOOL
        while peek(c.byref(message), None, 0, 0, 1):
            self.user.TranslateMessage(c.byref(message)); self.user.DispatchMessageW(c.byref(message))
        return self.events

    def __exit__(self, *error):
        self.user.UnhookWinEvent.argtypes = [w.HANDLE]
        if not self.user.UnhookWinEvent(self.hook):
            raise OSError(c.get_last_error(), 'cannot release status observer')
