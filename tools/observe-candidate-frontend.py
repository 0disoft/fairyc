"""Bounded native frontend observation, with optional Fairy phase markers.

This is verification transport, not compiler semantics or a product timeout.
The child owns no background helpers. Timeout/stream-limit termination is
recorded and drained before a durable receipt is published.
"""
from pathlib import Path
import argparse
import hashlib
import json
import queue
import re
import subprocess
import threading
import time

ROOT = Path(__file__).resolve().parents[1]
MARKER = re.compile(r'^FC_PROFILE\s+(\S+)\s+(\d+)\s+(\d+)(?:\s+(\d+))?\s*$')


def digest(path):
    result = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1 << 20), b''):
            result.update(chunk)
    return result.hexdigest()


def save(path, value):
    temporary = path.with_suffix(path.suffix + '.tmp')
    temporary.write_text(json.dumps(value, indent=2)+'\n', encoding='utf-8', newline='\n')
    temporary.replace(path)


def inventory(manifest):
    paths = [Path(line) for line in manifest.read_text(encoding='utf-8-sig').splitlines() if line]
    if not 0 < len(paths) <= 512:
        raise ValueError('manifest must contain 1..512 source files')
    return [dict(path=str((ROOT/p).resolve()), sha256=digest(ROOT/p)) for p in paths]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--manifest', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--timeout', type=int, default=180, help='verification allowance, 1..900 seconds')
    parser.add_argument('--require-profile', action='store_true')
    args = parser.parse_args()
    if not 1 <= args.timeout <= 900:
        parser.error('timeout must be in 1..900 seconds')
    candidate, manifest, output = args.candidate.resolve(), args.manifest.resolve(), args.output.resolve()
    if not candidate.is_file() or not manifest.is_file():
        parser.error('candidate and manifest must be regular files')
    output.mkdir(parents=True, exist_ok=True)
    if any(output.iterdir()):
        parser.error('output must be empty; completed and live observations are never overwritten')
    original = inventory(manifest)
    compiler_hash, manifest_hash = digest(candidate), digest(manifest)
    save(output/'source-inputs.json', original)
    argv = [str(candidate), '--c-project-file', str(manifest)]
    messages = queue.Queue()
    events, diagnostics = [], []
    started = time.monotonic()
    timed_out = stream_limit = False
    stderr_bytes = 0
    latest = None
    next_progress = 0
    with (output/'packet.bin').open('wb') as stdout, (output/'stderr.bin').open('wb') as stderr, (output/'events.jsonl').open('w', encoding='utf-8', newline='\n') as log:
        process = subprocess.Popen(argv, cwd=ROOT, stdin=subprocess.DEVNULL, stdout=stdout, stderr=subprocess.PIPE,
                                   creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))

        def read_errors():
            try:
                while True:
                    line = process.stderr.readline(65536)
                    if not line:
                        break
                    messages.put((time.monotonic()-started, line))
            finally:
                messages.put(None)

        reader = threading.Thread(target=read_errors, daemon=True)
        reader.start()
        reader_done = False
        try:
            while not reader_done or process.poll() is None:
                elapsed = time.monotonic()-started
                if elapsed >= next_progress:
                    save(output/'progress.json', dict(status='running', pid=process.pid,
                         seconds=elapsed, events=len(events), latest=latest))
                    next_progress = elapsed + 1
                if stdout.tell() > 96 << 20 or (output/'packet.bin').stat().st_size > 96 << 20:
                    stream_limit = True
                    if process.poll() is None:
                        process.kill()
                if elapsed >= args.timeout and process.poll() is None:
                    timed_out = True
                    process.kill()
                try:
                    message = messages.get(timeout=0.2)
                except queue.Empty:
                    continue
                if message is None:
                    reader_done = True
                    continue
                seconds, raw = message
                stderr_bytes += len(raw)
                if stderr_bytes > 8 << 20:
                    stream_limit = True
                    if process.poll() is None:
                        process.kill()
                    continue
                stderr.write(raw)
                stderr.flush()
                text = raw.decode('utf-8', errors='replace').rstrip('\r\n')
                match = MARKER.fullmatch(text)
                if match:
                    phase, first, stop, work = match.groups()
                    latest = dict(seconds=seconds, phase=phase, first=int(first), stop=int(stop), work=int(work) if work is not None else None)
                    events.append(latest)
                    log.write(json.dumps(latest)+'\n')
                    log.flush()
                elif text:
                    diagnostics.append(dict(seconds=seconds, text=text))
        finally:
            if process.poll() is None:
                process.kill()
            process.wait()
            reader.join()
            process.stderr.close()
    seconds = time.monotonic()-started
    try:
        unchanged = inventory(manifest) == original and digest(manifest) == manifest_hash and digest(candidate) == compiler_hash
    except (OSError, ValueError):
        unchanged = False
    packet = output/'packet.bin'
    packet_bytes = packet.stat().st_size
    completed = not timed_out and not stream_limit and process.returncode == 0 and 0 < packet_bytes <= 96 << 20
    passed = completed and unchanged and (bool(events) or not args.require_profile)
    receipt = dict(schema='fairyc.candidate-frontend-observation/v1', passed=passed, argv=argv,
                   candidateSHA256=compiler_hash, manifestSHA256=manifest_hash, sourceFiles=len(original),
                   sourceUnchanged=unchanged, timeoutSeconds=args.timeout, timedOut=timed_out,
                   streamLimitExceeded=stream_limit, exit=process.returncode, seconds=seconds,
                   stdoutComplete=completed, packetBytes=packet_bytes, packetSHA256=digest(packet),
                   profileEvents=len(events), diagnostics=diagnostics, latest=latest,
                   productTimeoutChanged=False, promotionGateReady=False)
    save(output/'receipt.json', receipt)
    save(output/'progress.json', dict(status='completed', pid=process.pid, seconds=seconds, events=len(events), latest=latest, passed=passed, timedOut=timed_out))
    print('frontend observation:', 'completed output' if completed else 'incomplete', round(seconds, 2), 'seconds;', len(events), 'phase markers')
    return 0 if passed else 1


if __name__ == '__main__':
    raise SystemExit(main())
