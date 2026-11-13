"""Build a differently branded copy and verify it using only PE/CLI operations."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
from app_settings import ROOT
from make_icon import render
from prepare_resources import prepare
from package_release import package


def verify(compiler):
    unchanged = {name:(ROOT/name).read_bytes() for name in ('app.toml','fairy.toml','native/app.rc','native/app.ico')}
    with tempfile.TemporaryDirectory(prefix='fairy-app-config-') as directory:
        root = Path(directory)/'repo/examples/win32_hash'
        root.mkdir(parents=True)
        for name in ('src','native','distribution'): shutil.copytree(ROOT/name,root/name)
        for name in ('fairy.toml','fairy.lock','fairy.windows.json'): shutil.copy2(ROOT/name,root/name)
        shutil.copy2(ROOT.parents[1]/'LICENSE',root.parents[1]/'LICENSE')
        settings = dict(product_name='검증 "해시" Ω',company_name='',description='테스트 "개발자" & 도구',
                        executable_name='BrandHash.exe',package_name='brand-hash',assembly_name='Brand.Hash',icon='custom/brand.ico')
        custom = root/'custom'; custom.mkdir()
        alternate = render((ROOT/'native/app.svg').read_bytes().replace(b'#286454',b'#4e507a'))
        (custom/'brand.ico').write_bytes(alternate)
        if alternate == unchanged['native/app.ico']: raise RuntimeError('custom icon fixture did not change')
        def write_settings(values):
            (root/'app.toml').write_text('[app]\n'+''.join(f'{key} = {json.dumps(value,ensure_ascii=False)}\n' for key,value in values.items()),encoding='utf-8',newline='\n')
        write_settings(settings); prepare(root)
        command = subprocess.run([str(compiler),'build','--locked','--release'],cwd=root,capture_output=True,timeout=180)
        if command.returncode: raise RuntimeError('custom-brand build failed: '+command.stderr.decode('utf-8',errors='replace'))
        result = package(root/'build/win32-hash.exe',root/'dist',root)
        if result['resources']['windowTitle'] != settings['product_name'] or result['resources']['strings']['CompanyName'] != '':
            raise RuntimeError('custom title/empty publisher was not preserved')
        if result['resources']['strings']['FileDescription'] != settings['description'] or result['resources']['strings']['InternalName'] != 'BrandHash':
            raise RuntimeError('custom description/internal name was not preserved')
        if result['files'][0] != 'BrandHash.exe' or Path(result['archive']).name != 'brand-hash-0.1.0-windows-x64.zip':
            raise RuntimeError('package names still use hardcoded defaults')
        saved_rc = (root/'native/app.rc').read_bytes()
        archive = Path(result['archive']); saved_archive = archive.read_bytes()
        write_settings(dict(settings,executable_name='../escape.exe'))
        try: prepare(root)
        except ValueError: pass
        else: raise RuntimeError('invalid executable path was accepted')
        if (root/'native/app.rc').read_bytes() != saved_rc or archive.read_bytes() != saved_archive:
            raise RuntimeError('invalid settings changed the generated resource or prior ZIP')
        source_unchanged = all((ROOT/name).read_bytes()==data for name,data in unchanged.items())
        if not source_unchanged: raise RuntimeError('verification changed the original app settings')
        return dict(ok=True,verification='fairy-hash-app-settings',customTitle=result['resources']['windowTitle'],
            emptyPublisherRoundTrip=True,customIconSha256=hashlib.sha256(alternate).hexdigest(),
            customNamesVerified=True,unicodeAndQuotesRoundTrip=True,cliCases=result['cliCases'],comparisonCases=result['comparisonCases'],checksumCases=result['checksumCases'],
            invalidPathRejected=True,priorArtifactsPreserved=True,sourceSettingsUnchanged=True,
            guiLaunched=False,inputInjected=False)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler',type=Path,required=True)
    args = parser.parse_args()
    try: print(json.dumps(verify(args.compiler.resolve())))
    except (OSError,ValueError,RuntimeError,subprocess.TimeoutExpired) as error:
        print(json.dumps(dict(ok=False,error=str(error)))); raise SystemExit(1)
