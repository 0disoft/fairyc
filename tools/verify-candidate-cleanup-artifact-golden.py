"""Write independent expected bytes for the FYCLEAN0 cleanup artifact fixture."""
import argparse
import hashlib
import json
from pathlib import Path
import struct


# Explicit fixture records, independent of the Fairy codec implementation.
# Integers below include nominal/provider coordinates, Option/Result glue,
# original field order, a cleanup receiver, a generic frame, and partial drops.
TABLES = [
    ('types', [
        (1, 1, 0, 0, 0, 0), (1, 7, 0, 0, 0, 0),
        (3, 0, 1, 3, 0, 0), (5, 0, 0, 0, 3, 0),
        (6, 0, 0, 0, 4, 1), (3, 0, 0, 9, 0, 0),
    ]),
    ('glues', [
        (0, 0, 0, 0, 0, 0, 0), (0, 0, 0, 0, 0, 0, 0),
        (1, 2, 1, 1, 7, 0, 1), (2, 0, 0, 0, 0, 0, 0),
        (3, 0, 0, 0, 0, 0, 0), (1, 0, 0, 0, 1, 1, 2),
    ]),
    ('frames', [
        (1, 1, 0, 1, 1, 1, 0, 0),
        (2, 2, 1, 3, 2, 1, 1, 3),
        (3, 3, 1, 7, 1, 1, 0, 0),
    ]),
    ('fields', [(3, 0, 1), (6, 0, 5), (6, 1, 3)]),
    ('operands', [(1, 1, 2, 6, 100, 110), (1, 2, 3, 3, 120, 130)]),
    ('drops', [(1, 1, 110, 111), (2, 6, 130, 131)]),
]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    data = bytearray(b'FYCLEAN0' + struct.pack('>Q', 0))
    rows = []
    for name, values in TABLES:
        if name != 'glues':
            data.extend(struct.pack('>Q', len(values)))
        start = len(data)
        for row in values:
            data.extend(struct.pack('>' + 'Q' * len(row), *row))
        rows.append(dict(table=name, firstRowOffset=start,
                         rows=len(values), rowBytes=8 * len(values[0])))
    assert len(data) == 1104
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(data)
    print(json.dumps(dict(schema='fairyc.candidate-cleanup-golden/v0',
                          bytes=len(data), sha256=hashlib.sha256(data).hexdigest(),
                          rows=rows,
                          independentEncoding='Python big-endian uint64 with explicit expected records')))


if __name__ == '__main__':
    main()
