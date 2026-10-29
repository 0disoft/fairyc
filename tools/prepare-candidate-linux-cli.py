"""Prepare an isolated Linux candidate workspace; never overwrite authored sources."""
import argparse
from pathlib import Path
import shutil
import re

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'examples/lexer_candidate'
WORKSPACE_READERS = ('project_directory', 'project_source_catalog',
                     'project_source_file', 'project_source_set',
                     'project_compiler_image_hash')


def select_workspace_io(output):
    for name in WORKSPACE_READERS:
        path = output / 'src' / (name + '.fy')
        source = path.read_text(encoding='utf-8')
        old = 'import std::fs as fs;'
        if source.count(old) != 1:
            raise ValueError(f'{name} filesystem import changed')
        source = source.replace(old, 'import linux_workspace_fs as fs;')
        source = re.sub(r'\bfs::(FileError|FileKind|FileId|File|ReadDir|DirEntry|PathKind)\b',
                        r'fs::Workspace\1', source)
        path.write_text(source,
                        encoding='utf-8', newline='\n')
    path = output / 'src/project_lock_write.fy'
    source = path.read_text(encoding='utf-8')
    if source.count('import std::fs as fs;') != 1 or source.count('return .err(io(4,error));') != 2:
        raise ValueError('lock writer filesystem boundary changed')
    source = source.replace('import std::fs as fs;',
                            'import std::fs as fs;\nimport linux_workspace_fs as directory;')
    for operation in ('read_dir_utf8', 'close_dir', 'path_kind'):
        if source.count('fs::' + operation + '(') != 1:
            raise ValueError(f'lock writer {operation} boundary changed')
        source = source.replace('fs::' + operation + '(', 'directory::' + operation + '(')
    source = source.replace('return .err(io(4,error));', 'return .err(directory_io(4,error));')
    source += '\nfn Problem directory_io(u32 stage,directory::WorkspaceFileError error){var Problem p=fail(stage,error.status);p.os_code=error.os_code;p.completed=error.completed;return p;}\n'
    path.write_text(source, encoding='utf-8', newline='\n')


def select_linux_emission(output):
    path = output / 'src/module_unit_c_sdk.fy'
    source = path.read_text(encoding='utf-8')
    old = 'import module_unit_c_file as file_storage;'
    if source.count(old) != 1:
        raise ValueError('SDK file wire selection changed')
    path.write_text(source.replace(old, 'import linux_c_file as file_storage;'), encoding='utf-8', newline='\n')
    path = output / 'src/module_unit_c_emit.fy'
    source = path.read_text(encoding='utf-8')
    old = '#define FY_CANDIDATE_PROJECT_HEADER_V1\\n'
    if source.count(old) != 1:
        raise ValueError('candidate project header marker changed')
    path.write_text(source.replace(old, old+'/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\\n'), encoding='utf-8', newline='\n')


def select_linux_tests(output):
    path = output / 'src/project_test_cli.fy'
    source = path.read_text(encoding='utf-8')
    old = 'import native_cli as native;'
    if source.count(old) != 1:
        raise ValueError('test executor selection changed')
    path.write_text(source.replace(old, 'import linux_test_cli as native;'), encoding='utf-8', newline='\n')


def select_linux_cache(output):
    selections = {
        'project_cache_disk': ('import windows_rooted_file as rooted;\nimport windows_rooted_file as handles;',
                               'import linux_cache_file as rooted;\nimport linux_rooted_file as handles;'),
        'project_cache_environment': ('import windows_host_paths as paths;',
                                      'import linux_host_paths as paths;'),
        'project_cache_profile': ('x86_64-windows-msvc', 'x86_64-linux-gnu'),
    }
    for name, (old, new) in selections.items():
        path = output / 'src' / (name + '.fy')
        source = path.read_text(encoding='utf-8')
        if source.count(old) != (2 if name == 'project_cache_profile' else 1):
            raise ValueError(f'{name} cache platform boundary changed')
        source = source.replace(old, new)
        if name == 'project_cache_profile':
            old_abi = ';llp64;microsoft-x64;msvc-x64-llp64;microsoft-x64;coff-x86-64;windows-x64-pdata-xdata'
            if source.count(old_abi) != 1:
                raise ValueError('cache ABI profile boundary changed')
            source = source.replace(old_abi, ';lp64;sysv-amd64;sysv-amd64-lp64;sysv-amd64;elf64;dwarf-cfi')
        path.write_text(source, encoding='utf-8', newline='\n')


def prepare(output):
    output = output.resolve()
    if not output.is_relative_to((ROOT / '.cache').resolve()):
        raise ValueError('output must be inside the repository .cache directory')
    # Refuse reuse rather than mixing different source snapshots.
    output.mkdir(parents=True, exist_ok=False)
    shutil.copyfile(SOURCE / 'fairy.toml', output / 'fairy.toml')
    shutil.copytree(SOURCE / 'src', output / 'src')
    main = output / 'src/main.fy'
    source = main.read_text(encoding='utf-8')
    old = 'import file_cli as file_cli;'
    if source.count(old) != 1:
        raise ValueError('candidate main dispatcher import changed')
    main.write_text(source.replace(old, 'import linux_file_cli as file_cli;'),
                    encoding='utf-8', newline='\n')
    select_workspace_io(output)
    select_linux_emission(output)
    select_linux_tests(output)
    select_linux_cache(output)
    return output


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    print(prepare(args.output))


if __name__ == '__main__':
    main()
