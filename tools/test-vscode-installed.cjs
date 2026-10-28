const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const { createHash } = require('node:crypto');
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
      completionCount: completion.items.length, unsavedDiagnosticsAndRecovery: true}, null, 2));
    console.log('Installed Fairy C: .fy association, activation, definition, completion, unsaved diagnostics and recovery passed');
  } catch (error) {
    fs.writeFileSync(receiptPath, JSON.stringify({passed: false, error: String(error.stack || error)}, null, 2));
    throw error;
  }
};
