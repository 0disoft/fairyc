"""Measure a local AMD64 PE image without running it or needing extra packages."""
import argparse
import hashlib
import json
from pathlib import Path
import struct


def inspect(path):
    data = path.read_bytes()
    def number(offset, kind='I'):
        return struct.unpack_from('<' + kind, data, offset)[0]
    pe = number(0x3c)
    if data[:2] != b'MZ' or data[pe:pe + 4] != b'PE\0\0' or number(pe + 4, 'H') != 0x8664:
        raise ValueError('expected an AMD64 PE executable')
    optional = pe + 24
    if number(optional, 'H') != 0x20b:
        raise ValueError('expected PE32+')
    headers = number(optional + 60)
    sections = []
    for i in range(number(pe + 6, 'H')):
        at = optional + number(pe + 20, 'H') + 40 * i
        name = data[at:at + 8].rstrip(b'\0').decode('ascii')
        virtual, rva, raw, offset = struct.unpack_from('<IIII', data, at + 8)
        if offset + raw > len(data):
            raise ValueError('section exceeds image')
        sections.append(dict(name=name, fileBytes=raw, virtualBytes=virtual,
                             alignmentPaddingBytes=max(0, raw - virtual), rva=rva, offset=offset))
    def locate(rva):
        if rva < headers:
            return rva
        for section in sections:
            delta = rva - section['rva']
            if 0 <= delta < section['fileBytes']:
                return section['offset'] + delta
        raise ValueError('RVA is outside file-backed sections')
    def string(rva):
        at = locate(rva)
        return data[at:data.index(b'\0', at)].decode('ascii')
    imports = {}
    import_rva = number(optional + 112 + 8)
    if import_rva:
        at = locate(import_rva)
        for _ in range(256):
            original, stamp, chain, name, thunk = struct.unpack_from('<IIIII', data, at)
            if not any((original, stamp, chain, name, thunk)):
                break
            symbols = []
            entry = locate(original or thunk)
            for _ in range(65536):
                value = number(entry, 'Q')
                if value == 0:
                    break
                symbols.append('ordinal:' + str(value & 0xffff) if value >> 63 else string(value + 2))
                entry += 8
            else:
                raise ValueError('unbounded import table')
            imports[string(name)] = symbols
            at += 20
        else:
            raise ValueError('unbounded import descriptors')
    raw_bytes = sum(section['fileBytes'] for section in sections)
    end = max([headers] + [section['offset'] + section['fileBytes'] for section in sections])
    return dict(schema='fairyc.win32-size/v1', image=str(path), sha256=hashlib.sha256(data).hexdigest(),
                fileBytes=len(data), headersBytes=headers, sectionFileBytes=raw_bytes,
                unassignedFileBytes=len(data) - headers - raw_bytes, overlayBytes=len(data) - end,
                mappedImageBytes=number(optional + 56), subsystem=number(optional + 68, 'H'),
                coffSymbolCount=number(pe + 16), sections=sections, imports=imports,
                attribution='Section/import measurements; linked CRT and function sizes are not separately attributed.')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('image', type=Path)
    args = parser.parse_args()
    print(json.dumps(inspect(args.image), ensure_ascii=True, indent=2))
