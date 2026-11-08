"""Control this example's real Win32 controls through a private stdio session."""
import argparse
import json
import os
from pathlib import Path
from queue import Empty, Queue
import subprocess
import sys
import time
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
        if command == 'drag-resize' and set(request) == {'command'}:
            return drag_window(self)
        if command in ('close-button', 'alt-f4') and set(request) == {'command'}:
            from desktop_input import close_input
            ready = self.wire(b'arm-close')
            if not ready.get('ok') or ready.get('pid') != self.process.pid:
                raise RuntimeError('GUI did not arm desktop close observation')
            with close_input(self.process.pid, command) as receipt:
                reply = self.receive()
                if not reply.get('ok') or reply.get('systemClose') is not True:
                    raise RuntimeError('desktop input did not reach native SC_CLOSE and Fairy close handling')
                if reply.get('closed'):
                    if self.process.wait(timeout=5) != 0:
                        raise RuntimeError('desktop close did not exit successfully')
                return dict(reply, desktopInput=receipt)
        if command in ('state', 'click', 'close') and set(request) == {'command'}:
            return self.wire(command.encode('ascii'))
        if command == 'resize' and set(request) == {'command', 'width', 'height'}:
            width, height = request['width'], request['height']
            if any(type(value) is not int or not 1 <= value <= 4096 for value in (width, height)):
                raise ValueError('resize width/height must be integers in 1..4096')
            return self.wire(f'resize\t{width}\t{height}'.encode('ascii'))
        if command == 'dpi-test' and set(request) == {'command', 'dpi', 'width', 'height'}:
            dpi, width, height = request['dpi'], request['width'], request['height']
            if type(dpi) is not int or not 96 <= dpi <= 384:
                raise ValueError('test DPI must be an integer in 96..384')
            if any(type(value) is not int or not 1 <= value <= 4096 for value in (width, height)):
                raise ValueError('suggested dimensions must be integers in 1..4096')
            return self.wire(f'dpi-test\t{dpi}\t{width}\t{height}'.encode('ascii'))
        if command == 'input' and set(request) == {'command', 'text'}:
            text = request['text']
            if not isinstance(text, str) or any(c in text for c in '\r\n\0'):
                raise ValueError('input must be a single-line string without NUL')
            if len(text.encode('utf-16-le')) > 254:
                raise ValueError('input exceeds 127 UTF-16 code units')
            return self.wire(b'input\t' + text.encode('utf-8'))
        raise ValueError('expected state, input, resize, dpi-test, drag-resize, click, close, close-button, or alt-f4')

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
    check(ready['clicks'] == 0 and ready['edits'] == 0 and ready['input'] == '', 'unexpected initial state')
    empty = request('click')
    check(empty['clicks'] == 1 and empty['edits'] == 0 and empty['result'] == 'Enter a message first.', 'empty click failed')
    sample = '제로디 Win32 "입력" \\ Ω'
    edited = request('input', text=sample)
    check(edited['input'] == sample and edited['edits'] == ready['edits'] + 1 and edited['clicks'] == empty['clicks'], 'Unicode input failed')
    check(edited['status'] == 'WM_COMMAND: EN_CHANGE', 'missing edit notification')
    check(edited['result'] == empty['result'], 'input applied without clicking')
    applied = request('click')
    check(applied['clicks'] == 2 and applied['edits'] == edited['edits'] and applied['result'] == 'Result: ' + sample, 'button result failed')
    check(applied['status'] == 'WM_COMMAND: BN_CLICKED', 'missing button notification')
    replaced = request('input', text='x' * 127)
    check(replaced['edits'] == edited['edits'] + 1 and replaced['clicks'] == applied['clicks'], 'replacement event count failed')
    check(replaced['status'] == 'WM_COMMAND: EN_CHANGE' and replaced['result'] == applied['result'], 'replacement applied before click')
    bounded = request('click')
    check(bounded['result'] == 'Result: ' + 'x' * 127 and bounded['edits'] == replaced['edits'], 'bounded input failed')
    for bad in (b'unknown', b'input\t' + b'x' * 128, b'input\t\xff', b'input\tbad\0text', b'x' * 1200):
        check(session.wire(bad).get('ok') is False, 'invalid native command accepted')
    unchanged = request('state')
    check(unchanged == bounded, 'invalid command or state read changed GUI state')
    cleared = request('input', text='')
    check(cleared['edits'] == replaced['edits'] + 1 and cleared['clicks'] == bounded['clicks'], 'clear event count failed')
    check(cleared['status'] == 'WM_COMMAND: EN_CHANGE' and cleared['result'] == bounded['result'], 'clear applied before click')
    final_click = request('click')
    check(final_click['result'] == 'Enter a message first.' and final_click['edits'] == cleared['edits'], 'cleared input failed')
    final = request('close')
    check(final.get('closed') is True, 'GUI close failed')
    check(session.process.wait(timeout=5) == 0, 'GUI exit failed')
    return dict(ok=True, verification='real-win32-controls', observations=rows,
                rejectedNativeCommands=5, processExited=True)


def output(value):
    print(json.dumps(value, ensure_ascii=False), flush=True)


def check_layout(row):
    if not row.get('ok') or row['resizes'] < 1:
        raise RuntimeError('missing Fairy layout acknowledgement')
    def scaled(value):
        return (value * row['dpi'] + 48) // 96
    width = max(scaled(320), row['client']['width'])
    inset = scaled(16)
    expected = dict(label=(inset, scaled(16), width - 2 * inset, scaled(20)),
                    input=(inset, scaled(44), width - scaled(132), scaled(28)),
                    apply=(width - scaled(100), scaled(44), scaled(84), scaled(28)),
                    result=(inset, scaled(94), width - 2 * inset, scaled(48)),
                    status=(inset, scaled(154), width - 2 * inset, scaled(24)))
    actual = {name: tuple(box[key] for key in ('x', 'y', 'width', 'height'))
              for name, box in row['layout'].items()}
    if actual != expected:
        raise RuntimeError('actual Win32 control rectangles differ from the Fairy layout')
    if row['minimum'] != dict(width=scaled(400), height=scaled(240)):
        raise RuntimeError('Fairy minimum size changed')
    if not row['perMonitorV2'] or row['windowDpi'] <= 0 or row['fonts'] != {name: scaled(12) for name in expected}:
        raise RuntimeError('DPI awareness or actual control font height is incorrect')


def verify_resize(session):
    rows = []

    def check(condition, message):
        if not condition:
            raise RuntimeError(message)

    check_layout(session.initial)
    session.request(dict(command='input', text='크기를 바꿔도 유지 Ω'))
    baseline = session.request(dict(command='click'))
    check(baseline['result'] == 'Result: 크기를 바꿔도 유지 Ω', 'resize setup did not apply input')
    prior = baseline
    # Widen, narrow to minimum, change height only, repeat unchanged, restore.
    for dip_width, dip_height in ((800, 480), (400, 240), (400, 480), (400, 480), (640, 360)):
        width = (dip_width * baseline['dpi'] + 48) // 96
        height = (dip_height * baseline['dpi'] + 48) // 96
        row = session.request(dict(command='resize', width=width, height=height))
        check_layout(row)
        check(row['window']['width'] == width and row['window']['height'] == height, 'window was not resized')
        check((row['window']['x'], row['window']['y']) ==
              (baseline['window']['x'], baseline['window']['y']), 'resize moved the window')
        for key in ('input', 'result', 'status', 'edits', 'clicks'):
            check(row[key] == baseline[key], 'resize changed application state: ' + key)
        changed = (width, height) != (prior['window']['width'], prior['window']['height'])
        check(row['resizes'] > prior['resizes'] if changed else row['resizes'] >= prior['resizes'],
              'actual size change did not reach Fairy')
        if width == prior['window']['width']:
            check(row['layout'] == prior['layout'], 'height-only resize changed fixed vertical layout')
        rows.append(row)
        prior = row
    minimum = baseline['minimum']
    invalid = (b'resize\t0\t240', f'resize\t{minimum["width"] - 1}\t{minimum["height"]}'.encode(),
               f'resize\t{minimum["width"]}\t{minimum["height"] - 1}'.encode(),
               b'resize\t4097\t240', b'resize\t400\t4097', b'resize\t-800\t480',
               b'resize\t800\t480junk', b'resize\t800', b'resize\t800\t480\textra')
    for command in invalid:
        check(session.wire(command).get('ok') is False, 'invalid resize was accepted')
    check(session.request(dict(command='state')) == prior, 'rejected resize changed window or state')
    for bad in (True, 3.5, '800', 0, 4097):
        try:
            session.request(dict(command='resize', width=bad, height=480))
        except ValueError:
            continue
        raise RuntimeError('invalid Python resize value was accepted')
    check(session.request(dict(command='state')) == prior, 'rejected Python request changed state')
    applied = session.request(dict(command='click'))
    check(applied['clicks'] == prior['clicks'] + 1 and applied['result'] == prior['result'],
          'button stopped working after resize')
    check(session.request(dict(command='close')).get('closed') is True, 'resized GUI did not close')
    check(session.process.wait(timeout=5) == 0, 'resized GUI did not exit successfully')
    return dict(ok=True, verification='real-win32-resize', observations=rows,
                rejectedNativeCommands=len(invalid), rejectedPythonCommands=5, processExited=True)


def verify_desktop_close(image):
    rows = []
    for command in ('close-button', 'alt-f4'):
        session = Session(image)
        try:
            session.request(dict(command='input', text='Fairy desktop close'))
            applied = session.request(dict(command='click'))
            if applied.get('result') != 'Result: Fairy desktop close':
                raise RuntimeError('desktop close setup did not apply input')
            reply = session.request(dict(command=command))
            if reply.get('closed') is not True or session.process.poll() != 0:
                raise RuntimeError('desktop close did not close the GUI with exit 0')
            rows.append(reply)
        finally:
            session.finish()
    return dict(ok=True, verification='win32-desktop-close', observations=rows,
                processExited=True)


def verify_dpi(session):
    from dpi_inventory import monitors
    baseline = session.initial
    check_layout(baseline)
    if baseline['dpi'] != baseline['windowDpi'] or baseline['dpiInjections'] != 0:
        raise RuntimeError('initial OS DPI was not used')
    session.request(dict(command='input', text='DPI를 바꿔도 유지 Ω'))
    applied = session.request(dict(command='click'))
    rows, prior = [], applied
    for dpi in (96, 144, 192, 96, baseline['windowDpi']):
        width, height = (640 * dpi + 48) // 96, (360 * dpi + 48) // 96
        row = session.request(dict(command='dpi-test', dpi=dpi, width=width, height=height))
        check_layout(row)
        if row['dpi'] != dpi or row['dpiChanges'] != prior['dpiChanges'] + 1 or row['dpiInjections'] != prior['dpiInjections'] + 1:
            raise RuntimeError('DPI event was not accepted by Fairy')
        expected = dict(x=applied['window']['x'], y=applied['window']['y'], width=width, height=height)
        if row['dpiSuggested'] != expected or row['dpiApplied'] != expected or row['window'] != expected:
            raise RuntimeError('Fairy did not apply the suggested DPI rectangle')
        if row['windowDpi'] != baseline['windowDpi']:
            raise RuntimeError('OS DPI changed during the injected-message test')
        for key in ('input', 'result', 'status', 'edits', 'clicks'):
            if row[key] != applied[key]:
                raise RuntimeError('DPI event changed application state: ' + key)
        rows.append(row)
        prior = row
    for command in (b'dpi-test\t0\t640\t360', b'dpi-test\t95\t640\t360',
                    b'dpi-test\t385\t640\t360', b'dpi-test\t144\t0\t360',
                    b'dpi-test\t144\t640\t360junk', b'dpi-test\t144\t640'):
        if session.wire(command).get('ok') is not False:
            raise RuntimeError('invalid DPI test request was accepted')
    if session.request(dict(command='state')) != prior:
        raise RuntimeError('rejected DPI test request changed state')
    clicked = session.request(dict(command='click'))
    if clicked['clicks'] != applied['clicks'] + 1 or clicked['result'] != applied['result']:
        raise RuntimeError('controls failed after DPI roundtrip')
    if not session.request(dict(command='close')).get('closed') or session.process.wait(timeout=5) != 0:
        raise RuntimeError('DPI test GUI did not exit successfully')
    return dict(ok=True, verification='injected-win32-dpi', observations=rows,
                monitors=monitors(), physicalMonitorTransitionTested=False,
                physicalMonitorTransitionReason='message injection verifies the handler; no monitor move was performed',
                rejectedNativeCommands=6, processExited=True)


def drag_window(session):
    from desktop_input import drag_input
    baseline = session.request(dict(command='state'))
    check_layout(baseline)
    rows, final = [], []

    def wait_released():
        deadline = time.monotonic() + 2
        while True:
            row = session.request(dict(command='state'))
            if not row['sizing'] and not row['mouseDown']:
                final.append(row)
                return
            if time.monotonic() >= deadline:
                raise RuntimeError('native drag loop did not exit after button release')
            time.sleep(0.02)

    with drag_input(session.process.pid, wait_released) as drag:
        deadline = time.monotonic() + 2
        while True:
            entered = session.request(dict(command='state'))
            if entered['sizing'] and entered['mouseDown']:
                break
            if time.monotonic() >= deadline:
                raise RuntimeError('mouse press did not enter the native sizing loop')
            time.sleep(0.02)
        prior, prior_dx = entered, 0
        for dx, dy in ((64, 32), (128, 64), (192, 96), (96, 48)):
            drag.move_to(drag.start[0] + dx, drag.start[1] + dy)
            deadline = time.monotonic() + 2
            while True:
                row = session.request(dict(command='state'))
                direction = 1 if dx > prior_dx else -1
                if (row['window']['width'] - prior['window']['width']) * direction > 1:
                    check_layout(row)
                    if (not row['sizing'] or not row['mouseDown'] or
                            row['liveLayouts'] <= prior['liveLayouts'] or row['resizes'] <= prior['resizes'] or
                            row['sizeEntries'] <= baseline['sizeEntries'] or row['sizeExits'] != baseline['sizeExits']):
                        raise RuntimeError('Fairy layout did not update while the mouse was held')
                    for key in ('input', 'result', 'status', 'edits', 'clicks'):
                        if row[key] != baseline[key]:
                            raise RuntimeError('drag changed application state: ' + key)
                    rows.append(row)
                    prior = row
                    prior_dx = dx
                    break
                if time.monotonic() >= deadline:
                    raise RuntimeError('real border drag did not change the native window size')
                time.sleep(0.02)
        receipt = drag.receipt
    check_layout(final[0])
    if final[0]['sizeExits'] <= baseline['sizeExits']:
        raise RuntimeError('native drag exit was not observed')
    return dict(ok=True, verification='win32-live-drag', desktopInput=receipt,
                observations=rows, finalState=final[0], released=True)


def verify_drag(session):
    session.request(dict(command='input', text='드래그 중에도 Fairy Ω'))
    applied = session.request(dict(command='click'))
    if applied['result'] != 'Result: 드래그 중에도 Fairy Ω':
        raise RuntimeError('drag setup did not apply input')
    report = drag_window(session)
    edited = session.request(dict(command='input', text='드래그 완료'))
    applied = session.request(dict(command='click'))
    if edited['edits'] != 2 or applied['clicks'] != 2 or applied['result'] != 'Result: 드래그 완료':
        raise RuntimeError('controls stopped working after live drag')
    if not session.request(dict(command='close')).get('closed') or session.process.wait(timeout=5) != 0:
        raise RuntimeError('live-drag GUI did not close with exit 0')
    return dict(report, processExited=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--exe', type=Path, default=Path(__file__).parent / 'build/win32-minimal.exe')
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument('--verify', action='store_true', help='verify events and close the owned GUI')
    mode.add_argument('--verify-desktop-close', action='store_true',
                      help='verify title-bar click and Alt+F4 via desktop input; requires an idle desktop')
    mode.add_argument('--verify-resize', action='store_true', help='resize the real window and verify native control rectangles')
    mode.add_argument('--verify-drag', action='store_true', help='verify live Fairy layout during actual border dragging; requires an idle desktop')
    mode.add_argument('--verify-dpi', action='store_true', help='inject DPI messages and verify actual geometry/fonts; does not change display settings')
    args = parser.parse_args()
    if os.name != 'nt':
        parser.error('this example requires Windows')
    session = None
    try:
        if args.verify_desktop_close:
            output(verify_desktop_close(args.exe.resolve()))
            return 0
        session = Session(args.exe.resolve())
        if args.verify:
            output(verify(session))
        elif args.verify_resize:
            output(verify_resize(session))
        elif args.verify_drag:
            output(verify_drag(session))
        elif args.verify_dpi:
            output(verify_dpi(session))
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
