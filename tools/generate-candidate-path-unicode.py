"""Generate compact Fairy lower-case metadata from the Go reference report."""
import argparse
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def tree(rows, indent=1):
    pad = '  ' * indent
    if not rows:
        return pad + 'return rune;\n'
    middle = len(rows) // 2
    row = rows[middle]
    lower, upper, delta = row['Lo'], row['Hi'], row['Delta']
    result = f'{lower}+((rune-{lower})/2)*2+1' if delta == 1114112 else f'rune+{delta}' if delta > 0 else f'rune-{abs(delta)}'
    return (pad + f'if(rune<{lower}){{\n' + tree(rows[:middle], indent+1) + pad + f'}}else if(rune>{upper}){{\n' + tree(rows[middle+1:], indent+1) + pad + f'}}else{{return {result};}}\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--reference', type=Path, required=True)
    args = parser.parse_args()
    report = json.loads(args.reference.read_bytes())
    assert report['Schema'] == 'fairyc.path-unicode-reference/v1'
    ranges = report['Ranges']
    assert all(r['Lo'] <= r['Hi'] <= 1114111 and r['Delta'] != 0 for r in ranges)
    assert all(a['Hi'] < b['Lo'] for a, b in zip(ranges, ranges[1:]))
    content = ('module project_path_unicode_data;\n\n// Generated metadata; runtime Unicode decoding and path comparison stay in Fairy.\n'
               f'// Go Unicode {report["UnicodeVersion"]}; report SHA-256 {hashlib.sha256(args.reference.read_bytes()).hexdigest()}.\n'
               f'pub fn ref str version(){{return "{report["UnicodeVersion"]}";}}\n'
               'pub fn u32 lower(u32 rune){\n' + tree(ranges) + '}\n'
               'pub fn u32 fold(u32 rune){\n' + ''.join(f'  if(rune=={r["From"]}){{return {r["To"]};}}\n' for r in report['Specials']) + '  return lower(rune);\n}\n')
    path = ROOT / 'examples/lexer_candidate/src/project_path_unicode_data.fy'
    path.write_text(content, encoding='utf-8', newline='\n')
    print(path.name, 'generated:', len(content.encode()), 'bytes')


if __name__ == '__main__':
    main()
