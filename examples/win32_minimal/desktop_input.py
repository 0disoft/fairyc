"""Send desktop input only to this controller's owned Fairy example window."""
import ctypes as c
from ctypes import wintypes as w
from contextlib import contextmanager
import time
from types import SimpleNamespace


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
def window_input(pid, method, released=None, focus_control=None):
    """Require foreground ownership; inject input without synthesizing UI messages."""
    keyboard = {'alt-f4': (0x12, 0x73), 'enter': (0x0D,), 'escape': (0x1B,),
                'ctrl-enter': (0x11, 0x0D)}
    if method not in ('close-button', 'drag-resize', *keyboard):
        raise ValueError('unknown desktop input method')
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
    window_rect = api('GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
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
    def wait_idle():
        # SendInput completion and OS key-state updates are asynchronous.
        # Wait for release; never inject a release for a user's held input.
        deadline = time.monotonic() + 2
        keys = (0x01, 0x02, 0x04, 0x10, 0x11, 0x12, 0x5B, 0x5C, 0x73, 0x0D, 0x1B)
        while any(pressed(key) & 0x8000 for key in keys):
            if time.monotonic() >= deadline:
                raise RuntimeError('a mouse button, modifier, F4, Return or Esc remains held; no desktop input sent')
            time.sleep(0.02)

    wait_idle()
    activate(hwnd)
    focus_attempts, attach_errors = 0, []
    deadline = time.monotonic() + 2
    while foreground() != hwnd and time.monotonic() < deadline:
        # A background CLI may need a temporary input-queue attachment to
        # activate its child. Always detach before sending desktop input.
        kernel = c.WinDLL('kernel32', use_last_error=True)
        kernel.GetCurrentThreadId.restype = w.DWORD
        kernel.GetCurrentThreadId.argtypes = []
        current_thread = kernel.GetCurrentThreadId()
        queued = w.MSG()
        peek = api('PeekMessageW', w.BOOL, c.POINTER(w.MSG), w.HWND, w.UINT, w.UINT, w.UINT)
        peek(c.byref(queued), None, 0, 0, 0)
        active_thread = owner(foreground(), None)
        target_thread = owner(hwnd, None)
        attach = api('AttachThreadInput', w.BOOL, w.DWORD, w.DWORD, w.BOOL)
        bring_to_top = api('BringWindowToTop', w.BOOL, w.HWND)
        attached = []
        focus_attempts += 1
        try:
            for thread in dict.fromkeys((active_thread, target_thread)):
                if thread and thread != current_thread:
                    if not attach(current_thread, thread, True):
                        # The foreground may have changed or be protected.
                        # This must not prevent an independent owned-window
                        # activation attempt; input still requires ownership.
                        attach_errors.append(c.get_last_error())
                        continue
                    attached.append(thread)
            bring_to_top(hwnd)
            activate(hwnd)
        finally:
            detached = [attach(current_thread, thread, False) for thread in reversed(attached)]
            if not all(detached):
                raise RuntimeError('cannot detach temporary foreground input queue')
        if foreground() != hwnd:
            time.sleep(0.05)
    if foreground() != hwnd:
        raise RuntimeError(f'owned GUI did not acquire foreground; attach errors={attach_errors}; no desktop input sent')
    # Queue detachment/activation can complete after SetForegroundWindow returns.
    # Require a stable owned foreground before beginning any input sequence.
    deadline = time.monotonic() + 0.1
    while time.monotonic() < deadline:
        if foreground() != hwnd:
            raise RuntimeError('owned GUI lost foreground during activation; no desktop input sent')
        time.sleep(0.01)
    # Do not combine our shortcut/click with keys or buttons held by the user.
    wait_idle()

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

    receipt = dict(method=method, pid=pid, hwnd=hwnd, foregroundVerified=True,
                   focusAttempts=focus_attempts, attachErrors=attach_errors)
    if method in keyboard:
        def key(vk, flags=0):
            return Input(1, Payload(keyboard=Keyboard(vk, 0, flags, 0, 0)))
        if focus_control is not None:
            focus = c.c_size_t()
            if not message(hwnd, 0x8001, 7, 0, 3, 2000, c.byref(focus)) or focus.value != focus_control:
                raise RuntimeError('owned GUI keyboard focus does not match the requested control')
        codes = keyboard[method]
        releases = [key(vk, 2) for vk in reversed(codes)]
        submit([key(vk) for vk in codes] + releases, releases)
        yield dict(receipt, injectedEvents=2 * len(codes))
        return

    previous_dpi = dpi(c.c_void_p(-4))  # Per-monitor aware v2, physical pixels.
    if not previous_dpi:
        raise RuntimeError('cannot use physical desktop coordinates')
    old, current = w.POINT(), w.POINT()
    point = None
    mouse_held = False
    try:
        if not cursor(c.byref(old)):
            raise RuntimeError('cannot read cursor position')
        result = c.c_size_t()
        if method == 'drag-resize':
            rect = w.RECT()
            if not window_rect(hwnd, c.byref(rect)):
                raise RuntimeError('cannot query native resize border')
            x, y = rect.right - 2, rect.bottom - 2
            hit, hit_name = 17, 'HTBOTTOMRIGHT'
        else:
            title = TitleBar()
            title.size = c.sizeof(title)
            if not message(hwnd, 0x033F, 0, c.addressof(title), 3, 2000, c.byref(result)):
                raise RuntimeError('cannot query native title-bar buttons')
            rect = title.rects[5]
            if title.states[5] & (0x8000 | 0x10000 | 1) or rect.right <= rect.left or rect.bottom <= rect.top:
                raise RuntimeError('native close button is unavailable')
            x, y = (rect.left + rect.right) // 2, (rect.top + rect.bottom) // 2
            hit, hit_name = 20, 'HTCLOSE'
        left, top, width, height = [metric(index) for index in (76, 77, 78, 79)]
        up = Input(0, Payload(mouse=Mouse(0, 0, 0, 4, 0, 0)))

        def pointer_input(x, y, flags, extra=0):
            if width <= 1 or height <= 1 or not (left <= x < left + width and top <= y < top + height):
                raise RuntimeError('mouse target lies outside the virtual desktop')
            normalized_x = (x - left) * 65535 // (width - 1)
            normalized_y = (y - top) * 65535 // (height - 1)
            return Input(0, Payload(mouse=Mouse(normalized_x, normalized_y, 0, flags, 0, extra)))

        def move_to(x, y):
            nonlocal point
            move = pointer_input(x, y, 0xC001)
            submit([move], [up])
            point = (x, y)
            deadline = time.monotonic() + 1
            while True:
                if not cursor(c.byref(current)):
                    raise RuntimeError('cannot read cursor during desktop input')
                if abs(current.x - x) <= 1 and abs(current.y - y) <= 1:
                    break
                if time.monotonic() >= deadline:
                    raise RuntimeError(f'cursor did not reach native target {(x, y)}; observed {(current.x, current.y)}')
                time.sleep(0.01)

        def probe(index):
            observed = c.c_size_t()
            if not message(hwnd, 0x8001, index, 0, 3, 2000, c.byref(observed)):
                raise RuntimeError('native input delivery probe failed')
            return observed.value

        if probe(0) != 0x46594931:
            raise RuntimeError('rebuild this example for native desktop input observation')
        before_hover = probe(5)
        move_to(x, y)
        if ancestor(at_point(current), 2) != hwnd:
            raise RuntimeError('native input target is occluded; mouse input aborted')
        packed = (current.x & 0xFFFF) | ((current.y & 0xFFFF) << 16)
        if not message(hwnd, 0x0084, 0, packed, 3, 2000, c.byref(result)) or result.value != hit:
            raise RuntimeError('native hit test did not identify ' + hit_name + '; mouse input aborted')

        deadline = time.monotonic() + 2
        while probe(5) <= before_hover or probe(1) != hit:
            require_target()
            if time.monotonic() >= deadline:
                raise RuntimeError('native mouse hover was not delivered to ' + hit_name)
            time.sleep(0.02)
        before = probe(2)
        token = (time.monotonic_ns() & 0x7FFFFFFF) | 1
        if not cursor(c.byref(current)) or abs(current.x - x) > 1 or abs(current.y - y) > 1:
            raise RuntimeError('cursor moved away from verified mouse target; no press sent')
        # Include absolute positioning in the press itself: an earlier move is
        # insufficient if the cursor changes between observation and SendInput.
        down = pointer_input(x, y, 0xC003, token)
        submit([down], [up])
        mouse_held = True
        deadline = time.monotonic() + 2
        while probe(2) <= before:
            require_target()
            if time.monotonic() >= deadline:
                raise RuntimeError(f'native non-client mouse press was not delivered; message={probe(3):#x}, hit={probe(4)}, capture={probe(6)}')
            time.sleep(0.02)
        packed_press = probe(8)
        receipt.update(hoverDelivered=True, pressDelivered=True, pressMessage=probe(3), pressHit=probe(4),
                       targetPoint=[x, y], pressPoint=[c.c_short(packed_press & 0xFFFF).value, c.c_short((packed_press >> 16) & 0xFFFF).value],
                       inputToken=token, pressToken=probe(9),
                       pressWindow=[c.c_ssize_t(probe(i)).value for i in range(10, 14)])
        if receipt['pressHit'] != hit:
            cursor(c.byref(current))
            raise RuntimeError(f'native press hit {receipt["pressHit"]}, expected {hit_name}; cursor={[current.x, current.y]}; observation={receipt}')
        if receipt['pressToken'] != token or any(abs(a - b) > 1 for a, b in zip(receipt['pressPoint'], (x, y))):
            raise RuntimeError(f'native press does not match injected input; observation={receipt}')
        if method == 'drag-resize':
            if receipt['pressMessage'] != 0x00A1:
                raise RuntimeError(f'native drag press was not WM_NCLBUTTONDOWN: {receipt["pressMessage"]:#x}')
            yield SimpleNamespace(receipt=dict(receipt, resizeBorder=[x, y], hitTest=hit_name),
                                  move_to=move_to, start=(x, y))
        else:
            submit([pointer_input(x, y, 0xC005, token)], [up])
            mouse_held = False
            yield dict(receipt, injectedEvents=3, closeButton=[x, y], hitTest=hit_name)
    finally:
        try:
            if mouse_held:
                cleanup = (Input * 1)(up)
                if send(1, cleanup, c.sizeof(Input)) != 1:
                    raise RuntimeError('cannot release injected drag button')
                if method == 'drag-resize' and released is not None:
                    released()
        finally:
            if point is not None and cursor(c.byref(current)):
                if abs(current.x - point[0]) <= 1 and abs(current.y - point[1]) <= 1:
                    restore(old.x, old.y)
            dpi(previous_dpi)


def close_input(pid, method):
    return window_input(pid, method)


def drag_input(pid, released):
    return window_input(pid, 'drag-resize', released)


def key_input(pid, method, focus_control):
    return window_input(pid, method, focus_control=focus_control)
