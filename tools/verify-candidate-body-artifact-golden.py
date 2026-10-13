"""Write independent FYBODY00 bytes and the authenticated allocator fixture."""
import argparse
import hashlib
import json
from pathlib import Path
import struct


# Explicit records for the synthetic verified return-42 body. Do not derive
# these bytes from the Fairy encoder or decoder: this is their independent oracle.
NODE = (
    0, 0, 1, 0, 1, 3, 5, 0, 0, 0,  # value / source expression
    0, 0, 0, 0, 0,                 # field
    0, 0, 0, 0, 0, 0, 0, 0,        # place
    0, 0, 0, 0,                    # builtin
    1, 1, 1, 1, 0,                 # expected / type / state / flexible / minimum
    1, 3, 5, 1, 0, 0,              # syntax expression
    3, 0, 2, 2, 2,                 # literal kind and raw / cooked text offsets
)
TABLES = [
    ('nodes', [NODE], 344),
    ('statements', [
        (1, 5, 2, 6, 1, 0, 0, 0, 0, 0, 2, 2),
        (2, 7, 2, 6, 0, 0, 1, 0, 0, 0, 2, 2),
        (3, 1, 0, 10, 0, 0, 2, 1, 0, 0, 2, 2),
    ], 96),
    ('locals', [], 144), ('calls', [], 88), ('arguments', [], 96),
    ('regions', [], 120), ('placements', [], 80), ('resets', [], 56),
]


def numbers(values):
    return struct.pack('>' + 'Q' * len(values), *values)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    data = bytearray(b'FYBODY00' + numbers((1, 3, 10, 0, 10)))
    rows = []
    for name, values, width in TABLES:
        data.extend(numbers((len(values),)))
        start = len(data)
        for row in values:
            assert len(row) * 8 == width
            data.extend(numbers(row))
        rows.append(dict(table=name, firstRowOffset=start,
                         rows=len(values), rowBytes=width))
    data.extend(numbers((2,)) + b'42')
    assert len(data) == 754
    root = Path(__file__).resolve().parents[1]
    allocator = root / 'internal/sdkstdlib/sources/std/alloc.fy'
    source = allocator.read_bytes()
    fixture = b'FYBGOLD0' + numbers((len(data),)) + data + numbers((len(source),)) + source
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_bytes(fixture)
    print(json.dumps(dict(schema='fairyc.candidate-body-golden/v0',
                          bodyBytes=len(data), bodySHA256=hashlib.sha256(data).hexdigest(),
                          fixtureBytes=len(fixture), fixtureSHA256=hashlib.sha256(fixture).hexdigest(),
                          allocatorSource=allocator.relative_to(root).as_posix(),
                          allocatorSHA256=hashlib.sha256(source).hexdigest(), rows=rows,
                          independentEncoding='Python big-endian uint64 with explicit expected records')))


if __name__ == '__main__':
    main()
