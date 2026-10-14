"""Create the nine source-owned workspaces for private project lookup tests."""
import argparse
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache'), 'fixture must stay under repository .cache'
    assert not output.exists(), 'use a fresh output directory to preserve existing fixtures'
    manifest = "schema='fairyc.manifest/v1'\n[package]\nuid='example.keys.{uid}'\nname='complete-keys'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n"
    left = "[dependencies.a]\npath='left'\nversion='1.0.0'\nfeatures=['fast']\n"
    right = "[dependencies.r]\npath='right'\nversion='1.0.0'\nfeatures=[]\n"
    alias = "[dependencies.alias-left]\npath='left'\nversion='1.0.0'\nfeatures=['fast']\n"
    root_manifest = manifest.format(uid='root') + "[workspace]\nmembers=['left','right']\n"
    ordinary = 'module main;import left as a;import right as r;fn i32 main(){return 42;}\n'
    generic = 'module main;import left as a;import left as b;import right as r;fn i32 constant<T>(){return 42;}fn i32 main(){discard constant<u16>();discard constant<u64>();return b::id<i32>(42);}\n'
    sources = {
        'Aliases': generic, 'Reordered': generic, 'Plain': ordinary, 'PrivateEdit': generic,
        'State': 'module main;import left as a;import right as r;@contract_uid("example.keys")state{cell u64 total;map bytes20 u64 balances;}fn i32 main(){return a::id<i32>(42);}\n',
        'Vec': 'module main;import left as a;import left as b;import right as r;import std::vec as v;fn i32 constant<T>(){return 42;}fn i32 main(){discard constant<u16>();discard constant<u64>();var v::Vec<i32> items=v::empty<i32>();discard v::push<i32>(ref mut items,a::id<i32>(1));return b::id<i32>(42);}\n',
        'BindingEdit': generic, 'FeatureEdit': generic, 'InvalidBody': ordinary.replace('return 42;', 'return true;'),
    }
    records = []
    for case, body in sources.items():
        dependencies = right + left + alias if case == 'Reordered' else left + right + alias
        if case == 'BindingEdit':
            dependencies = dependencies.replace('[dependencies.alias-left]', '[dependencies.other-left]')
        if case == 'FeatureEdit':
            dependencies = dependencies.replace("features=['fast']", "features=['slow']")
        provider = 'module left;pub fn T id<T>(T x){return copy<T>(x);}fn T copy<T>(T x){return x;}\n'
        if case == 'PrivateEdit':
            provider = provider.replace('return x;', 'let T result=x;return result;')
        files = {'fairy.toml': root_manifest + dependencies, 'src/main.fy': body,
                 'left/fairy.toml': manifest.format(uid='left'), 'left/src/left.fy': provider,
                 'right/fairy.toml': manifest.format(uid='right'), 'right/src/right.fy': 'module right;pub fn i32 value(){return 7;}\n'}
        for relative, text in files.items():
            path = output / 'fixture' / case / relative
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text, encoding='utf-8', newline='\n')
            records.append(dict(path=path.relative_to(ROOT).as_posix(), sha256=hashlib.sha256(path.read_bytes()).hexdigest()))
    (output / 'root.txt').write_text(str(output / 'fixture'), encoding='utf-8')
    (output / 'unchanged.json').write_text(json.dumps([row['path'] for row in records]) + '\n', encoding='utf-8')
    (output / 'fixture-receipt.json').write_text(json.dumps(dict(schema='fairyc.candidate-project-lookup-fixture/v0', sourceInputs=records), indent=2) + '\n', encoding='utf-8')
    print('project lookup fixture created:', len(sources), 'workspaces')


if __name__ == '__main__':
    main()
