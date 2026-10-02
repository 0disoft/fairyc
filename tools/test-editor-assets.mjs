import assert from 'node:assert/strict';
import fs from 'node:fs';
import {createRequire} from 'node:module';
const require=createRequire(new URL('../editors/vscode/package.json',import.meta.url));
const {Registry,parseRawGrammar}=require('vscode-textmate');
const onig=require('vscode-oniguruma');
await onig.loadWASM(fs.readFileSync(require.resolve('vscode-oniguruma/release/onig.wasm')));
const registry=new Registry({onigLib:Promise.resolve({createOnigScanner:p=>new onig.OnigScanner(p),createOnigString:s=>new onig.OnigString(s)}),
 loadGrammar:async scope=>scope==='source.fairyc'?parseRawGrammar(fs.readFileSync('syntaxes/fairyc.tmLanguage.json','utf8'),'fairyc.json'):null});
const grammar=await registry.loadGrammar('source.fairyc');
let stack=null;
function tokens(line){const r=grammar.tokenizeLine(line,stack);stack=r.ruleStack;return r.tokens.map(t=>({text:line.slice(t.startIndex,t.endIndex),scopes:t.scopes.join(' ')}));}
const has=(ts,text,scope)=>assert(ts.some(t=>t.text.includes(text)&&t.scopes.includes(scope)),JSON.stringify(ts));
function scopeAt(line, text, scope, expected=true) {
 stack=null;
 const at=line.indexOf(text);
 assert(at>=0,'Missing fixture text '+text);
 const result=grammar.tokenizeLine(line,stack);
 const token=result.tokens.find(t=>t.startIndex<=at&&at<t.endIndex);
 assert(token,'Missing token '+text);
 assert.equal(token.scopes.some(s=>s.startsWith(scope)),expected,JSON.stringify({line,text,token}));
}
has(tokens('fn i32 main() { return 0; }'),'fn','keyword');
has(tokens('let i64 total = 9223372036854775807;'),'i64','storage.type');
has(tokens('let message = "if // fake";'),'if','string');
has(tokens('/* outer /* inner */'),'inner','comment');
has(tokens('return still_comment; */ fn i32 next() {}'),'still_comment','comment');
has(tokens('let bytes = b"\\x41";'),'\\x41','escape');
has(tokens('let value = 0xA_B; // 한국어'),'0xA_B','numeric');
scopeAt('@effects(filesystem, console)','effects','entity.other.attribute-name');
scopeAt('@must_use','must_use','entity.other.attribute-name');
scopeAt('@link_name("native_symbol")','link_name','entity.other.attribute-name');
scopeAt('@effects(filesystem, console)','effects','entity.name.function',false);
for(const line of [
 'var bytes::Buffer input = bytes::empty();',
 'let bytes::Buffer saved = input;',
 'fn bytes::Buffer make() {',
 'fn void fill(ref mut bytes::Buffer input) {',
 'struct Holder { bytes::Buffer input; }',
 'pub bytes::Buffer input;',
 '    bytes::Buffer input;',
 'fn Result<bytes::Buffer, bytes::BufferError> make() {',
 'let Option<Result<bytes::Buffer, bytes::BufferError>> input = .none;',
]) {
 scopeAt(line,'Buffer','entity.name.type');
 scopeAt(line,'bytes','entity.name.namespace');
}
scopeAt('var bytes::Buffer input = bytes::empty();','empty','entity.name.function');
scopeAt('var bytes::Buffer input = bytes::empty();','empty','entity.name.type',false);
scopeAt('fn bytes::Buffer make() {','make','entity.name.function');
scopeAt('var bytes::Buffer[4] input;','Buffer','entity.name.type');
scopeAt('var bytes::Buffer[4] input;','4','constant.numeric');
scopeAt('fn void fill(ref mut bytes::Buffer input) {','ref','keyword.control');
scopeAt('let result<i32> input = value;','result','entity.name.type');
scopeAt('fn i32 read(i32 input, bytes::Buffer saved) {','i32','storage.type');
scopeAt('let count = bytes::len(ref input);','len','entity.name.function');
scopeAt('let count = bytes::len(ref input);','count','entity.name.type',false);
scopeAt('import std::bytes as bytes;','bytes','entity.name.type',false);
scopeAt('return bytes::len(ref input);','return','entity.name.type',false);
scopeAt('let message = "@effects bytes::Buffer input";','effects','entity.other.attribute-name',false);
scopeAt('// @effects bytes::Buffer input','Buffer','entity.name.type',false);
scopeAt('/* @effects bytes::Buffer input */','effects','entity.other.attribute-name',false);
scopeAt('let Result<i32, /* bytes::Buffer */ Error> input;','Buffer','comment.block');
scopeAt('let Result<i32, /* bytes::Buffer */ Error> input;','Buffer','entity.name.type',false);
scopeAt('let Container<"bytes::Buffer"> input;','Buffer','string.quoted');
scopeAt('let Container<"bytes::Buffer"> input;','Buffer','entity.name.type',false);
const metadata=JSON.parse(fs.readFileSync('../language.json','utf8'));
for(const word of metadata.reservedKeywords){stack=null;has(tokens(word+' '),word,['true','false','null'].includes(word)?'constant.language':'keyword');}
for(const snippet of Object.values(JSON.parse(fs.readFileSync('snippets/fairyc.json','utf8'))))assert(snippet.prefix&&snippet.body);
console.log('TextMate Oniguruma tokenization, attributes, nominal types, call distinctions, nested comments, strings, numbers and vocabulary passed');
