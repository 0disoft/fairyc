"""Owned OS named-pipe gates and pending-I/O observations for slow-I/O tests."""
import ctypes as c
from ctypes import wintypes as w


def api(dll, name, result, *types):
    fn = getattr(dll, name)
    fn.restype, fn.argtypes = result, types
    return fn


kernel = c.WinDLL('kernel32', use_last_error=True)
close = api(kernel, 'CloseHandle', w.BOOL, w.HANDLE)
wait = api(kernel, 'WaitForSingleObject', w.DWORD, w.HANDLE, w.DWORD)


class Overlap(c.Structure):
    _fields_ = [('internal', c.c_size_t), ('high', c.c_size_t),
                ('offset', w.DWORD), ('offsetHigh', w.DWORD), ('event', w.HANDLE)]


class Gate:
    def __init__(self, name):
        self.handle = api(kernel, 'CreateNamedPipeW', w.HANDLE, w.LPCWSTR, w.DWORD, w.DWORD,
                          w.DWORD, w.DWORD, w.DWORD, w.DWORD, c.c_void_p)(name, 0x40000002, 0, 1, 4096, 4096, 0, None)
        if self.handle == c.c_void_p(-1).value:
            raise OSError(c.get_last_error(), 'cannot create owned I/O gate')
        self.overlap = Overlap()
        self.overlap.event = api(kernel, 'CreateEventW', w.HANDLE, c.c_void_p, w.BOOL, w.BOOL, w.LPCWSTR)(None, True, False, None)
        if not self.overlap.event:
            close(self.handle)
            raise OSError(c.get_last_error(), 'cannot create I/O gate event')
        ok = api(kernel, 'ConnectNamedPipe', w.BOOL, w.HANDLE, c.POINTER(Overlap))(self.handle, c.byref(self.overlap))
        if not ok and c.get_last_error() != 997:
            self.release()
            raise OSError(c.get_last_error(), 'cannot arm owned I/O gate')

    def attached(self, pid):
        if not self.handle or wait(self.overlap.event, 0) != 0:
            return False
        actual, transferred = w.ULONG(), w.DWORD()
        if not api(kernel, 'GetOverlappedResult', w.BOOL, w.HANDLE, c.POINTER(Overlap), c.POINTER(w.DWORD), w.BOOL)(self.handle, c.byref(self.overlap), c.byref(transferred), False):
            raise RuntimeError('I/O gate connection failed')
        return bool(api(kernel, 'GetNamedPipeClientProcessId', w.BOOL, w.HANDLE, c.POINTER(w.ULONG))(self.handle, c.byref(actual))) and actual.value == pid

    def release(self):
        if self.handle:
            api(kernel, 'CancelIoEx', w.BOOL, w.HANDLE, c.POINTER(Overlap))(self.handle, c.byref(self.overlap))
            close(self.handle)
            self.handle = None
            # Keep the OVERLAPPED/event alive until a cancelled connect completes.
            if wait(self.overlap.event, 3000) != 0:
                raise RuntimeError('I/O gate connection did not finish cancelling')
            close(self.overlap.event)


def pending_threads(pid):
    class Entry(c.Structure):
        _fields_ = [('size', w.DWORD), ('usage', w.DWORD), ('id', w.DWORD), ('pid', w.DWORD),
                    ('base', w.LONG), ('delta', w.LONG), ('flags', w.DWORD)]
    snapshot = api(kernel, 'CreateToolhelp32Snapshot', w.HANDLE, w.DWORD, w.DWORD)(4, 0)
    if snapshot == c.c_void_p(-1).value:
        raise RuntimeError('cannot enumerate owned I/O threads')
    found, entry = [], Entry()
    entry.size = c.sizeof(entry)
    try:
        more = api(kernel, 'Thread32First', w.BOOL, w.HANDLE, c.POINTER(Entry))(snapshot, c.byref(entry))
        advance = api(kernel, 'Thread32Next', w.BOOL, w.HANDLE, c.POINTER(Entry))
        while more:
            if entry.pid == pid:
                thread = api(kernel, 'OpenThread', w.HANDLE, w.DWORD, w.BOOL, w.DWORD)(0x40, False, entry.id)
                if thread:
                    try:
                        pending = w.BOOL()
                        if api(kernel, 'GetThreadIOPendingFlag', w.BOOL, w.HANDLE, c.POINTER(w.BOOL))(thread, c.byref(pending)) and pending.value:
                            found.append(entry.id)
                    finally:
                        close(thread)
            more = advance(snapshot, c.byref(entry))
    finally:
        close(snapshot)
    return found


def threads_finished(identifiers):
    for identifier in identifiers:
        thread = api(kernel, 'OpenThread', w.HANDLE, w.DWORD, w.BOOL, w.DWORD)(0x100000, False, identifier)
        if thread:
            try:
                if wait(thread, 0) != 0:
                    return False
            finally:
                close(thread)
    return True
