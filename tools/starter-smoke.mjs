import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { resolve, join, relative } from 'node:path';
import { spawnSync } from 'node:child_process';

// Fixed local workflow; only a local packaging mode, no arbitrary paths,
// network, installation or publication.
const packageLocal = process.argv.length === 3 && process.argv[2] === '--package-local';
assert.ok(process.argv.length === 2 || packageLocal, 'only --package-local is accepted');
assert.equal(process.platform, 'win32', 'requires Windows');
assert.equal(process.arch, 'x64', 'requires AMD64');
const repo = resolve(import.meta.dirname, '..');
assert.equal(resolve(process.cwd()), repo, 'run from compiler repository');
const cache = join(repo, '.cache');
mkdirSync(cache, { recursive: true });
const root = mkdtempSync(join(cache, 'starter-smoke-'));
const scratch = join(root, 'scratch');
mkdirSync(scratch);
assert.deepEqual(readdirSync(scratch), []);
let exe = join(root, 'fairyc.exe');
const deadline = Date.now() + 150_000;
const steps = [];
function command(name, binary, args, cwd, timeout = 45_000) {
  const remaining = Math.min(timeout, deadline - Date.now());
  assert.ok(remaining > 0, 'workflow deadline exceeded');
  const started = Date.now();
  const result = spawnSync(binary, args, {
    cwd, shell: false, windowsHide: true, input: '', encoding: 'utf8',
    timeout: remaining, maxBuffer: 1024 * 1024,
    env: { ...process.env, TEMP: root, TMP: root },
  });
  if (result.error) throw result.error;
  assert.equal(result.status, 0, `${name}: ${result.stderr}\n${result.stdout}`);
  assert.equal(result.stderr, '', `${name} stderr`);
  const step = { name, exit: result.status, elapsed_ms: Date.now() - started, stdout: result.stdout };
  steps.push(step);
  console.log(JSON.stringify(step));
  return result.stdout;
}
const sha = path => createHash('sha256').update(readFileSync(path)).digest('hex');
command('build', 'bun', ['tools/run-go.ts', 'build', '-mod=readonly', '-o', exe, './cmd/fairyc'], repo, 60_000);
const executableHash = sha(exe);
const buildMetadata = command('build-metadata', 'bun', ['tools/run-go.ts', 'version', '-m', exe], repo);
const goVersion = buildMetadata.match(/\bgo1\.\d+(?:\.\d+)?\b/u)?.[0];
const vcsRevision = buildMetadata.match(/\bvcs\.revision=([0-9a-f]{40})\b/u)?.[1];
const vcsModified = buildMetadata.match(/\bvcs\.modified=(true|false)\b/u)?.[1];
assert.ok(goVersion && vcsRevision && vcsModified, 'missing Go build identity');
let archive;
if (packageLocal) {
  const stage = join(root, 'stage');
  const extracted = join(root, 'extracted');
  mkdirSync(stage);
  copyFileSync(exe, join(stage, 'fairyc.exe'));
  copyFileSync(join(repo, 'docs/ops/windows-preview-README.txt'), join(stage, 'README.txt'));
  const notices = ['LICENSE', 'LICENSING.md', 'THIRD_PARTY_NOTICES.md', 'GO-LICENSE.txt'];
  for (const name of notices) copyFileSync(join(repo, name), join(stage, name));
  const entries = ['README.txt', 'fairyc.exe', ...notices];
  const hashes = entries.map(name => sha(join(stage, name)));
  writeFileSync(join(stage, 'SHA256SUMS.txt'), entries.map((name, i) => `${hashes[i]}  ${name}`).join('\n') + '\n', { flag: 'wx' });
  const checksumHash = sha(join(stage, 'SHA256SUMS.txt'));
  const archiveName = 'fairyc-windows-amd64-preview-candidate.zip';
  archive = join(root, archiveName);
  // Paths originate exclusively from this run's unique cache directory. Avoid
  // interpolated shell paths, overwrites, deletion and external archive inputs.
  process.env.FAIRYC_PREVIEW_STAGE = stage;
  process.env.FAIRYC_PREVIEW_ZIP = archive;
  process.env.FAIRYC_PREVIEW_EXTRACTED = extracted;
  command('zip-roundtrip', 'powershell.exe', ['-NoProfile', '-NonInteractive', '-Command',
    "$ErrorActionPreference='Stop'; $names=@('fairyc.exe','README.txt','LICENSE','LICENSING.md','THIRD_PARTY_NOTICES.md','GO-LICENSE.txt','SHA256SUMS.txt'); $paths=@($names | ForEach-Object { Join-Path $env:FAIRYC_PREVIEW_STAGE $_ }); Compress-Archive -LiteralPath $paths -DestinationPath $env:FAIRYC_PREVIEW_ZIP; Expand-Archive -LiteralPath $env:FAIRYC_PREVIEW_ZIP -DestinationPath $env:FAIRYC_PREVIEW_EXTRACTED"], repo);
  delete process.env.FAIRYC_PREVIEW_STAGE;
  delete process.env.FAIRYC_PREVIEW_ZIP;
  delete process.env.FAIRYC_PREVIEW_EXTRACTED;
  assert.deepEqual(readdirSync(extracted).sort(), [...entries, 'SHA256SUMS.txt'].sort());
  assert.deepEqual(entries.map(name => sha(join(extracted, name))), hashes, 'ZIP changed contents');
  assert.equal(sha(join(extracted, 'SHA256SUMS.txt')), checksumHash);
  writeFileSync(archive + '.sha256', `${sha(archive)}  ${archiveName}\n`, { flag: 'wx' });
  exe = join(extracted, 'fairyc.exe');
}
assert.match(command('new', exe, ['new', 'hello-fairy', '--uid', 'example.hello-fairy'], scratch), /created Fairy C package/);
const workspace = join(scratch, 'hello-fairy');
const files = ['fairy.toml', 'src/main.fy', 'tests/main.fy', 'README.md'];
const original = files.map(path => sha(join(workspace, path)));
assert.match(command('check', exe, ['check'], workspace), /lock updated\n$/);
const lockHash = sha(join(workspace, 'fairy.lock'));
function unchanged() {
  assert.equal(sha(join(workspace, 'fairy.lock')), lockHash, 'locked workflow changed lock');
  assert.deepEqual(files.map(path => sha(join(workspace, path))), original, 'workflow changed starter files');
}
assert.match(command('check-locked', exe, ['check', '--locked'], workspace), /lock verified\n$/);
unchanged();
let testOutput;
for (const [name, args] of [
  ['test', ['test', '--locked']],
  ['native-test', ['test', '--native', '--locked']],
  ['native-test-release', ['test', '--native', '--release', '--locked']],
]) {
  const output = command(name, exe, args, workspace);
  assert.match(output, /1 passed; 0 failed/);
  if (testOutput === undefined) testOutput = output;
  else assert.equal(output, testOutput, 'test modes disagree');
  unchanged();
}
assert.equal(command('run', exe, ['run', '--locked'], workspace), '');
unchanged();
assert.equal(command('run-release', exe, ['run', '--release', '--locked'], workspace), '');
unchanged();
assert.equal(sha(exe), executableHash, 'compiler executable changed');
const receipt = { schema: 'fairyc.local-starter-smoke/v1', executable_sha256: executableHash,
  lock_sha256: lockHash, node: process.version, go_version: goVersion,
  vcs_revision: vcsRevision, vcs_modified: vcsModified === 'true',
  platform: `${process.platform}/${process.arch}`, steps,
  ...(archive ? { preview_candidate_archive: relative(repo, archive), archive_sha256: sha(archive) } : {}) };
writeFileSync(join(root, 'receipt.json'), JSON.stringify(receipt, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ receipt: relative(repo, join(root, 'receipt.json')), executable_sha256: executableHash }));
