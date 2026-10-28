import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs';
import { basename, join, relative, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { verifyCandidate } from '../windows-preview/verify.mjs';

const args = process.argv.slice(2);
assert.deepEqual([args.length, args[0], args[2], args[4]], [6, '--receipt', '--revision', '--code-root'],
  'usage: node tools/windows-editor/prepare.mjs --receipt <receipt.json> --revision <full-commit> --code-root <VS Code directory>');
assert.equal(process.platform, 'win32');
assert.equal(process.arch, 'x64');
const repo = resolve(import.meta.dirname, '../..');
assert.equal(resolve(process.cwd()), repo, 'run from the compiler repository');
const preview = await verifyCandidate({ repo, receipt: args[1], revision: args[3] });
assert.ok(preview.lspSHA256, 'requires the compiler and LSP candidate');
const codeRoot = resolve(args[5]);
const extension = JSON.parse(readFileSync(join(repo, 'editors/vscode/package.json'), 'utf8'));
const vsix = join(repo, '.cache/editor-release', `${extension.name}-${extension.version}.vsix`);
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
assert.equal(readFileSync(vsix + '.sha256', 'utf8').trim(), `${hash(vsix)}  ${basename(vsix)}`, 'VSIX checksum differs');
const root = mkdtempSync(join(repo, '.cache/windows-editor-'));
const stage = join(root, 'bundle');
mkdirSync(stage);
for (const source of [preview.archive, preview.archive + '.sha256', vsix, vsix + '.sha256']) {
  copyFileSync(source, join(stage, basename(source)));
}
for (const name of ['LICENSE', 'LICENSING.md', 'THIRD_PARTY_NOTICES.md', 'GO-LICENSE.txt']) {
  copyFileSync(join(repo, name), join(stage, name));
}
for (const name of ['test-vscode-install.ps1', 'test-vscode-installed.cjs']) {
  copyFileSync(join(repo, 'tools', name), join(stage, name));
}
for (const name of ['verify.ps1', 'README.txt']) copyFileSync(join(import.meta.dirname, name), join(stage, name));
copyFileSync(join(repo, 'editors/vscode/package.json'), join(stage, 'extension.json'));
const manifest = { schema: 'fairyc.windows-editor-bundle/v1', sourceRevision: preview.sourceRevision,
  preview: { name: basename(preview.archive), sha256: preview.archiveSHA256, lspSHA256: preview.lspSHA256 },
  vsix: { name: basename(vsix), sha256: hash(vsix), extensionId: `${extension.publisher}.${extension.name}`, version: extension.version },
  cleanMachineEvidence: false, publicationApproved: false,
  files: readdirSync(stage).sort().map(name => ({ name, bytes: statSync(join(stage, name)).size, sha256: hash(join(stage, name)) })) };
writeFileSync(join(stage, 'manifest.json'), JSON.stringify(manifest, null, 2) + '\n', { flag: 'wx' });
const archive = join(root, 'fairyc-windows-editor-verify.zip');
const extracted = join(root, 'extracted');
function run(name, argv, environment = {}) {
  const result = spawnSync(join(process.env.SystemRoot, 'System32/WindowsPowerShell/v1.0/powershell.exe'), argv,
    { cwd: root, shell: false, windowsHide: true, encoding: 'utf8', input: '', timeout: 120_000,
      maxBuffer: 1024 * 1024, env: { ...process.env,
        PSModulePath: join(process.env.SystemRoot, 'System32/WindowsPowerShell/v1.0/Modules'), ...environment } });
  if (result.error) throw result.error;
  assert.equal(result.status, 0, `${name}: ${result.stdout}\n${result.stderr}`);
  console.log(result.stdout.trim());
}
run('archive-roundtrip', ['-NoProfile', '-NonInteractive', '-Command',
  "$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.IO.Compression.FileSystem; [IO.Compression.ZipFile]::CreateFromDirectory($env:FAIRYC_EDITOR_BUNDLE,$env:FAIRYC_EDITOR_ZIP); [IO.Compression.ZipFile]::ExtractToDirectory($env:FAIRYC_EDITOR_ZIP,$env:FAIRYC_EDITOR_EXTRACTED)"],
{ FAIRYC_EDITOR_BUNDLE: stage, FAIRYC_EDITOR_ZIP: archive, FAIRYC_EDITOR_EXTRACTED: extracted });
assert.deepEqual(readdirSync(extracted).sort(), readdirSync(stage).sort());
for (const name of readdirSync(stage)) assert.equal(hash(join(extracted, name)), hash(join(stage, name)), `ZIP changed ${name}`);
const results = join(root, 'local-results');
run('standalone-editor-verification', ['-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass',
  '-File', join(extracted, 'verify.ps1'), '-CodeRoot', codeRoot, '-ResultsDirectory', results],
{ PATH: `${join(process.env.SystemRoot, 'System32')};${process.env.SystemRoot}` });
const receipt = JSON.parse(readFileSync(join(results, 'receipt.json'), 'utf8'));
assert.equal(receipt.passed, true);
assert.deepEqual(Object.values(receipt.toolPaths), [null, null, null], 'external development tool remains on PATH');
assert.equal(receipt.installation.host.version, extension.version);
assert.equal(receipt.installation.host.serverSHA256, preview.lspSHA256);
assert.equal(receipt.vsix.sha256, manifest.vsix.sha256);
assert.equal(receipt.preview.sha256, preview.archiveSHA256);
assert.equal(receipt.cleanMachineEvidence, false);
writeFileSync(archive + '.sha256', `${hash(archive)}  ${basename(archive)}\n`, { flag: 'wx' });
const preparation = { schema: 'fairyc.windows-editor-preparation/v1', archive: relative(repo, archive),
  archiveSHA256: hash(archive), archiveBytes: statSync(archive).size, sourceRevision: preview.sourceRevision,
  manifestSHA256: hash(join(stage, 'manifest.json')), localReceipt: relative(repo, join(results, 'receipt.json')),
  localReceiptSHA256: hash(join(results, 'receipt.json')), inputPreviewSHA256: preview.archiveSHA256,
  inputVSIXSHA256: manifest.vsix.sha256, verifiedFiles: manifest.files.length,
  externalToolsAbsentFromChildPATH: true, cleanMachineEvidence: false, publicationApproved: false };
writeFileSync(join(root, 'preparation.json'), JSON.stringify(preparation, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(preparation, null, 2));
