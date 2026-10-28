import {execFileSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {lstatSync, readFileSync, readlinkSync} from 'node:fs';
import path from 'node:path';

const hash = bytes => createHash('sha256').update(bytes).digest('hex');

// Include dirty contents, not just file names: edits can change while status stays M.
export function snapshotSource(root) {
  const git = args => execFileSync('git', args, {cwd: root, windowsHide: true});
  const status = git(['status', '--porcelain=v1', '-z', '--untracked-files=all']);
  const untracked = git(['ls-files', '--others', '--exclude-standard', '-z']).toString('utf8');
  return {
    revision: git(['rev-parse', '--verify', 'HEAD']).toString('utf8').trim(),
    modified: status.length !== 0,
    statusSHA256: hash(status),
    trackedDiffSHA256: hash(git(['diff', '--binary', '--no-ext-diff', '--no-textconv', 'HEAD', '--'])),
    untrackedFiles: untracked.split('\0').filter(Boolean).sort().map(name => {
      const file = path.join(root, name);
      const bytes = lstatSync(file).isSymbolicLink() ? readlinkSync(file, {encoding: 'buffer'}) : readFileSync(file);
      return {path: name, sha256: hash(bytes)};
    }),
  };
}
