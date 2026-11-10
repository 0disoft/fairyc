"""Verify the actual distribution image externally, without its private JSONL/probes."""
import argparse
import ctypes as c
from ctypes import wintypes as w
import json
from pathlib import Path
import subprocess
import time

from desktop_input import window_input
from size_report import inspect


class ThreadInfo(c.Structure):
    _fields_ = [('size', w.DWORD), ('flags', w.DWORD), ('active', w.HWND),
                ('focus', w.HWND), ('capture', w.HWND), ('menu', w.HWND),
                ('move', w.HWND), ('caret', w.HWND), ('caretRect', w.RECT)]


def verify(image, launch_flag=None):
    measurement = inspect(image)
    imported = {name for values in measurement['imports'].values() for name in values}
    # WriteFile remains in the static CRT/runtime and does not prove that the
    # private automation path is present. Require absence of its specific APIs.
    for name in ('PeekNamedPipe', 'ReadFile', 'SetTimer', 'KillTimer',
                 'ImmGetOpenStatus', 'ImmGetConversionStatus', 'GetMessageExtraInfo'):
        if name in imported:
            raise RuntimeError('distribution retains diagnostic import: ' + name)
    raw = image.read_bytes()
    for marker in (b'fairyc.win32-example-control/v1', b'dpi-test\t',
                   '--automation-stdio'.encode('utf-16-le'), '--smoke'.encode('utf-16-le')):
        if marker in raw:
            raise RuntimeError('distribution retains a diagnostic protocol/argument')
    user = c.WinDLL('user32', use_last_error=True)
    def api(name, result, *types):
        fn = getattr(user, name)
        fn.restype, fn.argtypes = result, types
        return fn
    enum_type = c.WINFUNCTYPE(w.BOOL, w.HWND, w.LPARAM)
    enum = api('EnumWindows', w.BOOL, enum_type, w.LPARAM)
    owner = api('GetWindowThreadProcessId', w.DWORD, w.HWND, c.POINTER(w.DWORD))
    visible = api('IsWindowVisible', w.BOOL, w.HWND)
    name = api('GetClassNameW', c.c_int, w.HWND, w.LPWSTR, c.c_int)
    child = api('GetDlgItem', w.HWND, w.HWND, c.c_int)
    info = api('GetGUIThreadInfo', w.BOOL, w.DWORD, c.POINTER(ThreadInfo))
    send = api('SendMessageTimeoutW', c.c_ssize_t, w.HWND, w.UINT, w.WPARAM, w.LPARAM,
               w.UINT, w.UINT, c.POINTER(c.c_size_t))
    resize = api('SetWindowPos', w.BOOL, w.HWND, w.HWND, c.c_int, c.c_int, c.c_int, c.c_int, w.UINT)
    rect = api('GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
    map_points = api('MapWindowPoints', c.c_int, w.HWND, w.HWND, c.POINTER(w.POINT), w.UINT)
    get_dpi = api('GetDpiForWindow', w.UINT, w.HWND)
    awareness = api('SetThreadDpiAwarenessContext', c.c_void_p, c.c_void_p)
    previous_dpi = awareness(c.c_void_p(-4))
    if not previous_dpi:
        raise RuntimeError('cannot use physical per-monitor coordinates')
    process = subprocess.Popen([str(image)] + ([launch_flag] if launch_flag else []),
                               stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    events = []
    def wait(read, predicate, label):
        deadline = time.monotonic() + 5
        while True:
            value = read()
            if predicate(value):
                return value
            if time.monotonic() >= deadline or process.poll() is not None:
                raise RuntimeError(label + ': ' + repr(value) + '; processExit=' + str(process.poll()) + '; events=' + json.dumps(events))
            time.sleep(0.03)
    try:
        windows = []
        @enum_type
        def visit(hwnd, unused):
            pid = w.DWORD()
            owner(hwnd, c.byref(pid))
            title = c.create_unicode_buffer(64)
            name(hwnd, title, len(title))
            if pid.value == process.pid and visible(hwnd) and title.value == 'FairyMinimalWindow':
                windows.append(hwnd)
            return True
        def find():
            windows.clear()
            enum(visit, 0)
            return list(windows)
        hwnd = wait(find, lambda v: len(v) == 1, 'owned visible window missing')[0]
        thread = owner(hwnd, None)
        def message(target, code, wp=0, lp=0):
            result = c.c_size_t()
            if not send(target, code, wp, lp, 3, 2000, c.byref(result)):
                raise RuntimeError('native read/control request timed out')
            return result.value
        def text(target):
            value = c.create_unicode_buffer(256)
            message(target, 0x000D, len(value), c.addressof(value))
            return value.value
        def focus():
            value = ThreadInfo(size=c.sizeof(ThreadInfo))
            if not info(thread, c.byref(value)):
                raise RuntimeError('native focus query failed')
            return value.focus
        def key(method, strokes=None):
            before = focus()
            with window_input(process.pid, method, strokes=strokes) as delivery:
                receipt = delivery.receipt if strokes else delivery
                time.sleep(0.08)
                events.append(dict(receipt, focusBefore=before, focusAfter=None if method == 'escape' else focus()))
        edit, apply, result = (child(hwnd, i) for i in (201, 202, 203))
        if text(hwnd) != 'Fairy C - Win32' or not all((edit, apply, result)):
            raise RuntimeError('native title/controls missing')
        if message(hwnd, 0x8001, 0) != 0:
            raise RuntimeError('private diagnostic probe remains enabled')
        wait(focus, lambda v: v == edit, 'initial input focus')
        sample = c.create_unicode_buffer('제로디 "배포" \\ Ω')
        if not message(edit, 0x000C, 0, c.addressof(sample)):
            raise RuntimeError('native Unicode edit failed')
        message(apply, 0x00F5)  # BM_CLICK exercises the ordinary native button.
        wait(lambda: text(result), lambda v: v == 'Result: ' + sample.value, 'Fairy Apply result')
        # A real native button activation can focus the button. Return through
        # Fairy's Tab policy before testing Enter from the edit.
        if focus() == apply:
            key('tab')
        wait(focus, lambda v: v == edit, 'input focus after native button activation')
        key('enter')
        wait(lambda: text(child(hwnd, 204)), lambda v: v == 'WM_KEYDOWN: Enter', 'Fairy Enter status')
        key('tab')
        wait(focus, lambda v: v == apply, 'Fairy Tab focus (expected HWND ' + str(apply) + ')')
        key('shift-tab')
        wait(focus, lambda v: v == edit, 'Fairy Shift+Tab focus')
        # Actual two-set keys determine the owned session's initial Hangul mode.
        empty = c.create_unicode_buffer('')
        message(edit, 0x000C, 0, c.addressof(empty))
        key('ime-strokes', [ord(v.upper()) for v in 'gksrmf'])
        typed = wait(lambda: text(edit), lambda v: v in ('gksrmf', '한글'), 'two-set input')
        toggled = typed == 'gksrmf'
        if toggled:
            key('ime-strokes', [21])
            message(edit, 0x000C, 0, c.addressof(empty))
            key('ime-strokes', [ord(v.upper()) for v in 'gksrmf'])
            wait(lambda: text(edit), lambda v: v == '한글', 'actual Hangul input')
        # An in-progress syllable must consume the first Enter, not Apply.
        key('enter')
        time.sleep(0.2)
        if text(result) != 'Result: ' + sample.value or focus() != edit:
            raise RuntimeError('IME Enter collided with application action')
        key('enter')
        wait(lambda: text(result), lambda v: v == 'Result: 한글', 'Enter after IME completion')
        if toggled:
            key('ime-strokes', [21])
        if not resize(hwnd, None, 0, 0, 800, 480, 0x16):
            raise RuntimeError('native resize failed')
        def input_rect():
            value = w.RECT()
            if not rect(edit, c.byref(value)):
                raise RuntimeError('cannot measure edit rectangle')
            map_points(None, hwnd, c.cast(c.byref(value), c.POINTER(w.POINT)), 2)
            return [value.left, value.top, value.right - value.left, value.bottom - value.top]
        dpi = get_dpi(hwnd)
        scaled = lambda n: (n * dpi + 48) // 96
        get_client = api('GetClientRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
        client = w.RECT()
        if not get_client(hwnd, c.byref(client)):
            raise RuntimeError('client rectangle unavailable')
        expected = [scaled(16), scaled(44), max(scaled(320), client.right) - scaled(132), scaled(28)]
        actual = wait(input_rect, lambda v: v == expected, 'Fairy resized input layout')
        key('escape')
        if process.wait(timeout=5) != 0:
            raise RuntimeError('Fairy Esc close failed')
        stdout, stderr = process.communicate(timeout=2)
        if stdout or stderr:
            raise RuntimeError('distribution unexpectedly wrote a diagnostic pipe')
        return dict(ok=True, verification='external-win32-distribution', imageSha256=measurement['sha256'],
                    fileBytes=measurement['fileBytes'], launchFlag=launch_flag,
                    diagnosticImportsAbsent=True, privateProbeAbsent=True, pipeOutputBytes=0,
                    input='한글', resizedInput=actual, desktopInput=events, processExited=True)
    finally:
        if process.poll() is None:
            process.terminate()
            process.wait(timeout=5)
        for stream in (process.stdin, process.stdout, process.stderr):
            stream.close()
        awareness(previous_dpi)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent / 'build/win32-minimal-distribution.exe')
    parser.add_argument('--launch-flag', choices=('smoke', 'automation-stdio'))
    args = parser.parse_args()
    try:
        print(json.dumps(verify(args.exe.resolve(), '--' + args.launch_flag if args.launch_flag else None), ensure_ascii=True))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
