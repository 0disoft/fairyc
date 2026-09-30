const vscode = require('vscode');
const { LanguageClient } = require('vscode-languageclient/node');
let client;
async function activate(context) {
  if (!vscode.workspace.isTrusted) {
    context.subscriptions.push(vscode.workspace.onDidGrantWorkspaceTrust(() => start(context)));
    return;
  }
  await start(context);
}
async function start(context) {
  if (client) return;
  const command = vscode.workspace.getConfiguration('fairyc').get('server.path', 'fairyc-lsp');
  const watcher = vscode.workspace.createFileSystemWatcher('**/{*.fy,fairy.toml}');
  context.subscriptions.push(watcher);
  client = new LanguageClient('fairyc', 'Fairy C', { command, args: [] }, {
    documentSelector: [{ scheme: 'file', language: 'fairyc' }],
    synchronize: { fileEvents: watcher },
    workspaceFolder: vscode.workspace.workspaceFolders?.[0]
  });
  try { await client.start(); }
  catch (error) { client = undefined; vscode.window.showErrorMessage('Fairy C language server could not start. Check fairyc.server.path.'); }
}
async function deactivate() { if (client) { await client.stop(); client = undefined; } }
module.exports = { activate, deactivate };
