"""Write independent FYSTATE0 bytes and canonical state hashes."""
import argparse
import hashlib
import json
from pathlib import Path
import struct


CODECS = ('', 'bool/v1', 'u32/be/v1', 'u64/be/v1', 'bytes20/v1', 'bytes32/v1')
UID = b'example.token'
FIELDS = [('total_supply', 1, 0, 3, 20, 30), ('balances', 2, 4, 3, 40, 50),
          ('paused', 1, 0, 1, 60, 70), ('count', 1, 0, 2, 80, 90),
          ('root', 1, 0, 5, 100, 110)]


def string(raw):
    return struct.pack('>Q', len(raw)) + raw


def identity(index, name, kind, key, value):
    return (struct.pack('>IB', index, kind) + string(name.encode())
            + string(CODECS[key].encode()) + string(CODECS[value].encode()))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    names = bytearray(UID)
    rows, prefixes = [], []
    for i, (name, kind, key, value, start, end) in enumerate(FIELDS):
        preimage = identity(i, name, kind, key, value)
        prefix = hashlib.sha256(b'fairyc-contract-state-field/v1' + string(UID) + preimage).digest()
        prefixes.append(prefix)
        first = len(names)
        names.extend(name.encode())
        rows.append((kind, key, value, first, len(names), start, end,
                     *struct.unpack('>8I', prefix)))
    layout_preimage = b'fairyc-contract-state-layout/v1' + string(UID) + struct.pack('>I', len(FIELDS))
    for i, (name, kind, key, value, _, _) in enumerate(FIELDS):
        layout_preimage += identity(i, name, kind, key, value) + prefixes[i]
    digest = hashlib.sha256(layout_preimage).digest()
    # Original owner/module/declaration and byte spans are transport data;
    # canonical identity hashes intentionally exclude these coordinates.
    layout = (7, 2, 3, 10, 150, 0, len(UID), 0, len(FIELDS), *struct.unpack('>8I', digest))
    data = bytearray(b'FYSTATE0' + struct.pack('>Q', 1) + struct.pack('>17Q', *layout)
                     + struct.pack('>Q', len(rows)))
    for row in rows:
        data.extend(struct.pack('>15Q', *row))
    data.extend(string(names))
    assert len(data) == 816
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(data)
    print(json.dumps(dict(schema='fairyc.candidate-state-golden/v0', bytes=len(data),
                          sha256=hashlib.sha256(data).hexdigest(), layoutDigest=digest.hex(),
                          fieldPrefixes=[p.hex() for p in prefixes],
                          independentEncoding='Python big-endian uint64 and hashlib canonical preimages')))


if __name__ == '__main__':
    main()
