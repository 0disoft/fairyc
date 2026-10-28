import assert from 'node:assert/strict';
import {execFileSync, spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync} from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import {snapshotSource} from './vsix-source.mjs';

test('VSIX provenance distinguishes clean, tracked edits, untracked edits and ignores output', t => {
  const root = mkdtempSync(path.join(os.tmpdir(), 'fairyc-vsix-source-'));
  t.after(() => {
    assert.equal(path.dirname(root), os.tmpdir());
    rmSync(root, {recursive: true, force: true});
  });
  const git = args => execFileSync('git', args, {cwd: root, windowsHide: true});
  git(['init', '--quiet']);
  writeFileSync(path.join(root, '.gitignore'), '*.vsix\n');
  writeFileSync(path.join(root, 'extension.js'), 'original\n');
  git(['add', '.']);
  git(['-c', 'user.name=VSIX test', '-c', 'user.email=vsix-test@example.invalid',
    '-c', 'commit.gpgsign=false', 'commit', '--quiet', '-m', 'fixture']);
  const clean = snapshotSource(root);
  assert.equal(clean.modified, false);
  assert.match(clean.revision, /^[a-f0-9]{40,64}$/);
  writeFileSync(path.join(root, 'output.vsix'), 'ignored output');
  assert.deepEqual(snapshotSource(root), clean);
  writeFileSync(path.join(root, 'extension.js'), 'first edit\n');
  const dirty = snapshotSource(root);
  assert.equal(dirty.modified, true);
  writeFileSync(path.join(root, 'extension.js'), 'second edit\n');
  const changed = snapshotSource(root);
  assert.equal(changed.statusSHA256, dirty.statusSHA256);
  assert.notEqual(changed.trackedDiffSHA256, dirty.trackedDiffSHA256);
  git(['add', 'extension.js']);
  assert.equal(snapshotSource(root).trackedDiffSHA256, changed.trackedDiffSHA256);
  writeFileSync(path.join(root, '새 소스.js'), 'first untracked edit\n');
  const untracked = snapshotSource(root);
  assert.equal(untracked.untrackedFiles[0].path, '새 소스.js');
  writeFileSync(path.join(root, '새 소스.js'), 'second untracked edit\n');
  assert.notDeepEqual(snapshotSource(root).untrackedFiles, untracked.untrackedFiles);
});

test('installer rejects a mismatched build receipt before reading VSIX or launching Code',
  {skip: process.platform !== 'win32'}, t => {
    const root = mkdtempSync(path.join(os.tmpdir(), 'fairyc-vsix-reject-'));
    t.after(() => {
      assert.equal(path.dirname(root), os.tmpdir());
      rmSync(root, {recursive: true, force: true});
    });
    const codeRoot = path.join(root, 'code');
    mkdirSync(path.join(codeRoot, 'bin'), {recursive: true});
    for (const name of ['Code.exe', 'bin/code.cmd']) writeFileSync(path.join(codeRoot, name), '');
    const vsix = path.join(root, 'test.vsix');
    const payload = 'not a VSIX; metadata must reject this before opening it';
    writeFileSync(vsix, payload);
    const digest = createHash('sha256').update(payload).digest('hex');
    writeFileSync(vsix + '.sha256', digest + '  test.vsix\n');
    const manifest = JSON.parse(readFileSync(new URL('../editors/vscode/package.json', import.meta.url), 'utf8'));
    writeFileSync(vsix + '.build.json', JSON.stringify({schema: 'fairyc.vsix-build/v1',
      extension: {id: `${manifest.publisher}.${manifest.name}`, version: manifest.version},
      artifact: {name: 'test.vsix', bytes: Buffer.byteLength(payload), sha256: '0'.repeat(64)}}));
    const result = spawnSync(path.join(process.env.SystemRoot, 'System32/WindowsPowerShell/v1.0/powershell.exe'),
      ['-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-File',
        path.join(import.meta.dirname, 'test-vscode-install.ps1'), '-CodeRoot', codeRoot, '-VSIXPath', vsix],
      {windowsHide: true, encoding: 'utf8', timeout: 20_000,
        env: {...process.env, PSModulePath: path.join(process.env.SystemRoot, 'System32/WindowsPowerShell/v1.0/Modules')}});
    if (result.error) throw result.error;
    assert.notEqual(result.status, 0);
    assert.match(result.stdout + result.stderr, /VSIX build receipt does not match package/);
  });
