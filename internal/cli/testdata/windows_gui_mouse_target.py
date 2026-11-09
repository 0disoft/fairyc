"""Perturb the real cursor immediately before injected button transitions."""
import ctypes as c
from ctypes import wintypes as w
import json
from pathlib import Path
import sys

sys.path.insert(0, sys.argv[1])
import desktop_input
from control import Session


def verify(session):
    original = c.WinDLL
    user = original('user32', use_last_error=True)

    def api(name, result, *args):
        fn = getattr(user, name)
        fn.restype, fn.argtypes = result, args
        return fn

    callback = c.WINFUNCTYPE(w.BOOL, w.HWND, w.LPARAM)
    owner = api('GetWindowThreadProcessId', w.DWORD, w.HWND, c.POINTER(w.DWORD))
    visible = api('IsWindowVisible', w.BOOL, w.HWND)
    rect = api('GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
    metric = api('GetSystemMetrics', c.c_int, c.c_int)
    hit_test = api('SendMessageTimeoutW', c.c_ssize_t, w.HWND, w.UINT,
                   w.WPARAM, w.LPARAM, w.UINT, w.UINT, c.POINTER(c.c_size_t))
    targets = []

    @callback
    def visit(hwnd, unused):
        pid = w.DWORD()
        owner(hwnd, c.byref(pid))
        if pid.value == session.process.pid and visible(hwnd):
            targets.append(hwnd)
        return True

    if not api('EnumWindows', w.BOOL, callback, w.LPARAM)(visit, 0) or len(targets) != 1:
        raise RuntimeError('expected one owned window for cursor perturbation')
    target = targets[0]
    transitions = []

    class Intercept:
        def __init__(self, function):
            object.__setattr__(self, 'function', function)

        def __getattr__(self, name):
            return getattr(self.function, name)

        def __setattr__(self, name, value):
            setattr(self.function, name, value)

        def __call__(self, count, events, size):
            for event in events:
                if event.kind != 0 or not event.payload.mouse.flags & 6:
                    continue
                transition = 'press' if event.payload.mouse.flags & 2 else 'release'
                if transition in transitions:
                    continue
                frame, hit = w.RECT(), c.c_size_t()
                if not rect(target, c.byref(frame)):
                    raise RuntimeError('cannot read owned window before perturbation')
                x, y = frame.right - 2, (frame.top + frame.bottom) // 2
                packed = (x & 0xFFFF) | ((y & 0xFFFF) << 16)
                if not hit_test(target, 0x84, 0, packed, 3, 2000, c.byref(hit)) or hit.value != 11:
                    raise RuntimeError('perturbation target must be the owned HTRIGHT border')
                left, top, width, height = [metric(i) for i in (76, 77, 78, 79)]
                move = desktop_input.Input(0, desktop_input.Payload(mouse=desktop_input.Mouse(
                    (x - left) * 65535 // (width - 1), (y - top) * 65535 // (height - 1), 0, 0xE001, 0, 0)))
                batch = (desktop_input.Input * 1)(move)
                if self.function(1, batch, c.sizeof(desktop_input.Input)) != 1:
                    raise RuntimeError('cannot perturb real cursor before input')
                transitions.append(transition)
            return self.function(count, events, size)

    class Proxy:
        send = Intercept(user.SendInput)

        def __getattr__(self, name):
            return self.send if name == 'SendInput' else getattr(user, name)

    c.WinDLL = lambda name, *args, **kwargs: Proxy() if name == 'user32' else original(name, *args, **kwargs)
    try:
        observation = session.request(dict(command='close-button'))
        if not observation.get('closed') or session.process.poll() != 0 or transitions != ['press', 'release']:
            raise RuntimeError('targeted close must survive both real cursor perturbations')
        return dict(ok=True, verification='win32-mouse-target', perturbations=transitions, observation=observation)
    except Exception as error:
        raise RuntimeError(f'{error}; perturbations={transitions}') from error
    finally:
        c.WinDLL = original


session = None
try:
    session = Session(Path(sys.argv[2]).resolve())
    result = verify(session)
except Exception as error:
    print(json.dumps(dict(ok=False, error=str(error)), ensure_ascii=False))
    raise SystemExit(1)
else:
    print(json.dumps(result, ensure_ascii=False))
finally:
    if session is not None:
        session.finish()
