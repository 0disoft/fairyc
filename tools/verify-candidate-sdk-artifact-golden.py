"""Write independent FYSDK000 bytes from exact current alloc/io SDK sources."""
import argparse
import hashlib
import json
from pathlib import Path
import struct


def numbers(*values):
    return struct.pack('>' + 'Q' * len(values), *values)


def blob(data):
    return numbers(len(data)) + data


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    paths = [root / 'internal/sdkstdlib/sources/std' / name for name in ('alloc.fy', 'io.fy')]
    alloc, io = [p.read_bytes() for p in paths]
    # User modules at 0 and 2 have no retained SDK source bytes.
    data = (b'FYSDK000' + numbers(4, 2) + numbers(1, 1, 0, len(alloc))
            + numbers(3, 4, len(alloc), len(alloc) + len(io)) + blob(alloc + io))
    fixture = b'FYSGOLD0' + blob(data) + blob(alloc) + blob(io)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(fixture)
    print(json.dumps(dict(schema='fairyc.candidate-sdk-golden/v0',
                          payloadBytes=len(data), payloadSHA256=hashlib.sha256(data).hexdigest(),
                          fixtureBytes=len(fixture), fixtureSHA256=hashlib.sha256(fixture).hexdigest(),
                          sources=[dict(path=p.relative_to(root).as_posix(), bytes=len(raw),
                                        sha256=hashlib.sha256(raw).hexdigest())
                                   for p, raw in zip(paths, (alloc, io))],
                          independentEncoding='Python big-endian uint64 with explicit module/SDK rows')))


if __name__ == '__main__':
    main()
