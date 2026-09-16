import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdirSync, mkdtempSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { resolve, join, relative } from 'node:path';
import { spawnSync } from 'node:child_process';

// Fixed local workflow; no user arguments, network, installation or publication.
assert.equal(process.argv.length, 2, 'no arguments accepted');
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
const exe = join(root, 'fairyc.exe');
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
  lock_sha256: lockHash, node: process.version, platform: `${process.platform}/${process.arch}`, steps };
writeFileSync(join(root, 'receipt.json'), JSON.stringify(receipt, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ receipt: relative(repo, join(root, 'receipt.json')), executable_sha256: executableHash }));
