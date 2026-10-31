import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
import test from 'node:test';

const requireExtension = createRequire(new URL('../editors/vscode/package.json', import.meta.url));
const requireClient = createRequire(requireExtension.resolve('vscode-languageclient/package.json'));
const requirePackager = createRequire(requireExtension.resolve('@vscode/vsce/package.json'));
const {prepareLanguageClientPatterns} = requireExtension('./minimatch-validation');

test('both minimatch consumers resolve a patched supported major', () => {
  for (const consumer of [requireClient, requirePackager]) {
    const [major, minor, patch] = consumer('minimatch/package.json').version.split('.').map(Number);
    assert.equal(major, 11);
    assert(minor > 0 || patch >= 1, 'minimatch must include the amplification fix');
  }
});

test('legacy client validation retains matching without regex generation', () => {
  const Minimatch = prepareLanguageClientPatterns();
  assert.equal(prepareLanguageClientPatterns(), Minimatch, 'preparation must be repeatable');
  const source = new Minimatch('**/{*.fy,fairy.toml}', {noext: true});
  assert.equal(source.makeRe(), true);
  for (const name of ['main.fy', 'src/main.fy', 'fairy.toml', 'client/fairy.toml']) {
    assert.equal(source.match(name), true, name);
  }
  for (const name of ['main.js', 'src/main.fy.bak', '.cache/main.fy']) {
    assert.equal(source.match(name), false, name);
  }
  const folders = new Minimatch('**/src/**', {nocase: true, dot: true});
  assert.equal(folders.makeRe(), true);
  assert.equal(folders.match('C:/project/SRC/.hidden/module.fy'), true);
  assert.equal(new Minimatch('#not-a-pattern').makeRe(), false);
  assert.equal(new Minimatch('').match(''), true);
  assert.equal(typeof source.makeRe(), 'boolean', 'never regenerate legacy regular expressions');
});

test('packager named matching API retains ignore and include behavior', () => {
  const {minimatch, Minimatch} = requirePackager('minimatch');
  assert.equal(typeof Minimatch.prototype.makeRe, 'undefined', 'client adapter must not affect the packager copy');
  assert.equal(minimatch('node_modules/vscode-textmate/release/main.js', 'node_modules/vscode-textmate/**', {dot: true}), true);
  assert.equal(minimatch('node_modules/vscode-languageclient/lib/node/main.js', 'node_modules/vscode-textmate/**', {dot: true}), false);
  assert.equal(minimatch('.vscodeignore', '**', {dot: true}), true);
});
