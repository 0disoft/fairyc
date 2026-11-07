import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { lstatSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, realpathSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, isAbsolute, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { lspProbeInput, verifyLspProbe } from './lsp-probe.mjs';
import { verifyCandidate } from './verify.mjs';

// Retained-ZIP execution only: no builds of the compiler, installation, network,
// publication or alternate packager. Adapted from the 2026-11-04 cache-only smoke.
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');
const within = (parent, path) => {
  const rel = relative(parent, path);
  return rel === '' || (!isAbsolute(rel) && rel !== '..' && !rel.startsWith('..' + sep));
};

export function previewEnvironment(source, root) {
  // Keep Windows discovery roots, not developer-shell overrides, credentials,
  // Fairy caches, Go configuration or user PATH entries.
  const allowed = /^(SystemRoot|SystemDrive|windir|ComSpec|ProgramFiles|ProgramFiles\(x86\)|ProgramW6432|ProgramData|ALLUSERSPROFILE|USERPROFILE|HOMEDRIVE|HOMEPATH|LOCALAPPDATA|APPDATA|PATHEXT|OS|NUMBER_OF_PROCESSORS|PROCESSOR_ARCHITECTURE)$/iu;
  const env = Object.fromEntries(Object.entries(source).filter(([key]) => allowed.test(key)));
  const systemRoot = Object.entries(env).find(([key]) => /^SystemRoot$/iu.test(key))?.[1];
  assert.ok(systemRoot && isAbsolute(systemRoot), 'missing absolute Windows SystemRoot');
  env.PATH = `${join(systemRoot, 'System32')};${systemRoot}`;
  env.TEMP = root;
  env.TMP = root;
  return { env, systemRoot };
}

export function verifyExtracted(extracted, retained) {
  const files = readdirSync(retained).sort();
  assert.deepEqual(readdirSync(extracted).sort(), files, 'unexpected external ZIP contents');
  for (const name of files) {
    const path = join(extracted, name);
    const info = lstatSync(path);
    assert.ok(info.isFile() && !info.isSymbolicLink(), `not a regular external ZIP file: ${name}`);
    assert.equal(hash(path), hash(join(retained, name)), `external ZIP file hash differs: ${name}`);
  }
  return files.length;
}

export function builtImage(output, workspace, profile) {
  // Accept only the stable public Windows build receipt, never an arbitrary path
  // supplied in process output or an experimental/private CLI build receipt.
  const match = /^built: (build\/([A-Za-z0-9_-]+)\.exe) \(x86_64-windows-msvc, (development|release), lock verified\)\r?\n$/u.exec(output);
  assert.ok(match && match[3] === profile, 'build did not report the expected locked Windows profile');
  return resolve(workspace, match[1]);
}

export function createRunner({ env, deadline, onStep }) {
  return (name, binary, args, cwd, childEnv = env, input = '') => {
    const timeout = Math.min(90_000, deadline - Date.now());
    assert.ok(timeout > 0, 'external smoke deadline exceeded');
    const started = Date.now();
    const child = spawnSync(binary, args, { cwd, env: childEnv, shell: false, windowsHide: true,
      encoding: 'utf8', timeout, maxBuffer: 1024 * 1024, input });
    const step = { name, binary, args, cwd, exit: child.status, stdout: child.stdout, stderr: child.stderr,
      elapsedMs: Date.now() - started, ...(child.error ? { error: child.error.message } : {}) };
    onStep(step);
    if (child.error) throw child.error;
    assert.equal(child.status, 0, `${name}: ${child.stderr}\n${child.stdout}`);
    assert.equal(child.stderr, '', `${name}: unexpected stderr`);
    return child.stdout;
  };
}

export async function externalSmoke({ repo, receipt, revision }) {
  assert.equal(process.platform, 'win32', 'requires Windows');
  assert.equal(process.arch, 'x64', 'requires AMD64');
  repo = realpathSync(repo);
  const verified = await verifyCandidate({ repo, receipt, revision });
  assert.ok(verified.lspSHA256, 'external smoke requires the public compiler-and-LSP v2 ZIP');
  const temporary = realpathSync(tmpdir());
  assert.ok(!within(repo, temporary), 'external scratch must be outside the compiler checkout');
  const root = realpathSync(mkdtempSync(join(temporary, 'fairyc ZIP 검증-')));
  const extracted = join(root, 'preview');
  const scratch = join(root, 'projects');
  mkdirSync(scratch);
  const output = join(root, 'external-zip-receipt.json');
  const { env, systemRoot } = previewEnvironment(process.env, root);
  const inputReceipt = realpathSync(resolve(repo, receipt));
  const inputReceiptHash = hash(inputReceipt);
  const result = { schema: 'fairyc.windows-external-zip-smoke/v1', passed: false,
    sourceRevision: revision, archive: verified.archive, archiveSHA256: verified.archiveSHA256,
    executableSHA256: verified.executableSHA256, lspSHA256: verified.lspSHA256,
    inputReceipt, inputReceiptSHA256: inputReceiptHash, verifierSHA256: hash(fileURLToPath(import.meta.url)),
    root, childPath: env.PATH, cleanMachineEvidence: false, publicationApproved: false, steps: [] };
  const command = createRunner({ env, deadline: Date.now() + 210_000, onStep: step => {
    result.steps.push(step);
    console.log(JSON.stringify(step));
  } });
  try {
    command('extract', join(systemRoot, 'System32/WindowsPowerShell/v1.0/powershell.exe'),
      ['-NoProfile', '-NonInteractive', '-Command',
        "$ErrorActionPreference='Stop'; Expand-Archive -LiteralPath $env:FAIRYC_INPUT_ZIP -DestinationPath $env:FAIRYC_EXTRACTED"],
      root, { ...env, FAIRYC_INPUT_ZIP: verified.archive, FAIRYC_EXTRACTED: extracted });
    result.verifiedFiles = verifyExtracted(extracted, join(dirname(inputReceipt), 'extracted'));
    const exe = join(extracted, 'fairyc.exe');
    const lsp = join(extracted, 'fairyc-lsp.exe');
    assert.match(command('lsp-version', lsp, ['--version'], scratch), /^fairyc-lsp \d+\.\d+\.\d+\r?\n$/u);
    verifyLspProbe(command('lsp-protocol', lsp, [], scratch, env, lspProbeInput(pathToFileURL(scratch).href)));
    assert.match(command('new', exe, ['new', 'hello-fairy', '--uid', 'example.external-zip'], scratch), /created Fairy C package/u);
    const workspace = join(scratch, 'hello-fairy');
    const sources = ['fairy.toml', 'README.md', 'src/main.fy', 'tests/main.fy'];
    const original = sources.map(path => hash(join(workspace, path)));
    assert.match(command('check', exe, ['check'], workspace), /lock updated\n$/u);
    const lock = hash(join(workspace, 'fairy.lock'));
    const unchanged = () => {
      assert.equal(hash(join(workspace, 'fairy.lock')), lock, 'locked workflow changed lock');
      assert.deepEqual(sources.map(path => hash(join(workspace, path))), original, 'workflow changed starter files');
    };
    assert.match(command('check-locked', exe, ['check', '--locked'], workspace), /lock verified\n$/u);
    unchanged();
    const tests = command('semantic-test', exe, ['test', '--locked'], workspace);
    assert.match(tests, /1 passed; 0 failed/u);
    unchanged();
    result.images = [];
    for (const profile of ['development', 'release']) {
      const flags = profile === 'release' ? ['--release', '--locked'] : ['--locked'];
      const image = builtImage(command(`build-${profile}`, exe, ['build', ...flags], workspace), workspace, profile);
      unchanged();
      const imageSHA256 = hash(image);
      assert.equal(command(`image-${profile}`, image, [], workspace), '');
      assert.equal(hash(image), imageSHA256, 'direct execution changed image');
      result.images.push({ profile, path: image, sha256: imageSHA256 });
      assert.equal(command(`run-${profile}`, exe, ['run', ...flags], workspace), '');
      unchanged();
      assert.equal(command(`native-test-${profile}`, exe, ['test', '--native', ...flags], workspace), tests, 'test modes disagree');
      unchanged();
    }
    result.verifiedFiles = verifyExtracted(extracted, join(dirname(inputReceipt), 'extracted'));
    assert.equal(hash(inputReceipt), inputReceiptHash, 'input receipt changed');
    await verifyCandidate({ repo, receipt, revision });
    result.sourceSHA256 = Object.fromEntries(sources.map((path, i) => [path, original[i]]));
    result.lockSHA256 = lock;
    result.passed = true;
  } catch (error) {
    result.error = error.stack;
  } finally {
    // Unique retained scratch/receipt per attempt; never replace past evidence.
    writeFileSync(output, JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
    console.log(JSON.stringify({ passed: result.passed, receipt: output, root }));
  }
  assert.ok(result.passed, result.error);
  return { ...result, receipt: output };
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const args = process.argv.slice(2);
    assert.ok(args.length === 4 && args[0] === '--receipt' && args[2] === '--revision',
      'usage: node tools/windows-preview/external-smoke.mjs --receipt <receipt.json> --revision <full-commit>');
    await externalSmoke({ repo: resolve(import.meta.dirname, '../..'), receipt: args[1], revision: args[3] });
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
