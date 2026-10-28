import assert from 'node:assert/strict';

export function lspProbeInput(rootUri) {
  return [
    { id: 1, method: 'initialize', params: { processId: null, rootUri, capabilities: {} } },
    { method: 'initialized', params: {} },
    { id: 2, method: 'shutdown', params: null },
    { method: 'exit' },
  ].map(message => {
    const body = JSON.stringify({ jsonrpc: '2.0', ...message });
    return `Content-Length: ${Buffer.byteLength(body)}\r\n\r\n${body}`;
  }).join('');
}

export function verifyLspProbe(output) {
  let data = Buffer.from(output);
  const messages = [];
  while (data.length) {
    const end = data.indexOf('\r\n\r\n');
    assert.ok(end >= 0, 'incomplete LSP header');
    const match = /^Content-Length: (\d+)$/iu.exec(data.subarray(0, end).toString());
    assert.ok(match, 'invalid LSP header');
    const length = Number(match[1]);
    const start = end + 4;
    assert.ok(data.length >= start + length, 'incomplete LSP body');
    messages.push(JSON.parse(data.subarray(start, start + length).toString()));
    data = data.subarray(start + length);
  }
  assert.deepEqual(messages.map(message => [message.jsonrpc, message.id]), [['2.0', 1], ['2.0', 2]]);
  assert.ok(messages.every(message => !Object.hasOwn(message, 'error')), 'LSP probe request failed');
  assert.equal(messages[0].result.serverInfo.name, 'fairyc-lsp');
  assert.equal(messages[0].result.capabilities.positionEncoding, 'utf-16');
  assert.equal(messages[0].result.capabilities.definitionProvider, true);
  assert.ok(messages[0].result.capabilities.completionProvider);
  assert.equal(messages[1].result, null);
}
