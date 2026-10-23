import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { basename, join, relative, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const args = process.argv.slice(2);
assert.equal(args.length, 4, 'usage: node tools/windows-portability/prepare.mjs --compiler <fairyc.exe> --dumpbin <dumpbin.exe>');
assert.equal(args[0], '--compiler');
assert.equal(args[2], '--dumpbin');
assert.equal(process.platform, 'win32');
assert.equal(process.arch, 'x64');
const compiler = resolve(args[1]);
const dumpbin = resolve(args[3]);
const repo = resolve(import.meta.dirname, '../..');
assert.equal(resolve(process.cwd()), repo, 'run from the compiler repository');
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');
const steps = [];
function run(name, binary, argv, cwd, environment = {}) {
  const result = spawnSync(binary, argv, { cwd, shell: false, windowsHide: true, encoding: 'utf8',
    input: '', timeout: 90_000, maxBuffer: 1024 * 1024, env: { ...process.env, ...environment } });
  if (result.error) throw result.error;
  assert.equal(result.status, 0, `${name}: ${result.stdout}\n${result.stderr}`);
  assert.equal(result.stderr, '', `${name} stderr`);
  steps.push({ name, exit: result.status, stdout: result.stdout });
  console.log(name);
  return result.stdout;
}
const version = run('compiler-version', compiler, ['--version'], repo);
assert.match(version, /\nmodified: false\n$/u, 'use a compiler built from a clean commit');
const revision = version.match(/\nrevision: ([0-9a-f]{40})\n/u)?.[1];
assert.ok(revision, 'compiler lacks source identity');
const buildMetadata = run('compiler-build-metadata', 'go', ['version', '-m', compiler], repo);
assert.match(buildMetadata, /\bCGO_ENABLED=0\b/u);
assert.match(buildMetadata, /\bGOARCH=amd64\b/u);
assert.match(buildMetadata, /\bGOOS=windows\b/u);
mkdirSync(join(repo, '.cache'), { recursive: true });
const root = mkdtempSync(join(repo, '.cache/windows-portability-'));
const stage = join(root, 'bundle');
const work = join(root, 'build-work');
mkdirSync(stage);
mkdirSync(work);
run('new-native-fixture', compiler, ['new', 'portability-probe', '--uid', 'example.portability'], work);
const project = join(work, 'portability-probe');
const source = join(import.meta.dirname, 'main.fy');
copyFileSync(source, join(project, 'src/main.fy'));
copyFileSync(compiler, join(stage, 'fairyc.exe'));
copyFileSync(join(import.meta.dirname, 'verify.ps1'), join(stage, 'verify.ps1'));
copyFileSync(join(import.meta.dirname, 'README.txt'), join(stage, 'README.txt'));
for (const name of ['LICENSE', 'LICENSING.md', 'THIRD_PARTY_NOTICES.md', 'GO-LICENSE.txt']) {
  copyFileSync(join(repo, name), join(stage, name));
}
for (const profile of ['development', 'release']) {
  const stdout = run(`build-${profile}`, compiler, ['build', ...(profile === 'release' ? ['--release', '--locked'] : [])], project);
  assert.match(stdout, /^built: build\/portability-probe\.exe /u);
  copyFileSync(join(project, 'build/portability-probe.exe'), join(stage, `native-${profile}.exe`));
}
const images = {};
for (const name of ['fairyc.exe', 'native-development.exe', 'native-release.exe']) {
  const output = run(`dependencies-${name}`, dumpbin, ['/nologo', '/dependents', join(stage, name)], repo);
  const imports = [...new Set([...output.matchAll(/^\s+([A-Za-z0-9_.-]+\.dll)\s*$/gimu)].map(match => match[1].toLowerCase()))].sort();
  // A new import must be investigated and added to the distribution contract.
  assert.deepEqual(imports, ['kernel32.dll'], `${name}: changed DLL dependencies`);
  images[name] = { imports, importScope: 'dumpbin direct and delay-load dependencies; dynamic LoadLibrary calls are not enumerated',
    crt: name === 'fairyc.exe' ? 'Go, CGO disabled' : 'MSVC /MT static CRT' };
}
const files = readdirSync(stage).sort().map(name => ({ name, bytes: statSync(join(stage, name)).size, sha256: hash(join(stage, name)) }));
const manifest = { schema: 'fairyc.windows-portability-bundle/v1', compilerRevision: revision, compilerVersion: version,
  sourceSHA256: hash(source), platform: 'windows/amd64', files, images,
  inputBytes: [0, 1, 127, 128, 255, 10, 42], nativeStdout: 'Fairy C portability\n',
  cleanMachineEvidence: false, lspIncluded: false };
writeFileSync(join(stage, 'manifest.json'), JSON.stringify(manifest, null, 2) + '\n', { flag: 'wx' });
const receipt = join(root, 'local-receipt.json');
const archive = join(root, 'fairyc-windows-portability.zip');
const extracted = join(root, 'extracted');
run('archive-roundtrip', 'powershell.exe', ['-NoProfile', '-NonInteractive', '-Command',
  "$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.IO.Compression.FileSystem; [IO.Compression.ZipFile]::CreateFromDirectory($env:FAIRYC_PORTABILITY_BUNDLE,$env:FAIRYC_PORTABILITY_ZIP); [IO.Compression.ZipFile]::ExtractToDirectory($env:FAIRYC_PORTABILITY_ZIP,$env:FAIRYC_PORTABILITY_EXTRACTED)"], repo,
{ FAIRYC_PORTABILITY_BUNDLE: stage, FAIRYC_PORTABILITY_ZIP: archive, FAIRYC_PORTABILITY_EXTRACTED: extracted });
assert.deepEqual(readdirSync(extracted).sort(), readdirSync(stage).sort());
for (const name of readdirSync(stage)) assert.equal(hash(join(extracted, name)), hash(join(stage, name)), `ZIP changed ${name}`);
run('isolated-extracted-execution', 'powershell.exe', ['-NoProfile', '-NonInteractive', '-File', join(extracted, 'verify.ps1'), '-Receipt', receipt], extracted);
const verified = JSON.parse(readFileSync(receipt, 'utf8'));
assert.equal(verified.passed, true);
assert.equal(verified.scratchRemoved, true);
assert.equal(verified.cleanMachineEvidence, false);
writeFileSync(archive + '.sha256', `${hash(archive)}  ${basename(archive)}\n`, { flag: 'wx' });
writeFileSync(join(root, 'preparation.json'), JSON.stringify({ schema: 'fairyc.windows-portability-preparation/v1',
  compilerRevision: revision, bundle: relative(repo, stage), extracted: relative(repo, extracted), manifestSHA256: hash(join(stage, 'manifest.json')),
  archive: relative(repo, archive), archiveSHA256: hash(archive), archiveBytes: statSync(archive).size,
  verifierSHA256: hash(join(stage, 'verify.ps1')), localReceiptSHA256: hash(receipt),
  cleanMachineEvidence: false, dumpbinSHA256: hash(dumpbin), steps }, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ preparation: relative(repo, join(root, 'preparation.json')), archive: relative(repo, archive) }));
