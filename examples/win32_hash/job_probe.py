"""Measure real large-file jobs through the existing native dialog driver."""
import hashlib
import re
import time


def measure(driver, mib):
    source = driver.root / 'large-job.bin'
    chunk = bytes(range(256)) * 4096
    expected = hashlib.sha256()
    with source.open('wb') as stream:
        for _ in range(mib):
            stream.write(chunk)
            expected.update(chunk)
    driver.choose(202, source)
    driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 202))), lambda v: not v,
                'large job did not become busy')
    started = time.perf_counter()
    latencies, summaries, progress = [], [], []
    has_progress = bool(driver.child(driver.main, 213))
    resized = False
    last_progress_read = 0
    saved_screenshot = False
    while not driver.enabled(driver.child(driver.main, 202)):
        if time.perf_counter() - started > 120:
            raise RuntimeError('large hash exceeded observation deadline')
        before = time.perf_counter()
        driver.message(driver.main, 0)  # WM_NULL on the real GUI thread.
        latencies.append((time.perf_counter()-before)*1000)
        summary = driver.text(driver.child(driver.main, 209))
        if not summaries or summaries[-1] != summary:
            summaries.append(summary)
        if has_progress and time.perf_counter()-last_progress_read >= 0.15:
            last_progress_read = time.perf_counter()
            status = driver.text(driver.child(driver.main, 205))
            position = driver.message(driver.child(driver.main, 213), 0x400+8)  # PBM_GETPOS.
            counts = re.fullmatch(r'(\d+) / (\d+) 바이트', summary)
            label = re.fullmatch(r'(\d+)% · 경과 (\d+):(\d{2}):(\d{2})', status)
            if counts and label and summary == driver.text(driver.child(driver.main, 209)):
                done, total = map(int, counts.groups())
                percent, hours, minutes, seconds = map(int, label.groups())
                # Cross-process reads can straddle one real render update. Only
                # retain coherent snapshots; never inject a synthetic revision.
                # Size is unknown until the background open/metadata completes.
                if total == mib*1048576 and percent == position == min(99, done*100//total):
                    progress.append((done, percent, hours*3600+minutes*60+seconds))
                    if not saved_screenshot and percent >= 10 and getattr(driver, 'screenshot', None):
                        from capture_window import capture
                        capture(driver.main, driver.screenshot.with_name(f'{driver.screenshot.stem}-working.png'), repaint=False)
                        saved_screenshot = True
        if not resized and len(latencies) >= 3:
            before = time.perf_counter()
            driver.resize()
            driver.wait(driver.layout, bool, 'large-job resize layout')
            resize_ms = (time.perf_counter()-before)*1000
            resized = True
        time.sleep(0.03)
    elapsed = time.perf_counter()-started
    actual = driver.text(driver.child(driver.main, 203)).replace('\r\n', '')
    if actual != expected.hexdigest() or not resized or not summaries:
        raise RuntimeError('large-job digest/resize/observations mismatch')
    driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 204))), bool, 'large job Save unavailable')
    if has_progress:
        if (len(progress) < 2 or progress[-1][1] <= progress[0][1] or
                elapsed >= 2 and progress[-1][2] < 1):
            raise RuntimeError('large job did not show advancing percentage/time')
        if any(a[0] > b[0] or a[1] > b[1] or a[2] > b[2] for a, b in zip(progress, progress[1:])):
            raise RuntimeError('large job percentage/time moved backwards')
        if driver.message(driver.child(driver.main, 213), 0x400+8) != 100:
            raise RuntimeError('completed job progress is not 100 percent')
        status = driver.text(driver.child(driver.main, 205))
        final = re.fullmatch(r'✓ 계산 완료 · 경과 (\d+):(\d{2}):(\d{2})', status)
        if not final:
            raise RuntimeError('completed job lost elapsed time')
        hours, minutes, seconds = map(int, final.groups())
        if abs(hours*3600+minutes*60+seconds-elapsed) > 2:
            raise RuntimeError('displayed elapsed time differs from observed wall time')
    cancellation = []
    if driver.child(driver.main, 212):
        cancelled = driver.root / 'cancel-job.bin'
        for cycle in range(2):
            with cancelled.open('wb') as stream:
                for _ in range(16):
                    stream.write(chunk)
            driver.choose(202, cancelled)
            driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 202))), lambda v: not v,
                        'cancel job did not become busy')
            button = driver.child(driver.main, 212)
            driver.wait(lambda: (bool(driver.visible(button)), bool(driver.enabled(button)),
                                bool(driver.visible(driver.child(driver.main, 202)))),
                        lambda v: v == (True, True, False), 'Cancel did not replace Open during hashing')
            driver.wait(lambda: driver.message(driver.child(driver.main, 213), 0x400+8),
                        lambda v: 0 < v < 100, 'cancel fixture did not show partial progress')
            before = time.perf_counter()
            driver.message(button, 0xF5)  # Actual native Cancel button BM_CLICK.
            driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 202))), bool,
                        'Cancel did not return to idle', seconds=3)
            cancel_ms = (time.perf_counter()-before)*1000
            if cancel_ms > 500:
                raise RuntimeError(f'local-file cancellation exceeded 500ms: {cancel_ms}')
            if driver.process.poll() is not None or not driver.visible(driver.main):
                raise RuntimeError('Cancel closed the owned window/process')
            if not driver.text(driver.child(driver.main, 205)).startswith('취소됨 · 경과 '):
                raise RuntimeError('Cancel did not show elapsed cancellation status')
            driver.wait(lambda: (bool(driver.visible(button)), bool(driver.enabled(button)),
                                bool(driver.visible(driver.child(driver.main, 202)))),
                        lambda v: v == (False, False, True), 'Cancel did not restore Open')
            if (driver.enabled(driver.child(driver.main, 204)) or driver.enabled(driver.child(driver.main, 208)) or
                    driver.text(driver.child(driver.main, 203))):
                raise RuntimeError('Cancel exposed a partial/stale digest')
            if driver.message(driver.child(driver.main, 213), 0x400+8) >= 100:
                raise RuntimeError('cancelled work was displayed as complete')
            if driver.screenshot and cycle == 0:
                from capture_window import capture
                capture(driver.main, driver.screenshot.with_name(f'{driver.screenshot.stem}-cancelled.png'), repaint=False)
            cancelled.unlink()
            cancellation.append(dict(cycle=cycle, responseMs=cancel_ms, windowKept=True, inputRemoved=True))
        # The same native window must compute a different file after both cancels.
        followup = driver.root / 'after-cancel.txt'
        followup.write_bytes(b'abc')
        driver.choose(202, followup)
        driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 204))), bool,
                    'a new job after cancellation did not enable Save')
        if driver.text(driver.child(driver.main, 203)).replace('\r\n', '') != hashlib.sha256(b'abc').hexdigest():
            raise RuntimeError('next job inherited cancellation or stale hash state')
        if not driver.enabled(driver.child(driver.main, 208)) or driver.message(driver.child(driver.main, 213), 0x400+8) != 100:
            raise RuntimeError('next job did not return to completed state')
    ordered = sorted(latencies)
    if max(latencies) > 500 or resize_ms > 1000:
        raise RuntimeError('local-file GUI ping/resize responsiveness exceeded bounds')
    return dict(bytes=source.stat().st_size, observedBusySeconds=elapsed,
                observedMiBPerSecond=mib/elapsed, sha256=actual,
                pingSamples=len(latencies), pingMaxMs=max(latencies),
                pingP95Ms=ordered[min(len(ordered)-1, int(len(ordered)*0.95))],
                resizeMs=resize_ms, summaryUpdates=len(summaries),
                progressObserved=has_progress, progressSamples=len(progress),
                cancellation=cancellation, cancelAndReuse=bool(cancellation),
                progressFirst=progress[0] if progress else None,
                progressLast=progress[-1] if progress else None)
