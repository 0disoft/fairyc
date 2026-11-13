"""Prepare and validate a portable ZIP with an optional app-specific CLI check."""
import hashlib
import os
from pathlib import Path
import tempfile
import zipfile
from .resources import metadata
from .settings import load
from .pe import verify as resources

def license_path(root):
    local = root/'LICENSE'
    return local if local.is_file() else root.parents[1]/'LICENSE'


def package_files(root):
    return (load(root).executable_name,'README.txt','LICENSE.txt')


def instructions(root):
    app = load(root)
    version,_,_ = metadata(root)
    guide = (root/'distribution/README.ko.txt').read_text(encoding='utf-8')
    for name,value in {'PRODUCT':app.product_name,'VERSION':version,'EXECUTABLE':app.executable_name}.items():
        guide = guide.replace('@'+name+'@',value)
    return guide.encode('utf-8-sig')


def payload(image, root):
    names = package_files(root)
    files = dict(zip(names,(image.read_bytes(),instructions(root),license_path(root).read_bytes())))
    files['SHA256SUMS.txt'] = ''.join(f'{hashlib.sha256(files[name]).hexdigest()}  {name}\n' for name in names).encode('ascii')
    return files


def verify(archive, root, validate=None):
    names = package_files(root)
    with zipfile.ZipFile(archive) as package:
        entries = package.infolist()
        if len(entries) != 4 or {entry.filename for entry in entries} != {*names,'SHA256SUMS.txt'}:
            raise RuntimeError('unexpected/duplicate files in portable ZIP')
        if sum(entry.file_size for entry in entries) > 32*1024*1024:
            raise RuntimeError('portable ZIP exceeds expected size bound')
        if any(entry.flag_bits & 1 or (entry.external_attr >> 16) & 0o170000 == 0o120000 for entry in entries):
            raise RuntimeError('encrypted/symlink archive entry is not supported')
        if package.testzip() is not None: raise RuntimeError('ZIP CRC mismatch')
        files = {entry.filename:package.read(entry) for entry in entries}
    expected = ''.join(f'{hashlib.sha256(files[name]).hexdigest()}  {name}\n' for name in names).encode('ascii')
    if files['SHA256SUMS.txt'] != expected: raise RuntimeError('package SHA-256 checksum mismatch')
    if files['LICENSE.txt'] != license_path(root).read_bytes(): raise RuntimeError('license text mismatch')
    version,_,_ = metadata(root)
    if files['README.txt'] != instructions(root): raise RuntimeError('packaged instructions/version mismatch')
    with tempfile.TemporaryDirectory(prefix='Fairy Win32 패키지-') as directory:
        target = Path(directory)/names[0]
        target.write_bytes(files[names[0]])
        pe = resources(target,root)
        checks = validate(target) if validate else {}
    return dict(ok=True,verification='fairy-win32-portable-package',version=version,
                files=list(files),zipBytes=archive.stat().st_size,exeBytes=len(files[names[0]]),
                zipSha256=hashlib.sha256(archive.read_bytes()).hexdigest(),resources=pe,
                temporaryExtractionRemoved=True,guiLaunched=False,inputInjected=False,**checks)



def package(image, output, root, validate=None):
    app = load(root)
    version,_,_ = metadata(root)
    pe = resources(image,root)
    if pe['certificateTablePresent']:
        raise RuntimeError('this unsigned-package template does not describe signed images; update distribution instructions first')
    files = payload(image,root)
    output.mkdir(parents=True,exist_ok=True)
    archive = output/f'{app.package_name}-{version}-windows-x64.zip'
    with tempfile.TemporaryDirectory(prefix='.fairy-package-',dir=output) as directory:
        pending = Path(directory)/archive.name
        with zipfile.ZipFile(pending,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as bundle:
            for name,data in files.items():
                entry = zipfile.ZipInfo(name,date_time=(1980,1,1,0,0,0))
                entry.create_system = 3
                entry.external_attr = (0o100755 if name.endswith('.exe') else 0o100644)<<16
                entry.compress_type = zipfile.ZIP_DEFLATED
                bundle.writestr(entry,data,compresslevel=9)
        result = verify(pending,root,validate)
        checksum = archive.with_suffix(archive.suffix+'.sha256')
        sidecar = Path(directory)/checksum.name
        sidecar.write_text(f'{result["zipSha256"]}  {archive.name}\n',encoding='ascii',newline='\n')
        os.replace(pending,archive)
        os.replace(sidecar,checksum)
    result.update(archive=str(archive),checksumFile=str(checksum),published=False,signed=False)
    return result


