const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { createHash } = require('node:crypto');
const { createRequire } = require('node:module');
const vscode = require('vscode');

async function eventually(read, accept, label) {
  const deadline = Date.now() + 20000;
  while (Date.now() < deadline) {
    const value = await read();
    if (accept(value)) return value;
    await new Promise(resolve => setTimeout(resolve, 150));
  }
  throw new Error(`Timed out waiting for ${label}`);
}

exports.run = async function () {
  const workspace = vscode.workspace.workspaceFolders[0].uri.fsPath;
  const receiptPath = path.join(workspace, '..', 'host-receipt.json');
  try {
    const extension = vscode.extensions.getExtension('0disoft.fairyc');
    assert(extension, 'Installed Fairy C extension is missing');
    const relative = path.relative(path.join(workspace, '..', 'extensions'), extension.extensionPath);
    assert(!relative.startsWith('..') && !path.isAbsolute(relative), 'Extension must load from the isolated installed directory');
    const uri = vscode.Uri.file(path.join(workspace, 'main.fy'));
    const original = fs.readFileSync(uri.fsPath, 'utf8');
    const document = await vscode.workspace.openTextDocument(uri);
    assert.equal(document.languageId, 'fairyc');
    assert.equal(vscode.workspace.isTrusted, true);
    const serverPath = vscode.workspace.getConfiguration('fairyc').get('server.path');
    assert(path.isAbsolute(serverPath), 'Language server path must be absolute');
    const serverSHA256 = createHash('sha256').update(fs.readFileSync(serverPath)).digest('hex');
    await vscode.window.showTextDocument(document);
    await extension.activate();
    assert.equal(extension.isActive, true);

    // Exercise real client consumers, including their legacy makeRe guards.
    // Ordinary definition/completion alone would miss the overridden API break.
    const requireExtension = createRequire(path.join(extension.extensionPath, 'package.json'));
    // The test loader has a separate module cache from VS Code's extension
    // loader. Apply the same installed startup adapter to this test copy too.
    requireExtension('./minimatch-validation').prepareLanguageClientPatterns();
    const requireClient = createRequire(requireExtension.resolve('vscode-languageclient/package.json'));
    const { DidCreateFilesFeature } = requireClient('./lib/common/fileOperations');
    const feature = new DidCreateFilesFeature({code2ProtocolConverter: {asDidCreateFilesParams() {}}});
    try {
      feature.register({id: 'glob-compat', registerOptions: {
        filters: [{scheme: 'file', pattern: {glob: '**/*.fy'}}]
      }});
      const ignored = vscode.Uri.file(path.join(workspace, 'notes.txt'));
      const filtered = await feature.filter({files: [uri, ignored]}, item => item);
      assert.deepEqual(filtered.files, [uri]);
    } finally {
      feature.clear();
    }
    const { NotebookDocumentSyncFeature } = requireClient('./lib/common/notebook');
    const notebooks = new NotebookDocumentSyncFeature({
      clientOptions: {}, protocol2CodeConverter: {asDocumentSelector() { return []; }}
    });
    try {
      notebooks.register({id: 'glob-compat', registerOptions: {
        notebookSelector: [{notebook: {pattern: '**/*.fy'}}]
      }});
      const provider = notebooks.registrations.get('glob-compat');
      const cells = [{document: {languageId: 'fairyc'}}];
      // Notebook filters keep dot=false; the harness itself lives in .cache.
      // These synthetic notebook URIs need no backing notebook editor or file.
      const notebookURI = vscode.Uri.file(path.join(path.parse(workspace).root, 'fairyc-glob-smoke', 'main.fy'));
      assert.deepEqual(provider.getMatchingCells({uri: notebookURI, getCells: () => cells}), cells);
      assert.equal(provider.getMatchingCells({uri, getCells: () => cells}), undefined);
      const ignored = vscode.Uri.file(path.join(path.dirname(notebookURI.fsPath), 'notes.txt'));
      assert.equal(provider.getMatchingCells({uri: ignored, getCells: () => cells}), undefined);
    } finally {
      notebooks.clear();
    }

    const character = document.lineAt(2).text.indexOf('helper');
    const definitions = await eventually(
      () => vscode.commands.executeCommand('vscode.executeDefinitionProvider', uri, new vscode.Position(2, character + 1)),
      values => values?.some(value => value.range?.start.line === 1), 'definition');
    const completion = await eventually(
      () => vscode.commands.executeCommand('vscode.executeCompletionItemProvider', uri, new vscode.Position(2, character + 3)),
      value => value?.items.some(item => (typeof item.label === 'string' ? item.label : item.label.label) === 'helper'), 'completion');

    async function replace(text) {
      const edit = new vscode.WorkspaceEdit();
      edit.replace(uri, new vscode.Range(document.positionAt(0), document.positionAt(document.getText().length)), text);
      assert.equal(await vscode.workspace.applyEdit(edit), true);
    }
    await replace(original.replace('return 7', 'return false'));
    await eventually(() => vscode.languages.getDiagnostics(uri), values => values.some(value => value.severity === vscode.DiagnosticSeverity.Error), 'unsaved diagnostics');
    await replace(original);
    await eventually(() => vscode.languages.getDiagnostics(uri), values => values.length === 0, 'diagnostic recovery');
    assert.equal(fs.readFileSync(uri.fsPath, 'utf8'), original);
    fs.writeFileSync(receiptPath, JSON.stringify({passed: true, vscode: vscode.version,
      extension: extension.id, version: extension.packageJSON.version, extensionPath: extension.extensionPath,
      serverPath, serverSHA256, trustedWorkspace: vscode.workspace.isTrusted,
      languageId: document.languageId, definitionCount: definitions.length,
      completionCount: completion.items.length, unsavedDiagnosticsAndRecovery: true,
      clientGlobCompatibility: true}, null, 2));
    console.log('Installed Fairy C: .fy association, activation, definition, completion, unsaved diagnostics and recovery passed');
  } catch (error) {
    fs.writeFileSync(receiptPath, JSON.stringify({passed: false, error: String(error.stack || error)}, null, 2));
    throw error;
  }
};
