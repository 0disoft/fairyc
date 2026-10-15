"""Create private Windows source-catalog fixtures without removing existing paths."""
import argparse
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--root', type=Path, required=True)
    parser.add_argument('--root-bin', type=Path, required=True)
    parser.add_argument('--wide', action='store_true', help='include a 512-file shared-work regression directory')
    args = parser.parse_args()
    if os.name != 'nt':
        parser.error('the current SDK directory adapter requires Windows')
    base = args.root.resolve()
    for name in ['Good/nested', 'Empty', 'Collision', 'Reparse', 'IgnoredReparse', 'Reads', 'Package/src/nested',
                 'ModuleOrder/a', 'BadModule', 'BadCase', 'SecondModule', 'MissingModule', 'BrokenSyntax',
                 'InvalidUtf8', 'UnicodeFile', 'UnicodeDir/Ü', 'HyphenDir/bad-name', 'LargeModules', 'DuplicateDeclared', 'LongModuleTrivia/net']:
        (base / name).mkdir(parents=True, exist_ok=True)
    sources = {
        'Good/a.fy': 'module a;\n',
        'Good/z.fy': 'module z;\n',
        'Good/nested/z.fy': 'module nested::z;\n',
        'Good/ignored.FY': 'not source',
        'Good/readme.txt': 'not source',
        'Collision/İ.fy': 'module first;\n',
        'Collision/i.fy': 'module second;\n',
        'Reparse/a.fy': 'module a;\n',
        'IgnoredReparse/a.fy': 'module a;\n',
        'Package/src/a.fy': 'module a;\n',
        'Package/src/z.fy': 'module z;\n',
        'Package/src/nested/z.fy': 'module nested::z;\n',
        'Package/fairy.toml': "schema='fairyc.manifest/v1'\n[package]\nuid='example.catalog'\nname='catalog'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n",
        'ModuleOrder/a/x.fy': 'module a::x;\n',
        'ModuleOrder/a0.fy': 'module a0;\n',
        'BadModule/a.fy': 'module other;\n',
        'BadCase/a.fy': 'module A;\n',
        'SecondModule/a.fy': 'module a;\nmodule b;\n',
        'MissingModule/a.fy': 'fn i32 main(){return 0;}\n',
        'BrokenSyntax/a.fy': 'module a;\nbogus;\n',
        'UnicodeFile/Ü.fy': 'module a;\n',
        'UnicodeDir/Ü/a.fy': 'module a;\n',
        'HyphenDir/bad-name/a.fy': 'module a;\n',
        'DuplicateDeclared/a.fy': 'module a;\n',
        'DuplicateDeclared/b.fy': 'module a;\n',
    }
    for name, source in sources.items():
        (base / name).write_text(source, encoding='utf-8', newline='\n')
    if args.wide:
        (base / 'Wide').mkdir(exist_ok=True)
        for index in range(512):
            (base / 'Wide' / f'p{index}_module_unit_long_source_fixture_for_work.fy').write_bytes(b'module fixture;\n')
    (base / 'Reads/binary.fy').write_bytes(b'\x00\xff\r')
    (base / 'Reads/large.fy').write_bytes(b'a' * 65536 + b'z')
    (base / 'Reads/empty.fy').write_bytes(b'')
    (base / 'InvalidUtf8/a.fy').write_bytes(b'module a;\n\xff')
    (base / 'LargeModules/large.fy').write_bytes(b'module large;\n' + b' ' * (65537 - 14))
    (base / 'LongModuleTrivia/net/http.fy').write_bytes(b'module net /*' + b't' * 70000 + b'*/ :: http;\n')
    for name, size in [('maximum.fy', 8388608), ('too-large.fy', 8388609)]:
        with (base / 'Reads' / name).open('wb') as file:
            file.truncate(size)
    if len(list((base / 'Collision').glob('*.fy'))) != 2:
        raise RuntimeError('fixture filesystem must retain both dotted I and i')
    target = base / 'Good'
    for name in ['Reparse/linked', 'IgnoredReparse/linked.txt']:
        path = base / name
        if os.path.lexists(path):
            if not os.path.isjunction(path) or path.resolve() != target:
                raise RuntimeError(f'conflicting existing fixture path: {path}')
            continue
        quote = lambda value: "'" + str(value).replace("'", "''") + "'"
        subprocess.run(['powershell', '-NoProfile', '-NonInteractive', '-Command',
                        'New-Item -ItemType Junction -Path ' + quote(path) +
                        ' -Target ' + quote(target) + ' -ErrorAction Stop | Out-Null'],
                       check=True, capture_output=True)
    args.root_bin.parent.mkdir(parents=True, exist_ok=True)
    args.root_bin.write_bytes(str(base).encode('utf-8'))


if __name__ == '__main__':
    main()
