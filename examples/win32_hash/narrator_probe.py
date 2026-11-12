"""Opt-in Narrator speech-history check on the verifier's temporary desktop."""
import ctypes as c
from ctypes import wintypes as w
import os
from pathlib import Path
import subprocess
import time
from desktop_process import DesktopProcess
from native_keyboard import DesktopKeys


def verify(driver):
    from verify_desktop import api, input_name, object_name, user
    from usability_probe import clipboard_text
    desktop = input_name()
    if not desktop.startswith('FairyHashVerify-'):
        raise RuntimeError('Narrator probe requires an isolated owned input desktop')
    station = object_name(api(user, 'GetProcessWindowStation', w.HANDLE)())
    narrator = None
    result = dict(screenReader='Windows Narrator', speechHistoryTested=False,
                  launchAttempted=True, started=False, audioRecorded=False, observations=[])
    try:
        narrator = DesktopProcess([str(Path(os.environ['WINDIR'])/'System32/Narrator.exe')],
            desktop=station+'\\'+desktop, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        result['started'] = True
        time.sleep(2)
        if narrator.poll() is not None:
            raise RuntimeError(f'Narrator exited before speech observation: {narrator.returncode}')
        keys = DesktopKeys(driver.process.pid)
        def phrase(label):
            keys.activate(driver.main)
            time.sleep(2)
            # Documented Narrator+Ctrl+X copies the last spoken phrase.
            if not api(user, 'OpenClipboard', w.BOOL, w.HWND)(None):
                raise RuntimeError('cannot prepare speech clipboard')
            try:
                if not api(user, 'EmptyClipboard', w.BOOL)():
                    raise RuntimeError('cannot reset speech clipboard')
            finally:
                api(user, 'CloseClipboard', w.BOOL)()
            keys.chord(driver.main, [45, 17, 88])
            text = driver.wait(clipboard_text, bool, 'Narrator did not expose its last spoken phrase')
            result['observations'].append(dict(step=label, spoken=text))
            return text
        initial = phrase('focused Open')
        if '파일' not in initial or '열기' not in initial:
            raise RuntimeError('Narrator did not speak the focused Open button')
        source = driver.root/'내레이터-입력.txt'
        source.write_bytes(b'abc')
        driver.choose(202, source)
        driver.wait(lambda: bool(driver.enabled(driver.child(driver.main, 204))), bool, 'Narrator fixture did not finish')
        completed = phrase('hash completion live region')
        if '계산 완료' not in completed:
            raise RuntimeError('Narrator speech did not confirm hash completion')
        result['speechHistoryTested'] = True
        result['receipts'] = keys.receipts
    except OSError as error:
        if narrator is not None or error.errno != 740:
            raise
        result['launchWinError'] = error.errno
        result['skipReason'] = 'Windows refused UIAccess Narrator via CreateProcess on the isolated desktop (740); speech was not tested'
    finally:
        if narrator:
            if narrator.poll() is None: narrator.terminate()
            narrator.wait(timeout=5)
            for stream in (narrator.stdin, narrator.stdout, narrator.stderr): stream.close()
            result['ownedNarratorStopped'] = True
    return result
