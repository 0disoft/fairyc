"""Compare generated C and native results before/after project literal accounting changes."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--seed', type=Path, required=True)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--builder', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    output.mkdir(parents=True)
    tools = {key: dict(path=str(path.resolve()), sha256=sha(path))
             for key, path in [('seed', args.seed), ('candidate', args.candidate),
                               ('builder', args.builder)]}
    cases = {
        'duplicate-empty-nul': {'main.fy':
            'module main;fn i32 main(){let ref u8[] a=b"shared\\x00bytes";'
            'let ref u8[] b=b"shared\\x00bytes";let ref str empty="";'
            'return if(a.len==12 && b[6]==0 && empty.len==0){0}else{1};}\n'},
        'shared-unicode': {
            'main.fy': 'module main;import dependency as dep;fn i32 main(){'
                       'let ref str a="한글😀";let b=dep::value();'
                       'return if(a.len==10 && b.len==10){0}else{1};}\n',
            'dependency.fy': 'module dependency;pub fn ref str value(){return "한글😀";}\n'},
    }
    rows = []
    for name, files in cases.items():
        directory = output / name
        directory.mkdir()
        paths = []
        for filename, text in files.items():
            path = directory / filename
            path.write_text(text, encoding='utf-8', newline='\n')
            paths.append(path)
        manifest = directory / 'sources.list'
        manifest.write_text(''.join(path.relative_to(ROOT).as_posix()+'\n' for path in paths),
                            encoding='utf-8', newline='\n')
        packets = []
        for engine in ['seed', 'candidate']:
            result = subprocess.run([tools[engine]['path'], '--c-project-file', str(manifest)],
                                    cwd=ROOT, capture_output=True, timeout=90)
            assert result.returncode == 0 and not result.stderr, (name, engine, result.stderr)
            assert result.stdout.startswith(b'FYCPARTS1\n')
            packet = directory / (engine+'.bin')
            packet.write_bytes(result.stdout)
            packets.append(packet)
        assert packets[0].read_bytes() == packets[1].read_bytes(), name
        image = directory / 'program.exe'
        built = subprocess.run([tools['builder']['path'], '--packet', str(packets[1]),
                                '--packet-sha256', sha(packets[1]), '--output', str(image),
                                '--release'], cwd=ROOT, capture_output=True, timeout=120)
        assert built.returncode == 0 and not built.stderr, (name, built.stderr)
        ran = subprocess.run([str(image)], cwd=directory, capture_output=True, timeout=10)
        assert ran.returncode == 0 and not ran.stdout and not ran.stderr, name
        rows.append(dict(case=name, byteIdentical=True, packetSHA256=sha(packets[1]),
                         nativeExit=ran.returncode, sourceInputs=[
                             dict(path=path.relative_to(ROOT).as_posix(), sha256=sha(path))
                             for path in paths]))
        print('passed:', name, flush=True)
    assert all(sha(Path(row['path'])) == row['sha256'] for row in tools.values())
    receipt = dict(schema='fairyc.candidate-project-literal-controls/v1', passed=True,
                   tools=tools, cases=rows, goFrontendInvoked=False)
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2)+'\n', encoding='utf-8')


if __name__ == '__main__':
    main()
