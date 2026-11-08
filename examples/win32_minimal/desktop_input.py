"""Send desktop input only to this controller's owned Fairy example window."""
import ctypes as c
from ctypes import wintypes as w
from contextlib import contextmanager
import time


class Mouse(c.Structure):
    _fields_ = [('dx', w.LONG), ('dy', w.LONG), ('data', w.DWORD),
                ('flags', w.DWORD), ('time', w.DWORD), ('extra', c.c_size_t)]


class Keyboard(c.Structure):
    _fields_ = [('key', w.WORD), ('scan', w.WORD), ('flags', w.DWORD),
                ('time', w.DWORD), ('extra', c.c_size_t)]


class Hardware(c.Structure):
    _fields_ = [('message', w.DWORD), ('low', w.WORD), ('high', w.WORD)]


class Payload(c.Union):
    _fields_ = [('mouse', Mouse), ('keyboard', Keyboard), ('hardware', Hardware)]


class Input(c.Structure):
    _fields_ = [('kind', w.DWORD), ('payload', Payload)]


class TitleBar(c.Structure):
    _fields_ = [('size', w.DWORD), ('title', w.RECT),
                ('states', w.DWORD * 6), ('rects', w.RECT * 6)]


@contextmanager
def close_input(pid, method):
    """Require foreground ownership; return a receipt, never send WM_CLOSE."""
    if method not in ('close-button', 'alt-f4'):
        raise ValueError('unknown desktop close method')
    user = c.WinDLL('user32', use_last_error=True)
    callback = c.WINFUNCTYPE(w.BOOL, w.HWND, w.LPARAM)

    def api(name, result, *args):
        fn = getattr(user, name)
        fn.restype, fn.argtypes = result, args
        return fn

    enum = api('EnumWindows', w.BOOL, callback, w.LPARAM)
    owner = api('GetWindowThreadProcessId', w.DWORD, w.HWND, c.POINTER(w.DWORD))
    visible = api('IsWindowVisible', w.BOOL, w.HWND)
    name = api('GetClassNameW', c.c_int, w.HWND, w.LPWSTR, c.c_int)
    foreground = api('GetForegroundWindow', w.HWND)
    activate = api('SetForegroundWindow', w.BOOL, w.HWND)
    pressed = api('GetAsyncKeyState', w.SHORT, c.c_int)
    send = api('SendInput', w.UINT, w.UINT, c.POINTER(Input), c.c_int)
    dpi = api('SetThreadDpiAwarenessContext', c.c_void_p, c.c_void_p)
    message = api('SendMessageTimeoutW', c.c_ssize_t, w.HWND, w.UINT,
                  w.WPARAM, w.LPARAM, w.UINT, w.UINT, c.POINTER(c.c_size_t))
    cursor = api('GetCursorPos', w.BOOL, c.POINTER(w.POINT))
    restore = api('SetCursorPos', w.BOOL, c.c_int, c.c_int)
    at_point = api('WindowFromPoint', w.HWND, w.POINT)
    ancestor = api('GetAncestor', w.HWND, w.HWND, w.UINT)
    metric = api('GetSystemMetrics', c.c_int, c.c_int)
    windows = []

    @callback
    def visit(hwnd, unused):
        process = w.DWORD()
        owner(hwnd, c.byref(process))
        text = c.create_unicode_buffer(64)
        name(hwnd, text, len(text))
        if process.value == pid and visible(hwnd) and text.value == 'FairyMinimalWindow':
            windows.append(hwnd)
        return True

    if not enum(visit, 0) or len(windows) != 1:
        raise RuntimeError('expected exactly one visible window owned by the GUI process')
    hwnd = windows[0]
    for key in (0x01, 0x02, 0x04, 0x10, 0x11, 0x12, 0x5B, 0x5C, 0x73):
        if pressed(key) & 0x8000:
            raise RuntimeError('a mouse button or modifier/F4 key is held; no desktop input sent')
    activate(hwnd)
    if foreground() != hwnd:
        # A background CLI may need a temporary input-queue attachment to
        # activate its child. Always detach before sending desktop input.
        kernel = c.WinDLL('kernel32', use_last_error=True)
        kernel.GetCurrentThreadId.restype = w.DWORD
        kernel.GetCurrentThreadId.argtypes = []
        current_thread = kernel.GetCurrentThreadId()
        active_thread = owner(foreground(), None)
        target_thread = owner(hwnd, None)
        attach = api('AttachThreadInput', w.BOOL, w.DWORD, w.DWORD, w.BOOL)
        bring_to_top = api('BringWindowToTop', w.BOOL, w.HWND)
        attached = []
        try:
            for thread in dict.fromkeys((active_thread, target_thread)):
                if thread and thread != current_thread:
                    if not attach(current_thread, thread, True):
                        raise RuntimeError('cannot attach temporary foreground input queue')
                    attached.append(thread)
            bring_to_top(hwnd)
            activate(hwnd)
        finally:
            detached = [attach(current_thread, thread, False) for thread in reversed(attached)]
            if not all(detached):
                raise RuntimeError('cannot detach temporary foreground input queue')
    deadline = time.monotonic() + 2
    while foreground() != hwnd:
        if time.monotonic() >= deadline:
            raise RuntimeError('owned GUI did not acquire foreground; no desktop input sent')
        time.sleep(0.02)
    # Do not combine our shortcut/click with keys or buttons held by the user.
    for key in (0x01, 0x02, 0x04, 0x10, 0x11, 0x12, 0x5B, 0x5C, 0x73):
        if pressed(key) & 0x8000:
            raise RuntimeError('a mouse button or modifier/F4 key is held; no desktop input sent')

    def require_target():
        process = w.DWORD()
        owner(hwnd, c.byref(process))
        if process.value != pid or foreground() != hwnd:
            raise RuntimeError('owned GUI lost foreground; desktop input aborted')

    def submit(events, releases):
        require_target()
        batch = (Input * len(events))(*events)
        if send(len(batch), batch, c.sizeof(Input)) != len(batch):
            # Release our own potentially partially injected keys/buttons.
            cleanup = (Input * len(releases))(*releases)
            send(len(cleanup), cleanup, c.sizeof(Input))
            raise RuntimeError('SendInput did not insert every event')

    receipt = dict(method=method, pid=pid, hwnd=hwnd, foregroundVerified=True)
    if method == 'alt-f4':
        def key(vk, flags=0):
            return Input(1, Payload(keyboard=Keyboard(vk, 0, flags, 0, 0)))
        releases = [key(0x73, 2), key(0x12, 2)]
        submit([key(0x12), key(0x73), *releases], releases)
        yield dict(receipt, injectedEvents=4)
        return

    previous_dpi = dpi(c.c_void_p(-4))  # Per-monitor aware v2, physical pixels.
    if not previous_dpi:
        raise RuntimeError('cannot use physical desktop coordinates')
    old, current = w.POINT(), w.POINT()
    point = None
    try:
        if not cursor(c.byref(old)):
            raise RuntimeError('cannot read cursor position')
        title = TitleBar()
        title.size = c.sizeof(title)
        result = c.c_size_t()
        if not message(hwnd, 0x033F, 0, c.addressof(title), 3, 2000, c.byref(result)):
            raise RuntimeError('cannot query native title-bar buttons')
        rect = title.rects[5]
        if title.states[5] & (0x8000 | 0x10000 | 1) or rect.right <= rect.left or rect.bottom <= rect.top:
            raise RuntimeError('native close button is unavailable')
        x, y = (rect.left + rect.right) // 2, (rect.top + rect.bottom) // 2
        left, top, width, height = [metric(index) for index in (76, 77, 78, 79)]
        if width <= 1 or height <= 1 or not (left <= x < left + width and top <= y < top + height):
            raise RuntimeError('close button lies outside the virtual desktop')
        normalized_x = (x - left) * 65535 // (width - 1)
        normalized_y = (y - top) * 65535 // (height - 1)
        move = Input(0, Payload(mouse=Mouse(normalized_x, normalized_y, 0, 0xC001, 0, 0)))
        up = Input(0, Payload(mouse=Mouse(0, 0, 0, 4, 0, 0)))
        submit([move], [up])
        point = (x, y)
        deadline = time.monotonic() + 1
        while True:
            cursor(c.byref(current))
            if abs(current.x - x) <= 1 and abs(current.y - y) <= 1:
                break
            if time.monotonic() >= deadline:
                raise RuntimeError('cursor did not reach the native close button')
            time.sleep(0.01)
        if ancestor(at_point(current), 2) != hwnd:
            raise RuntimeError('native close button is occluded; click aborted')
        packed = (current.x & 0xFFFF) | ((current.y & 0xFFFF) << 16)
        if not message(hwnd, 0x0084, 0, packed, 3, 2000, c.byref(result)) or result.value != 20:
            raise RuntimeError('native hit test did not identify HTCLOSE; click aborted')
        down = Input(0, Payload(mouse=Mouse(0, 0, 0, 2, 0, 0)))
        submit([down, up], [up])
        yield dict(receipt, injectedEvents=3, closeButton=[x, y], hitTest='HTCLOSE')
    finally:
        if point is not None and cursor(c.byref(current)):
            if abs(current.x - point[0]) <= 1 and abs(current.y - point[1]) <= 1:
                restore(old.x, old.y)
        dpi(previous_dpi)
