import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtempSync, mkdirSync, readFileSync, realpathSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, relative, sep } from 'node:path';
import test from 'node:test';
import { verifyCandidate } from './verify.mjs';
import { lspProbeInput, verifyLspProbe } from './lsp-probe.mjs';

const revision = 'a'.repeat(40);
const archiveName = 'fairyc-windows-amd64-preview-candidate.zip';
const files = ['fairyc.exe', 'fairyc-lsp.exe', 'README.txt', 'LICENSE', 'LICENSING.md', 'THIRD_PARTY_NOTICES.md', 'GO-LICENSE.txt'];
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');
const frame = message => {
  const body = JSON.stringify({ jsonrpc: '2.0', ...message });
  return `Content-Length: ${Buffer.byteLength(body)}\r\n\r\n${body}`;
};
const probeOutput = frame({ id: 1, result: { serverInfo: { name: 'fairyc-lsp', version: '0.1.0', label: '프리뷰' },
  capabilities: { positionEncoding: 'utf-16', definitionProvider: true, completionProvider: {} } } }) +
  frame({ id: 2, result: null });

function fixture() {
  const repo = realpathSync(mkdtempSync(join(tmpdir(), 'fairyc-preview-check-')));
  const root = join(repo, '.cache', 'candidate');
  const extracted = join(root, 'extracted');
  mkdirSync(extracted, { recursive: true });
  // Byte-only fixtures exercise evidence validation; no executable is launched.
  for (const name of files) writeFileSync(join(extracted, name), `fixture bytes: ${name}\n`);
  const sums = join(extracted, 'SHA256SUMS.txt');
  writeFileSync(sums, files.map(name => `${hash(join(extracted, name))}  ${name}\n`).join(''));
  const archive = join(root, archiveName);
  writeFileSync(archive, 'archive identity fixture\n');
  writeFileSync(archive + '.sha256', `${hash(archive)}  ${archiveName}\n`);
  const image = join(extracted, 'fairyc.exe');
  const server = join(extracted, 'fairyc-lsp.exe');
  const input = { schema: 'fairyc.local-starter-smoke/v2', platform: 'win32/x64', vcs_modified: false,
    vcs_revision: revision, build_flags: ['-trimpath', '-ldflags=-s -w'],
    go_version: 'go1.27.1',
    steps: ['build', 'build-metadata', 'lsp-build', 'lsp-build-metadata', 'go-root', 'zip-roundtrip',
      'lsp-version', 'lsp-protocol', 'new', 'check', 'check-locked',
      'test', 'native-test', 'native-test-release', 'run', 'run-release'].map(name => ({ name, exit: 0,
      stdout: name === 'lsp-build-metadata' ? `go1.27.1 vcs.revision=${revision} vcs.modified=false CGO_ENABLED=0` :
        name === 'lsp-version' ? 'fairyc-lsp 0.1.0\n' : name === 'lsp-protocol' ? probeOutput : '' })),
    preview_candidate_archive: relative(repo, archive), archive_bytes: statSync(archive).size,
    archive_sha256: hash(archive), executable_bytes: statSync(image).size, executable_sha256: hash(image),
    preview_checksums_sha256: hash(sums), lsp_sha256: hash(server), lsp_bytes: statSync(server).size };
  return { repo, root, extracted, archive, sums, input, receipt: join(root, 'receipt.json') };
}

for (const [name, edit, rejection] of [
  ['valid retained candidate', () => {}, null],
  ['valid legacy compiler-only candidate', f => {
    f.input.schema = 'fairyc.local-starter-smoke/v1';
    f.input.steps = f.input.steps.filter(step => !step.name.startsWith('lsp-'));
    rmSync(join(f.extracted, 'fairyc-lsp.exe'));
    writeFileSync(f.sums, files.filter(name => name !== 'fairyc-lsp.exe').map(name => `${hash(join(f.extracted, name))}  ${name}\n`).join(''));
    f.input.preview_checksums_sha256 = hash(f.sums);
    delete f.input.lsp_sha256;
    delete f.input.lsp_bytes;
  }, null],
  ['different revision', f => { f.input.vcs_revision = 'b'.repeat(40); }, /source revision differs/u],
  ['dirty compiler', f => { f.input.vcs_modified = true; }, /modified source/u],
  ['missing native step', f => { f.input.steps = f.input.steps.filter(step => step.name !== 'native-test'); }, /starter checks/u],
  ['failed native step', f => { f.input.steps.find(step => step.name === 'native-test').exit = 1; }, /starter check failed/u],
  ['missing LSP build step', f => { f.input.steps = f.input.steps.filter(step => step.name !== 'lsp-build'); }, /starter checks/u],
  ['different LSP source revision', f => { f.input.steps.find(step => step.name === 'lsp-build-metadata').stdout = 'go1.27.1 vcs.revision=wrong'; }, /LSP build identity differs/u],
  ['failed LSP protocol', f => { f.input.steps.find(step => step.name === 'lsp-protocol').stdout = frame({ id: 1, error: { code: -32603 } }) + frame({ id: 2, result: null }); }, /LSP probe request failed/u],
  ['changed LSP', f => { writeFileSync(join(f.extracted, 'fairyc-lsp.exe'), 'changed\n'); }, /preview file hash differs/u],
  ['missing LSP', f => { rmSync(join(f.extracted, 'fairyc-lsp.exe')); }, /unexpected preview contents/u],
  ['wrong LSP size', f => { f.input.lsp_bytes++; }, /LSP size differs/u],
  ['missing LSP hash', f => { delete f.input.lsp_sha256; }, /LSP hash differs/u],
  ['changed archive', f => { writeFileSync(f.archive, 'corrupted\n'); }, /archive size differs/u],
  ['changed checksum sidecar', f => { writeFileSync(f.archive + '.sha256', 'changed\n'); }, /archive checksum differs/u],
  ['changed compiler', f => { writeFileSync(join(f.extracted, 'fairyc.exe'), 'changed\n'); }, /preview file hash differs/u],
  ['missing checksum identity', f => { delete f.input.preview_checksums_sha256; }, /checksums differ/u],
  ['rewritten license and checksums', f => {
    writeFileSync(join(f.extracted, 'LICENSE'), 'changed license\n');
    writeFileSync(f.sums, files.map(name => `${hash(join(f.extracted, name))}  ${name}\n`).join(''));
  }, /checksums differ/u],
  ['unlisted file', f => { writeFileSync(join(f.extracted, 'extra.txt'), 'extra'); }, /unexpected preview contents/u],
  ['archive outside candidate', f => { f.input.preview_candidate_archive = '../elsewhere.zip'; }, /receipt-owned/u],
]) {
  test(name, async () => {
    const f = fixture();
    try {
      edit(f);
      writeFileSync(f.receipt, JSON.stringify(f.input));
      const options = { repo: f.repo, receipt: f.receipt, revision };
      if (rejection) await assert.rejects(verifyCandidate(options), rejection);
      else {
        const before = hash(f.receipt);
        const result = await verifyCandidate(options);
        assert.equal(result.passed, true);
        assert.equal(result.starterSteps, f.input.schema.endsWith('/v2') ? 16 : 12);
        assert.equal(result.verifiedFiles, f.input.schema.endsWith('/v2') ? 7 : 6);
        assert.equal(result.lspSHA256, f.input.lsp_sha256);
        assert.equal(result.cleanMachineEvidence, false);
        assert.equal(result.publicationApproved, false);
        assert.equal(hash(f.receipt), before, 'recheck modified the retained receipt');
      }
    } finally {
      assert.ok(f.root.startsWith(realpathSync(f.repo) + sep));
      assert.equal(realpathSync(f.repo), f.repo);
      rmSync(f.repo, { recursive: true });
    }
  });
}

test('LSP probe uses byte lengths and rejects incomplete responses', () => {
  const input = lspProbeInput('file:///C:/프리뷰');
  assert.ok(input.includes('"method":"initialize"') && input.includes('"method":"exit"'));
  const bytes = Buffer.from(input);
  const headerEnd = bytes.indexOf('\r\n\r\n');
  const length = Number(bytes.subarray(0, headerEnd).toString().split(': ')[1]);
  assert.equal(JSON.parse(bytes.subarray(headerEnd + 4, headerEnd + 4 + length).toString()).params.rootUri, 'file:///C:/프리뷰');
  verifyLspProbe(probeOutput);
  assert.throws(() => verifyLspProbe(probeOutput.slice(0, -1)), /incomplete LSP body/u);
});
