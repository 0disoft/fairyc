"""Owned native process launch with an explicit STARTUPINFOW desktop."""
import ctypes as c
from ctypes import wintypes as w
import msvcrt
import os
import subprocess
import tempfile
import weakref


class Startup(c.Structure):
    _fields_ = [('size', w.DWORD), ('reserved', w.LPWSTR), ('desktop', w.LPWSTR), ('title', w.LPWSTR),
                ('x', w.DWORD), ('y', w.DWORD), ('width', w.DWORD), ('height', w.DWORD),
                ('columns', w.DWORD), ('rows', w.DWORD), ('fill', w.DWORD), ('flags', w.DWORD),
                ('show', w.WORD), ('reservedSize', w.WORD), ('reservedBytes', c.c_void_p),
                ('input', w.HANDLE), ('output', w.HANDLE), ('error', w.HANDLE)]


class Information(c.Structure):
    _fields_ = [('process', w.HANDLE), ('thread', w.HANDLE), ('pid', w.DWORD), ('tid', w.DWORD)]


kernel = c.WinDLL('kernel32', use_last_error=True)
def api(name, result, *types):
    fn = getattr(kernel, name); fn.restype, fn.argtypes = result, types; return fn

close = api('CloseHandle', w.BOOL, w.HANDLE)
wait = api('WaitForSingleObject', w.DWORD, w.HANDLE, w.DWORD)


class DesktopProcess:
    def __init__(self, args, *, desktop, stdin, stdout, stderr):
        if (stdin, stdout, stderr) != (subprocess.PIPE,)*3:
            raise ValueError('owned desktop launch requires captured streams')
        self.args, self.returncode, self._handle = args, None, None
        self.stdin = open(os.devnull, 'rb')
        self.stdout, self.stderr = tempfile.TemporaryFile(), tempfile.TemporaryFile()
        handles = [msvcrt.get_osfhandle(stream.fileno()) for stream in (self.stdin, self.stdout, self.stderr)]
        info, startup = Information(), Startup()
        startup.size, startup.desktop, startup.flags = c.sizeof(startup), desktop, 0x100
        startup.input, startup.output, startup.error = handles
        create = api('CreateProcessW', w.BOOL, w.LPCWSTR, w.LPWSTR, c.c_void_p, c.c_void_p,
                     w.BOOL, w.DWORD, c.c_void_p, w.LPCWSTR, c.POINTER(Startup), c.POINTER(Information))
        try:
            for handle in handles: os.set_handle_inheritable(handle, True)
            command = c.create_unicode_buffer(subprocess.list2cmdline([str(value) for value in args]))
            if not create(str(args[0]), command, None, None, True, 0x08000000, None, None, c.byref(startup), c.byref(info)):
                raise OSError(c.get_last_error(), 'owned desktop process launch failed')
            self.pid, self._handle = info.pid, info.process
            close(info.thread)
            self._finalizer = weakref.finalize(self, close, self._handle)
        except BaseException:
            for stream in (self.stdin, self.stdout, self.stderr): stream.close()
            raise
        finally:
            for handle in handles:
                try: os.set_handle_inheritable(handle, False)
                except OSError: pass

    def poll(self):
        if self.returncode is None and wait(self._handle, 0) == 0:
            code = w.DWORD()
            if not api('GetExitCodeProcess', w.BOOL, w.HANDLE, c.POINTER(w.DWORD))(self._handle, c.byref(code)):
                raise OSError(c.get_last_error(), 'owned desktop process exit unavailable')
            self.returncode = code.value
        return self.returncode

    def wait(self, timeout=None):
        state = wait(self._handle, 0xffffffff if timeout is None else round(timeout*1000))
        if state == 258: raise subprocess.TimeoutExpired(self.args, timeout)
        if state != 0: raise OSError(c.get_last_error(), 'owned desktop process wait failed')
        return self.poll()

    def terminate(self):
        if self.poll() is None and not api('TerminateProcess', w.BOOL, w.HANDLE, w.UINT)(self._handle, 1):
            raise OSError(c.get_last_error(), 'owned desktop process termination failed')

    def communicate(self, timeout=None):
        self.wait(timeout)
        self.stdout.seek(0); self.stderr.seek(0)
        return self.stdout.read(), self.stderr.read()
