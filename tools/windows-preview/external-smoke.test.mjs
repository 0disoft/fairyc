import assert from 'node:assert/strict';
import { copyFileSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { builtImage, createRunner, previewEnvironment, verifyExtracted } from './external-smoke.mjs';

test('minimal environment retains Windows discovery roots and drops shell overrides', () => {
  const windows = process.platform === 'win32' ? 'C:\\Windows' : '/Windows';
  const { env, systemRoot } = previewEnvironment({
    sYsTeMrOoT: windows, 'ProgramFiles(x86)': 'discovery-root', USERPROFILE: 'profile',
    Path: 'developer-path', PATH: 'another-path', Include: 'headers', LIB: 'libraries',
    CC: 'override', CXX: 'override', CL: 'override', _CL_: 'override', LINK: 'override',
    GOFLAGS: '-changed', GOPATH: 'developer-go', VSINSTALLDIR: 'developer-vs',
    WindowsSdkDir: 'developer-sdk', VCToolsInstallDir: 'developer-vc',
    FAIRYC_CACHE_DIR: 'old-cache', FAIRYC_INPUT_ZIP: 'old-zip', SECRET_TOKEN: 'excluded',
    TEMP: 'old-temp', TMP: 'old-temp',
  }, 'fresh-scratch');
  assert.equal(systemRoot, windows);
  assert.equal(env.PATH, `${join(windows, 'System32')};${windows}`);
  assert.equal(env.TEMP, 'fresh-scratch');
  assert.equal(env.TMP, 'fresh-scratch');
  assert.equal(env['ProgramFiles(x86)'], 'discovery-root');
  assert.equal(env.USERPROFILE, 'profile');
  assert.deepEqual(Object.keys(env).sort(), ['PATH', 'ProgramFiles(x86)', 'TEMP', 'TMP', 'USERPROFILE', 'sYsTeMrOoT'].sort());
  assert.throws(() => previewEnvironment({}, 'scratch'), /SystemRoot/u);
  assert.throws(() => previewEnvironment({ SystemRoot: 'relative' }, 'scratch'), /SystemRoot/u);
});

test('build receipt confines direct execution to the expected public profile and build path', () => {
  const workspace = join(tmpdir(), 'fairyc workspace 한글');
  assert.equal(builtImage('built: build/hello-fairy.exe (x86_64-windows-msvc, development, lock verified)\n', workspace, 'development'), join(workspace, 'build', 'hello-fairy.exe'));
  assert.equal(builtImage('built: build/hello-fairy.exe (x86_64-windows-msvc, release, lock verified)\r\n', workspace, 'release'), join(workspace, 'build', 'hello-fairy.exe'));
  for (const output of [
    'built: build/../../elsewhere.exe (x86_64-windows-msvc, release, lock verified)\n',
    'built: C:/elsewhere.exe (x86_64-windows-msvc, release, lock verified)\n',
    'built: build/hello-fairy.exe (x86_64-windows-msvc, development, lock verified)\n',
    'built: build/hello-fairy.exe (x86_64-windows-msvc, release, lock updated)\n',
    'built: build/hello-fairy.exe (x86_64-linux-gnu, release, lock verified)\n',
    'built: build/hello-fairy.exe (x86_64-windows-msvc, release, lock verified)\nextra\n',
  ]) assert.throws(() => builtImage(output, workspace, 'release'), /expected locked Windows profile/u);
});

test('fresh ZIP expansion must match every retained file including notices and checksum bytes', () => {
  const root = mkdtempSync(join(tmpdir(), 'fairyc-external-test-'));
  try {
    const retained = join(root, 'retained');
    const expanded = join(root, 'expanded');
    mkdirSync(retained);
    mkdirSync(expanded);
    const files = ['fairyc.exe', 'fairyc-lsp.exe', 'LICENSE', 'SHA256SUMS.txt'];
    for (const name of files) {
      writeFileSync(join(retained, name), `original ${name}`);
      copyFileSync(join(retained, name), join(expanded, name));
    }
    assert.equal(verifyExtracted(expanded, retained), 4);
    for (const name of files) {
      writeFileSync(join(expanded, name), `tampered ${name}`);
      assert.throws(() => verifyExtracted(expanded, retained), /file hash differs/u);
      copyFileSync(join(retained, name), join(expanded, name));
    }
    writeFileSync(join(expanded, 'extra'), 'unlisted');
    assert.throws(() => verifyExtracted(expanded, retained), /unexpected external ZIP contents/u);
    rmSync(join(expanded, 'extra'));
    rmSync(join(expanded, 'fairyc.exe'));
    mkdirSync(join(expanded, 'fairyc.exe'));
    assert.throws(() => verifyExtracted(expanded, retained), /not a regular/u);
  } finally {
    rmSync(root, { recursive: true });
  }
});

test('runner retains actual failures, rejects stderr and stops before an expired deadline', () => {
  const steps = [];
  const command = createRunner({ env: process.env, deadline: Date.now() + 20_000, onStep: step => steps.push(step) });
  assert.equal(command('success', process.execPath, ['-e', 'process.stdout.write("ok")'], tmpdir()), 'ok');
  assert.throws(() => command('failure', process.execPath, ['-e', 'process.exit(7)'], tmpdir()), /failure/u);
  assert.equal(steps.at(-1).exit, 7);
  assert.throws(() => command('stderr', process.execPath, ['-e', 'process.stderr.write("diagnostic")'], tmpdir()), /unexpected stderr/u);
  assert.equal(steps.at(-1).stderr, 'diagnostic');
  const expired = createRunner({ env: process.env, deadline: Date.now() - 1, onStep: () => assert.fail('must not spawn') });
  assert.throws(() => expired('expired', process.execPath, [], tmpdir()), /deadline exceeded/u);
  const timeoutSteps = [];
  const bounded = createRunner({ env: process.env, deadline: Date.now() + 200, onStep: step => timeoutSteps.push(step) });
  assert.throws(() => bounded('timeout', process.execPath, ['-e', 'setTimeout(() => {}, 5000)'], tmpdir()), /ETIMEDOUT/u);
  assert.equal(timeoutSteps.length, 1);
  assert.ok(timeoutSteps[0].error.includes('ETIMEDOUT'));
});
