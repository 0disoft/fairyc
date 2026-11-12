"""Actual keyboard and native accessibility checks on the owned Hash utility."""
import ctypes as c
from ctypes import wintypes as w
import hashlib
import json
import time
from native_keyboard import DesktopKeys
from clipboard_snapshot import preserve_clipboard
from accessibility_probe import inspect


def clipboard_text():
    user, kernel = c.WinDLL('user32'), c.WinDLL('kernel32')
    user.OpenClipboard.argtypes, user.OpenClipboard.restype = [w.HWND], w.BOOL
    user.GetClipboardData.argtypes, user.GetClipboardData.restype = [w.UINT], w.HANDLE
    kernel.GlobalLock.argtypes, kernel.GlobalLock.restype = [w.HANDLE], c.c_void_p
    kernel.GlobalUnlock.argtypes = [w.HANDLE]
    if not user.OpenClipboard(None):
        return None
    try:
        handle = user.GetClipboardData(13)
        pointer = kernel.GlobalLock(handle) if handle else None
        if not pointer:
            return None
        try:
            return c.wstring_at(pointer)
        finally:
            kernel.GlobalUnlock(handle)
    finally:
        user.CloseClipboard()


def keyboard(driver):
    d, keys = driver, DesktopKeys(driver.process.pid)
    keys.activate(d.main)
    d.wait(lambda: keys.focus(d.main), lambda v: v == 202, 'initial Open focus')
    trace = []
    def focus_after(chord, expected):
        keys.chord(d.main, chord)
        def observe():
            if keys.foreground() != d.main:
                raise RuntimeError(f'foreground lost after keys={chord}, trace={trace}, focus={keys.focus(d.main)}, actual={keys.describe_foreground()}')
            return keys.focus(d.main)
        actual = d.wait(observe, lambda v: v == expected, f'keyboard focus expected {expected}, trace={trace}')
        trace.append(actual)
    for identifier in (201, 203, 202):
        focus_after([9], identifier)
    for identifier in (203, 201, 202):
        focus_after([16, 9], identifier)
    def opened():
        return d.wait(lambda: d.windows('#32770'), lambda v: len(v) == 1, 'keyboard did not open native dialog')[0]
    def accept(hwnd):
        keys.chord(hwnd, [18, 78])  # Native Korean/English filename Alt+N.
        keys.chord(hwnd, [13])
    def ready(expected):
        d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'keyboard-selected file did not finish')
        if d.text(d.child(d.main, 203)).replace('\r\n', '') != expected:
            raise RuntimeError('keyboard-selected hash mismatch')
    source = d.root / '키보드-입력&.txt'
    source.write_bytes(b'abc')
    expected = hashlib.sha256(b'abc').hexdigest()
    keys.chord(d.main, [13])
    d.choose(202, source, active=opened(), accept=accept)
    ready(expected)
    for identifier in (201, 203, 208, 204, 202):
        focus_after([9], identifier)
    for identifier in (204, 208, 203, 201, 202):
        focus_after([16, 9], identifier)
    with preserve_clipboard():
        for identifier in (201, 203, 208):
            focus_after([9], identifier)
        keys.chord(d.main, [13])  # Enter on focused Copy, not default Open.
        d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v == '✓ 해시 복사 완료', 'Enter Copy status missing')
        d.wait(clipboard_text, lambda v: v == expected, 'Enter did not activate focused Copy')
        focus_after([16, 9], 203)
        keys.chord(d.main, [18, 67])
        d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v == '✓ 해시 복사 완료', 'Alt+C did not copy')
        d.wait(clipboard_text, lambda v: v == expected, 'Alt+C clipboard mismatch')
    target = d.root / '키보드-보고서.json'
    keys.chord(d.main, [18, 83])
    d.choose(204, target, active=opened(), accept=accept)
    d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v == '✓ 결과 저장 완료', 'Alt+S/Enter did not save report')
    d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'keyboard report cleanup unfinished')
    if json.loads(target.read_bytes())['sha256'] != expected:
        raise RuntimeError('keyboard report differs from computed digest')
    # Reach owner-drawn Save with Tab, then Enter and real Escape cancellation.
    current = keys.focus(d.main)
    for _ in range(6):
        if current == 204:
            break
        keys.chord(d.main, [9])
        time.sleep(0.05)
        current = keys.focus(d.main)
    if current != 204:
        raise RuntimeError('owner-drawn Save is absent from tab navigation')
    keys.chord(d.main, [13])
    active = opened()
    keys.chord(active, [27])
    d.wait(lambda: d.windows('#32770'), lambda v: not v, 'Escape did not cancel native Save dialog')
    large = d.root / 'keyboard-cancel.bin'
    with large.open('wb') as stream:
        for _ in range(256):
            stream.write(bytes(range(256))*4096)
    for shortcut in ([13], [18, 88]):
        keys.chord(d.main, [18, 79])
        d.choose(202, large, active=opened(), accept=accept)
        d.wait(lambda: bool(d.visible(d.child(d.main, 212))), bool, 'keyboard Cancel not visible')
        d.wait(lambda: keys.focus(d.main), lambda v: v == 212, 'Cancel did not receive focus')
        for identifier in (201, 203, 212):
            focus_after([9], identifier)
        for identifier in (203, 201, 212):
            focus_after([16, 9], identifier)
        keys.chord(d.main, shortcut)
        d.wait(lambda: bool(d.enabled(d.child(d.main, 202))), bool, 'keyboard cancellation did not restore Open')
        if not d.text(d.child(d.main, 205)).startswith('취소됨') or d.process.poll() is not None:
            raise RuntimeError('keyboard cancellation lost window/state')
    return dict(actualSendInput=True, focusTrace=trace, shortcuts=['Alt+O', 'Alt+C', 'Alt+S', 'Alt+X'],
                enterButtons=['Open', 'Copy', 'Save', 'Cancel'], receipts=keys.receipts)


def accessibility(driver):
    user = c.WinDLL('user32')
    next_tab = user.GetNextDlgTabItem
    next_tab.argtypes, next_tab.restype = [w.HWND, w.HWND, w.BOOL], w.HWND
    control_id = user.GetDlgCtrlID
    control_id.argtypes, control_id.restype = [w.HWND], c.c_int
    def order(start, expected, reverse=False):
        control, observed = driver.child(driver.main, start), []
        for _ in expected:
            control = next_tab(driver.main, control, reverse)
            observed.append(control_id(control))
        if observed != expected:
            raise RuntimeError(f'native tab order mismatch: {observed} expected {expected}')
        return observed
    result = inspect(driver.main, driver.process.pid)
    tab_orders = dict(initial=order(202, [201, 203, 202]),
                      initialReverse=order(202, [203, 201, 202], True))
    rows = {row['id']: row for row in result['controls']}
    for identifier, role in (('202', 50000), ('201', 50004), ('203', 50004), ('213', 50012)):
        if rows[identifier]['type'] != role or not rows[identifier]['name']:
            raise RuntimeError(f'accessibility role/name missing: control={identifier}, observed={rows[identifier]}')
    if not rows['201']['valuePattern'] or not rows['203']['valuePattern'] or not rows['201']['readOnly'] or not rows['203']['readOnly']:
        raise RuntimeError('read-only UI fields were exposed as writable')
    if not rows['213']['rangePattern'] or rows['213']['rangeMin'] != 0 or rows['213']['rangeMax'] != 100:
        raise RuntimeError('accessible progress range invalid')
    for identifier, shortcut in (('202', 'Alt+o'), ('204', 'Alt+s'), ('208', 'Alt+c')):
        if rows[identifier]['accessKey'].lower() != shortcut.lower():
            raise RuntimeError(f'accessibility shortcut missing: {identifier}')
    if rows['204']['enabled'] or rows['208']['enabled']:
        raise RuntimeError('initial inaccessible Save/Copy state')
    source = driver.root/'접근성-파일.txt'
    source.write_bytes(b'abc')
    driver.choose(202, source)
    driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 204))), bool, 'accessibility fixture did not finish')
    ready = inspect(driver.main, driver.process.pid)
    after = {row['id']: row for row in ready['controls']}
    if after['201']['value'] != str(source) or after['203']['value'].replace('\r\n', '') != hashlib.sha256(b'abc').hexdigest():
        raise RuntimeError('accessible values do not match actual selected file/hash')
    if after['213']['rangeValue'] != 100 or not after['204']['enabled'] or not after['208']['enabled']:
        raise RuntimeError('accessible completed state invalid')
    tab_orders['completed'] = order(202, [201, 203, 208, 204, 202])
    tab_orders['completedReverse'] = order(202, [204, 208, 203, 201, 202], True)
    large = driver.root/'accessible-busy.bin'
    with large.open('wb') as stream:
        stream.seek(64*1048576-1)
        stream.write(b'\0')
    driver.choose(202, large)
    driver.wait(lambda: bool(driver.visible(driver.child(driver.main, 212))), bool, 'accessible busy state missing')
    busy = inspect(driver.main, driver.process.pid)
    active = {row['id']: row for row in busy['controls']}
    if not active['212']['enabled'] or not active['212']['name'] or active['212']['accessKey'].lower() != 'alt+x':
        raise RuntimeError('Cancel accessible name/action missing')
    if active['204']['enabled'] or active['208']['enabled']:
        raise RuntimeError('accessible busy state exposed partial-result actions')
    tab_orders['busy'] = order(212, [201, 203, 212])
    tab_orders['busyReverse'] = order(212, [203, 201, 212], True)
    driver.message(driver.child(driver.main, 212), 0xf5)
    driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 202))), bool, 'accessible cancel did not return to idle')
    result['completed'] = ready['controls']
    result['busy'] = busy['controls']
    result['screenReaderVoiceTested'] = False
    result['nativeTabOrders'] = tab_orders
    result['nativeOrderIsPhysicalInput'] = False
    return result


def run(driver, mode, high_contrast=False):
    result = {}
    if mode in ('keyboard', 'all'):
        result['keyboard'] = keyboard(driver)
    if mode in ('accessibility', 'all'):
        result['accessibility'] = accessibility(driver)
    if mode in ('display', 'all'):
        from display_probe import verify
        result['display'] = verify(driver, high_contrast)
    return result
