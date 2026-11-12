"""Run physical-key verification on a temporary input desktop, then restore it."""
import argparse
import ctypes as c
from ctypes import wintypes as w
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import uuid
from clipboard_snapshot import preserve_clipboard


user, kernel = c.WinDLL('user32', use_last_error=True), c.WinDLL('kernel32', use_last_error=True)
def api(dll, name, result, *types):
    fn = getattr(dll, name); fn.restype, fn.argtypes = result, types; return fn

close = api(kernel, 'CloseHandle', w.BOOL, w.HANDLE)
switch = api(user, 'SwitchDesktop', w.BOOL, w.HANDLE)
close_desktop = api(user, 'CloseDesktop', w.BOOL, w.HANDLE)
wait = api(kernel, 'WaitForSingleObject', w.DWORD, w.HANDLE, w.DWORD)


def object_name(handle):
    out, needed = c.create_unicode_buffer(256), w.DWORD()
    if not api(user, 'GetUserObjectInformationW', w.BOOL, w.HANDLE, c.c_int, c.c_void_p, w.DWORD, c.POINTER(w.DWORD))(handle, 2, out, c.sizeof(out), c.byref(needed)):
        raise OSError(c.get_last_error(), 'desktop name unavailable')
    return out.value


def input_name():
    desktop = api(user, 'OpenInputDesktop', w.HANDLE, w.DWORD, w.BOOL, w.DWORD)(0, False, 0x101)
    if not desktop: raise OSError(c.get_last_error(), 'input desktop unavailable')
    try: return object_name(desktop)
    finally: close_desktop(desktop)


def open_event(name):
    event = api(kernel, 'OpenEventW', w.HANDLE, w.DWORD, w.BOOL, w.LPCWSTR)(0x100000, False, name)
    if not event: raise OSError(c.get_last_error(), 'owned desktop event unavailable')
    return event


def watchdog(parent, desktop_name, event_name, ready):
    desktop = api(user, 'OpenDesktopW', w.HANDLE, w.LPCWSTR, w.DWORD, w.BOOL, w.DWORD)(desktop_name, 0, False, 0x101)
    process = api(kernel, 'OpenProcess', w.HANDLE, w.DWORD, w.BOOL, w.DWORD)(0x100000, False, parent)
    event = open_event(event_name)
    if not desktop or not process: raise RuntimeError('desktop restoration watchdog unavailable')
    try:
        Path(ready).write_text('ready', encoding='ascii')
        api(kernel, 'WaitForMultipleObjects', w.DWORD, w.DWORD, c.POINTER(w.HANDLE), w.BOOL, w.DWORD)(2, (w.HANDLE*2)(process, event), False, 90000)
        if not switch(desktop): raise OSError(c.get_last_error(), 'watchdog desktop restore failed')
    finally:
        close(event); close(process); close_desktop(desktop)


class Limits(c.Structure):
    _fields_ = [('processTime', c.c_int64), ('jobTime', c.c_int64), ('flags', w.DWORD),
                ('minimum', c.c_size_t), ('maximum', c.c_size_t), ('processes', w.DWORD),
                ('affinity', c.c_size_t), ('priority', w.DWORD), ('scheduling', w.DWORD)]


class JobLimits(c.Structure):
    _fields_ = [('basic', Limits), ('io', c.c_uint64*6), ('processMemory', c.c_size_t),
                ('jobMemory', c.c_size_t), ('peakProcess', c.c_size_t), ('peakJob', c.c_size_t)]


def verify(image, mode='keyboard'):
    original_name = input_name()
    station = api(user, 'GetProcessWindowStation', w.HANDLE)()
    station_name = object_name(station)
    token = uuid.uuid4().hex
    desktop_name = 'FairyHashVerify-'+token
    desktop = api(user, 'CreateDesktopW', w.HANDLE, w.LPCWSTR, w.LPCWSTR, c.c_void_p, w.DWORD, w.DWORD, c.c_void_p)(desktop_name, None, None, 0, 0x1ff, None)
    original = api(user, 'OpenDesktopW', w.HANDLE, w.LPCWSTR, w.DWORD, w.BOOL, w.DWORD)(original_name, 0, False, 0x101)
    create_event = api(kernel, 'CreateEventW', w.HANDLE, c.c_void_p, w.BOOL, w.BOOL, w.LPCWSTR)
    restore_name, start_name = 'Local\\FairyRestore-'+token, 'Local\\FairyStart-'+token
    restore_event, start_event = create_event(None, True, False, restore_name), create_event(None, True, False, start_name)
    job = api(kernel, 'CreateJobObjectW', w.HANDLE, c.c_void_p, w.LPCWSTR)(None, None)
    limits = JobLimits(); limits.basic.flags = 0x2000
    worker = guardian = None
    switched = False
    try:
        if not all((desktop, original, restore_event, start_event, job)):
            raise OSError(c.get_last_error(), 'cannot allocate owned desktop guard')
        if not api(kernel, 'SetInformationJobObject', w.BOOL, w.HANDLE, c.c_int, c.c_void_p, w.DWORD)(job, 9, c.byref(limits), c.sizeof(limits)):
            raise OSError(c.get_last_error(), 'cannot contain desktop test processes')
        with tempfile.TemporaryDirectory(prefix='fairy-desktop-guard-') as directory, preserve_clipboard():
            # Materialize user data above, then release the previous clipboard
            # owner while its desktop is still active. A delayed-render owner
            # on the inactive desktop must not block the app's first Copy.
            if not api(user, 'OpenClipboard', w.BOOL, w.HWND)(None):
                raise OSError(c.get_last_error(), 'cannot prepare isolated clipboard')
            try:
                if not api(user, 'EmptyClipboard', w.BOOL)():
                    raise OSError(c.get_last_error(), 'cannot clear isolated clipboard')
            finally:
                api(user, 'CloseClipboard', w.BOOL)()
            ready = Path(directory)/'ready'
            guardian = subprocess.Popen([sys.executable, '-B', str(Path(__file__).resolve()), '--watchdog', str(os.getpid()), original_name, restore_name, str(ready)],
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE, creationflags=0x08000000)
            import time
            until = time.monotonic()+5
            while not ready.exists():
                if guardian.poll() is not None or time.monotonic() > until: raise RuntimeError('desktop restoration watchdog did not arm')
                time.sleep(0.02)
            worker = subprocess.Popen([sys.executable, '-B', str(Path(__file__).resolve()), '--worker', start_name, str(image), mode, desktop_name],
                                      stdout=subprocess.PIPE, stderr=subprocess.PIPE, creationflags=0x08000000)
            if not api(kernel, 'AssignProcessToJobObject', w.BOOL, w.HANDLE, w.HANDLE)(job, int(worker._handle)):
                raise OSError(c.get_last_error(), 'cannot contain owned desktop worker')
            try:
                if not switch(desktop): raise OSError(c.get_last_error(), 'cannot select owned input desktop')
                switched = True
                if input_name() != desktop_name: raise RuntimeError('owned input desktop not selected')
                api(kernel, 'SetEvent', w.BOOL, w.HANDLE)(start_event)
                out, err = worker.communicate(timeout=75)
            finally:
                # Restore before returning success or propagating any failure.
                if not switch(original): raise OSError(c.get_last_error(), 'original input desktop restore failed')
                switched = False
                api(kernel, 'SetEvent', w.BOOL, w.HANDLE)(restore_event)
                guardian_out, guardian_err = guardian.communicate(timeout=5)
                if guardian.returncode or guardian_out or guardian_err: raise RuntimeError(f'desktop restoration watchdog failed: {guardian_err!r}')
                if input_name() != original_name: raise RuntimeError('original input desktop restoration not observed')
                close(job); job = None
                if worker.poll() is None: worker.wait(timeout=5)
            if err or worker.returncode:
                raise RuntimeError(f'isolated desktop verification failed: {out!r} {err!r}')
            result = json.loads(out)
            if not result.get('ok'): raise RuntimeError('isolated desktop verifier did not pass')
            result['isolatedInputDesktop'] = True
            result['originalDesktopRestored'] = True
            result['processTreeContained'] = True
            return result
    finally:
        if switched and original: switch(original)
        if restore_event: api(kernel, 'SetEvent', w.BOOL, w.HANDLE)(restore_event)
        if job: close(job)
        if worker and worker.poll() is None: worker.terminate(); worker.wait(timeout=5)
        if guardian and guardian.poll() is None: guardian.communicate(timeout=5)
        for handle in (restore_event, start_event):
            if handle: close(handle)
        for handle in (desktop, original):
            if handle: close_desktop(handle)


if __name__ == '__main__':
    try:
        if len(sys.argv) == 6 and sys.argv[1] == '--watchdog':
            watchdog(int(sys.argv[2]), sys.argv[3], sys.argv[4], sys.argv[5])
        elif len(sys.argv) == 6 and sys.argv[1] == '--worker':
            desktop = api(user, 'OpenDesktopW', w.HANDLE, w.LPCWSTR, w.DWORD, w.BOOL, w.DWORD)(sys.argv[5], 0, False, 0x1ff)
            if not desktop or not api(user, 'SetThreadDesktop', w.BOOL, w.HANDLE)(desktop):
                raise OSError(c.get_last_error(), 'owned worker cannot select its desktop')
            event = open_event(sys.argv[2])
            try:
                if wait(event, 10000): raise RuntimeError('owned desktop start not authorized')
            finally: close(event)
            from verify_gui import verify as verify_gui
            from desktop_process import DesktopProcess
            station_name = object_name(api(user, 'GetProcessWindowStation', w.HANDLE)())
            target_desktop = station_name+'\\'+sys.argv[5]
            def launch(args, **options): return DesktopProcess(args, desktop=target_desktop, **options)
            print(json.dumps(verify_gui(Path(sys.argv[3]), usability=sys.argv[4], process_factory=launch)))
        else:
            parser = argparse.ArgumentParser(description=__doc__)
            parser.add_argument('--exe', type=Path, default=Path(__file__).parent/'build/win32-hash.exe')
            parser.add_argument('--mode', choices=('keyboard',), default='keyboard')
            args = parser.parse_args()
            print(json.dumps(verify(args.exe.resolve(), args.mode)))
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False, error=str(error)))); raise SystemExit(1)
