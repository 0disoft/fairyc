"""Native GUI file/report/identity checks without foreground or clipboard access."""
import hashlib
import json
import os


def verify(driver):
    d, rows = driver, []
    for index, payload in enumerate((b'', b'abc', bytes(range(256))*256+b'\xff')):
        source = d.root/f'파일-Ω-{index}.bin'
        source.write_bytes(payload)
        d.choose(202, source)
        expected_hash = hashlib.sha256(payload).hexdigest()
        d.wait(lambda: d.text(d.child(d.main, 203)).replace('\r\n', ''),
               lambda v: v == expected_hash, 'background reader GUI digest mismatch')
        d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'background result Save unavailable')
        actual_path = d.text(d.child(d.main, 201))
        if not os.path.samefile(actual_path, source) or d.message(d.child(d.main, 213), 0x400+8) != 100:
            raise RuntimeError('background reader path/completion mismatch')
        target = d.root/f'보고서-{index}.json'
        d.choose(204, target)
        d.wait(target.exists, bool, 'background result report not saved')
        expected = dict(algorithm='SHA-256', file=actual_path, bytes=len(payload), sha256=expected_hash)
        if json.loads(target.read_bytes()) != expected:
            raise RuntimeError('background GUI report differs from source/reference')
        if index == 1:
            alias = d.root/'원본-하드링크.bin'
            os.link(source, alias)
            for destination in (source, alias):
                d.choose(204, destination, overwrite=True)
                d.wait(lambda: d.text(d.child(d.main, 205)),
                       lambda v: v == '원본 파일에는 덮어쓸 수 없어요.', 'background file identity did not protect source/alias')
                if destination.read_bytes() != payload or source.read_bytes() != payload:
                    raise RuntimeError('background source/alias was overwritten')
        rows.append(dict(bytes=len(payload), sha256=expected_hash, reportSaved=True))
    return dict(cases=len(rows), observations=rows, sourceAndHardLinkProtected=True)
