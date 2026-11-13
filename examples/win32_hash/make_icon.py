"""Render the example's simple rect/polygon SVG to a multi-resolution PNG ICO.

Uses only Python's standard library; no renderer ships with the app. Other apps
may replace native/app.ico directly with their own multi-resolution ICO.
"""
import argparse
from pathlib import Path
import struct
import xml.etree.ElementTree as xml
import zlib

SIZES = (16, 20, 24, 32, 40, 48, 64, 128, 256)


def shapes(source):
    root = xml.fromstring(source)
    if root.attrib.get('viewBox') != '0 0 256 256': raise ValueError('expected a 256px viewBox')
    result = []
    for node in root:
        color = bytes.fromhex(node.attrib['fill'].removeprefix('#'))
        if len(color) != 3: raise ValueError('expected RGB fill')
        kind = node.tag.rsplit('}', 1)[-1]
        if kind == 'rect':
            x, y, width, height, radius = (float(node.attrib[key]) for key in ('x', 'y', 'width', 'height', 'rx'))
            def contains(px, py, x=x, y=y, width=width, height=height, radius=radius):
                dx = max(x+radius-px, 0, px-(x+width-radius))
                dy = max(y+radius-py, 0, py-(y+height-radius))
                return dx*dx+dy*dy <= radius*radius
        elif kind == 'polygon':
            points = [tuple(map(float, pair.split(','))) for pair in node.attrib['points'].split()]
            def contains(px, py, points=points):
                inside = False
                for (ax, ay), (bx, by) in zip(points, points[1:]+points[:1]):
                    if (ay > py) != (by > py) and px < (bx-ax)*(py-ay)/(by-ay)+ax:
                        inside = not inside
                return inside
        else: raise ValueError(f'unsupported SVG element {kind}')
        result.append((contains, color))
    return result


def png(size, layers):
    def chunk(kind, data):
        return struct.pack('>I', len(data))+kind+data+struct.pack('>I', zlib.crc32(kind+data))
    rows = bytearray()
    for y in range(size):
        rows.append(0)
        for x in range(size):
            channels, covered = [0, 0, 0], 0
            for sy in range(4):
                for sx in range(4):
                    px, py = (x+(sx+.5)/4)*256/size, (y+(sy+.5)/4)*256/size
                    for contains, color in reversed(layers):
                        if contains(px, py):
                            covered += 1
                            for index in range(3): channels[index] += color[index]
                            break
            rows.extend([round(value/covered) if covered else 0 for value in channels]+[round(255*covered/16)])
    return (b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR', struct.pack('>IIBBBBB',size,size,8,6,0,0,0))+
            chunk(b'IDAT', zlib.compress(rows,9))+chunk(b'IEND', b''))


def render(source):
    layers = shapes(source)
    frames = [png(size, layers) for size in SIZES]
    offset, directory = 6+16*len(frames), bytearray(struct.pack('<HHH', 0, 1, len(frames)))
    for size, frame in zip(SIZES, frames):
        directory.extend(struct.pack('<BBBBHHII',size%256,size%256,0,0,1,32,len(frame),offset))
        offset += len(frame)
    return bytes(directory)+b''.join(frames)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    root = Path(__file__).resolve().parent
    parser.add_argument('--source', type=Path, default=root/'native/app.svg')
    parser.add_argument('--output', type=Path, default=root/'native/app.ico')
    args = parser.parse_args()
    data = render(args.source.read_bytes()); args.output.write_bytes(data)
    print(f'wrote {args.output}: {len(data)} bytes, sizes {SIZES}')
