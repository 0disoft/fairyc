"""App identity shared by resource generation, packaging and CLI verification."""
from dataclasses import dataclass
from pathlib import Path, PurePosixPath
import re
import tomllib

ROOT = Path(__file__).resolve().parent


@dataclass(frozen=True)
class App:
    product_name: str
    company_name: str
    description: str
    executable_name: str
    package_name: str
    assembly_name: str
    icon: str

    def icon_path(self, root=ROOT):
        return root/self.icon


def load(root=ROOT):
    document = tomllib.loads((root/'app.toml').read_text(encoding='utf-8'))
    if set(document) != {'app'} or not isinstance(document['app'],dict):
        raise ValueError('app.toml must contain one [app] table')
    values = document['app']
    fields = set(App.__annotations__)
    if set(values) != fields:
        raise ValueError(f'app.toml fields: missing={sorted(fields-set(values))}, unknown={sorted(set(values)-fields)}')
    for name,value in values.items():
        limit = 127 if name == 'product_name' else 255
        if not isinstance(value,str) or any(ord(char)<32 or ord(char)==127 for char in value):
            raise ValueError(f'app.toml {name} must be single-line text without control characters')
        if (not value.strip() and name != 'company_name') or len(value.encode('utf-16-le')) > limit*2:
            raise ValueError(f'app.toml {name} must fit {limit} UTF-16 characters')
    app = App(**values)
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]*\.exe',app.executable_name):
        raise ValueError('executable_name must be an ASCII .exe filename without directories or spaces')
    if app.executable_name.split('.')[0].upper() in {'CON','PRN','AUX','NUL',*(f'COM{i}' for i in range(1,10)),*(f'LPT{i}' for i in range(1,10))}:
        raise ValueError('executable_name is a reserved Windows device name')
    if not re.fullmatch(r'[a-z0-9]+(?:-[a-z0-9]+)*',app.package_name):
        raise ValueError('package_name must be a lowercase ASCII slug')
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]*',app.assembly_name):
        raise ValueError('assembly_name must be a Windows assembly identifier')
    icon = PurePosixPath(app.icon)
    if icon.is_absolute() or '..' in icon.parts or icon.suffix.lower() != '.ico' or any(char in app.icon for char in '\\<>:"|?*'):
        raise ValueError('icon must be a local relative .ico path using forward slashes')
    if any(part.endswith((' ','.')) for part in icon.parts) or not app.icon_path(root).is_file():
        raise ValueError('configured icon does not name an existing local ICO file')
    return app


def resource_string(value):
    """RC uses doubled quotes; keep Unicode explicit and backslashes literal."""
    return 'L"'+value.replace('\\','\\\\').replace('"','""')+'"'
