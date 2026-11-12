"""Owned Narrator launch through the normal Windows shell, without runas."""
import ctypes as c
from ctypes import wintypes as w
from pathlib import Path
import os
import time


kernel = c.WinDLL('kernel32', use_last_error=True)
user = c.WinDLL('user32', use_last_error=True)


def api(dll, name, result, *types):
    fn = getattr(dll, name); fn.restype, fn.argtypes = result, types; return fn


class Entry(c.Structure):
    _fields_ = [('size', w.DWORD), ('usage', w.DWORD), ('pid', w.DWORD),
                ('heap', c.c_size_t), ('module', w.DWORD), ('threads', w.DWORD),
                ('parent', w.DWORD), ('priority', w.LONG), ('flags', w.DWORD), ('exe', w.WCHAR*260)]


def narrator_pids():
    snapshot = api(kernel, 'CreateToolhelp32Snapshot', w.HANDLE, w.DWORD, w.DWORD)(2, 0)
    if snapshot == c.c_void_p(-1).value: raise OSError(c.get_last_error(), 'process inventory failed')
    try:
        entry = Entry(); entry.size = c.sizeof(entry)
        first = api(kernel, 'Process32FirstW', w.BOOL, w.HANDLE, c.POINTER(Entry))
        following = api(kernel, 'Process32NextW', w.BOOL, w.HANDLE, c.POINTER(Entry))
        found, valid = [], first(snapshot, c.byref(entry))
        while valid:
            if entry.exe.casefold() == 'narrator.exe': found.append(entry.pid)
            valid = following(snapshot, c.byref(entry))
        return found
    finally:
        api(kernel, 'CloseHandle', w.BOOL, w.HANDLE)(snapshot)


class Execute(c.Structure):
    _fields_ = [('size', w.DWORD), ('mask', w.ULONG), ('window', w.HWND),
                ('verb', w.LPCWSTR), ('file', w.LPCWSTR), ('parameters', w.LPCWSTR),
                ('directory', w.LPCWSTR), ('show', c.c_int), ('instance', w.HINSTANCE),
                ('idlist', c.c_void_p), ('className', w.LPCWSTR), ('classKey', w.HKEY),
                ('hotkey', w.DWORD), ('icon', w.HANDLE), ('process', w.HANDLE)]


class NarratorSession:
    def __enter__(self):
        if narrator_pids():
            raise RuntimeError('Narrator is already running; leave the existing reader untouched')
        self.handle = None
        ole = c.WinDLL('ole32')
        initialized = api(ole, 'CoInitializeEx', w.LONG, c.c_void_p, w.DWORD)(None, 6)
        if initialized < 0: raise OSError(initialized, 'shell COM initialization failed')
        try:
            info = Execute(); info.size = c.sizeof(info)
            info.mask, info.verb = 0x40 | 0x100 | 0x400, 'open'
            info.file, info.show = str(Path(os.environ['WINDIR'])/'System32/Narrator.exe'), 0
            if not api(c.WinDLL('shell32', use_last_error=True), 'ShellExecuteExW', w.BOOL, c.POINTER(Execute))(c.byref(info)):
                raise OSError(c.get_last_error(), 'normal-shell Narrator launch failed')
            self.handle = info.process
            if not self.handle: raise RuntimeError('shell returned no owned Narrator process handle')
            self.pid = api(kernel, 'GetProcessId', w.DWORD, w.HANDLE)(self.handle)
            if not self.pid: raise OSError(c.get_last_error(), 'owned Narrator PID unavailable')
            self.receipt = dict(method='ShellExecuteExW/open', pid=self.pid, runas=False)
            return self
        except BaseException:
            if self.handle: self.close()
            raise
        finally:
            api(ole, 'CoUninitialize', None)()

    def running(self):
        return api(kernel, 'WaitForSingleObject', w.DWORD, w.HANDLE, w.DWORD)(self.handle, 0) == 258

    def close(self):
        try:
            if self.running():
                # The handle came from this exact shell launch; never stop an
                # existing reader or a process selected only by executable name.
                if not api(kernel, 'TerminateProcess', w.BOOL, w.HANDLE, w.UINT)(self.handle, 0):
                    raise OSError(c.get_last_error(), 'owned Narrator cleanup failed')
                if api(kernel, 'WaitForSingleObject', w.DWORD, w.HANDLE, w.DWORD)(self.handle, 5000):
                    raise RuntimeError('owned Narrator did not stop')
            self.receipt['ownedNarratorStopped'] = True
        finally:
            api(kernel, 'CloseHandle', w.BOOL, w.HANDLE)(self.handle)
            self.handle = None

    def __exit__(self, *error):
        self.close()


if __name__ == '__main__':
    import json
    with NarratorSession() as session:
        time.sleep(3)
        if not session.running(): raise RuntimeError('Narrator exited during launch smoke')
    print(json.dumps(dict(ok=True, **session.receipt)))
