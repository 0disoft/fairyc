module module_source_artifact_model;
import generic_package_bindings as bindings;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import project_cache_encoding as budget;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Detached transport records carry no executable or Complete authority.
pub resource Snapshot{bytes::Buffer names;v::Vec<bindings::Module> modules;v::Vec<bindings::Import> imports;cleanup{}}
pub fn usize count(ref Snapshot p){return v::len<bindings::Module>(ref p.modules);}
pub fn usize import_count(ref Snapshot p){return v::len<bindings::Import>(ref p.imports);}
pub fn bindings::Module module_at(ref Snapshot p,usize i){return *v::at<bindings::Module>(ref p.modules,i);}
pub fn bindings::Import import_at(ref Snapshot p,usize i){return *v::at<bindings::Import>(ref p.imports,i);}
pub fn ref u8[] names(ref Snapshot p){return bytes::view(ref p.names);}
pub fn ref u8[] text(ref Snapshot p,bindings::Text t){return names(p)[t.start..t.end];}
fn u32 cover(ref mut bytes::Buffer seen,bindings::Text t,ref mut usize work){if(t.start>=t.end || t.end>bytes::len(seen)){return 1001;}if(!budget::charge(work,t.end-t.start+1)){return 1002;}let raw=bytes::view_mut(seen);var usize i=t.start;while(i<t.end){if(raw[i]!=0){return 1001;}raw[i]=1;i+=1;}return 0;}
fn bool namespace(ref u8[] value){let prefix=b"package_coordinate_v1_";if(value.len!=prefix.len+64 || !bytes::equal(value[..prefix.len],prefix)){return false;}var bool nonzero=false;var usize i=prefix.len;while(i<value.len){let c=value[i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}if(c!=48){nonzero=true;}i+=1;}return nonzero;}
fn bool alias(ref u8[] value){if(!scanner::canonical_path(value)){return false;}var usize i=0;while(i<value.len){if(value[i]==58){return false;}i+=1;}return true;}
pub fn u32 check(ref Snapshot p,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(count(p)>512 || import_count(p)>32768 || names(p).len>1048576){return 1002;}if(!budget::charge(work,names(p).len+1)){return 1002;}var bytes::Buffer seen=bytes::empty();if(!bytes::resize_zeroed(ref mut seen,names(p).len).is_ok){return 1003;}
    var usize[512] owners=[0;512];var usize next=0;var usize i=0;
    while(i<count(p)){if(!budget::charge(work,1)){return 1002;}let m=module_at(p,i);if(m.owner>=512 || m.first!=next || m.first>import_count(p) || m.count>import_count(p)-m.first || m.source_size>8388608){return 1001;}
        var u32 code=cover(ref mut seen,m.namespace,work);if(code!=0){return code;}code=cover(ref mut seen,m.path,work);if(code!=0){return code;}let ns=text(p,m.namespace);let path=text(p,m.path);if(!namespace(ns) || !scanner::canonical_path(path)){return 1001;}
        if(owners[m.owner]!=0){let previous=module_at(p,owners[m.owner]-1);if(!bytes::equal(ns,text(p,previous.namespace))){return 1001;}}
        else{var usize owner=0;while(owner<512){if(owners[owner]!=0){if(!budget::charge(work,1)){return 1002;}if(bytes::equal(ns,text(p,module_at(p,owners[owner]-1).namespace))){return 1001;}}owner+=1;}owners[m.owner]=i+1;}
        var usize prior=0;while(prior<i){if(!budget::charge(work,1)){return 1002;}let other=module_at(p,prior);if(other.owner==m.owner && bytes::equal(path,text(p,other.path))){return 1001;}prior+=1;}
        next+=m.count;i+=1;
    }if(next!=import_count(p)){return 1001;}
    i=0;while(i<count(p)){let m=module_at(p,i);var usize j=0;while(j<m.count){if(!budget::charge(work,1)){return 1002;}let row=import_at(p,m.first+j);if(row.module_index!=i || row.declaration<2 || row.declaration>1048576 || row.target>=count(p)){return 1001;}
        var u32 code=cover(ref mut seen,row.alias,work);if(code!=0){return code;}code=cover(ref mut seen,row.binding,work);if(code!=0){return code;}let name=text(p,row.alias);let value=text(p,row.binding);let provider=module_at(p,row.target);let ns=text(p,provider.namespace);let path=text(p,provider.path);
        if(!alias(name) || value.len!=name.len+ns.len+path.len+3 || !bytes::equal(value[..name.len],name) || value[name.len]!=61 || !bytes::equal(value[name.len+1..name.len+1+ns.len],ns) || !bytes::equal(value[name.len+1+ns.len..name.len+3+ns.len],b"::") || !bytes::equal(value[name.len+3+ns.len..],path)){return 1001;}
        if(j!=0){let previous=import_at(p,m.first+j-1);if(!budget::before(text(p,previous.binding),value) || bytes::equal(text(p,previous.alias),name)){return 1001;}}j+=1;}i+=1;
    }
    if(!budget::charge(work,names(p).len+1)){return 1002;}let coverage=bytes::view(ref seen);i=0;while(i<coverage.len){if(coverage[i]!=1){return 1001;}i+=1;}return 0;
}
pub fn Result<Snapshot,u32> assemble(bytes::Buffer names,v::Vec<bindings::Module> modules,v::Vec<bindings::Import> imports,ref mut usize work){let p=Snapshot{.names,.modules,.imports};let code=check(ref p,work);if(code!=0){return .err(code);}return .ok(p);}
pub fn Result<Snapshot,u32> extract(ref bindings::Bindings input,ref mut usize work){if(!budget::charge(work,1)){return .err(1002);}if(bindings::count(input)>512 || bindings::import_count(input)>32768 || bindings::names_view(input).len>1048576){return .err(1002);}if(!budget::charge(work,bindings::names_view(input).len+1)){return .err(1002);}let names=switch(bytes::copy(bindings::names_view(input))){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var v::Vec<bindings::Module> modules=v::empty<bindings::Module>();var v::Vec<bindings::Import> imports=v::empty<bindings::Import>();var usize i=0;while(i<bindings::count(input)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<bindings::Module>(ref mut modules,bindings::module_at(input,i)).is_ok){return .err(1003);}i+=1;}i=0;while(i<bindings::import_count(input)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<bindings::Import>(ref mut imports,bindings::import_at(input,i)).is_ok){return .err(1003);}i+=1;}return assemble(names,modules,imports,work);
}
