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
    while not driver.enabled(driver.child(driver.main, 202)):
        if time.perf_counter() - started > 120:
            raise RuntimeError('large hash exceeded observation deadline')
        before = time.perf_counter()
        driver.message(driver.main, 0)  # WM_NULL on the real GUI thread.
        latencies.append((time.perf_counter()-before)*1000)
        summary = driver.text(driver.child(driver.main, 209))
        if not summaries or summaries[-1] != summary:
            summaries.append(summary)
        if has_progress:
            status = driver.text(driver.child(driver.main, 205))
            position = driver.message(driver.child(driver.main, 213), 0x400+8)  # PBM_GETPOS.
            counts = re.fullmatch(r'(\d+) / (\d+) 바이트', summary)
            label = re.fullmatch(r'(\d+)% · 경과 (\d+):(\d{2}):(\d{2})', status)
            if counts and label and summary == driver.text(driver.child(driver.main, 209)):
                done, total = map(int, counts.groups())
                percent, hours, minutes, seconds = map(int, label.groups())
                # Cross-process reads can straddle one real render update. Only
                # retain coherent snapshots; never inject a synthetic revision.
                if percent == position == min(99, done*100//total) and total == mib*1048576:
                    progress.append((done, percent, hours*3600+minutes*60+seconds))
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
        if len(progress) < 2 or progress[-1][1] <= progress[0][1] or progress[-1][2] < 1:
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
    ordered = sorted(latencies)
    return dict(bytes=source.stat().st_size, observedBusySeconds=elapsed,
                observedMiBPerSecond=mib/elapsed, sha256=actual,
                pingSamples=len(latencies), pingMaxMs=max(latencies),
                pingP95Ms=ordered[min(len(ordered)-1, int(len(ordered)*0.95))],
                resizeMs=resize_ms, summaryUpdates=len(summaries),
                progressObserved=has_progress, progressSamples=len(progress),
                progressFirst=progress[0] if progress else None,
                progressLast=progress[-1] if progress else None)
