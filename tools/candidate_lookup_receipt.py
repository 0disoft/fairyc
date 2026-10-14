"""Independent decoder for private candidate project-lookup observations."""
import hashlib
import re


class Reader:
    def __init__(self, raw):
        self.raw, self.at = raw, 0

    def number(self):
        assert self.at + 8 <= len(self.raw), 'short lookup number'
        value = int.from_bytes(self.raw[self.at:self.at + 8], 'big')
        self.at += 8
        return value

    def field(self):
        size = self.number()
        assert size <= 16777216 and self.at + size <= len(self.raw), 'short lookup field'
        value = self.raw[self.at:self.at + size]
        self.at += size
        return value


def lookup_ledger(raw, image):
    assert len(raw) <= 16777216 and raw[:8] == b'FYLOOK00'
    reader = Reader(raw); reader.at = 8
    count = reader.number(); assert count == 9
    compiler = b'fairyc.compiler-executable-sha256/v1:' + hashlib.sha256(image.read_bytes()).hexdigest().encode()
    rows = []
    for _ in range(count):
        name, canonical, digest = reader.field(), reader.field(), reader.field()
        assert canonical.startswith(b'fairyc.candidate-project-lookup\0\0')
        assert digest == b'fylookup0-sha256-' + hashlib.sha256(canonical).hexdigest().encode(), 'lookup hash contradicts preimage'
        entry = Reader(canonical); entry.at = len(b'fairyc.candidate-project-lookup\0\0')
        assert entry.field() == compiler, 'lookup does not bind exact executing image'
        # All other bytes must agree across independently compiled native images.
        normalized = canonical[entry.at:]
        fields = [entry.field() for _ in range(7)]
        assert all(fields)
        assert entry.field() == b'x86_64-windows-msvc'
        assert re.fullmatch(rb'fypi1-sha256-[0-9a-f]{64}', entry.field())
        root = entry.field(); packages = entry.number(); assert 1 <= packages <= 512
        ids = []; targets = []; sdk_count = 0
        for _ in range(packages):
            instance = entry.field(); assert re.fullmatch(rb'fypi1-sha256-[0-9a-f]{64}', instance)
            assert not ids or ids[-1] < instance
            ids.append(instance)
            is_sdk = entry.number(); assert is_sdk in (0, 1); sdk_count += is_sdk
            modules = entry.number(); assert 1 <= modules <= 4096
            names = []
            for _ in range(modules):
                module, path = entry.field(), entry.field()
                assert module and path and not path.startswith((b'/', b'\\')) and b':' not in path
                assert not names or names[-1] < module
                names.append(module)
            edges = entry.number(); assert edges <= 4096
            bindings = []
            for _ in range(edges):
                binding, target = entry.field(), entry.field()
                assert binding and (not bindings or bindings[-1] < binding)
                bindings.append(binding); targets.append(target)
        assert root in ids and all(target in ids for target in targets) and sdk_count <= 1 and entry.at == len(canonical)
        rows.append(dict(case=name.decode(), root=root.decode(), packages=packages, sdkPackages=sdk_count, canonicalBytes=len(canonical), key=digest.decode(), normalizedSHA256=hashlib.sha256(normalized).hexdigest()))
    assert reader.at == len(raw) and len({row['case'] for row in rows}) == count
    by_name = {row['case']: row for row in rows}
    assert by_name['Aliases']['key'] == by_name['Reordered']['key']
    assert by_name['Aliases']['root'] == by_name['PrivateEdit']['root']
    assert by_name['Aliases']['key'] != by_name['PrivateEdit']['key'], 'private provider body did not invalidate lookup'
    assert by_name['Aliases']['root'] == by_name['BindingEdit']['root'] == by_name['FeatureEdit']['root']
    assert len({by_name[name]['key'] for name in ('Plain', 'PrivateEdit', 'State', 'Vec', 'InvalidBody')}) == 5
    assert len({by_name[name]['key'] for name in ('Aliases', 'BindingEdit', 'FeatureEdit')}) == 3
    assert by_name['Vec']['sdkPackages'] == 1 and by_name['Plain']['sdkPackages'] == 0
    return dict(compiler=compiler.decode(), rows=rows)
