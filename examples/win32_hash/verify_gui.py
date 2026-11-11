"""Exercise real Open/Save dialogs and Fairy's hash workflow from a CLI."""
import argparse
import ctypes as c
from ctypes import wintypes as w
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time
from contextlib import ExitStack
from clipboard_snapshot import preserve_clipboard


def themed_controls(pid):
    """Observe the app's loaded v6 Common Controls, not just manifest text."""
    kernel, psapi = c.WinDLL('kernel32', use_last_error=True), c.WinDLL('psapi', use_last_error=True)
    kernel.OpenProcess.argtypes, kernel.OpenProcess.restype = [w.DWORD, w.BOOL, w.DWORD], w.HANDLE
    kernel.CloseHandle.argtypes = [w.HANDLE]
    psapi.EnumProcessModules.argtypes = [w.HANDLE, c.POINTER(w.HMODULE), w.DWORD, c.POINTER(w.DWORD)]
    psapi.GetModuleFileNameExW.argtypes = [w.HANDLE, w.HMODULE, w.LPWSTR, w.DWORD]
    process = kernel.OpenProcess(0x410, False, pid)
    if not process:
        raise RuntimeError('cannot inspect owned GUI modules')
    try:
        modules, needed = (w.HMODULE * 256)(), w.DWORD()
        if not psapi.EnumProcessModules(process, modules, c.sizeof(modules), c.byref(needed)) or needed.value > c.sizeof(modules):
            raise RuntimeError('cannot enumerate owned GUI modules')
        for module in modules[:needed.value // c.sizeof(w.HMODULE)]:
            path = c.create_unicode_buffer(32768)
            if psapi.GetModuleFileNameExW(process, module, path, len(path)) and (
                    'microsoft.windows.common-controls' in path.value.lower() and path.value.lower().endswith('comctl32.dll')):
                return True
        return False
    finally:
        kernel.CloseHandle(process)


def verify(image, screenshot=None):
    user = c.WinDLL('user32', use_last_error=True)
    def api(name, result, *types):
        fn = getattr(user, name)
        fn.restype, fn.argtypes = result, types
        return fn
    callback = c.WINFUNCTYPE(w.BOOL, w.HWND, w.LPARAM)
    enum = api('EnumWindows', w.BOOL, callback, w.LPARAM)
    children = api('EnumChildWindows', w.BOOL, w.HWND, callback, w.LPARAM)
    owner = api('GetWindowThreadProcessId', w.DWORD, w.HWND, c.POINTER(w.DWORD))
    visible = api('IsWindowVisible', w.BOOL, w.HWND)
    enabled = api('IsWindowEnabled', w.BOOL, w.HWND)
    child = api('GetDlgItem', w.HWND, w.HWND, c.c_int)
    ident = api('GetDlgCtrlID', c.c_int, w.HWND)
    parent = api('GetParent', w.HWND, w.HWND)
    class_name = api('GetClassNameW', c.c_int, w.HWND, w.LPWSTR, c.c_int)
    send = api('SendMessageTimeoutW', c.c_ssize_t, w.HWND, w.UINT, w.WPARAM, w.LPARAM,
               w.UINT, w.UINT, c.POINTER(c.c_size_t))
    post = api('PostMessageW', w.BOOL, w.HWND, w.UINT, w.WPARAM, w.LPARAM)
    resize = api('SetWindowPos', w.BOOL, w.HWND, w.HWND, c.c_int, c.c_int, c.c_int, c.c_int, w.UINT)
    window_rect = api('GetWindowRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
    client_rect = api('GetClientRect', w.BOOL, w.HWND, c.POINTER(w.RECT))
    map_points = api('MapWindowPoints', c.c_int, w.HWND, w.HWND, c.c_void_p, w.UINT)
    window_dpi = api('GetDpiForWindow', w.UINT, w.HWND)
    show = api('ShowWindow', w.BOOL, w.HWND, c.c_int)
    dpi = api('SetThreadDpiAwarenessContext', c.c_void_p, c.c_void_p)
    previous = dpi(c.c_void_p(-4))
    if not previous:
        raise RuntimeError('cannot use physical desktop coordinates')
    process = None
    rows = []
    try:
        with tempfile.TemporaryDirectory(prefix='fairy-hash-dialog-한글-') as directory, ExitStack() as cleanup:
            root = Path(directory)
            process = subprocess.Popen([str(image)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            def stop_before_fixture_cleanup():
                if process.poll() is None:
                    process.terminate()
                    process.wait(timeout=5)
            cleanup.callback(stop_before_fixture_cleanup)
            def wait(read, accept, label, seconds=12):
                deadline = time.monotonic() + seconds
                while True:
                    value = read()
                    if accept(value):
                        return value
                    if time.monotonic() >= deadline or process.poll() is not None:
                        raise RuntimeError(label + ': ' + repr(value) + '; exit=' + str(process.poll()))
                    time.sleep(0.02)
            def kind(hwnd):
                text = c.create_unicode_buffer(64)
                class_name(hwnd, text, len(text))
                return text.value
            def windows(type_name):
                found = []
                @callback
                def visit(hwnd, unused):
                    pid = w.DWORD()
                    owner(hwnd, c.byref(pid))
                    if pid.value == process.pid and visible(hwnd) and (type_name is None or kind(hwnd) == type_name):
                        found.append(hwnd)
                    return True
                enum(visit, 0)
                return found
            def message(hwnd, code, wp=0, lp=0):
                value = c.c_size_t()
                if not hwnd or not send(hwnd, code, wp, lp, 3, 3000, c.byref(value)):
                    raise RuntimeError(f'owned native control request failed: message={code:#x}, control={ident(hwnd) if hwnd else None}, winError={c.get_last_error()}, exit={process.poll()}')
                return value.value
            def text(hwnd):
                value = c.create_unicode_buffer(32768)
                message(hwnd, 13, len(value), c.addressof(value))
                return value.value
            def click(hwnd):
                message(hwnd, 0xF5)  # BM_CLICK, not a fabricated result/path event.
            main = wait(lambda: windows('FairyHashWindow'), lambda v: len(v) == 1, 'GUI missing')[0]
            if not themed_controls(process.pid):
                raise RuntimeError('GUI did not load side-by-side Common Controls v6')
            if text(main) != 'Fairy Hash' or enabled(child(main, 204)) or enabled(child(main, 208)):
                raise RuntimeError('initial title/save state invalid')
            digest_height = None
            def layout():
                area = w.RECT()
                if not client_rect(main, c.byref(area)):
                    raise RuntimeError('missing client area')
                boxes = []
                for control_id in range(201, 212):
                    control = child(main, control_id)
                    r = w.RECT()
                    if not control or not window_rect(control, c.byref(r)):
                        return False
                    map_points(None, main, c.byref(r), 2)
                    if not (0 <= r.left < r.right <= area.right and 0 <= r.top < r.bottom <= area.bottom):
                        return False
                    for other in boxes:
                        if max(r.left, other.left) < min(r.right, other.right) and max(r.top, other.top) < min(r.bottom, other.bottom):
                            return False
                    boxes.append(r)
                digest, save = boxes[2], boxes[3]  # IDs 203/204, not child order.
                if digest_height is not None and (digest.bottom-digest.top != digest_height or
                        save.top-digest.bottom > round(24*window_dpi(main)/96)):
                    return False
                return abs(min(r.left for r in boxes) - (area.right - max(r.right for r in boxes))) <= 1
            wait(layout, bool, 'initial layout overlaps or clips controls')
            digest_bounds = w.RECT()
            window_rect(child(main, 203), c.byref(digest_bounds))
            digest_height = digest_bounds.bottom-digest_bounds.top
            def dialog(button):
                click(child(main, button))
                return wait(lambda: windows('#32770'), lambda v: len(v) == 1,
                            f'native file dialog missing: button={button}, enabled={enabled(child(main, button))}, status={text(child(main, 205))}')[0]
            def cancel(button):
                active = dialog(button)
                click(child(active, 2))
                wait(lambda: windows('#32770'), lambda v: not v, 'file dialog did not cancel')
            def choose(button, path, overwrite=False):
                active = dialog(button)
                edit = child(active, 1152)
                if kind(edit) != 'Edit':
                    combo = child(active, 1148)
                    candidates = []
                    @callback
                    def visit(hwnd, unused):
                        if kind(hwnd) == 'Edit':
                            candidates.append(hwnd)
                        return True
                    if combo:
                        children(combo, visit, 0)
                    if not candidates:
                        # Windows 11's DirectUI dialog hosts the filename edit
                        # in a ComboBox without the legacy 1148 parent ID.
                        @callback
                        def filename(hwnd, unused):
                            if kind(hwnd) == 'Edit' and ident(hwnd) == 1001 and kind(parent(hwnd)) == 'ComboBox':
                                candidates.append(hwnd)
                            return True
                        children(active, filename, 0)
                    if len(candidates) != 1:
                        structure = []
                        @callback
                        def describe(hwnd, unused):
                            structure.append((ident(hwnd), kind(hwnd)))
                            return True
                        children(active, describe, 0)
                        raise RuntimeError('native filename edit not found: ' + repr(structure))
                    edit = candidates[0]
                value = c.create_unicode_buffer(str(path))
                if not message(edit, 12, 0, c.addressof(value)):
                    raise RuntimeError('native filename edit rejected path')
                # An OK click can synchronously enter the native overwrite
                # confirmation. Post the real button click so this controller
                # can observe/respond to that nested modal dialog.
                if not post(child(active, 1), 0xF5, 0, 0):
                    raise RuntimeError('could not click native dialog OK')
                if overwrite:
                    def confirmations():
                        buttons = []
                        for popup in windows('#32770'):
                            if child(popup, 6):
                                buttons.append(child(popup, 6))
                                continue
                            @callback
                            def yes_button(control, unused):
                                if kind(control) == 'Button':
                                    label = text(control).replace('&', '').split('(')[0].strip().casefold()
                                    if label in ('예', 'yes'):
                                        buttons.append(control)
                                return True
                            children(popup, yes_button, 0)
                        return buttons
                    try:
                        confirmation = wait(confirmations, lambda v: len(v) == 1, 'overwrite confirmation missing')[0]
                    except RuntimeError as error:
                        structures = []
                        for hwnd in windows(None):
                            buttons = []
                            @callback
                            def describe_button(control, unused):
                                if kind(control) == 'Button':
                                    buttons.append((ident(control), text(control)))
                                return True
                            children(hwnd, describe_button, 0)
                            structures.append((kind(hwnd), buttons))
                        raise RuntimeError(str(error) + '; native windows=' + repr(structures)) from error
                    if not post(confirmation, 0xF5, 0, 0):
                        raise RuntimeError('could not confirm owned fixture overwrite')
                wait(lambda: windows('#32770'), lambda v: not v, 'file dialog did not accept path')
            cancel(202)
            if text(child(main, 201)) or text(child(main, 203)) or enabled(child(main, 204)):
                raise RuntimeError('Open cancellation changed state')
            for index, data in enumerate((b'', b'abc', bytes(range(256)) * 4097)):
                source = root / ('입력 Ω-' + str(index) + '-' + '긴이름' * 20 + '.bin')
                source.write_bytes(data)
                choose(202, source)
                expected_hash = hashlib.sha256(data).hexdigest()
                expected_display = expected_hash[:32] + '\r\n' + expected_hash[32:]
                actual_display = wait(lambda: text(child(main, 203)), lambda v: v == expected_display, 'Fairy SHA-256 display')
                actual_hash = actual_display.replace('\r\n', '')
                wait(lambda: bool(enabled(child(main, 204))), bool, 'Save did not enable')
                wait(lambda: bool(enabled(child(main, 208))), bool, 'Copy did not enable')
                if text(child(main, 206)) != source.name or text(child(main, 209)) != f'{len(data)} 바이트':
                    raise RuntimeError('Fairy file summary differs from selected source')
                with preserve_clipboard():
                    open_clipboard = api('OpenClipboard', w.BOOL, w.HWND)
                    if index == 1:
                        wait(lambda: bool(open_clipboard(None)), bool, 'clipboard lock fixture')
                        try:
                            click(child(main, 208))
                            wait(lambda: text(child(main, 205)), lambda v: v == '복사 실패: 다시 시도하세요.', 'busy clipboard error')
                        finally:
                            api('CloseClipboard', w.BOOL)()
                    click(child(main, 208))
                    wait(lambda: text(child(main, 205)), lambda v: v == '✓ 해시 복사 완료', 'Fairy copy status')
                    wait(lambda: bool(open_clipboard(None)), bool, 'clipboard locked')
                    try:
                        data_handle = api('GetClipboardData', w.HANDLE, w.UINT)(13)
                        kernel = c.WinDLL('kernel32', use_last_error=True)
                        kernel.GlobalLock.argtypes, kernel.GlobalLock.restype = [w.HANDLE], c.c_void_p
                        kernel.GlobalUnlock.argtypes = [w.HANDLE]
                        pointer = kernel.GlobalLock(data_handle)
                        if not pointer:
                            raise RuntimeError('clipboard has no Unicode digest')
                        try:
                            if c.wstring_at(pointer) != expected_hash:
                                raise RuntimeError('clipboard digest is not the original 64 hex digits')
                        finally:
                            kernel.GlobalUnlock(data_handle)
                    finally:
                        api('CloseClipboard', w.BOOL)()
                actual_path = text(child(main, 201))
                if len(actual_path) <= 127:
                    raise RuntimeError('fixture did not exercise paths beyond the old edit limit')
                if not os.path.samefile(actual_path, source):
                    raise RuntimeError('displayed path does not identify the selected file')
                expected = dict(algorithm='SHA-256', file=actual_path, bytes=len(data), sha256=expected_hash)
                if index == 0:
                    # The same file at the same wide size must look identical
                    # after expanding and after selecting it while already wide.
                    # Text-only checks miss cached ellipses left in native paint.
                    from capture_window import capture
                    title = child(main, 206)
                    # Only this owned verifier window is temporarily topmost,
                    # keeping visible-pixel reads independent of occlusion.
                    resize(main, w.HWND(-1), 0, 0, 0, 0, 0x13)
                    for cycle in range(2):
                        time.sleep(0.1)
                        show(main, 3)
                        wait(layout, bool, 'maximized layout invalid')
                        time.sleep(0.1)
                        before_title = root / 'expanded-title.png'
                        capture(title, before_title, repaint=False)
                        if screenshot:
                            capture(main, screenshot.with_name(f'{screenshot.stem}-maximized.png'), repaint=False)
                        # Change selection first so readiness cannot be mistaken
                        # for the previous identical file/hash's completed state.
                        reset = root / 'refresh.txt'
                        reset.write_bytes(b'repaint')
                        choose(202, reset)
                        wait(lambda: text(child(main, 201)), lambda v: bool(v) and os.path.samefile(v, reset), 'refresh selection')
                        wait(lambda: bool(enabled(child(main, 202))), bool, 'refresh hash not complete')
                        choose(202, source)
                        wait(lambda: text(child(main, 201)), lambda v: bool(v) and os.path.samefile(v, source), 'wide reselect path')
                        wait(lambda: text(child(main, 203)), lambda v: v == expected_display, 'wide reselect hash')
                        wait(lambda: bool(enabled(child(main, 202))), bool, 'wide reselect not complete')
                        time.sleep(0.1)
                        fresh_title = root / 'fresh-title.png'
                        capture(title, fresh_title, repaint=False)
                        if before_title.read_bytes() != fresh_title.read_bytes():
                            raise RuntimeError('filename paint retains ellipsis from the previous window width')
                        if text(title) != source.name:
                            raise RuntimeError('filename text changed during ellipsis painting')
                        show(main, 9)
                        wait(layout, bool, 'restored filename layout invalid')
                    resize(main, w.HWND(-2), 0, 0, 0, 0, 0x13)
                if index == 1:
                    cancel(202)
                    cancel(204)
                    if text(child(main, 201)) != actual_path or text(child(main, 203)) != actual_display:
                        raise RuntimeError('dialog cancellation discarded a completed digest')
                    if screenshot:
                        # Show a small ordinary filename in the actual app, then
                        # return to the long-path fixture for the remaining checks.
                        sample = root / 'sample.txt'
                        sample.write_bytes(b'abc')
                        choose(202, sample)
                        wait(lambda: text(child(main, 201)), lambda v: bool(v) and os.path.samefile(v, sample), 'screenshot sample selection')
                        wait(lambda: text(child(main, 203)), lambda v: v == expected_display, 'screenshot sample hash')
                        wait(lambda: bool(enabled(child(main, 204))), bool, 'screenshot sample not ready')
                        from capture_window import capture
                        capture(main, screenshot)
                        choose(202, source)
                        wait(lambda: text(child(main, 201)), lambda v: bool(v) and os.path.samefile(v, source), 'long-path selection restore')
                        wait(lambda: text(child(main, 203)), lambda v: v == expected_display, 'long-path restore')
                        wait(lambda: bool(enabled(child(main, 204))), bool, 'restored sample not ready')
                target = root / f'보고서-{index}.json'
                if index == 2:
                    target.write_bytes(b'owned old report')
                choose(204, target, overwrite=index == 2)
                wait(lambda: text(child(main, 205)), lambda v: v == '✓ 결과 저장 완료', 'Fairy save status')
                if json.loads(target.read_bytes()) != expected:
                    raise RuntimeError('saved report differs from selected-file reference')
                rows.append(dict(bytes=len(data), sha256=actual_hash, unicodePath=True, saved=True,
                                 inputPathCharacters=len(actual_path), overwritten=index == 2))
            before = source.read_bytes()
            choose(204, source, overwrite=True)
            wait(lambda: text(child(main, 205)), lambda v: v == '원본 파일에는 덮어쓸 수 없어요.', 'source protection status')
            if source.read_bytes() != before:
                raise RuntimeError('GUI save damaged the source')
            area, outer = w.RECT(), w.RECT()
            client_rect(main, c.byref(area))
            window_rect(main, c.byref(outer))
            scale = window_dpi(main) / 96
            for width, height in ((520, 276), (800, 450)):
                physical_width = round(width*scale) + outer.right-outer.left-area.right
                physical_height = round(height*scale) + outer.bottom-outer.top-area.bottom
                if not resize(main, None, 0, 0, physical_width, physical_height, 0x16):
                    raise RuntimeError('GUI resize failed')
                wait(layout, bool, 'resized layout overlaps or clips controls')
                if text(child(main, 203)) != expected_display or message(child(main, 203), 0xBA) != 2:
                    raise RuntimeError('resize lost/wrapped the two-line digest')
            show(main, 6)  # Minimize must not make a zero-size layout terminate Fairy.
            time.sleep(0.1)
            if process.poll() is not None:
                raise RuntimeError('minimizing terminated the application')
            show(main, 9)
            wait(layout, bool, 'restore layout invalid')
            # A close request must interrupt streaming and release its file.
            large = root / 'close-during-hash.bin'
            with large.open('wb') as stream:
                stream.seek(64 * 1024 * 1024 - 1)
                stream.write(b'\0')
            choose(202, large)
            wait(lambda: bool(enabled(child(main, 202))), lambda v: not v, 'busy controls were not disabled')
            message(main, 0x10)  # Real WM_CLOSE handler; Fairy resolves it after a chunk.
            if process.wait(timeout=12) != 0:
                raise RuntimeError('close during hashing failed')
            stdout, stderr = process.communicate(timeout=2)
            if stdout or stderr:
                raise RuntimeError('ordinary GUI unexpectedly wrote CLI output')
            large.unlink()  # Windows deletion also catches leaked input handles.
            if list(root.glob('*.tmp')):
                raise RuntimeError('temporary report leaked')
    finally:
        if process is not None:
            if process.poll() is None:
                process.terminate()
                process.wait(timeout=5)
            for stream in (process.stdin, process.stdout, process.stderr):
                stream.close()
        dpi(previous)
    return dict(ok=True, verification='actual-win32-hash-dialogs', observations=rows,
                nativeOpenAndSave=True, cancellationPreservesResult=True, sourceProtected=True,
                closeDuringHash=True, twoLineDigest=True, clipboardDigest=True, clipboardRestored=True,
                clipboardBusyRecovery=True, nativeVisualStyles=True, responsiveLayout=True, minimizeRestore=True,
                filenameResizeRepaint=True,
                boundedDigestLayout=True,
                privateAutomationProtocol=False, processExited=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent / 'build/win32-hash.exe')
    parser.add_argument('--screenshot', type=Path, help='optional actual sample.txt window capture')
    args = parser.parse_args()
    try:
        print(json.dumps(verify(args.exe.resolve(), args.screenshot), ensure_ascii=True))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
