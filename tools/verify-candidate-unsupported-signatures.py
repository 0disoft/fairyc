"""Capture original generic signature diagnostics from the owning Go CLI."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import struct
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compiler', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    compiler = args.compiler.resolve()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    cases = []
    for scalar in ['i8', 'i16', 'isize', 'u16', 'u64', 'f32', 'f64', 'bool', 'u8', 'i32', 'i64', 'u32', 'usize']:
        cases.append((scalar, {'root': f'fn T id<T>(T x){{return x;}}fn void main(){{discard id<{scalar}>(42);}}'}))
    cases += [
        ('Repeated', {'root': 'fn T id<T>(T x){return x;}fn void main(){discard id<u16>(42);discard id<u16>(43);discard id<u64>(44);}'}),
        ('Unused', {'root': 'fn T unused<T>(T x){return x;}fn i32 main(){return 42;}'}),
        ('UnusedParameter', {'root': 'fn i32 constant<T>(){return 42;}fn i32 main(){return constant<u16>();}'}),
        ('Reference', {'root': 'fn ref T id<T>(ref T x){return x;}fn void main(){discard id<u16>(42);}'}),
        ('Pointer', {'root': 'fn ptr T id<T>(ptr T x){return x;}fn void main(){discard id<u16>(42);}'}),
        ('Option', {'root': 'fn Option<T> id<T>(Option<T> x){return x;}fn void main(){discard id<u16>(42);}'}),
        ('Array', {'root': 'fn T[2] id<T>(T[2] x){return x;}fn void main(){discard id<u16>(42);}'}),
        ('Forward', {'root': 'import helper as h;fn T outer<T>(T x){return h::id<T>(x);}fn void main(){discard outer<u16>(42);}',
                     'helper': 'pub fn T id<T>(T x){return x;}'}),
        ('Pair', {'root': 'fn A pair<A,B>(A x,B y){return x;}fn void main(){discard pair<u16,u64>(42,43);}'}),
    ]
    wire = bytearray(b'FYUSIG01')

    def number(value):
        wire.extend(struct.pack('>Q', value))

    def field(value):
        raw = value.encode('utf-8')
        number(len(raw))
        wire.extend(raw)

    number(len(cases))
    observations = []
    for label, modules in cases:
        root = output / label
        (root / 'src').mkdir(parents=True)
        (root / 'fairy.toml').write_text("schema='fairyc.manifest/v1'\n[package]\nuid='example.unsupported.signatures'\nname='unsupported-signatures'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='src'\n", encoding='utf-8', newline='\n')
        sources = {name: f'module {name}; // 한글\r\n{body}' for name, body in sorted(modules.items())}
        for name, raw in sources.items():
            (root / 'src' / (name + '.fy')).write_bytes(raw.encode('utf-8'))
        result = subprocess.run([str(compiler), 'check'], cwd=root, capture_output=True, timeout=120)
        assert result.returncode in (0, 3), (label, result)
        stderr = result.stderr.decode('utf-8')
        diagnostics = []
        for line in stderr.splitlines():
            match = re.fullmatch(r'fairyc: semantic subset does not support type (.+) at ([a-z]+)\.fy \[bytes ([0-9]+)\.\.([0-9]+)\]', line)
            if match:
                feature, module, start, end = match.groups()
                diagnostics.append(dict(module=module, feature=feature, start=int(start), end=int(end)))
        expected = {'Repeated': 4, 'Forward': 4, 'Pair': 3}.get(label,
            2 if label in {'i8', 'i16', 'isize', 'u16', 'u64', 'f32', 'f64',
                           'Reference', 'Pointer', 'Option', 'Array'} else 0)
        assert len(diagnostics) == expected, (label, stderr)
        if expected:
            assert result.returncode == 3 and not result.stdout and len(stderr.splitlines()) == expected, (label, result)
        else:
            assert result.returncode == (3 if label == 'bool' else 0), (label, result)
        # Supported instantiations can fail ordinary argument typing; these are
        # intentionally not relabeled as unsupported signature diagnostics.
        number(len(sources))
        field(label)
        for module, raw in sources.items():
            field(module)
            field(raw)
            imports = re.findall(r'import ([a-z]+) as [a-z]+;', raw)
            number(len(imports))
            for imported in imports:
                number(list(sources).index(imported))
        number(len(diagnostics))
        for row in diagnostics:
            number(list(sources).index(row['module']))
            number(row['start'])
            number(row['end'])
            field(row['feature'])
        observations.append(dict(name=label, exit=result.returncode, stdout=result.stdout.decode('utf-8'), stderr=stderr,
                                 sources=sources, diagnostics=diagnostics))
        print(label, result.returncode, len(diagnostics), flush=True)
    (output / 'fixture.bin').write_bytes(wire)
    (output / 'observations.json').write_text(json.dumps(dict(schema='fairyc.unsupported-signature-oracle/v1',
        compiler=dict(path=str(compiler), sha256=hashlib.sha256(compiler.read_bytes()).hexdigest()),
        fixtureSHA256=hashlib.sha256(wire).hexdigest(), cases=observations), indent=2) + '\n', encoding='utf-8', newline='\n')


if __name__ == '__main__':
    main()
