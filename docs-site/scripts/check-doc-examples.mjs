import { spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { docExamples } from '../src/config/doc-examples.mjs';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const docsSiteDir = resolve(scriptDir, '..');
const repoRoot = resolve(docsSiteDir, '..');

const byId = new Map(docExamples.map((entry) => [entry.id, entry.source]));
const requireSnippet = (id) => {
	const value = byId.get(id);
	if (!value) {
		throw new Error(`missing canonical doc example "${id}"`);
	}
	return value;
};

const resolveCompilerExecutable = (tempRoot) => {
	const envBin = process.env.FAIRYC_BIN?.trim();
	if (envBin) {
		return resolve(envBin);
	}

	const exeName = process.platform === 'win32' ? 'fairyc.exe' : 'fairyc';
	const builtExe = join(repoRoot, 'build', exeName);
	if (existsSync(builtExe)) {
		return builtExe;
	}

	const tempExe = join(tempRoot, exeName);
	const runGoScript = join(repoRoot, 'tools', 'run-go.ts');
	const buildResult = spawnSync(
		'bun',
		[runGoScript, 'build', '-mod=readonly', '-trimpath', '-o', tempExe, './cmd/fairyc'],
		{ cwd: repoRoot, encoding: 'utf8' }
	);
	if (buildResult.error) {
		throw new Error(`failed to build fairyc for doc check: ${buildResult.error.message}`);
	}
	if (buildResult.status !== 0) {
		throw new Error(
			`failed to build fairyc for doc check (exit ${buildResult.status}):\n${buildResult.stdout ?? ''}${buildResult.stderr ?? ''}`
		);
	}
	return tempExe;
};

const runInWorkspace = (compilerExe, workspaceDir, args, label) => {
	const result = spawnSync(compilerExe, args, { cwd: workspaceDir, encoding: 'utf8' });
	if (result.error) {
		throw new Error(`${label}: failed to execute compiler: ${result.error.message}`);
	}
	if (result.status !== 0) {
		throw new Error(
			`${label}: fairyc ${args.join(' ')} exited with code ${result.status}\n${result.stdout ?? ''}${result.stderr ?? ''}`
		);
	}
	return result.stdout.trim();
};

const writeFiles = (baseDir, files) => {
	for (const [relPath, content] of Object.entries(files)) {
		const fullPath = join(baseDir, relPath);
		mkdirSync(dirname(fullPath), { recursive: true });
		writeFileSync(fullPath, `${content.trim()}\n`, 'utf8');
	}
};

const tempRoot = mkdtempSync(join(tmpdir(), 'fairyc-doc-examples-'));

try {
	const compilerExe = resolveCompilerExecutable(tempRoot);

	// 1. Validate starter snippets from start/first-program.md
	const starterDir = join(tempRoot, 'hello-fairy');
	writeFiles(starterDir, {
		'fairy.toml': requireSnippet('first-program-manifest'),
		'src/main.fy': requireSnippet('first-program-main'),
		'tests/main.fy': requireSnippet('first-program-test'),
	});
	runInWorkspace(compilerExe, starterDir, ['check'], 'starter check');
	runInWorkspace(compilerExe, starterDir, ['test', '--locked'], 'starter test');

	// 2. Validate multi-package manifest and language/memory snippets together
	const projectDir = join(tempRoot, 'my-app');
	writeFiles(projectDir, {
		'fairy.toml': requireSnippet('modules-manifest'),
		'src/main.fy': requireSnippet('modules-main'),
		'src/app/math.fy': requireSnippet('modules-app-math'),
		'src/doc_closure.fy': `module doc_closure;\n\n${requireSnippet('control-flow-closure')}`,
		'src/doc_errors.fy': [
			'module doc_errors;',
			'',
			requireSnippet('structs-parse-error-enum'),
			'',
			requireSnippet('structs-parse-error-switch'),
			'',
			requireSnippet('structs-add-one-result'),
		].join('\n'),
		'src/doc_region.fy': `module doc_region;\n\n${requireSnippet('memory-region-alloc')}`,
		'libs/answer/fairy.toml': `schema = "fairyc.manifest/v1"

[package]
uid = "example.answer"
name = "answer"
version = "0.1.0"
language = "fairyc-language/20"
source = "src"`,
		'libs/answer/src/answer.fy': `module answer;

pub fn i32 value() {
    return 77;
}`,
	});
	runInWorkspace(compilerExe, projectDir, ['check'], 'language & manifest check');

	console.log('doc examples ok (starter check + test, language & manifest workspace check)');
} finally {
	rmSync(tempRoot, { recursive: true, force: true });
}
