"""Observe actual GUI save jobs at real pending Win32 I/O gates."""
import hashlib
import json
import os
import time
from contextlib import ExitStack
from io_gate import Gate, pending_threads


def verify(d, mode):
    rows, pings = [], []
    text = d.text
    def observed_text(window):
        try:
            return text(window)
        except RuntimeError as error:
            raise RuntimeError(f'{error}; completed={rows}; pending={pending_threads(d.process.pid)}') from error
    d.text = observed_text
    source = d.root/'저장 원본 Ω.bin'
    payload = b'abc\x00\xff'
    source.write_bytes(payload)
    digest = hashlib.sha256(payload).hexdigest()
    d.choose(202, source)
    d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'hash result unavailable')
    expected = dict(algorithm='SHA-256', file=d.text(d.child(d.main, 201)), bytes=len(payload), sha256=digest)
    prefix = os.environ['FAIRY_HASH_SAVE_TEST_PIPE']
    choose, first = d.choose, True
    def traced_choose(button, path, **options):
        nonlocal first
        # Identical success/failure text from a previous save must not satisfy
        # this save's completion observation before its dialog unwinds.
        if button == 204:
            if not first:
                choose(202, source)
                d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v.startswith('✓ 계산 완료'), 'fresh save baseline')
                d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'fresh save hash not ready')
            first = False
        try:
            return choose(button, path, **options)
        except RuntimeError as error:
            raise RuntimeError(f'{error}; target={path}; completed={rows}') from error
    d.choose = traced_choose

    def responsive():
        for _ in range(3):
            start = time.perf_counter(); d.message(d.main, 0)
            pings.append((time.perf_counter()-start)*1000)
        start = time.perf_counter()
        if not d.resize(d.main, None, 0, 0, 900, 550, 0x16):
            raise RuntimeError('pending-save resize failed')
        d.wait(d.layout, bool, 'pending-save layout failed')
        duration = (time.perf_counter()-start)*1000
        if duration > 500 or max(pings) > 500:
            raise RuntimeError('pending save blocked UI messages/layout')
        return duration

    def unchanged(target):
        if target.read_bytes() != b'old report' or source.read_bytes() != payload:
            raise RuntimeError('pending/failed save changed existing data')

    def clean(target):
        if list(d.root.glob(target.name+'.fairy-save-*.tmp')):
            raise RuntimeError('owned save temporary survived completion')
        if d.text(d.child(d.main, 203)).replace('\r\n', '') != digest:
            raise RuntimeError('save changed the completed hash')

    def idle(status):
        d.wait(lambda: d.text(d.child(d.main, 205)), lambda text: text == status, 'save status: '+status)
        d.wait(lambda: bool(d.enabled(d.child(d.main, 204))) and bool(d.enabled(d.child(d.main, 202))), bool, 'save controls not restored')

    with ExitStack() as cleanup:
        def blocked(stage, suffix):
            target = d.root/f'slow-save-{stage}-{suffix}.json'
            target.write_bytes(b'old report')
            gate = Gate(prefix+'-'+target.name); cleanup.callback(gate.release)
            d.choose(204, target, overwrite=True)
            d.wait(lambda: gate.attached(d.process.pid) and bool(pending_threads(d.process.pid)), bool, 'writer not observed in actual pending I/O')
            if d.enabled(d.child(d.main, 204)) or d.enabled(d.child(d.main, 202)) or d.enabled(d.child(d.main, 208)):
                raise RuntimeError('pending save allows conflicting actions')
            return target, gate, responsive()

        if mode != 'main':
            stage = mode.removeprefix('close-')
            target, gate, resize_ms = blocked(stage, 'retain-close')
            if stage != 'cleanup':
                unchanged(target)
            if stage != 'write' and d.enabled(d.child(d.main, 212)):
                raise RuntimeError('commit/cleanup still cancellable')
            start = time.perf_counter(); d.message(d.main, 0x10)
            d.wait(lambda: d.text(d.child(d.main, 205)),
                   lambda text: text.startswith('저장 취소·정리') if stage == 'write' else text.startswith('저장 마무리'), 'close cleanup status')
            time.sleep(0.1)
            if d.process.poll() is not None:
                raise RuntimeError('Close exited before owned save/cleanup finished')
            responsive()
            gate.release()
            if d.process.wait(timeout=3):
                raise RuntimeError('deferred save Close failed')
            elapsed = (time.perf_counter()-start)*1000
            if stage == 'write':
                unchanged(target)
            elif json.loads(target.read_bytes()) != expected:
                raise RuntimeError('Close during commit lost complete report')
            if list(d.root.glob(target.name+'.fairy-save-*.tmp')):
                raise RuntimeError('Close leaked an owned temporary')
            rows.append(dict(case=mode, closeIncludingHeldGateMs=elapsed, resizeMs=resize_ms, deferredClose=True, temporaryRemoved=True))
        else:
            for stage in ('check', 'create', 'write', 'flush', 'publish'):
                target, gate, resize_ms = blocked(stage, 'complete')
                unchanged(target); gate.release(); idle('✓ 결과 저장 완료')
                if json.loads(target.read_bytes()) != expected:
                    raise RuntimeError('save report mismatch after pending '+stage)
                clean(target)
                rows.append(dict(case='pending-'+stage+'-completion', resizeMs=resize_ms, oldResultPreservedUntilCommit=True))
            for stage in ('check', 'create', 'write', 'flush'):
                target, gate, resize_ms = blocked(stage, 'cancel')
                start = time.perf_counter(); d.message(d.child(d.main, 212), 0xf5)
                idle('저장 취소됨')
                elapsed = (time.perf_counter()-start)*1000
                if elapsed > 500:
                    raise RuntimeError('cancellable save did not settle promptly')
                gate.release(); unchanged(target); clean(target)
                rows.append(dict(case='pending-'+stage+'-cancel', cancelMs=elapsed, temporaryRemoved=True))
            for stage in ('write', 'flush', 'publish'):
                target = d.root/f'fail-save-{stage}.json'; target.write_bytes(b'old report')
                d.choose(204, target, overwrite=True); idle('저장 실패: 다른 위치를 선택하세요.')
                unchanged(target); clean(target)
                rows.append(dict(case=stage+'-failure', existingResultPreserved=True, temporaryRemoved=True))
            target, gate, _ = blocked('write', 'retain-cancel')
            d.message(d.child(d.main, 212), 0xf5)
            d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v.startswith('저장 취소·정리'), 'retained-save cleanup not shown')
            responsive(); unchanged(target)
            if d.enabled(d.child(d.main, 204)):
                raise RuntimeError('another save can overlap unfinished cleanup')
            gate.release(); idle('저장 취소됨'); unchanged(target); clean(target)
            followup = d.root/'after-cancel.json'
            d.choose(204, followup); idle('✓ 결과 저장 완료')
            if json.loads(followup.read_bytes()) != expected:
                raise RuntimeError('cancelled writer contaminated a later result')
            rows.append(dict(case='retained-cancel-and-reuse', workerLimit=1, latePublishPrevented=True))
            target, gate, _ = blocked('write', 'alias-change')
            target.unlink(); os.link(source, target)
            gate.release(); idle('원본 파일에는 덮어쓸 수 없어요.'); clean(target)
            if target.read_bytes() != payload or source.read_bytes() != payload:
                raise RuntimeError('late hard-link change damaged source')
            rows.append(dict(case='source-alias-rechecked-before-commit', sourceProtected=True))
            target = d.root/'collision.json'; target.write_bytes(b'old report')
            foreign = d.root/(target.name+'.fairy-save-0.tmp'); foreign.write_bytes(b'foreign')
            d.choose(204, target, overwrite=True); idle('✓ 결과 저장 완료')
            if foreign.read_bytes() != b'foreign' or json.loads(target.read_bytes()) != expected:
                raise RuntimeError('exclusive save temporary collision damaged data')
            foreign.unlink(); clean(target)
            rows.append(dict(case='exclusive-temporary-collision', foreignTemporaryPreserved=True))
            d.message(d.main, 0x10)
            if d.process.wait(timeout=3):
                raise RuntimeError('normal save window exit failed')
    return dict(cases=len(rows), mode=mode, observations=rows, actualPendingIO=True, pingMaxMs=max(pings),
                oldResultsProtected=True, temporaryCleanup=True, physicalInput=False, pixelInspection=False)
