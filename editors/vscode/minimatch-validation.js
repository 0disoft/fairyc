const { createRequire } = require('node:module');

// languageclient 9 uses makeRe() only as a truthy pattern-validity check in
// fileOperations, diagnostic and notebook, then calls match() for matching.
// minimatch 11 deliberately removed regex generation. Keep those three checks
// working with its parsed set, without recreating the unsafe regex backend.
// Resolve from the client so the packaging tool's separate copy is untouched.
function prepareLanguageClientPatterns() {
  const requireClient = createRequire(require.resolve('vscode-languageclient/package.json'));
  const { Minimatch } = requireClient('minimatch');
  if (typeof Minimatch.prototype.makeRe !== 'function') {
    Object.defineProperty(Minimatch.prototype, 'makeRe', {
      value() { return this.set.length > 0; }
    });
  }
  return Minimatch;
}

module.exports = { prepareLanguageClientPatterns };
