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


def verify(image):
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
            if text(main) != 'Fairy Hash' or enabled(child(main, 204)):
                raise RuntimeError('initial title/save state invalid')
            def dialog(button):
                click(child(main, button))
                return wait(lambda: windows('#32770'), lambda v: len(v) == 1, 'native file dialog missing')[0]
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
                actual_hash = wait(lambda: text(child(main, 203)), lambda v: v == expected_hash, 'Fairy SHA-256 display')
                wait(lambda: bool(enabled(child(main, 204))), bool, 'Save did not enable')
                actual_path = text(child(main, 201))
                if len(actual_path) <= 127:
                    raise RuntimeError('fixture did not exercise paths beyond the old edit limit')
                if not os.path.samefile(actual_path, source):
                    raise RuntimeError('displayed path does not identify the selected file')
                expected = dict(algorithm='SHA-256', file=actual_path, bytes=len(data), sha256=expected_hash)
                if index == 1:
                    cancel(202)
                    cancel(204)
                    if text(child(main, 201)) != actual_path or text(child(main, 203)) != actual_hash:
                        raise RuntimeError('dialog cancellation discarded a completed digest')
                target = root / f'보고서-{index}.json'
                if index == 2:
                    target.write_bytes(b'owned old report')
                choose(204, target, overwrite=index == 2)
                wait(lambda: text(child(main, 205)), lambda v: v == 'Report saved.', 'Fairy save status')
                if json.loads(target.read_bytes()) != expected:
                    raise RuntimeError('saved report differs from selected-file reference')
                rows.append(dict(bytes=len(data), sha256=actual_hash, unicodePath=True, saved=True,
                                 inputPathCharacters=len(actual_path), overwritten=index == 2))
            before = source.read_bytes()
            choose(204, source, overwrite=True)
            wait(lambda: text(child(main, 205)), lambda v: 'Cannot overwrite the input' in v, 'source protection status')
            if source.read_bytes() != before:
                raise RuntimeError('GUI save damaged the source')
            if not resize(main, None, 0, 0, 1000, 500, 0x16):
                raise RuntimeError('GUI resize failed')
            if text(child(main, 203)) != expected_hash:
                raise RuntimeError('resize lost digest')
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
                closeDuringHash=True, privateAutomationProtocol=False, processExited=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent / 'build/win32-hash.exe')
    args = parser.parse_args()
    try:
        print(json.dumps(verify(args.exe.resolve()), ensure_ascii=True))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error))))
        raise SystemExit(1)
