// Prepare a local syntax-only preview. This does not install or launch anything.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {spawnSync} from 'node:child_process';

if (process.argv.length !== 2) throw Error('This helper accepts no arguments');
const root = fileURLToPath(new URL('../../', import.meta.url));
const git = args => {
  const result = spawnSync('git', args, {cwd: root, encoding: 'utf8', shell: false, timeout: 10000});
  if (result.status !== 0) throw Error(result.stderr || result.error || 'Git read failed');
  return result.stdout.trim();
};
const parserPaths = ['editors/tree-sitter-fairyc/grammar.js', 'editors/tree-sitter-fairyc/src'];
if (git(['diff', '--name-only', 'HEAD', '--', ...parserPaths])) {
  throw Error('Commit grammar/parser changes before preparing the pinned preview');
}
const revision = git(['rev-parse', 'HEAD']);
const output = path.join(root, '.cache/zed-syntax-dev');
const source = path.join(root, 'editors/zed');
let manifest = fs.readFileSync(path.join(source, 'extension.toml'), 'utf8');
manifest = manifest.split('[language_servers.fairyc-lsp]')[0];
manifest = manifest.replace(/(\[grammars\.fairyc\]\s*repository = )"[^"]+"/, `$1"${pathToFileURL(root).href.replace(/\/$/, '')}"`);
manifest = manifest.replace(/^rev = "[^"]+"/m, `rev = "${revision}"`);
fs.mkdirSync(path.join(output, 'languages/fairyc'), {recursive: true});
fs.writeFileSync(path.join(output, 'extension.toml'), manifest);
for (const name of ['config.toml', 'highlights.scm', 'brackets.scm', 'indents.scm', 'outline.scm']) {
  fs.copyFileSync(path.join(source, 'languages/fairyc', name), path.join(output, 'languages/fairyc', name));
}
console.log(`Syntax-only Zed dev extension: ${output}\nLocal grammar revision: ${revision}\nNo Rust/LSP adapter included; install explicitly through Zed.`);
