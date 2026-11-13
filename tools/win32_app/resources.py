"""Synchronize Windows release metadata with the existing fairy.toml version."""
from pathlib import Path
import re
import tomllib
from .settings import load,resource_string

ICON_ID = 1


def metadata(root):
    app = load(root)
    version = tomllib.loads((root/'fairy.toml').read_text(encoding='utf-8'))['package']['version']
    if not re.fullmatch(r'(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)', version):
        raise ValueError('Windows packaging currently requires a stable major.minor.patch version')
    numbers = tuple(map(int, version.split('.')))+(0,)
    if max(numbers) > 65535: raise ValueError('Windows version component exceeds 65535')
    return version, numbers, dict(CompanyName=app.company_name, FileDescription=app.description,
        FileVersion=version, InternalName=Path(app.executable_name).stem, OriginalFilename=app.executable_name,
        ProductName=app.product_name, ProductVersion=version)


def resource_text(root):
    app = load(root)
    version, numbers, strings = metadata(root)
    numeric, assembly = ','.join(map(str,numbers)), '.'.join(map(str,numbers))
    values = '\n'.join(f'      VALUE "{name}", {resource_string(value)}' for name,value in strings.items())
    return f'''// Generated from app.toml and fairy.toml by prepare_resources.py; version is not bumped.
#include <windows.h>
LANGUAGE LANG_ENGLISH, SUBLANG_ENGLISH_US
{ICON_ID} ICON "{app.icon}"
1 VERSIONINFO
 FILEVERSION {numeric}
 PRODUCTVERSION {numeric}
 FILEFLAGSMASK VS_FFI_FILEFLAGSMASK
 FILEFLAGS 0
 FILEOS VOS_NT_WINDOWS32
 FILETYPE VFT_APP
 FILESUBTYPE 0
BEGIN
  BLOCK "StringFileInfo"
  BEGIN
    BLOCK "040904b0"
    BEGIN
{values}
    END
  END
  BLOCK "VarFileInfo"
  BEGIN
    VALUE "Translation", 0x0409, 1200
  END
END
STRINGTABLE
BEGIN
  101 {resource_string(app.product_name)}
END
1 RT_MANIFEST
BEGIN
  "<?xml version='1.0' encoding='UTF-8' standalone='yes'?><assembly xmlns='urn:schemas-microsoft-com:asm.v1' manifestVersion='1.0'><assemblyIdentity type='win32' name='{app.assembly_name}' version='{assembly}' processorArchitecture='amd64'/><dependency><dependentAssembly><assemblyIdentity type='win32' name='Microsoft.Windows.Common-Controls' version='6.0.0.0' processorArchitecture='amd64' publicKeyToken='6595b64144ccf1df' language='*'/></dependentAssembly></dependency></assembly>"
END
'''


def prepare(root):
    target, expected = root/'native/app.rc', resource_text(root)
    if not target.exists() or target.read_text(encoding='utf-8') != expected:
        target.write_text(expected,encoding='utf-8',newline='\n')


