module generic_package_bindings;
import project_compiler_input as input;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import project_import_path as imports;
import project_path as budget;
import generic_source_json as json;
import generic_source_function_json as functions;
import generic_source_aggregate_json as aggregates;
import generic_identity as identity;
import syntax as syntax;
import scanner as scanner;
import sha256 as hash;
import std::bytes as bytes;
import std::vec as v;

// Resolved source coordinates are retained separately from canonical identity.
pub struct Text{usize start;usize end;}
pub struct Module{usize owner;Text namespace;Text path;usize first;usize count;usize source_size;u32[8] digest;}
pub struct Import{usize module_index;usize declaration;usize target;Text alias;Text binding;}
pub resource Bindings{bytes::Buffer names;v::Vec<Module> modules;v::Vec<Import> imports;cleanup{}}
// 9 invalid plumbing, 10 bounded work/output, 11 allocation, 12 stale source.
pub struct Problem{u32 code;usize module_index;usize declaration;}
fn Problem fail(u32 code,usize module_index,usize declaration){return Problem{.code,.module_index,.declaration};}
pub fn usize count(ref Bindings p){return v::len<Module>(ref p.modules);}
pub fn ref u8[] names_view(ref Bindings p){return bytes::view(ref p.names);}
pub fn Module module_at(ref Bindings p,usize at){assert(at<count(p));return *v::at<Module>(ref p.modules,at);}
pub fn ref u8[] text(ref Bindings p,Text t){assert(t.start<=t.end && t.end<=bytes::len(ref p.names));return bytes::view(ref p.names)[t.start..t.end];}
pub fn usize import_count(ref Bindings p){return v::len<Import>(ref p.imports);}
pub fn Import import_at(ref Bindings p,usize at){assert(at<import_count(p));return *v::at<Import>(ref p.imports,at);}
pub fn Bindings empty(){return Bindings{.names=bytes::empty(),.modules=v::empty<Module>(),.imports=v::empty<Import>()};}
// Detached provenance keeps exact source stamps and resolved import/namespace
// records without retaining parser arenas or source bytes.
pub fn Result<Bindings,u32> copy(ref Bindings input,ref mut usize meter){
    if(count(input)>512 || import_count(input)>32768 || bytes::len(ref input.names)>1048576){return .err(9);}if(!budget::charge(meter,bytes::len(ref input.names)+1)){return .err(10);}let names=switch(bytes::copy(bytes::view(ref input.names))){case .ok(p):{p}case .err(_):{return .err(11);}};var v::Vec<Module> modules=v::empty<Module>();var v::Vec<Import> imports=v::empty<Import>();var usize i=0;
    while(i<count(input)){if(!budget::charge(meter,1)){return .err(10);}if(!v::push<Module>(ref mut modules,module_at(input,i)).is_ok){return .err(11);}i+=1;}i=0;while(i<import_count(input)){if(!budget::charge(meter,1)){return .err(10);}if(!v::push<Import>(ref mut imports,import_at(input,i)).is_ok){return .err(11);}i+=1;}return .ok(Bindings{.names,.modules,.imports});
}
fn Result<Text,u32> retain(ref mut bytes::Buffer names,ref u8[] data,ref mut usize meter){if(data.len>1048576 || bytes::len(names)>1048576-data.len || !budget::charge(meter,data.len+1)){return .err(10);}let start=bytes::len(names);if(!bytes::append(names,data).is_ok){return .err(11);}return .ok(Text{.start,.end=bytes::len(names)});}
fn Result<bytes::Buffer,u32> namespace(ref coordinates::Coordinates table,usize owner,ref mut usize meter){var usize checked=0;let code=coordinates::validate_owner(table,owner,ref mut checked);if(!budget::charge(meter,checked)){return .err(10);}if(code!=0){return .err(code);}var usize found=coordinates::coordinate_count(table);var usize i=0;
    while(i<coordinates::coordinate_count(table)){if(!budget::charge(meter,1)){return .err(10);}let row=coordinates::coordinate_at(table,i);if(row.owner==owner){if(found!=coordinates::coordinate_count(table) || row.start>row.end || row.end-row.start!=77){return .err(9);}found=i;}i+=1;}
    if(found==coordinates::coordinate_count(table)){return .err(9);}let value=coordinates::coordinate_text(table,found);let prefix=b"fypc1-sha256-";if(!budget::charge(meter,value.len+1)){return .err(10);}if(!coordinates::valid_coordinate(value)){return .err(9);}var bool nonzero=false;i=prefix.len;while(i<value.len){if(value[i]!=48){nonzero=true;}i+=1;}if(!nonzero){return .err(9);}var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,b"package_coordinate_v1_").is_ok || !bytes::append(ref mut out,value[prefix.len..]).is_ok){return .err(11);}return .ok(out);
}
fn Result<bytes::Buffer,u32> module_path(ref syntax::Parsed p,ref mut usize meter){if(syntax::problem_count(p)!=0 || syntax::declaration_count(p)==0){return .err(9);}let d=syntax::declaration_at(p,0);let source=syntax::source_view(p);if(d.kind!=1 || d.name_start>=d.name_end || d.name_end>source.len){return .err(9);}var usize low=0;var usize high=syntax::token_count(p);
    while(low<high){if(!budget::charge(meter,1)){return .err(10);}let mid=low+(high-low)/2;if(syntax::token_info(p,mid).start<d.name_start){low=mid+1;}else{high=mid;}}
    var bytes::Buffer out=bytes::empty();while(low<syntax::token_count(p)){if(!budget::charge(meter,1)){return .err(10);}let t=syntax::token_info(p,low);if(t.start>=d.name_end){break;}if(t.start>t.end || t.end>d.name_end){return .err(9);}if(t.kind!=2){let raw=source[t.start..t.end];if(!budget::charge(meter,raw.len+1)){return .err(10);}if((t.kind!=3 && !bytes::equal(raw,b"::")) || !bytes::append(ref mut out,raw).is_ok){return .err(9);}}low+=1;}
    if(!scanner::canonical_path(bytes::view(ref out))){return .err(9);}return .ok(out);
}
fn Result<bool,u32> less(ref bytes::Buffer names,Text left,Text right,ref mut usize meter){let a=bytes::view(names)[left.start..left.end];let b=bytes::view(names)[right.start..right.end];var usize i=0;while(i<a.len && i<b.len){if(!budget::charge(meter,1)){return .err(10);}if(a[i]!=b[i]){return .ok(a[i]<b[i]);}i+=1;}return .ok(a.len<b.len);}
fn u32 sort(ref mut v::Vec<Import> rows,ref bytes::Buffer names,usize first,usize end,ref mut usize meter){var usize i=first+1;while(i<end){let value=*v::at<Import>(rows,i);var usize at=i;while(at>first){if(!budget::charge(meter,1)){return 10;}let previous=*v::at<Import>(rows,at-1);let before=switch(less(names,value.binding,previous.binding,meter)){case .ok(value):{value}case .err(code):{return code;}};if(!before){break;}let slot=v::at_mut<Import>(rows,at);*slot=previous;at-=1;}let slot=v::at_mut<Import>(rows,at);*slot=value;i+=1;}return 0;}
// Workspace collection uses the existing 16MiB ledger; each definition/JSON
// request keeps the existing independent 1MiB source-identity ledger.
pub fn Result<Bindings,Problem> build(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> resolved,ref coordinates::Coordinates table,ref mut usize meter){let total=v::len<syntax::Parsed>(sources);if(total==0 || total>512 || v::len<usize>(owners)!=total || v::len<catalog::Binding>(resolved)>4096){return .err(fail(9,0,0));}if(!budget::charge(meter,total+1)){return .err(fail(10,0,0));}
    var bytes::Buffer names=bytes::empty();var v::Vec<Module> modules=v::empty<Module>();var v::Vec<Import> rows=v::empty<Import>();var usize module_index=0;
    while(module_index<total){let owner=*v::at<usize>(owners,module_index);if(owner>=512){return .err(fail(9,module_index,0));}let p=v::at<syntax::Parsed>(sources,module_index);let source=syntax::source_view(p);if(source.len>8388608 || !budget::charge(meter,source.len+1)){return .err(fail(10,module_index,0));}
        let owner_text=switch(namespace(table,owner,meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,module_index,0));}};let path=switch(module_path(p,meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,module_index,0));}};var usize i=0;
        while(i<module_index){if(!budget::charge(meter,1)){return .err(fail(10,module_index,0));}let old=*v::at<Module>(ref modules,i);let same_owner=bytes::equal(bytes::view(ref names)[old.namespace.start..old.namespace.end],bytes::view(ref owner_text));if(same_owner && old.owner!=owner || old.owner==owner && (!same_owner || bytes::equal(bytes::view(ref names)[old.path.start..old.path.end],bytes::view(ref path)))){return .err(fail(9,module_index,0));}i+=1;}
        let ns=switch(retain(ref mut names,bytes::view(ref owner_text),meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,module_index,0));}};let name=switch(retain(ref mut names,bytes::view(ref path),meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,module_index,0));}};
        if(!v::push<Module>(ref mut modules,Module{.owner,.namespace=ns,.path=name,.first=0,.count=0,.source_size=source.len,.digest=hash::digest(source)}).is_ok){return .err(fail(11,module_index,0));}module_index+=1;
    }
    var usize next=0;module_index=0;while(module_index<total){let p=v::at<syntax::Parsed>(sources,module_index);let first=v::len<Import>(ref rows);var usize d_index=1;while(d_index<syntax::declaration_count(p)){if(!budget::charge(meter,1)){return .err(fail(10,module_index,d_index));}let d=syntax::declaration_at(p,d_index);if(d.kind!=2){d_index+=1;continue;}
        if(next>=v::len<catalog::Binding>(resolved)){return .err(fail(9,module_index,d_index));}let binding=*v::at<catalog::Binding>(resolved,next);if(binding.source!=module_index || binding.declaration!=d_index+1 || binding.target>=total || d.name_start>=d.name_end || d.name_end>syntax::source_view(p).len){return .err(fail(9,module_index,d_index));}next+=1;
        let path=switch(imports::read(p,d_index,meter)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==3){10}else if(code==2){11}else{9},module_index,d_index));}};let target=*v::at<Module>(ref modules,binding.target);if(!bytes::equal(imports::name(ref path),bytes::view(ref names)[target.path.start..target.path.end])){return .err(fail(9,module_index,d_index));}
        let alias=syntax::source_view(p)[d.name_start..d.name_end];var usize i=0;while(i<v::len<Import>(ref rows)){if(!budget::charge(meter,1)){return .err(fail(10,module_index,d_index));}let old=*v::at<Import>(ref rows,i);if(old.module_index==module_index){if(!budget::charge(meter,alias.len+1)){return .err(fail(10,module_index,d_index));}if(bytes::equal(bytes::view(ref names)[old.alias.start..old.alias.end],alias)){return .err(fail(9,module_index,d_index));}}let prior=*v::at<Module>(ref modules,old.module_index);if(prior.owner==v::at<Module>(ref modules,module_index).owner){let provider=*v::at<Module>(ref modules,old.target);if(!budget::charge(meter,imports::name(ref path).len+1)){return .err(fail(10,module_index,d_index));}if(bytes::equal(bytes::view(ref names)[provider.path.start..provider.path.end],imports::name(ref path)) && provider.owner!=target.owner){return .err(fail(9,module_index,d_index));}}i+=1;}
        var bytes::Buffer record=bytes::empty();if(!bytes::append(ref mut record,alias).is_ok || !bytes::append(ref mut record,b"=").is_ok || !bytes::append(ref mut record,bytes::view(ref names)[target.namespace.start..target.namespace.end]).is_ok || !bytes::append(ref mut record,b"::").is_ok || !bytes::append(ref mut record,imports::name(ref path)).is_ok){return .err(fail(11,module_index,d_index));}
        let a=switch(retain(ref mut names,alias,meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,module_index,d_index));}};let r=switch(retain(ref mut names,bytes::view(ref record),meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,module_index,d_index));}};if(!v::push<Import>(ref mut rows,Import{.module_index,.declaration=d_index+1,.target=binding.target,.alias=a,.binding=r}).is_ok){return .err(fail(11,module_index,d_index));}d_index+=1;
    }let end=v::len<Import>(ref rows);let code=sort(ref mut rows,ref names,first,end,meter);if(code!=0){return .err(fail(code,module_index,0));}let slot=v::at_mut<Module>(ref mut modules,module_index);slot.first=first;slot.count=end-first;module_index+=1;}
    if(next!=v::len<catalog::Binding>(resolved)){return .err(fail(9,0,0));}return .ok(Bindings{.names,.modules,.imports=rows});
}
pub fn Result<Bindings,Problem> collect(ref input::Input sources,ref coordinates::Coordinates table,ref mut usize meter){return build(input::sources(sources),input::owners(sources),input::bindings(sources),table,meter);}
pub fn bool value(ref Bindings p,usize module_index,ref mut bytes::Buffer out,ref mut json::Ledger work){if(!json::charge(work,1)){return false;}if(module_index>=count(p)){return json::reject(work,9);}let m=module_at(p,module_index);if(m.count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}var usize i=0;while(i<m.count){if(!json::charge(work,1)){return false;}if(i!=0 && !json::raw(out,b",",work)){return false;}let row=import_at(p,m.first+i);if(!json::quoted(out,text(p,row.binding),work)){return false;}i+=1;}return json::raw(out,b"]",work);}
pub fn Result<bytes::Buffer,u32> encode(ref Bindings p,usize module_index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=value(p,module_index,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
fn u32 current(ref Bindings p,usize module_index,ref syntax::Parsed source,ref mut usize work){if(module_index>=count(p)){return 9;}var json::Ledger ledger=json::Ledger{.used=*work,.code=0};let raw=syntax::source_view(source);let ok=json::charge(ref mut ledger,raw.len+1);*work=ledger.used;if(!ok){return ledger.code;}let m=module_at(p,module_index);if(m.source_size!=raw.len){return 12;}let digest=hash::digest(raw);var usize i=0;while(i<8){if(digest[i]!=m.digest[i]){return 12;}i+=1;}return 0;}
pub fn Result<bytes::Buffer,u32> definition(ref Bindings p,usize module_index,ref syntax::Parsed source,usize declaration,ref mut usize work){let code=current(p,module_index,source,work);if(code!=0){return .err(code);}if(declaration>=syntax::declaration_count(source)){return .err(9);}let d=syntax::declaration_at(source,declaration);if(d.generic_parameters==0 || (d.kind!=3 && d.kind!=4 && d.kind!=6)){return .err(9);}var encoded=switch(if(d.kind==3){functions::encode(source,declaration,work)}else{aggregates::template(source,declaration,work)}){case .ok(value):{value}case .err(code):{return .err(code);}};
    var json::Ledger ledger=json::Ledger{.used=*work,.code=0};let ok=json::raw(ref mut encoded,b"\n",ref mut ledger) && value(p,module_index,ref mut encoded,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return identity::digest_text(if(d.kind==3){b"source-package-function-v0"}else{b"source-package-aggregate-v0"},bytes::view(ref encoded),work);
}
