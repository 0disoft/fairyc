import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtempSync, mkdirSync, readFileSync, realpathSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, relative, sep } from 'node:path';
import test from 'node:test';
import { verifyCandidate } from './verify.mjs';

const revision = 'a'.repeat(40);
const archiveName = 'fairyc-windows-amd64-preview-candidate.zip';
const files = ['fairyc.exe', 'README.txt', 'LICENSE', 'LICENSING.md', 'THIRD_PARTY_NOTICES.md', 'GO-LICENSE.txt'];
const hash = path => createHash('sha256').update(readFileSync(path)).digest('hex');

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
  const input = { schema: 'fairyc.local-starter-smoke/v1', platform: 'win32/x64', vcs_modified: false,
    vcs_revision: revision, build_flags: ['-trimpath', '-ldflags=-s -w'],
    steps: ['build', 'build-metadata', 'go-root', 'zip-roundtrip', 'new', 'check', 'check-locked',
      'test', 'native-test', 'native-test-release', 'run', 'run-release'].map(name => ({ name, exit: 0 })),
    preview_candidate_archive: relative(repo, archive), archive_bytes: statSync(archive).size,
    archive_sha256: hash(archive), executable_bytes: statSync(image).size, executable_sha256: hash(image),
    preview_checksums_sha256: hash(sums) };
  return { repo, root, extracted, archive, sums, input, receipt: join(root, 'receipt.json') };
}

for (const [name, edit, rejection] of [
  ['valid retained candidate', () => {}, null],
  ['different revision', f => { f.input.vcs_revision = 'b'.repeat(40); }, /source revision differs/u],
  ['dirty compiler', f => { f.input.vcs_modified = true; }, /modified source/u],
  ['missing native step', f => { f.input.steps.splice(8, 1); }, /starter checks/u],
  ['failed native step', f => { f.input.steps[8].exit = 1; }, /starter check failed/u],
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
        assert.equal(result.starterSteps, 12);
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
