"""Actual file changes, protected writes and rejected GUI/CLI hash results."""
import ctypes as c
from ctypes import wintypes as w
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time
from contextlib import ExitStack
from io_gate import Gate, api, kernel, close, pending_threads


class Basic(c.Structure):
    _fields_ = [('created', c.c_int64), ('accessed', c.c_int64), ('written', c.c_int64), ('changed', c.c_int64), ('attributes', w.DWORD)]


def metadata(path):
    handle = api(kernel, 'CreateFileW', w.HANDLE, w.LPCWSTR, w.DWORD, w.DWORD, c.c_void_p, w.DWORD, w.DWORD, w.HANDLE)(str(path), 0x80, 7, None, 3, 0, None)
    if handle == c.c_void_p(-1).value:
        raise OSError(c.get_last_error(), 'fixture metadata open failed')
    try:
        out = Basic()
        if not api(kernel, 'GetFileInformationByHandleEx', w.BOOL, w.HANDLE, c.c_int, c.c_void_p, w.DWORD)(handle, 0, c.byref(out), c.sizeof(out)):
            raise OSError(c.get_last_error(), 'fixture metadata failed')
        return out
    finally:
        close(handle)


def write_time(path, value):
    handle = api(kernel, 'CreateFileW', w.HANDLE, w.LPCWSTR, w.DWORD, w.DWORD, c.c_void_p, w.DWORD, w.DWORD, w.HANDLE)(str(path), 0x100, 7, None, 3, 0, None)
    if handle == c.c_void_p(-1).value:
        raise OSError(c.get_last_error(), 'fixture attribute open failed')
    try:
        when = c.c_uint64(value)
        if not api(kernel, 'SetFileTime', w.BOOL, w.HANDLE, c.c_void_p, c.c_void_p, c.c_void_p)(handle, None, None, c.byref(when)):
            raise OSError(c.get_last_error(), 'fixture timestamp update failed')
    finally:
        close(handle)


def deny_write(path):
    handle = api(kernel, 'CreateFileW', w.HANDLE, w.LPCWSTR, w.DWORD, w.DWORD, c.c_void_p, w.DWORD, w.DWORD, w.HANDLE)(str(path), 0x40000000, 7, None, 3, 0, None)
    error = c.get_last_error()
    if handle != c.c_void_p(-1).value:
        close(handle)
        raise RuntimeError('production sharing unexpectedly allows overwrite')
    if error != 32:
        raise RuntimeError(f'overwrite was not rejected by sharing protection: {error}')


def replace_open_file(replacement, target):
    # Windows POSIX rename allows replacing the deliberately delete-shared test
    # fixture while its old handle stays open. MoveFileEx can reject that case.
    class Rename(c.Structure):
        _fields_ = [('flags', w.DWORD), ('root', w.HANDLE), ('length', w.DWORD), ('name', w.WCHAR*1)]
    handle = api(kernel, 'CreateFileW', w.HANDLE, w.LPCWSTR, w.DWORD, w.DWORD, c.c_void_p, w.DWORD, w.DWORD, w.HANDLE)(str(replacement), 0x10000, 7, None, 3, 0, None)
    if handle == c.c_void_p(-1).value: raise OSError(c.get_last_error(), 'replacement fixture open failed')
    try:
        name = str(target.resolve()).encode('utf-16-le')
        buffer = c.create_string_buffer(c.sizeof(Rename)+len(name))
        info = Rename.from_buffer(buffer); info.flags = 3; info.length = len(name)
        c.memmove(c.addressof(buffer)+Rename.name.offset, name, len(name))
        if not api(kernel, 'SetFileInformationByHandle', w.BOOL, w.HANDLE, c.c_int, c.c_void_p, w.DWORD)(handle, 22, buffer, len(buffer)):
            raise OSError(c.get_last_error(), 'replacement fixture POSIX rename failed')
    finally:
        close(handle)


def verify(d):
    rows, pings = [], []
    prefix = os.environ['FAIRY_HASH_CHANGE_PIPE']
    payload = bytes(range(256))*768
    seed = d.root/'clean-seed.bin'; seed.write_bytes(b'known good')
    with ExitStack() as cleanup:
        for name in ('protected-overwrite', 'protected-timestamp', 'change-same-size', 'change-restored-mtime',
                     'change-grow', 'change-shrink', 'change-replace-eof'):
            d.choose(202, seed)
            d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v.startswith('✓ 계산 완료'), 'seed hash not ready')
            d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'seed Save not ready')
            d.wait(lambda: d.text(d.child(d.main, 201)), lambda v: v == str(seed), 'fresh seed path not selected')
            d.wait(lambda: d.text(d.child(d.main, 203)).replace('\r\n', ''), lambda v: v == hashlib.sha256(b'known good').hexdigest(), 'fresh seed digest not ready')
            source = d.root/(name+'.bin'); source.write_bytes(payload)
            first = metadata(source)
            gate = Gate(prefix+'-'+source.name); cleanup.callback(gate.release)
            d.choose(202, source)
            d.wait(lambda: gate.attached(d.process.pid) and bool(pending_threads(d.process.pid)), bool, 'change gate not pending in owned process')
            if d.text(d.child(d.main, 203)) or d.enabled(d.child(d.main, 204)) or d.enabled(d.child(d.main, 208)):
                raise RuntimeError('previous result survives into changed-file job')
            for _ in range(3):
                start = time.perf_counter(); d.message(d.main, 0); pings.append((time.perf_counter()-start)*1000)
            if name == 'protected-overwrite':
                deny_write(source)
                if source.read_bytes() != payload:
                    raise RuntimeError('protected overwrite changed file bytes')
            elif name == 'protected-timestamp':
                deny_write(source); write_time(source, first.written+10000000)
            elif name in ('change-same-size', 'change-restored-mtime'):
                with source.open('r+b') as stream:
                    stream.write(b'X'*4096); stream.flush(); os.fsync(stream.fileno())
                if source.stat().st_size != len(payload) or source.read_bytes() == payload:
                    raise RuntimeError('same-size overwrite fixture did not change real bytes')
                if name == 'change-restored-mtime':
                    write_time(source, first.written)
                    last = metadata(source)
                    if last.written != first.written or last.changed == first.changed:
                        raise RuntimeError('restored-mtime fixture did not isolate ChangeTime')
            elif name == 'change-grow':
                with source.open('ab') as stream: stream.write(b'!')
            elif name == 'change-shrink':
                with source.open('r+b') as stream: stream.truncate(128)
            else:
                replacement = d.root/'replacement.bin'; replacement.write_bytes(b'R'*len(payload)); replace_open_file(replacement, source)
            gate.release()
            if name == 'protected-overwrite':
                d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v.startswith('✓ 계산 완료'), 'protected hash did not finish')
                d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'protected result unavailable')
                if d.text(d.child(d.main, 203)).replace('\r\n', '') != hashlib.sha256(payload).hexdigest():
                    raise RuntimeError('protected result digest mismatch')
                rows.append(dict(case=name, overwriteDenied=True, correctDigest=True)); continue
            d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v.startswith('계산 중 원본이 변경'), 'source change not detected')
            d.wait(lambda: bool(d.enabled(d.child(d.main, 202))), bool, 'source-change window did not recover')
            if d.text(d.child(d.main, 203)) or d.enabled(d.child(d.main, 204)) or d.enabled(d.child(d.main, 208)):
                raise RuntimeError('changed result can be copied/saved')
            sequence = api(c.WinDLL('user32'), 'GetClipboardSequenceNumber', w.DWORD)()
            d.message(d.child(d.main, 208), 0xf5); d.message(d.child(d.main, 204), 0xf5)
            time.sleep(0.05)
            if d.windows('#32770') or api(c.WinDLL('user32'), 'GetClipboardSequenceNumber', w.DWORD)() != sequence:
                raise RuntimeError('disabled changed-result action produced an effect')
            rows.append(dict(case=name, changeDetected=True, digestCleared=True, copyAndSaveBlocked=True,
                             relaxedTestSharing=name.startswith('change-')))
        d.choose(202, seed)
        d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v.startswith('✓ 계산 완료'), 'window reuse after change failed')
        d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'window reuse result unavailable')
        target = d.root/'reused.json'; d.choose(204, target)
        d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v == '✓ 결과 저장 완료', 'reused result save failed')
        if json.loads(target.read_bytes())['sha256'] != hashlib.sha256(b'known good').hexdigest():
            raise RuntimeError('changed digest contaminated a later report')
        rows.append(dict(case='window-reuse', correctReport=True))
        d.message(d.main, 0x10)
        if d.process.wait(timeout=3): raise RuntimeError('source-change window exit failed')
    if max(pings) > 500: raise RuntimeError('source observation blocked UI')
    return dict(cases=len(rows), observations=rows, pingMaxMs=max(pings), actualPendingIO=True,
                physicalInput=False, pixelInspection=False, productionWriteProtectionKept=True)


def verify_cli(image, root, prefix):
    source = root/'cli-timestamp.bin'; source.write_bytes(bytes(range(256))*1024)
    target = root/'previous-report.json'; target.write_bytes(b'previous report')
    first = metadata(source)
    with ExitStack() as cleanup:
        gate = Gate(prefix+'-'+source.name); cleanup.callback(gate.release)
        process = subprocess.Popen([str(image), '--hash', str(source), str(target)], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        def stop():
            if process.poll() is None: process.kill(); process.wait(timeout=3)
        cleanup.callback(stop)
        deadline = time.monotonic()+10
        while not gate.attached(process.pid) or not pending_threads(process.pid):
            if time.monotonic() > deadline or process.poll() is not None: raise RuntimeError('CLI final guard gate not reached')
            time.sleep(0.01)
        deny_write(source); write_time(source, first.written+10000000); gate.release()
        out, err = process.communicate(timeout=5)
        if process.returncode != 1 or err or json.loads(out) != dict(ok=False, errorCode=11):
            raise RuntimeError(f'CLI emitted a digest after source change: {out!r} {err!r}')
        if target.read_bytes() != b'previous report' or list(root.glob('*.fairy-save-*.tmp')):
            raise RuntimeError('CLI source change altered an existing report/left temporaries')
    return dict(case='cli-source-change', errorCode=11, overwriteDenied=True, digestNotEmitted=True, oldReportPreserved=True)
