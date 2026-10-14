"""Create an isolated locked generic/Vec/state/field workspace for IR rebinding."""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import subprocess

ROOT = Path(__file__).resolve().parents[1]
def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()
def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache'), 'verification fixture must stay under the repository cache'
    workspace = output / 'workspace'
    source = workspace / 'src'
    source.mkdir(parents=True, exist_ok=True)
    (workspace / 'fairy.toml').write_text("schema='fairyc.manifest/v1'\n[package]\nuid='example.artifact.rebase'\nname='artifact-rebase'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n", encoding='utf-8', newline='\n')
    (source / 'helper.fy').write_text('module helper;pub struct S{i32 n;}pub fn T id<T>(T value){return value;}\n', encoding='utf-8', newline='\n')
    (source / 'main.fy').write_text('module main;import helper as h;import std::vec as vec;@contract_uid("example.artifact.rebase")state{cell u64 supply;map bytes20 u64 balances;}fn i32 main(){let h::S s=h::S{.n=42};var vec::Vec<i32> values=vec::empty<i32>();assert(vec::push<i32>(ref mut values,h::id<i32>(s.n)).is_ok);return *vec::at<i32>(ref values,0);}\n', encoding='utf-8', newline='\n')
    observations = []
    for flags in ([], ['--locked']):
        result = subprocess.run([str(args.compiler.resolve()), 'check', *flags], cwd=workspace, capture_output=True, timeout=120)
        row = dict(argv=['check', *flags], exit=result.returncode, stdout=result.stdout.decode(), stderr=result.stderr.decode())
        observations.append(row)
        assert result.returncode == 0 and not result.stderr, row
    # The existing Fairy fixture reader expects a framed golden member and path.
    # Rebinding tests use the locked workspace; they never consume golden rows.
    path = workspace.as_posix().encode()
    payload = b'FYIDFIX0' + struct.pack('>Q', 0) + struct.pack('>Q', len(path)) + path
    fixture = output / 'fixture.bin'
    fixture.write_bytes(payload)
    paths = [workspace/'fairy.toml', workspace/'fairy.lock', source/'helper.fy', source/'main.fy']
    ledger = [p.relative_to(ROOT).as_posix() for p in paths]
    (output / 'unchanged-workspace.json').write_text(json.dumps(ledger)+'\n', encoding='utf-8')
    record = dict(schema='fairyc.candidate-artifact-rebase-fixture/v0', passed=True,
                  compiler=dict(path=str(args.compiler.resolve()), sha256=digest(args.compiler.resolve())),
                  workspace=workspace.relative_to(ROOT).as_posix(), observations=observations,
                  fixture=dict(path=fixture.relative_to(ROOT).as_posix(), sha256=digest(fixture)),
                  workspaceInputs=[dict(path=p.relative_to(ROOT).as_posix(), sha256=digest(p)) for p in paths])
    (output/'fixture-receipt.json').write_text(json.dumps(record, indent=2)+'\n', encoding='utf-8')
    print(json.dumps(dict(passed=True, fixture=str(fixture), observations=observations)))
if __name__ == '__main__':
    main()
