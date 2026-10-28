import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { createReadStream, lstatSync, readFileSync, readdirSync, realpathSync } from 'node:fs';
import { basename, dirname, join, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyLspProbe } from './lsp-probe.mjs';

const legacySteps = ['build', 'build-metadata', 'go-root', 'zip-roundtrip', 'new', 'check',
  'check-locked', 'test', 'native-test', 'native-test-release', 'run', 'run-release'];
const legacyFiles = ['fairyc.exe', 'README.txt', 'LICENSE', 'LICENSING.md', 'THIRD_PARTY_NOTICES.md', 'GO-LICENSE.txt'];
const archiveName = 'fairyc-windows-amd64-preview-candidate.zip';

async function sha(path) {
  const digest = createHash('sha256');
  for await (const chunk of createReadStream(path)) digest.update(chunk);
  return digest.digest('hex');
}

function regular(path) {
  const info = lstatSync(path);
  assert.ok(info.isFile() && !info.isSymbolicLink(), `not a regular file: ${path}`);
  return info;
}

// Recheck retained local evidence and bytes; never run a compiler, publish, or
// substitute this check for exact-commit CI or a separate clean-machine run.
export async function verifyCandidate({ repo, receipt, revision }) {
  assert.match(revision, /^[0-9a-f]{40}$/u, 'expected full source revision');
  const cache = realpathSync(join(repo, '.cache'));
  receipt = realpathSync(resolve(repo, receipt));
  assert.ok(receipt.startsWith(cache + sep), 'receipt must be inside the repository cache');
  assert.ok(regular(receipt).size <= 64 * 1024, 'oversized receipt');
  const input = JSON.parse(readFileSync(receipt, 'utf8'));
  assert.ok(['fairyc.local-starter-smoke/v1', 'fairyc.local-starter-smoke/v2'].includes(input.schema), 'unsupported smoke schema');
  const hasLsp = input.schema === 'fairyc.local-starter-smoke/v2';
  const files = hasLsp ? [...legacyFiles, 'fairyc-lsp.exe'] : legacyFiles;
  const steps = hasLsp ? [...legacySteps.slice(0, 2), 'lsp-build', 'lsp-build-metadata',
    ...legacySteps.slice(2, 4), 'lsp-version', 'lsp-protocol', ...legacySteps.slice(4)] : legacySteps;
  assert.equal(input.platform, 'win32/x64');
  assert.equal(input.vcs_modified, false, 'candidate was built from modified source');
  assert.equal(input.vcs_revision, revision, 'candidate source revision differs');
  assert.deepEqual(input.build_flags, ['-trimpath', '-ldflags=-s -w']);
  assert.deepEqual(input.steps.map(step => step.name), steps, 'incomplete or reordered starter checks');
  assert.ok(input.steps.every(step => step.exit === 0), 'starter check failed');
  if (hasLsp) {
    const metadata = input.steps.find(step => step.name === 'lsp-build-metadata').stdout;
    for (const identity of [input.go_version, `vcs.revision=${revision}`, 'vcs.modified=false', 'CGO_ENABLED=0']) {
      assert.ok(typeof identity === 'string' && typeof metadata === 'string' && metadata.includes(identity), 'LSP build identity differs');
    }
    assert.match(input.steps.find(step => step.name === 'lsp-version').stdout, /^fairyc-lsp \d+\.\d+\.\d+\r?\n$/u);
    verifyLspProbe(input.steps.find(step => step.name === 'lsp-protocol').stdout);
  }
  const root = dirname(receipt);
  const archive = resolve(repo, input.preview_candidate_archive);
  assert.equal(archive, join(root, archiveName), 'archive is not the receipt-owned candidate');
  assert.equal(regular(archive).size, input.archive_bytes, 'archive size differs');
  assert.equal(await sha(archive), input.archive_sha256, 'archive hash differs');
  regular(archive + '.sha256');
  assert.equal(readFileSync(archive + '.sha256', 'utf8'), `${input.archive_sha256}  ${basename(archive)}\n`, 'archive checksum differs');
  const extracted = join(root, 'extracted');
  assert.equal(realpathSync(extracted), extracted, 'extracted directory is redirected');
  assert.deepEqual(readdirSync(extracted).sort(), [...files, 'SHA256SUMS.txt'].sort(), 'unexpected preview contents');
  const sums = join(extracted, 'SHA256SUMS.txt');
  assert.ok(regular(sums).size <= 4096, 'oversized checksum file');
  assert.equal(await sha(sums), input.preview_checksums_sha256, 'preview checksums differ from the smoke receipt');
  const checked = new Set();
  for (const line of readFileSync(sums, 'utf8').trimEnd().split('\n')) {
    const match = /^([0-9a-f]{64})  ([A-Za-z0-9_.-]+)$/u.exec(line);
    assert.ok(match && files.includes(match[2]) && !checked.has(match[2]), 'invalid or duplicate checksum entry');
    const path = join(extracted, match[2]);
    regular(path);
    assert.equal(await sha(path), match[1], `preview file hash differs: ${match[2]}`);
    checked.add(match[2]);
  }
  assert.equal(checked.size, files.length, 'missing preview checksum');
  const executable = join(extracted, 'fairyc.exe');
  assert.equal(regular(executable).size, input.executable_bytes, 'compiler size differs');
  assert.equal(await sha(executable), input.executable_sha256, 'compiler hash differs from the smoke receipt');
  if (hasLsp) {
    const server = join(extracted, 'fairyc-lsp.exe');
    assert.equal(regular(server).size, input.lsp_bytes, 'LSP size differs');
    assert.equal(await sha(server), input.lsp_sha256, 'LSP hash differs from the smoke receipt');
  }
  return { schema: 'fairyc.windows-preview-recheck/v1', passed: true, sourceRevision: revision,
    archive, archiveSHA256: input.archive_sha256, executableSHA256: input.executable_sha256,
    ...(hasLsp ? { lspSHA256: input.lsp_sha256 } : {}),
    starterSteps: steps.length, verifiedFiles: checked.size, cleanMachineEvidence: false, publicationApproved: false };
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const args = process.argv.slice(2);
    assert.ok(args.length === 4 && args[0] === '--receipt' && args[2] === '--revision',
      'usage: node tools/windows-preview/verify.mjs --receipt <receipt.json> --revision <full-commit>');
    console.log(JSON.stringify(await verifyCandidate({ repo: resolve(import.meta.dirname, '../..'), receipt: args[1], revision: args[3] }), null, 2));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
