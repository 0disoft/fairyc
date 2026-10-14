"""Create isolated native handle-relative file fixtures, including real junctions."""
import argparse
import ctypes
from ctypes import wintypes
import hashlib
import json
import os
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parents[1]


def junction(path, target):
    path.mkdir()
    kernel = ctypes.WinDLL('kernel32', use_last_error=True)
    create = kernel.CreateFileW
    create.argtypes = [wintypes.LPCWSTR, wintypes.DWORD, wintypes.DWORD, ctypes.c_void_p, wintypes.DWORD, wintypes.DWORD, wintypes.HANDLE]
    create.restype = wintypes.HANDLE
    control = kernel.DeviceIoControl
    control.argtypes = [wintypes.HANDLE, wintypes.DWORD, ctypes.c_void_p, wintypes.DWORD, ctypes.c_void_p, wintypes.DWORD, ctypes.POINTER(wintypes.DWORD), ctypes.c_void_p]
    control.restype = wintypes.BOOL
    close = kernel.CloseHandle
    close.argtypes = [wintypes.HANDLE]; close.restype = wintypes.BOOL
    substitute = ('\\??\\' + str(target)).encode('utf-16-le')
    printed = str(target).encode('utf-16-le')
    names = substitute + b'\0\0' + printed + b'\0\0'
    raw = struct.pack('<IHHHHHH', 0xA0000003, 8 + len(names), 0, 0, len(substitute), len(substitute) + 2, len(printed)) + names
    data = ctypes.create_string_buffer(raw)
    handle = create(str(path), 0x40000000, 0, None, 3, 0x02200000, None)
    if handle == ctypes.c_void_p(-1).value:
        raise ctypes.WinError(ctypes.get_last_error())
    try:
        returned = wintypes.DWORD()
        if not control(handle, 0x900A4, data, len(raw), None, 0, ctypes.byref(returned), None):
            raise ctypes.WinError(ctypes.get_last_error())
    finally:
        assert close(handle)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert os.name == 'nt'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    base = output / 'rooted 파일'
    actual = base / 'reparse/actual'; external = base / 'reparse/external'
    actual.mkdir(parents=True); external.mkdir()
    marker = external / 'marker.bin'; marker.write_bytes(b'untouched\0\xff')
    junction(actual / 'redirect', external)
    junction(base / 'reparse/root-link', external)
    optional = False
    try:
        os.symlink(marker, actual / 'file-link')
        optional = True
    except OSError:
        pass
    (output / 'root.txt').write_text(str(base), encoding='utf-8')
    (output / 'unchanged.json').write_text(json.dumps([marker.relative_to(ROOT).as_posix()])+'\n', encoding='utf-8')
    (output / 'fixture-receipt.json').write_text(json.dumps(dict(schema='fairyc.candidate-rooted-file-fixture/v0', base=base.relative_to(ROOT).as_posix(), junctionsCreated=2, optionalFileSymlinkCreated=optional, markerSHA256=hashlib.sha256(marker.read_bytes()).hexdigest()),indent=2)+'\n',encoding='utf-8')
    print('rooted file fixture: two junctions, optional file symlink:', optional)


if __name__ == '__main__':
    main()
