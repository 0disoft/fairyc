import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import fs from 'node:fs/promises';
import {createRequire} from 'node:module';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {snapshotSource} from './vsix-source.mjs';

const root = fileURLToPath(new URL('../', import.meta.url));
const extensionRoot = path.join(root, 'editors', 'vscode');
const require = createRequire(path.join(extensionRoot, 'package.json'));
const {createVSIX} = require('@vscode/vsce');
const source = snapshotSource(root);
const lockfile = await fs.readFile(path.join(extensionRoot, 'package-lock.json'));
const manifest = JSON.parse(await fs.readFile(path.join(extensionRoot, 'package.json'), 'utf8'));
assert.equal(manifest.license, 'Apache-2.0');
assert.deepEqual(await fs.readFile(path.join(extensionRoot, 'LICENSE')), await fs.readFile(path.join(root, 'LICENSE')));
const releaseRoot = path.join(root, '.cache', 'editor-release');
await fs.mkdir(releaseRoot, {recursive: true});
const outputRoot = await fs.mkdtemp(path.join(releaseRoot, 'candidate-'));
const filename = `${manifest.name}-${manifest.version}.vsix`;
const packagePath = path.join(outputRoot, filename);
await createVSIX({cwd: extensionRoot, packagePath, dependencies: true});
assert.deepEqual(snapshotSource(root), source, 'Source changed during packaging; discard this candidate');
assert.deepEqual(await fs.readFile(path.join(extensionRoot, 'package-lock.json')), lockfile,
  'Dependency lockfile changed during packaging; discard this candidate');
const digest = createHash('sha256').update(await fs.readFile(packagePath)).digest('hex');
const build = {
  schema: 'fairyc.vsix-build/v1',
  source,
  extension: {id: `${manifest.publisher}.${manifest.name}`, version: manifest.version},
  toolchain: {node: process.version, vsce: require('@vscode/vsce/package.json').version,
    lockfileSHA256: createHash('sha256').update(lockfile).digest('hex')},
  artifact: {name: filename, bytes: (await fs.stat(packagePath)).size, sha256: digest},
};
await fs.writeFile(`${packagePath}.build.json`, JSON.stringify(build, null, 2) + '\n', {flag: 'wx'});
await fs.writeFile(`${packagePath}.sha256`, `${digest}  ${filename}\n`, {flag: 'wx'});
console.log(`VSIX: ${packagePath}\nSHA-256: ${digest}\nBuild receipt: ${packagePath}.build.json`);
