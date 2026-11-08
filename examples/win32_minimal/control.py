"""Control this example's real Win32 controls through a private stdio session."""
import argparse
import json
import os
from pathlib import Path
from queue import Empty, Queue
import subprocess
import sys
from threading import Thread


class Session:
    def __init__(self, image):
        self.process = subprocess.Popen(
            [str(image), '--automation-stdio'], stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
        self.replies = Queue()
        Thread(target=self._read, daemon=True).start()
        try:
            self.initial = self.receive()
        except BaseException:
            self.finish()
            raise

    def _read(self):
        try:
            while True:
                raw = self.process.stdout.readline(8193)
                if not raw:
                    raise RuntimeError('GUI closed its reply pipe')
                if len(raw) > 8192 or not raw.endswith(b'\n'):
                    raise RuntimeError('oversized or incomplete GUI reply')
                self.replies.put(json.loads(raw))
        except Exception as error:
            self.replies.put(error)

    def receive(self):
        try:
            result = self.replies.get(timeout=5)
        except Empty as error:
            raise RuntimeError('GUI reply timed out after 5 seconds') from error
        if isinstance(result, Exception):
            raise result
        return result

    def wire(self, command):
        self.process.stdin.write(command + b'\n')
        self.process.stdin.flush()
        return self.receive()

    def request(self, request):
        if not isinstance(request, dict):
            raise ValueError('request must be an object')
        command = request.get('command')
        if command in ('state', 'click', 'close') and set(request) == {'command'}:
            return self.wire(command.encode('ascii'))
        if command == 'input' and set(request) == {'command', 'text'}:
            text = request['text']
            if not isinstance(text, str) or any(c in text for c in '\r\n\0'):
                raise ValueError('input must be a single-line string without NUL')
            if len(text.encode('utf-16-le')) > 254:
                raise ValueError('input exceeds 127 UTF-16 code units')
            return self.wire(b'input\t' + text.encode('utf-8'))
        raise ValueError('expected state, input with text, click, or close')

    def finish(self):
        if self.process.poll() is None:
            self.process.stdin.close()
            try:
                self.process.wait(timeout=2)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=2)
        self.process.stdout.close()


def verify(session):
    rows = [session.initial]

    def check(condition, message):
        if not condition:
            raise RuntimeError(message)

    def request(command, **values):
        row = session.request(dict(command=command, **values))
        rows.append(row)
        check(row.get('ok'), 'GUI rejected verification command')
        return row

    ready = rows[0]
    check(ready.get('schema') == 'fairyc.win32-example-control/v1' and ready['ok'], 'missing control schema')
    check(ready['title'] == 'Fairy C - Win32' and ready['visible'], 'window title or visibility failed')
    check(ready['clicks'] == 0 and ready['input'] == '', 'unexpected initial state')
    empty = request('click')
    check(empty['clicks'] == 1 and empty['result'] == 'Enter a message first.', 'empty click failed')
    sample = '제로디 Win32 "입력" \\ Ω'
    edited = request('input', text=sample)
    check(edited['input'] == sample and edited['edits'] > ready['edits'], 'Unicode input failed')
    check(edited['status'] == 'WM_COMMAND: EN_CHANGE', 'missing edit notification')
    check(edited['result'] == empty['result'], 'input applied without clicking')
    applied = request('click')
    check(applied['clicks'] == 2 and applied['result'] == 'Result: ' + sample, 'button result failed')
    check(applied['status'] == 'WM_COMMAND: BN_CLICKED', 'missing button notification')
    request('input', text='x' * 127)
    check(request('click')['result'] == 'Result: ' + 'x' * 127, 'bounded input failed')
    for bad in (b'unknown', b'input\t' + b'x' * 128, b'input\t\xff', b'input\tbad\0text', b'x' * 1200):
        check(session.wire(bad).get('ok') is False, 'invalid native command accepted')
    check(request('state')['input'] == 'x' * 127, 'invalid command changed input')
    request('input', text='')
    check(request('click')['result'] == 'Enter a message first.', 'cleared input failed')
    final = request('close')
    check(final.get('closed') is True, 'GUI close failed')
    check(session.process.wait(timeout=5) == 0, 'GUI exit failed')
    return dict(ok=True, verification='real-win32-controls', observations=rows,
                rejectedNativeCommands=5, processExited=True)


def output(value):
    print(json.dumps(value, ensure_ascii=False), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent / 'build/win32-minimal.exe')
    parser.add_argument('--verify', action='store_true', help='verify events and close the owned GUI')
    args = parser.parse_args()
    if os.name != 'nt':
        parser.error('this example requires Windows')
    session = None
    try:
        session = Session(args.exe.resolve())
        if args.verify:
            output(verify(session))
        else:
            output(session.initial)
            while True:
                raw = sys.stdin.buffer.readline(4097)
                if not raw:
                    break
                if len(raw) > 4096:
                    raise ValueError('request exceeds 4096 bytes')
                try:
                    reply = session.request(json.loads(raw))
                except (ValueError, UnicodeError) as error:
                    output(dict(ok=False, error=str(error)))
                    continue
                output(reply)
                if reply.get('closed'):
                    break
        return 0
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired) as error:
        output(dict(ok=False, error=str(error)))
        return 1
    finally:
        if session is not None:
            session.finish()


if __name__ == '__main__':
    sys.stdout.reconfigure(encoding='utf-8')
    raise SystemExit(main())
