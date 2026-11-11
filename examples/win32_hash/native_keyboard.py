"""Real SendInput with owned-window, idle-key and stable-foreground guards."""
import ctypes as c
from ctypes import wintypes as w
import time


class Keyboard(c.Structure):
    _fields_ = [('key', w.WORD), ('scan', w.WORD), ('flags', w.DWORD),
                ('time', w.DWORD), ('extra', c.c_size_t)]


class Payload(c.Union):
    _fields_ = [('keyboard', Keyboard), ('padding', c.c_byte*32)]


class Input(c.Structure):
    _fields_ = [('kind', w.DWORD), ('payload', Payload)]


class DesktopKeys:
    def __init__(self, pid):
        self.pid, self.receipts = pid, []
        self.user = c.WinDLL('user32', use_last_error=True)
        self.foreground = self.api('GetForegroundWindow', w.HWND)
        self.owner = self.api('GetWindowThreadProcessId', w.DWORD, w.HWND, c.POINTER(w.DWORD))
        self.pressed = self.api('GetAsyncKeyState', w.SHORT, c.c_int)

    def api(self, name, result, *types):
        fn = getattr(self.user, name)
        fn.restype, fn.argtypes = result, types
        return fn

    def require_owned(self, hwnd):
        pid = w.DWORD()
        thread = self.owner(hwnd, c.byref(pid))
        if not thread or pid.value != self.pid:
            raise RuntimeError('keyboard target is not the owned native process')
        return thread

    def describe_foreground(self):
        pid, name = w.DWORD(), c.create_unicode_buffer(128)
        self.owner(self.foreground(), c.byref(pid))
        self.api('GetClassNameW', c.c_int, w.HWND, w.LPWSTR, c.c_int)(self.foreground(), name, len(name))
        return dict(pid=pid.value, windowClass=name.value)

    def activate(self, hwnd):
        self.idle(())
        target_thread = self.require_owned(hwnd)
        activate = self.api('SetForegroundWindow', w.BOOL, w.HWND)
        activate(hwnd)
        kernel = c.WinDLL('kernel32')
        kernel.GetCurrentThreadId.restype = w.DWORD
        current = kernel.GetCurrentThreadId()
        queued = w.MSG()
        self.api('PeekMessageW', w.BOOL, c.POINTER(w.MSG), w.HWND, w.UINT, w.UINT, w.UINT)(c.byref(queued), None, 0, 0, 0)
        attach = self.api('AttachThreadInput', w.BOOL, w.DWORD, w.DWORD, w.BOOL)
        deadline, stable = time.monotonic()+2, None
        while time.monotonic() < deadline:
            if self.foreground() == hwnd:
                stable = stable or time.monotonic()
                if time.monotonic()-stable >= 0.1:
                    return
                time.sleep(0.01)
                continue
            stable, attached = None, []
            try:
                for thread in dict.fromkeys((self.owner(self.foreground(), None), target_thread)):
                    if thread and thread != current and attach(current, thread, True):
                        attached.append(thread)
                self.api('BringWindowToTop', w.BOOL, w.HWND)(hwnd)
                activate(hwnd)
            finally:
                if not all([attach(current, thread, False) for thread in reversed(attached)]):
                    raise RuntimeError('cannot detach keyboard input queues')
            time.sleep(0.02)
        actual = self.foreground()
        pid, name = w.DWORD(), c.create_unicode_buffer(128)
        self.owner(actual, c.byref(pid))
        self.api('GetClassNameW', c.c_int, w.HWND, w.LPWSTR, c.c_int)(actual, name, len(name))
        raise RuntimeError(f'foreground denied: pid={pid.value}, class={name.value}; no keys sent')

    def idle(self, keys):
        held = (1, 2, 4, 5, 6, 9, 13, 16, 17, 18, 27, 91, 92, 115, *keys)
        deadline = time.monotonic()+2
        while any(self.pressed(key) & 0x8000 for key in held):
            if time.monotonic() >= deadline:
                raise RuntimeError('user input remains held; no keys sent')
            time.sleep(0.02)

    def chord(self, hwnd, keys):
        if not keys or any(key not in (9, 13, 16, 17, 18, 27, 32, 65, 67, 78, 79, 83, 88) for key in keys):
            raise ValueError('unsupported verification key chord')
        self.activate(hwnd)
        self.idle(keys)
        self.require_owned(hwnd)
        if self.foreground() != hwnd:
            raise RuntimeError(f'owned window lost foreground; actual={self.describe_foreground()}; no keys sent')
        def key(code, flags=0):
            return Input(1, Payload(keyboard=Keyboard(code, 0, flags, 0, 0)))
        releases = [key(code, 2) for code in reversed(keys)]
        events = [key(code) for code in keys] + releases
        send = self.api('SendInput', w.UINT, w.UINT, c.POINTER(Input), c.c_int)
        if send(len(events), (Input*len(events))(*events), c.sizeof(Input)) != len(events):
            send(len(releases), (Input*len(releases))(*releases), c.sizeof(Input))
            raise RuntimeError('incomplete SendInput; released only test-owned keys')
        self.receipts.append(dict(hwnd=hwnd, keys=keys, events=len(events), foregroundVerified=True))

    def focus(self, hwnd):
        class Info(c.Structure):
            _fields_ = [('size', w.DWORD), ('flags', w.DWORD), ('active', w.HWND),
                        ('focus', w.HWND), ('capture', w.HWND), ('menu', w.HWND),
                        ('move', w.HWND), ('caret', w.HWND), ('rect', w.RECT)]
        info = Info()
        info.size = c.sizeof(info)
        if not self.api('GetGUIThreadInfo', w.BOOL, w.DWORD, c.POINTER(Info))(self.require_owned(hwnd), c.byref(info)):
            raise RuntimeError('cannot observe native keyboard focus')
        return self.api('GetDlgCtrlID', c.c_int, w.HWND)(info.focus)
