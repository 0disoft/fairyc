"""Inspect PE resources as data and decode icons, without launching the GUI."""
import argparse
import ctypes as c
from ctypes import wintypes as w
import hashlib
import json
from pathlib import Path
import struct
from prepare_resources import ROOT, metadata, resource_text


def api(dll, name, result, *types):
    fn = getattr(dll,name); fn.restype,fn.argtypes = result,types; return fn


def verify(image, root=ROOT):
    version, numbers, expected_strings = metadata(root)
    if (root/'native/app.rc').read_text(encoding='utf-8') != resource_text(root):
        raise RuntimeError('Windows metadata is stale; run prepare_resources.py')
    binary = image.read_bytes()
    pe = struct.unpack_from('<I',binary,0x3c)[0]
    if binary[:2] != b'MZ' or binary[pe:pe+4] != b'PE\0\0': raise RuntimeError('invalid PE image')
    machine, magic = struct.unpack_from('<H',binary,pe+4)[0],struct.unpack_from('<H',binary,pe+24)[0]
    subsystem = struct.unpack_from('<H',binary,pe+24+68)[0]
    certificate_offset,certificate_bytes = struct.unpack_from('<II',binary,pe+24+112+4*8)
    if (machine,magic,subsystem) != (0x8664,0x20b,2): raise RuntimeError('expected x64 Windows-subsystem image')
    kernel,user,gdi,versions = (c.WinDLL(name,use_last_error=True) for name in ('kernel32','user32','gdi32','version'))
    # Both flags map resources only: no app entry point, imports or windows.
    module = api(kernel,'LoadLibraryExW',w.HMODULE,w.LPCWSTR,w.HANDLE,w.DWORD)(str(image),None,0x60)
    if not module: raise OSError(c.get_last_error(),'cannot map PE resources as data')
    sizes = []
    try:
        def resource(kind, identifier):
            found = api(kernel,'FindResourceW',w.HANDLE,w.HMODULE,c.c_void_p,c.c_void_p)(module,identifier,kind)
            if not found: raise RuntimeError(f'missing resource {kind}/{identifier}')
            length = api(kernel,'SizeofResource',w.DWORD,w.HMODULE,w.HANDLE)(module,found)
            loaded = api(kernel,'LoadResource',w.HANDLE,w.HMODULE,w.HANDLE)(module,found)
            pointer = api(kernel,'LockResource',c.c_void_p,w.HANDLE)(loaded)
            if not pointer or not length: raise RuntimeError('invalid resource data')
            return c.string_at(pointer,length)
        group, ico = resource(14,1),(root/'native/app.ico').read_bytes()
        reserved, kind, count = struct.unpack_from('<HHH',ico)
        if (reserved,kind) != (0,1) or not count or group[:6] != ico[:6]: raise RuntimeError('icon group mismatch')
        for index in range(count):
            entry = struct.unpack_from('<BBBBHHII',ico,6+16*index)
            embedded = struct.unpack_from('<BBBBHHIH',group,6+14*index)
            if entry[:7] != embedded[:7] or resource(3,embedded[-1]) != ico[entry[-1]:entry[-1]+entry[-2]]:
                raise RuntimeError('embedded icon differs from source ICO')
            if entry[0] != entry[1]: raise RuntimeError('expected square icon')
            sizes.append(entry[0] or 256)
        if not {16,32,48,256}.issubset(sizes): raise RuntimeError('missing standard icon sizes')
        class Icon(c.Structure):
            _fields_ = [('icon',w.BOOL),('x',w.DWORD),('y',w.DWORD),('mask',w.HBITMAP),('color',w.HBITMAP)]
        class Bitmap(c.Structure):
            _fields_ = [('type',w.LONG),('width',w.LONG),('height',w.LONG),('stride',w.LONG),
                        ('planes',w.WORD),('bits',w.WORD),('data',c.c_void_p)]
        for size in sizes:
            handle = api(user,'LoadImageW',w.HANDLE,w.HINSTANCE,c.c_void_p,w.UINT,c.c_int,c.c_int,w.UINT)(module,1,1,size,size,0)
            if not handle: raise RuntimeError(f'Windows could not decode icon at {size}px')
            info = Icon()
            try:
                if not api(user,'GetIconInfo',w.BOOL,w.HANDLE,c.POINTER(Icon))(handle,c.byref(info)):
                    raise RuntimeError('cannot inspect decoded icon')
                bitmap = Bitmap()
                if not api(gdi,'GetObjectW',c.c_int,w.HANDLE,c.c_int,c.c_void_p)(info.color,c.sizeof(bitmap),c.byref(bitmap)):
                    raise RuntimeError('cannot read icon dimensions')
                if (bitmap.width,bitmap.height) != (size,size): raise RuntimeError('decoded icon size mismatch')
            finally:
                for bitmap_handle in (info.color,info.mask):
                    if bitmap_handle: api(gdi,'DeleteObject',w.BOOL,w.HANDLE)(bitmap_handle)
                api(user,'DestroyIcon',w.BOOL,w.HANDLE)(handle)
        manifest = resource(24,1).decode('utf-8').rstrip('\0')
        if '.'.join(map(str,numbers)) not in manifest or 'Microsoft.Windows.Common-Controls' not in manifest:
            raise RuntimeError('manifest version/theme dependency mismatch')
    finally:
        api(kernel,'FreeLibrary',w.BOOL,w.HMODULE)(module)
    length = api(versions,'GetFileVersionInfoSizeW',w.DWORD,w.LPCWSTR,c.POINTER(w.DWORD))(str(image),None)
    if not length: raise RuntimeError('VERSIONINFO missing')
    data = c.create_string_buffer(length)
    if not api(versions,'GetFileVersionInfoW',w.BOOL,w.LPCWSTR,w.DWORD,w.DWORD,c.c_void_p)(str(image),0,length,data):
        raise RuntimeError('cannot read VERSIONINFO')
    def query(path):
        pointer,count = c.c_void_p(),w.UINT()
        if not api(versions,'VerQueryValueW',w.BOOL,c.c_void_p,w.LPCWSTR,c.POINTER(c.c_void_p),c.POINTER(w.UINT))(data,path,c.byref(pointer),c.byref(count)):
            raise RuntimeError(f'version field missing: {path}')
        return pointer,count.value
    pointer,length = query('\\')
    if length < 52: raise RuntimeError('truncated fixed version info')
    fixed = struct.unpack('<13I',c.string_at(pointer,52))
    words = (numbers[0]<<16|numbers[1], numbers[2]<<16|numbers[3])
    if fixed[0] != 0xfeef04bd or fixed[2:4] != words or fixed[4:6] != words:
        raise RuntimeError('fixed version differs from fairy.toml')
    strings = {}
    for name,expected in expected_strings.items():
        pointer,length = query('\\StringFileInfo\\040904b0\\'+name)
        strings[name] = c.wstring_at(pointer)
        if strings[name] != expected: raise RuntimeError(f'version text mismatch: {name}')
    return dict(ok=True,verification='fairy-hash-pe-resources',version=version,strings=strings,
                iconSizes=sizes,iconBytes=len(ico),exeBytes=len(binary),sha256=hashlib.sha256(binary).hexdigest(),
                subsystem='windows',architecture='x64',certificateTablePresent=bool(certificate_offset or certificate_bytes),
                guiLaunched=False,inputInjected=False)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe',type=Path,default=ROOT/'build/win32-hash.exe')
    args = parser.parse_args()
    try: print(json.dumps(verify(args.exe.resolve())))
    except (OSError,ValueError,RuntimeError,struct.error) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
