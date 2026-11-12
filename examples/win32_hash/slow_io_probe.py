"""Actual GUI cancellation/close while native I/O is pending in a test build."""
import hashlib
import os
import re
import time
from contextlib import ExitStack
from io_gate import Gate, pending_threads, threads_finished


def verify(driver):
    d, pid = driver, driver.process.pid
    prefix = os.environ['FAIRY_HASH_TEST_PIPE']
    rows, ping = [], []
    def ready(payload):
        expected = hashlib.sha256(payload).hexdigest()
        d.wait(lambda: bool(d.enabled(d.child(d.main, 204))), bool, 'slow-I/O followup did not finish')
        if d.text(d.child(d.main, 203)).replace('\r\n', '') != expected:
            raise RuntimeError('slow-I/O digest mismatch')
        return expected
    with ExitStack() as cleanup:
        def blocked(name):
            path = d.root/name
            path.write_bytes(b'abc')
            gate = Gate(prefix+'-'+name)
            cleanup.callback(gate.release)
            d.choose(202, path)
            d.wait(lambda: gate.attached(pid) and bool(pending_threads(pid)), bool, 'owned worker is not blocked in actual I/O')
            if d.enabled(d.child(d.main, 204)) or d.enabled(d.child(d.main, 208)) or d.text(d.child(d.main, 203)):
                raise RuntimeError('pending I/O exposed a result')
            for _ in range(3):
                started = time.perf_counter()
                d.message(d.main, 0)
                ping.append((time.perf_counter()-started)*1000)
            started = time.perf_counter()
            if not d.resize(d.main, None, 0, 0, 900, 550, 0x16):
                raise RuntimeError('pending I/O resize failed')
            d.wait(d.layout, bool, 'pending I/O resize layout')
            resize_ms = (time.perf_counter()-started)*1000
            if resize_ms > 500:
                raise RuntimeError('pending I/O blocked resize')
            return gate, resize_ms
        def cancel():
            started = time.perf_counter()
            d.message(d.child(d.main, 212), 0xf5)
            try:
                d.wait(lambda: bool(d.enabled(d.child(d.main, 202))), bool, 'pending-I/O cancel did not return to idle', seconds=2)
            except RuntimeError as error:
                raise RuntimeError(f'{error}; completed={rows}; status={d.text(d.child(d.main, 205))}; '
                                   f'cancelEnabled={bool(d.enabled(d.child(d.main, 212)))}; pending={pending_threads(pid)}') from error
            elapsed = (time.perf_counter()-started)*1000
            if elapsed > 500 or not d.text(d.child(d.main, 205)).startswith('취소됨') or d.process.poll() is not None:
                raise RuntimeError('pending-I/O cancel blocked/closed the window')
            if d.enabled(d.child(d.main, 204)) or d.enabled(d.child(d.main, 208)) or d.text(d.child(d.main, 203)):
                raise RuntimeError('cancelled I/O exposed a partial result')
            return elapsed
        gate, resized = blocked('slow-read-complete.bin')
        d.wait(lambda: d.text(d.child(d.main, 205)),
               lambda v: bool(re.fullmatch(r'0% · 경과 \d+:\d{2}:0[1-9]', v)), 'elapsed time stopped while read pending', seconds=3)
        if not pending_threads(pid):
            raise RuntimeError('read was no longer pending before release')
        gate.release(); ready(b'abc')
        rows.append(dict(case='pending-read-completion', resizeMs=resized, correctDigest=True, elapsedAdvances=True))
        for stage in ('read', 'open', 'meta'):
            gate, resized = blocked(f'slow-{stage}-cancel.bin')
            elapsed = cancel(); gate.release()
            rows.append(dict(case=f'pending-{stage}-cancel', cancelMs=elapsed, resizeMs=resized))
        normal = d.root/'after-slow.txt'
        payload = b'new job after cancelled I/O'
        normal.write_bytes(payload)
        gate, _ = blocked('slow-read-retain-late.bin')
        elapsed = cancel()
        late_threads = pending_threads(pid)
        d.choose(202, normal); expected = ready(payload)
        if not pending_threads(pid):
            raise RuntimeError('old read was not pending while the new job completed')
        gate.release()
        d.wait(lambda: threads_finished(late_threads), bool, 'late worker did not finish after release')
        if d.text(d.child(d.main, 203)).replace('\r\n', '') != expected or d.text(d.child(d.main, 201)) != str(normal):
            raise RuntimeError('late cancelled result overwrote the new job')
        rows.append(dict(case='late-result-isolation', cancelMs=elapsed, newJobFinishedBeforeOldRead=True))
        retired = []
        for number in (1, 2):
            gate, _ = blocked(f'slow-read-retain-{number}.bin')
            elapsed = cancel()
            d.wait(lambda: len(pending_threads(pid)), lambda n: n >= number, 'cancel-unresponsive fixture did not retain its worker')
            retired.append(gate)
            rows.append(dict(case=f'retained-read-{number}', cancelMs=elapsed))
        d.choose(202, normal)
        d.wait(lambda: d.text(d.child(d.main, 205)), lambda v: v.startswith('이전 파일 읽기가'), 'retired-worker limit not enforced')
        if len(pending_threads(pid)) < 2:
            raise RuntimeError('retired-worker limit was not tested while reads were pending')
        retired_threads = pending_threads(pid)
        for gate in retired:
            gate.release()
        d.wait(lambda: threads_finished(retired_threads), bool, 'retired reads did not finish after release')
        d.choose(202, normal); expected = ready(payload)
        time.sleep(0.15)
        if d.text(d.child(d.main, 203)).replace('\r\n', '') != expected or d.text(d.child(d.main, 201)) != str(normal):
            raise RuntimeError('late cancelled result overwrote the new job')
        rows.append(dict(case='bounded-workers-and-reuse', workerLimit=2, lateResultsIgnored=True))
        gate, resized = blocked('slow-read-close.bin')
        started = time.perf_counter()
        d.message(d.main, 0x10)
        code = d.process.wait(timeout=2)
        closed = (time.perf_counter()-started)*1000
        if code or closed > 500:
            raise RuntimeError('closing a pending read blocked/failed')
        rows.append(dict(case='pending-read-close', closeMs=closed, resizeMs=resized))
    if max(ping) > 500:
        raise RuntimeError('pending read blocked GUI thread pings')
    return dict(cases=len(rows), observations=rows, actualPendingRead=True, pingMaxMs=max(ping), cancelledWindowKept=True,
                staleResultsIgnored=True, workerLimit=2, closeDuringPendingRead=True,
                physicalInput=False, pixelInspection=False)
