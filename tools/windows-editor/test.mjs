import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { copyFileSync, mkdtempSync, readFileSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve, sep } from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

for (const [name, change, expected] of [
  ['unsupported manifest', manifest => { manifest.schema = 'unknown'; }, /Unsupported editor bundle manifest/u],
  ['changed payload', manifest => { manifest.files[0].sha256 = '0'.repeat(64); }, /Bundle integrity mismatch/u],
  ['unlisted file', () => {}, /Unexpected bundle contents/u],
]) {
  test(`receiver rejects ${name} before launching VS Code`, { skip: process.platform !== 'win32' }, () => {
    const parent = realpathSync(tmpdir());
    const root = realpathSync(mkdtempSync(join(parent, 'fairyc-editor-reject-')));
    try {
      const script = join(root, 'verify.ps1');
      copyFileSync(join(import.meta.dirname, 'verify.ps1'), script);
      const payload = 'test payload\n';
      writeFileSync(join(root, 'payload.txt'), payload);
      const manifest = { schema: 'fairyc.windows-editor-bundle/v1', files: [{ name: 'payload.txt',
        bytes: Buffer.byteLength(payload), sha256: createHash('sha256').update(payload).digest('hex') }] };
      change(manifest);
      writeFileSync(join(root, 'manifest.json'), JSON.stringify(manifest));
      const results = join(root, 'results');
      const child = spawnSync(join(process.env.SystemRoot, 'System32/WindowsPowerShell/v1.0/powershell.exe'),
        ['-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-File', script,
          '-CodeRoot', join(root, 'no-vscode'), '-ResultsDirectory', results],
        { windowsHide: true, encoding: 'utf8', timeout: 20_000,
          env: { ...process.env, PSModulePath: join(process.env.SystemRoot, 'System32/WindowsPowerShell/v1.0/Modules') } });
      if (child.error) throw child.error;
      assert.notEqual(child.status, 0, 'invalid bundle was accepted');
      const receipt = JSON.parse(readFileSync(join(results, 'receipt.json'), 'utf8'));
      assert.equal(receipt.passed, false);
      assert.match(receipt.error, expected);
      assert.equal(receipt.installation, undefined);
    } finally {
      assert.ok(resolve(root).startsWith(parent + sep));
      assert.equal(realpathSync(root), root);
      rmSync(root, { recursive: true });
    }
  });
}
