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
 const init=await connection.sendRequest('initialize',{processId:process.pid,rootUri:pathToFileURL(root).href,capabilities:{}});
 assert.equal(init.capabilities.positionEncoding,'utf-16');
 assert.equal(init.capabilities.textDocumentSync.change,2);
 for(const capability of ['definitionProvider','hoverProvider','referencesProvider','documentFormattingProvider']){
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
 const returnColumn=text.split('\n')[1].indexOf('7');
 const replaceReturn=(width,newText)=>({
  range:{start:{line:1,character:returnColumn},end:{line:1,character:returnColumn+width}},
  text:newText,
 });
 wait=diagnostics(2);
 connection.sendNotification('textDocument/didChange',{textDocument:{uri,version:2},contentChanges:[replaceReturn(1,'false')]});
 assert((await wait).diagnostics.length>0);
 wait=diagnostics(3);
 connection.sendNotification('textDocument/didChange',{textDocument:{uri,version:3},contentChanges:[
  replaceReturn(5,'7'),
  {range:{start:{line:0,character:12},end:{line:0,character:12}},text:'  '},
 ]});
 assert.equal((await wait).diagnostics.length,0);
 const formatting=await connection.sendRequest('textDocument/formatting',{textDocument:{uri},options:{tabSize:4,insertSpaces:true}});
 assert.equal(formatting.length,1);
 assert.equal(formatting[0].newText,text);
 assert.equal(fs.readFileSync(filename,'utf8'),'module main; fn i32 main() { return 0; }');
 assert.equal(await connection.sendRequest('shutdown'),null);
 const exited=new Promise((resolve,reject)=>{child.once('exit',code=>code===0?resolve():reject(Error('server exit '+code+' '+stderr)));});
 connection.sendNotification('exit');await exited;
 assert.equal(stderr,'');
 console.log('Native fairyc-lsp + vscode-jsonrpc: initialize, diagnostics, definition, completion, hover, references, formatting, incremental edits, shutdown passed');
}finally{clearTimeout(timeout);connection.dispose();if(child.exitCode===null)child.kill();}
