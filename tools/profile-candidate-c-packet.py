"""Add diagnostic-only inclusive function timers to an authenticated C packet.

This changes generated C, not compiler sources. The resulting image is a profiling
probe, never evidence that a candidate was rebuilt from current Fairy sources.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
PREFIX = b'/* fairyc.candidate-project-unit-c/v1 */\n#include "fy_project.h"\n'


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--packet', type=Path, required=True)
    parser.add_argument('--packet-sha256', required=True)
    parser.add_argument('--function', action='append', required=True, metavar='LABEL=SYMBOL')
    parser.add_argument('--within', help='only report calls inside this selected function (single-threaded probe)')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__
    raw = args.packet.read_bytes()
    assert hashlib.sha256(raw).hexdigest() == args.packet_sha256
    assert raw.startswith(b'FYCPARTS1\n') and len(raw) <= 67108864
    selected = {}
    for value in args.function:
        label, symbol = value.split('=')
        assert re.fullmatch(r'[a-z0-9_-]+', label)
        assert re.fullmatch(r'fy_fn_[A-Za-z0-9_]+', symbol) and symbol not in selected
        selected[symbol] = label
    assert args.within is None or args.within in selected
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    offset = 10

    def number():
        nonlocal offset
        end = raw.index(b'\n', offset)
        text = raw[offset:end]
        assert re.fullmatch(rb'0|[1-9][0-9]{0,7}', text)
        offset = end + 1
        return int(text)

    def block():
        nonlocal offset
        length = number()
        assert 0 < length <= 2097152 and offset + length <= len(raw)
        data = raw[offset:offset + length]
        offset += length
        return data

    header, literals = block(), block()
    count = number()
    assert 0 < count <= 4096
    units = [block() for _ in range(count)]
    assert offset == len(raw) and b'fairyc_private_windows_ticks_v1(void)' in header
    depth = 'fy_native_profile_depth_v1'
    if args.within:
        assert depth.encode() not in raw
        assert header.endswith(b'#endif\n')
        header = header[:-7] + ('extern uintptr_t ' + depth + ';\n#endif\n').encode()
    found = {}
    pattern = re.compile(rb'^extern ([^\n]+?) (fy_fn_[A-Za-z0-9_]+)\(([^\n]*)\)\{', re.M)

    def wrap(match):
        symbol = match[2].decode()
        if symbol not in selected:
            return match[0]
        assert symbol not in found
        result_type, parameters = match[1].decode(), match[3].decode()
        arguments = [] if parameters == 'void' else [p.rsplit(' ', 1)[-1] for p in parameters.split(',')]
        assert all(re.fullmatch(r'fy_l_[0-9]+', p) for p in arguments)
        label = selected[symbol]
        implementation = 'fy_profile_impl_' + symbol.removeprefix('fy_fn_')
        invoke = implementation + '(' + ','.join(arguments) + ');'
        returned = '' if result_type == 'void' else result_type + ' result='
        finish = 'return;' if result_type == 'void' else 'return result;'
        declaration = result_type + ' ' + implementation + '(' + parameters + ')'
        scoped = symbol == args.within
        enter = depth + '++;' if scoped else ''
        leave = depth + '--;' if scoped else ''
        condition = 'if(' + depth + '){' if args.within else ''
        end_condition = '}' if args.within else ''
        wrapper = ('static ' + declaration + ';\nextern ' + result_type + ' ' + symbol + '(' + parameters + '){'
                   + enter + 'uintptr_t started=fairyc_private_windows_ticks_v1();' + returned + invoke + condition +
                   'fprintf(stderr,"FY_NATIVE_PROFILE ' + label + ' %llu\\n",'
                   '(unsigned long long)(fairyc_private_windows_ticks_v1()-started));' + end_condition + leave + finish + '}\n'
                   'static ' + declaration + '{')
        found[symbol] = dict(label=label, signature=match[0].decode())
        return wrapper.encode()

    for index, unit in enumerate(units):
        assert unit.startswith(PREFIX)
        patched = pattern.sub(wrap, unit)
        if patched != unit:
            units[index] = PREFIX + b'#include <stdio.h>\n' + patched[len(PREFIX):]
            assert len(units[index]) <= 2097152
    assert set(found) == set(selected), (set(selected) - set(found))
    if args.within:
        units[0] = PREFIX + ('uintptr_t ' + depth + '=0;\n').encode() + units[0][len(PREFIX):]

    def encoded(data):
        return str(len(data)).encode() + b'\n' + data

    packet = (b'FYCPARTS1\n' + encoded(header) + encoded(literals) +
              str(count).encode() + b'\n' + b''.join(encoded(unit) for unit in units))
    assert len(header) <= 2097152 and all(len(unit) <= 2097152 for unit in units)
    assert len(packet) <= 67108864
    output.mkdir(parents=True)
    (output / 'profile-packet.bin').write_bytes(packet)
    receipt = dict(schema='fairyc.candidate-c-packet-profile/v1', diagnosticOnly=True,
                   sourcePacket=str(args.packet.resolve()), sourcePacketSHA256=args.packet_sha256,
                   packetSHA256=hashlib.sha256(packet).hexdigest(), functions=found,
                   timer='GetTickCount64 through existing private runtime; inclusive milliseconds',
                   within=args.within, singleThreadedScope=bool(args.within),
                   toolSHA256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest())
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8', newline='\n')
    print(json.dumps(receipt, indent=2))


if __name__ == '__main__':
    main()
