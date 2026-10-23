import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, realpathSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { join, resolve, sep } from 'node:path';
import { spawnSync } from 'node:child_process';

assert.equal(process.argv.length, 4, 'usage: node tools/windows-portability/test.mjs --bundle <bundle>');
assert.equal(process.argv[2], '--bundle');
const bundle = resolve(process.argv[3]);
const cache = resolve(import.meta.dirname, '../../.cache');
mkdirSync(cache, { recursive: true });
const root = mkdtempSync(join(cache, 'portability-negative-'));
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');
function copy(name) {
  const target = join(root, name);
  mkdirSync(target);
  for (const file of readdirSync(bundle)) copyFileSync(join(bundle, file), join(target, file));
  return target;
}
function rejected(target, receipt, error) {
  const child = spawnSync('powershell.exe', ['-NoProfile', '-NonInteractive', '-File', join(target, 'verify.ps1'), '-Receipt', receipt],
    { cwd: target, shell: false, windowsHide: true, input: '', encoding: 'utf8', timeout: 90_000, maxBuffer: 1024 * 1024 });
  if (child.error) throw child.error;
  assert.notEqual(child.status, 0, 'invalid fixture passed');
  assert.match(child.stderr, error);
}
try {
  const corrupt = copy('corrupt-image');
  const native = join(corrupt, 'native-release.exe');
  const bytes = readFileSync(native); bytes[0] ^= 1;
  writeFileSync(native, bytes);
  const corruptReceipt = join(root, 'corrupt-receipt.json');
  rejected(corrupt, corruptReceipt, /Bundle file differs/u);
  assert.equal(existsSync(corruptReceipt), false, 'hash rejection reached execution');

  const behavior = copy('한글 wrong-behavior');
  copyFileSync(join(behavior, 'fairyc.exe'), join(behavior, 'native-release.exe'));
  const manifestPath = join(behavior, 'manifest.json');
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  const image = manifest.files.find(file => file.name === 'native-release.exe');
  image.sha256 = hash(join(behavior, image.name)); image.bytes = statSync(join(behavior, image.name)).size;
  writeFileSync(manifestPath, JSON.stringify(manifest) + '\n');
  const behaviorReceipt = join(root, 'behavior-receipt.json');
  rejected(behavior, behaviorReceipt, /Fixture failed: native-release/u);
  const failed = JSON.parse(readFileSync(behaviorReceipt, 'utf8'));
  assert.equal(failed.passed, false);
  assert.equal(failed.scratchRemoved, true);
  assert.equal(failed.executions.at(-1).name, 'native-release');
  assert.notEqual(failed.executions.at(-1).exit, 0);

  const retained = join(root, 'existing-receipt.json');
  writeFileSync(retained, 'retain this receipt\n');
  const before = hash(retained);
  rejected(bundle, retained, /Receipt already exists/u);
  assert.equal(hash(retained), before);
  console.log('passed corrupt image, wrong native behavior, and existing receipt checks');
} finally {
  // This directory was created here, never supplied by the caller.
  assert.equal(realpathSync(root), root);
  assert.ok(root.startsWith(realpathSync(cache) + sep));
  rmSync(root, { recursive: true });
}
