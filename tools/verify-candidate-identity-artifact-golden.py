"""Independent FYIDS000 row encoding plus an existing locked Vec workspace."""
import argparse
import hashlib
import json
from pathlib import Path
import struct


def golden():
    a = b'g0_' + b'a' * 64
    b = b'g0_' + b'b' * 64
    types = [(0, 0, 0), (0, 0, 0), (17, 0, 67)]
    frames = [(23, 67, 134), (17, 134, 201), (0, 0, 0)]
    data = bytearray(b'FYIDS000' + struct.pack('>Q', len(types)))
    for row in types:
        data.extend(struct.pack('>3Q', *row))
    data.extend(struct.pack('>Q', len(frames)))
    for row in frames:
        data.extend(struct.pack('>3Q', *row))
    names = a + b + a
    data.extend(struct.pack('>Q', len(names)) + names)
    assert len(data) == 377
    return bytes(data)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--workspace', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    workspace = args.workspace.resolve()
    assert (workspace / 'fairy.toml').is_file() and (workspace / 'fairy.lock').is_file()
    payload = golden()
    path = workspace.as_posix().encode('utf-8')
    fixture = b'FYIDFIX0' + struct.pack('>Q', len(payload)) + payload + struct.pack('>Q', len(path)) + path
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(fixture)
    print(json.dumps(dict(schema='fairyc.candidate-identity-golden/v0', goldenBytes=len(payload),
                          goldenSHA256=hashlib.sha256(payload).hexdigest(), fixtureBytes=len(fixture),
                          fixtureSHA256=hashlib.sha256(fixture).hexdigest(), types=3, frames=3,
                          workspace=workspace.as_posix(),
                          workspaceInputs=[dict(path=p.relative_to(workspace).as_posix(),
                                                sha256=hashlib.sha256(p.read_bytes()).hexdigest())
                                           for p in sorted(workspace.rglob('*'))
                                           if p.is_file() and (p.suffix == '.fy' or p.name in ('fairy.toml', 'fairy.lock'))],
                          independentEncoding='Python struct big-endian uint64 and explicit row/name constants')))


if __name__ == '__main__':
    main()
