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
has(tokens('fn i32 main() { return 0; }'),'fn','keyword');
has(tokens('let message = "if // fake";'),'if','string');
has(tokens('/* outer /* inner */'),'inner','comment');
has(tokens('return still_comment; */ fn i32 next() {}'),'still_comment','comment');
has(tokens('let bytes = b"\\x41";'),'\\x41','escape');
has(tokens('let value = 0xA_B; // 한국어'),'0xA_B','numeric');
const metadata=JSON.parse(fs.readFileSync('../language.json','utf8'));
for(const word of metadata.reservedKeywords){stack=null;has(tokens(word+' '),word,['true','false','null'].includes(word)?'constant.language':'keyword');}
for(const snippet of Object.values(JSON.parse(fs.readFileSync('snippets/fairyc.json','utf8'))))assert(snippet.prefix&&snippet.body);
console.log('TextMate Oniguruma tokenization, nested comments, strings, numbers and vocabulary passed');
