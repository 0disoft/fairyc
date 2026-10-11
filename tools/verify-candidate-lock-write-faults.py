"""Build the Fairy-owned writer probe and inject test-only Win32 failures."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--input', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=True)
    main_source = ROOT / 'examples/lexer_candidate/probes/project_lock_write_faults.fy'
    modules = {'main': main_source}
    for base in (ROOT / 'examples/lexer_candidate/src', ROOT / 'internal/sdkstdlib/sources'):
        for path in base.rglob('*.fy'):
            name = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]
            if name != 'main':
                modules[name] = path
    pending, seen = ['main'], set()
    while pending:
        name = pending.pop()
        if name not in seen:
            seen.add(name)
            pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    paths = [main_source] + [modules[name] for name in sorted(seen - {'main'})]
    manifest = output / 'sources.list'
    manifest.write_text(''.join(p.relative_to(ROOT).as_posix() + '\n' for p in paths), encoding='utf-8')
    tracked = paths + [Path(__file__), ROOT / 'cmd/fairyc-candidate-lock-write-faults/main.go', args.candidate.resolve(), args.input.resolve()]
    hashes = {str(p): hashlib.sha256(p.read_bytes()).hexdigest() for p in tracked}
    packet = output / 'packet.bin'
    emitted = subprocess.run([str(args.candidate.resolve()), '--c-project-file', str(manifest)], cwd=ROOT, capture_output=True, timeout=600)
    if emitted.returncode != 0 or emitted.stderr:
        raise RuntimeError('candidate emission failed: ' + emitted.stderr.decode('utf-8', errors='replace'))
    packet.write_bytes(emitted.stdout)
    subprocess.run(['bun', 'tools/run-go.ts', 'run', './cmd/fairyc-candidate-lock-write-faults', str(packet), str(args.input.resolve()), str(output)], cwd=ROOT, check=True)
    for path, digest in hashes.items():
        if hashlib.sha256(Path(path).read_bytes()).hexdigest() != digest:
            raise RuntimeError('verification input changed: ' + path)
    receipt_path = output / 'receipt.json'
    receipt = json.loads(receipt_path.read_text(encoding='utf-8'))
    receipt['sourceInputs'] = [{'path': Path(p).relative_to(ROOT).as_posix(), 'sha256': h} for p, h in hashes.items()]
    receipt_path.write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print('writer faults and termination passed:', len(receipt['observations']), 'observations', flush=True)


if __name__ == '__main__':
    main()
