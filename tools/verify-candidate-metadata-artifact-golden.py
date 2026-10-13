"""Write independent FYMETA00 metadata bytes with explicit expected records."""
import argparse
import hashlib
import json
from pathlib import Path
import struct


TABLES = [
    ('layouts', [
        (4, 1, 3, 0, 2, 0, 1, 10, 50, 0, 0, 0, 1),
        (5, 0, 4, 2, 1, 1, 0, 60, 100, 0, 0, 3, 4),
    ]),
    ('fields', [
        (1, 1, 1, 3, 1, 20, 25, 1, 2, 0, 0),
        (1, 3, 1, 3, 2, 30, 35, 2, 3, 0, 0),
        (1, 1, 0, 4, 1, 70, 75, 4, 5, 0, 0),
    ]),
    ('callables', [
        (0, 1, 0, 1, 0, 1, 0, 1, 0, 0, 0, 0),
        (1, 8, 0, 1, 1, 1, 4, 0, 1, 1, 5, 19),
    ]),
    ('parameters', [(6,), (1,)]),
]
NAMES = b'SxyRnforeign_number'


def numbers(values):
    return struct.pack('>' + 'Q' * len(values), *values)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    data = bytearray(b'FYMETA00' + numbers((2, 6)))
    rows = []
    for name, values in TABLES:
        data.extend(numbers((len(values),)))
        start = len(data)
        for row in values:
            data.extend(numbers(row))
        rows.append(dict(table=name, firstRowOffset=start, rows=len(values),
                         rowBytes=len(values[0]) * 8))
    data.extend(numbers((len(NAMES),)) + NAMES)
    assert len(data) == 763
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(data)
    print(json.dumps(dict(schema='fairyc.candidate-metadata-golden/v0',
                          bytes=len(data), sha256=hashlib.sha256(data).hexdigest(),
                          rows=rows,
                          independentEncoding='Python big-endian uint64 with explicit expected records')))


if __name__ == '__main__':
    main()
