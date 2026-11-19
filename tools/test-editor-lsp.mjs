import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import {spawn} from 'node:child_process';
import {createRequire} from 'node:module';
import {pathToFileURL} from 'node:url';
const require=createRequire(new URL('../editors/vscode/package.json',import.meta.url));
const rpc=require('vscode-jsonrpc/node');
const root=path.resolve('.cache/editor-inputs/native-lsp');
fs.mkdirSync(path.join(root,'src'),{recursive:true});
fs.writeFileSync(path.join(root,'fairy.toml'),'schema = "fairyc.manifest/v1"\n[package]\nuid = "editor.native-test"\nname = "editor-native-test"\nversion = "1.0.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n');
const filename=path.join(root,'src','main.fy'),uri=pathToFileURL(filename).href;
const text='module main;\nfn i32 helper() { return 7; }\nfn i32 main() { return helper(); }\n';
fs.writeFileSync(filename,'module main; fn i32 main() { return 0; }');
const child=spawn(path.resolve('.cache/fairyc-lsp.exe'),[],{stdio:['pipe','pipe','pipe'],shell:false});
let stderr='';child.stderr.on('data',x=>{stderr=(stderr+x.toString()).slice(-4096);});
const connection=rpc.createMessageConnection(new rpc.StreamMessageReader(child.stdout),new rpc.StreamMessageWriter(child.stdin));
const diagnosticWaiters=new Map();
connection.onNotification('textDocument/publishDiagnostics',p=>{const resolve=diagnosticWaiters.get(p.version);if(resolve){diagnosticWaiters.delete(p.version);resolve(p);}});
const diagnostics=version=>new Promise(resolve=>diagnosticWaiters.set(version,resolve));
const timeout=setTimeout(()=>{child.kill();throw Error('Native LSP smoke timed out '+stderr);},15000);
try{
 connection.listen();
 const capabilities={
  workspace:{workspaceEdit:{documentChanges:true}},
  textDocument:{codeAction:{codeActionLiteralSupport:{codeActionKind:{valueSet:['quickfix']}}}},
 };
 const init=await connection.sendRequest('initialize',{
  processId:process.pid,rootUri:pathToFileURL(root).href,capabilities,
  initializationOptions:{fairycSdkSourceNavigation:true},
 });
 assert.equal(init.capabilities.experimental.fairycSdkSourceNavigation,true);
 assert.equal(init.capabilities.positionEncoding,'utf-16');
 assert.equal(init.capabilities.textDocumentSync.change,2);
 assert(init.capabilities.completionProvider.triggerCharacters.includes('.'));
 for(const capability of ['definitionProvider','hoverProvider','referencesProvider','documentFormattingProvider','documentRangeFormattingProvider']){
  assert.equal(init.capabilities[capability],true);
 }
 connection.sendNotification('initialized',{});
 let wait=diagnostics(1);
 connection.sendNotification('textDocument/didOpen',{textDocument:{uri,languageId:'fairyc',version:1,text}});
 assert.equal((await wait).diagnostics.length,0);
 const character=text.split('\n')[2].indexOf('helper');
 const definition=await connection.sendRequest('textDocument/definition',{textDocument:{uri},position:{line:2,character:character+1}});
 assert.equal(definition.length,1);assert.equal(definition[0].range.start.line,1);
 const completion=await connection.sendRequest('textDocument/completion',{textDocument:{uri},position:{line:2,character:character+3}});
 assert(completion.some(c=>c.label==='helper'));
 const position={line:2,character:character+1};
 const hover=await connection.sendRequest('textDocument/hover',{textDocument:{uri},position});
 assert.match(hover.contents.value,/helper/);
 const references=await connection.sendRequest('textDocument/references',{textDocument:{uri},position,context:{includeDeclaration:true}});
 assert.deepEqual(references.map(ref=>ref.range.start.line),[1,2]);
 assert.equal(init.capabilities.renameProvider.prepareProvider,true);
 const prepared=await connection.sendRequest('textDocument/prepareRename',{textDocument:{uri},position});
 assert.equal(prepared.placeholder,'helper');
 const renamed=await connection.sendRequest('textDocument/rename',{textDocument:{uri},position,newName:'answer'});
 assert.deepEqual(renamed.documentChanges,[{
  textDocument:{uri,version:1},edits:references.map(ref=>({range:ref.range,newText:'answer'})),
 }]);
 const legend=init.capabilities.semanticTokensProvider.legend;
 const functionToken=legend.tokenTypes.indexOf('function');
 assert(functionToken >= 0);
 const tokens=await connection.sendRequest('textDocument/semanticTokens/full',{textDocument:{uri}});
 assert.deepEqual(tokens.data.filter((_,index)=>index%5===3),[functionToken,functionToken,functionToken]);
 const returnColumn=text.split('\n')[1].indexOf('7');
 const replaceReturn=(width,newText)=>({
  range:{start:{line:1,character:returnColumn},end:{line:1,character:returnColumn+width}},
  text:newText,
 });
 wait=diagnostics(2);
 connection.sendNotification('textDocument/didChange',{textDocument:{uri,version:2},contentChanges:[replaceReturn(1,'false')]});
 assert((await wait).diagnostics.length>0);
 assert.deepEqual((await connection.sendRequest('textDocument/semanticTokens/full',{textDocument:{uri}})).data,[]);
 wait=diagnostics(3);
 connection.sendNotification('textDocument/didChange',{textDocument:{uri,version:3},contentChanges:[
  replaceReturn(5,'7'),
  {range:{start:{line:0,character:12},end:{line:0,character:12}},text:'  '},
 ]});
 assert.equal((await wait).diagnostics.length,0);
 const formatting=await connection.sendRequest('textDocument/formatting',{textDocument:{uri},options:{tabSize:4,insertSpaces:true}});
 assert.equal(formatting.length,1);
 assert.equal(formatting[0].newText,text);
 const selection={start:{line:0,character:12},end:{line:1,character:0}};
 const rangeFormatting=await connection.sendRequest('textDocument/rangeFormatting',{
  textDocument:{uri},range:selection,options:{tabSize:4,insertSpaces:true},
 });
 assert.deepEqual(rangeFormatting,[{range:selection,newText:'\n'}]);
 const memberText='module main; struct Point { i32 row; } fn i32 read(ref Point point) { return point.; }';
 wait=diagnostics(4);
 connection.sendNotification('textDocument/didChange',{textDocument:{uri,version:4},contentChanges:[{text:memberText}]});
 assert((await wait).diagnostics.length>0);
 const members=await connection.sendRequest('textDocument/completion',{
  textDocument:{uri},position:{line:0,character:memberText.lastIndexOf('point.')+'point.'.length},
 });
 assert.deepEqual(members,[{label:'row',kind:5,detail:'field'}]);
 const missingSemicolon='module main; fn i32 main() { return 42 }';
 wait=diagnostics(5);
 connection.sendNotification('textDocument/didChange',{textDocument:{uri,version:5},contentChanges:[{text:missingSemicolon}]});
 const damaged=(await wait).diagnostics;
 assert.deepEqual(damaged.map(d=>d.code),['E0007']);
 const fixes=await connection.sendRequest('textDocument/codeAction',{
  textDocument:{uri},range:damaged[0].range,context:{diagnostics:damaged,only:['quickfix']},
 });
 assert.equal(fixes.length,1);
 assert.deepEqual(fixes[0].edit.documentChanges,[{
  textDocument:{uri,version:5},edits:[{range:damaged[0].range,newText:';'}],
 }]);
 const sdkText='module main; import std::bytes as bytes; fn bool same() { return bytes::equal(b"x", b"x"); }';
 wait=diagnostics(6);
 connection.sendNotification('textDocument/didChange',{textDocument:{uri,version:6},contentChanges:[{text:sdkText}]});
 assert.equal((await wait).diagnostics.length,0);
 const sdkLocations=await connection.sendRequest('textDocument/definition',{
  textDocument:{uri},position:{line:0,character:sdkText.indexOf('bytes::equal')+'bytes::'.length},
 });
 assert.equal(sdkLocations.length,1);
 const sdkLocation=sdkLocations[0];
 assert.match(sdkLocation.uri,/^fairyc-sdk:\/\/stdlib\/[a-f0-9]{64}\/std\/bytes\.fy$/);
 const sdkSource=await connection.sendRequest('fairyc/sdkSource',{uri:sdkLocation.uri});
 const sdkLine=sdkSource.split('\n')[sdkLocation.range.start.line];
 assert.equal(sdkLine.slice(sdkLocation.range.start.character,sdkLocation.range.end.character),'equal');
 assert.equal(fs.readFileSync(filename,'utf8'),'module main; fn i32 main() { return 0; }');
 assert.equal(await connection.sendRequest('shutdown'),null);
 const exited=new Promise((resolve,reject)=>{child.once('exit',code=>code===0?resolve():reject(Error('server exit '+code+' '+stderr)));});
 connection.sendNotification('exit');await exited;
 assert.equal(stderr,'');
 console.log('Native fairyc-lsp + vscode-jsonrpc: initialize, diagnostics, definition, completion, semantic tokens, quick fixes, SDK source, hover, references, document/range formatting, incremental edits, shutdown passed');
}finally{clearTimeout(timeout);connection.dispose();if(child.exitCode===null)child.kill();}
