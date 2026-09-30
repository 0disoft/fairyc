import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
const root=path.resolve('.');
const grammar=path.join(root,'editors/tree-sitter-fairyc');
const cli=path.join(root,'.cache/editor-tools/tree-sitter.exe');
const run=args=>{const r=spawnSync(cli,args,{cwd:grammar,encoding:'utf8',shell:false,timeout:60000});if(r.error)throw r.error;return r;};
const manifest=JSON.parse(fs.readFileSync('spec/fixtures/parser/manifest-v0.json','utf8'));
const pass=manifest.cases.filter(c=>c.expect.status==='pass').map(c=>path.join(root,'spec/fixtures/parser',c.sourceRoot,c.source));
const failures=[];
for(const file of pass){const r=run(['parse','--quiet',file]);if(r.status!==0)failures.push(file+'\n'+r.stdout);}
assert.deepEqual(failures,[]);
for(const name of ['highlights','brackets','indents','outline']){
 const r=run(['query',path.join(grammar,'queries',name+'.scm'),pass[0]]);
 assert.equal(r.status,0,name+'\n'+r.stderr);
 assert.equal(fs.readFileSync(path.join(grammar,'queries',name+'.scm'),'utf8'),fs.readFileSync('editors/zed/languages/fairyc/'+name+'.scm','utf8'));
}
fs.mkdirSync('.cache/editor-inputs',{recursive:true});
fs.writeFileSync('.cache/editor-inputs/incomplete.fy','module main; fn i32 main() { return');
assert.notEqual(run(['parse','--quiet',path.join(root,'.cache/editor-inputs/incomplete.fy')]).status,0);
fs.writeFileSync('.cache/editor-inputs/editing.fy','module main; /* unfinished');
const editing=run(['parse','--quiet',path.join(root,'.cache/editor-inputs/editing.fy')]);assert.equal(editing.status,0,editing.stderr);
console.log('Tree-sitter compiled parser: '+pass.length+' compiler parser pass fixtures, four query sets and editing recovery passed');
