"""Create bounded source workspaces for authored mixed-package cache tests."""
import argparse
import hashlib
import json
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    output = args.output.resolve()
    assert output.is_relative_to(root / '.cache') and not output.exists()
    output.mkdir(parents=True)
    manifest = "schema='fairyc.manifest/v1'\n[package]\nuid='example.mixed.{uid}'\nname='mixed-cache'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n"
    dependencies = "[workspace]\nmembers=['helper','other']\n[dependencies.helper]\npath='helper'\nversion='1.0.0'\nfeatures=[]\n[dependencies.other]\npath='other'\nversion='1.0.0'\nfeatures=[]\n"
    rows = []
    for case in ['Base', 'Private', 'Root', 'Public', 'Invalid', 'Generic', 'GenericEdit', 'Vec', 'VecEdit']:
        expression = 'h::get()+o::one()'
        helper = 'module helper;pub fn i32 get(){return secret();}fn i32 secret(){return 42;}\n'
        if case in ('Private', 'VecEdit'):
            helper = helper.replace('42', '43')
        if case == 'Root':
            expression += '+1'
        if case == 'Public':
            helper = helper.replace('pub fn', '@effects(console) pub fn')
        if case == 'Invalid':
            helper = helper.replace('return 42', 'return true')
        if case.startswith('Generic'):
            expression = 'h::id<i32>(41)+o::one()'
            helper = 'module helper;pub fn T id<T>(T n){return '+('n+1' if case == 'GenericEdit' else 'n')+';}\n'
        source = 'module main;import helper as h;import other as o;@effects(console) fn i32 main(){return '+expression+';}\n'
        if case.startswith('Vec'):
            source = 'module main;import helper as h;import other as o;import std::vec as v;@effects(console) fn i32 main(){var v::Vec<i32> items=v::empty<i32>();discard v::push<i32>(ref mut items,h::get());return *v::at<i32>(ref items,0)+o::one();}\n'
        files = {'fairy.toml': manifest.format(uid='root')+dependencies,
                 'src/main.fy': source,
                 'helper/fairy.toml': manifest.format(uid='helper'),
                 'helper/src/helper.fy': helper,
                 'other/fairy.toml': manifest.format(uid='other'),
                 'other/src/other.fy': 'module other;pub fn i32 one(){return 1;}\n'}
        for name, text in files.items():
            path = output / case / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding='utf-8', newline='\n')
            rows.append(dict(path=path.relative_to(root).as_posix(), sha256=hashlib.sha256(path.read_bytes()).hexdigest()))
    (output / 'root.txt').write_text(output.as_posix(), encoding='utf-8', newline='\n')
    (output / 'inputs.json').write_text(json.dumps(rows, indent=2)+'\n', encoding='utf-8', newline='\n')
    print('Created', len(rows), 'mixed-cache fixture inputs')


if __name__ == '__main__':
    main()
