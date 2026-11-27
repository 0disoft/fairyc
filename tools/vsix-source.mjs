import {execFileSync, spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
import {once} from 'node:events';
import {lstatSync, readFileSync, readlinkSync} from 'node:fs';
import path from 'node:path';
import {pipeline} from 'node:stream/promises';

const hash = bytes => createHash('sha256').update(bytes).digest('hex');

async function hashTrackedDiff(root) {
  const digest = createHash('sha256');
  const git = spawn('git', ['diff', '--binary', '--no-ext-diff', '--no-textconv', 'HEAD', '--'], {
    cwd: root, windowsHide: true, stdio: ['ignore', 'pipe', 'inherit'],
  });
  const [[code, signal]] = await Promise.all([
    once(git, 'close'),
    pipeline(git.stdout, async chunks => {
      for await (const chunk of chunks) digest.update(chunk);
    }),
  ]);
  if (code !== 0) throw new Error(`git diff failed (${signal ?? `exit ${code}`})`);
  return digest.digest('hex');
}

// Include dirty contents, not just file names: edits can change while status stays M.
export async function snapshotSource(root) {
  const git = args => execFileSync('git', args, {cwd: root, windowsHide: true});
  const status = git(['status', '--porcelain=v1', '-z', '--untracked-files=all']);
  const untracked = git(['ls-files', '--others', '--exclude-standard', '-z']).toString('utf8');
  return {
    revision: git(['rev-parse', '--verify', 'HEAD']).toString('utf8').trim(),
    modified: status.length !== 0,
    statusSHA256: hash(status),
    trackedDiffSHA256: await hashTrackedDiff(root),
    untrackedFiles: untracked.split('\0').filter(Boolean).sort().map(name => {
      const file = path.join(root, name);
      const bytes = lstatSync(file).isSymbolicLink() ? readlinkSync(file, {encoding: 'buffer'}) : readFileSync(file);
      return {path: name, sha256: hash(bytes)};
    }),
  };
}
