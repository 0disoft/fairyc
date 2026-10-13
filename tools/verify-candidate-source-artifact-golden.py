"""Independent FYPROV00 encoding of exact source/coordinate/import stamps."""
import argparse
import hashlib
import json
from pathlib import Path
import struct

SOURCES = [
    b'module root;import common as a;import common as a0;pub fn T id<T>(T value){return value;}',
    b'module common;pub fn T first<T>(T value){return value;}',
    b'module common;pub fn T second<T>(T value){return value;}',
]


def golden():
    names = bytearray()
    def retain(raw):
        first = len(names)
        names.extend(raw)
        return first, len(names)
    modules = []
    namespaces = []
    for i, source in enumerate(SOURCES):
        ns = b'package_coordinate_v1_' + str(i + 1).encode() * 64
        namespaces.append(ns)
        namespace = retain(ns)
        path = retain(b'root' if i == 0 else b'common')
        modules.append((i, *namespace, *path, 0 if i == 0 else 2,
                        2 if i == 0 else 0, len(source),
                        *struct.unpack('>8I', hashlib.sha256(source).digest())))
    rows = []
    for declaration, alias in [(2, b'a'), (3, b'a0')]:
        a = retain(alias)
        b = retain(alias + b'=' + namespaces[1] + b'::common')
        rows.append((0, declaration, 1, *a, *b))
    rows.reverse()  # a0= sorts before a=; packed text retains declaration order.
    wire = bytearray(b'FYPROV00' + struct.pack('>Q', len(modules)))
    for row in modules:
        wire.extend(struct.pack('>16Q', *row))
    wire.extend(struct.pack('>Q', len(rows)))
    for row in rows:
        wire.extend(struct.pack('>7Q', *row))
    wire.extend(struct.pack('>Q', len(names)) + names)
    return bytes(wire)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    data = golden()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(data)
    print(json.dumps(dict(schema='fairyc.candidate-source-golden/v0', bytes=len(data),
                          sha256=hashlib.sha256(data).hexdigest(), modules=3, imports=2,
                          sourceDigests=[hashlib.sha256(s).hexdigest() for s in SOURCES],
                          independentEncoding='Python struct big-endian uint64 and hashlib source bytes')))


if __name__ == '__main__':
    main()
