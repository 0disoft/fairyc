module module_catalog;
import candidate_limits as limits;
import sha256 as hash;
import syntax as syntax;
import module_registry as scalar_modules;
import std::bytes as bytes;
import std::vec as v;
import sdk_authority as sdk;

// Program-local declaration coordinates. Names and aliases never define type
// equality. This owner contains no borrowed syntax or source buffers.
pub struct Problem {u32 code;usize module_index;usize start;usize end;}
pub struct Module {usize owner;usize start;usize end;usize source_start;usize source_end;u32[8] digest;bool vector_sdk;}
pub struct Declaration {u32 kind;usize module_index;usize declaration;bool public;usize start;usize end;usize source_start;usize source_end;}
pub struct Import {usize module_index;usize target;usize start;usize end;usize path_start;usize path_end;usize source_start;usize source_end;}
// Explicit checked source-set coordinates, sorted by source/declaration. The
// package loader resolves visibility; this catalog authenticates exact names,
// declaration spans and provider coordinates without rewriting source bytes.
pub struct Binding{usize source;usize declaration;usize target;}
pub struct Lookup {u32 code;usize item;}
pub resource Catalog {bytes::Buffer names;bytes::Buffer sources;v::Vec<Module> modules;v::Vec<Declaration> declarations;v::Vec<Import> imports;cleanup{}}
fn Problem fail(u32 code,usize module_index,usize start,usize end){return Problem{.code,.module_index,.start,.end};}
fn bool append(ref mut bytes::Buffer out,ref u8[] value){return bytes::len(out)<=limits::program_bytes() && value.len<=limits::program_bytes()-bytes::len(out) && bytes::append(out,value).is_ok;}
fn bool same(ref Catalog c,usize start,usize end,ref u8[] name){return bytes::equal(bytes::view(ref c.names)[start..end],name);}
fn bool spelling_same(ref bytes::Buffer names,usize start,usize end,ref u8[] name){return bytes::equal(bytes::view(names)[start..end],name);}
pub fn usize module_count(ref Catalog c){return v::len<Module>(ref c.modules);}
pub fn usize module_owner(ref Catalog c,usize module_index){assert(module_index<module_count(c));return v::at<Module>(ref c.modules,module_index).owner;}
pub fn usize declaration_count(ref Catalog c){return v::len<Declaration>(ref c.declarations);}
// Digest belongs to the retained exact source snapshot, not a caller claim.
pub fn u32[8] source_digest(ref Catalog c,usize module_index){assert(module_index<module_count(c));return v::at<Module>(ref c.modules,module_index).digest;}
pub fn bool source_matches(ref Catalog c,usize module_index,ref syntax::Parsed p){if(module_index>=module_count(c)){return false;}let m=*v::at<Module>(ref c.modules,module_index);return bytes::equal(bytes::view(ref c.sources)[m.source_start..m.source_end],syntax::source_view(p));}
// Build authenticates reserved source names before retaining this flag.
pub fn bool sdk_vector_source(ref Catalog c,usize module_index){return module_index<module_count(c) && v::at<Module>(ref c.modules,module_index).vector_sdk;}
pub fn bool sdk_vector(ref Catalog c,usize module_index,usize declaration){
    if(!sdk_vector_source(c,module_index)){return false;}
    let usize id=identity(c,module_index,declaration);return id!=0 && declaration_at(c,id).kind==6 && same(c,declaration_at(c,id).start,declaration_at(c,id).end,b"Vec");
}
pub fn Declaration declaration_at(ref Catalog c,usize id){assert(id>0 && id<=declaration_count(c));return *v::at<Declaration>(ref c.declarations,id-1);}
pub fn usize identity(ref Catalog c,usize module_index,usize declaration){var usize i=1;while(i<=declaration_count(c)){let d=declaration_at(c,i);if(d.module_index==module_index && d.declaration==declaration){return i;}i+=1;}return 0;}
pub fn bool same_declaration(Declaration a,Declaration b){return a.kind==b.kind && a.module_index==b.module_index && a.declaration==b.declaration;}
fn usize destination(ref Catalog c,Import i,ref mut usize probes){if(i.target!=0){*probes=*probes+1;if(i.target>module_count(c)){return 0;}let m=*v::at<Module>(ref c.modules,i.target-1);return if(same(c,m.start,m.end,bytes::view(ref c.names)[i.path_start..i.path_end])){i.target}else{0};}var usize n=0;while(n<module_count(c)){*probes=*probes+1;let m=*v::at<Module>(ref c.modules,n);if(same(c,m.start,m.end,bytes::view(ref c.names)[i.path_start..i.path_end])){return n+1;}n+=1;}return 0;}
fn usize first(ref Catalog c,usize module_index,bool imports,ref mut usize probes){
    var usize low=0;var usize high=if(imports){v::len<Import>(ref c.imports)}else{declaration_count(c)};
    while(low<high){*probes=*probes+1;let usize mid=low+(high-low)/2;let usize owner=if(imports){v::at<Import>(ref c.imports,mid).module_index}else{declaration_at(c,mid+1).module_index};if(owner<module_index){low=mid+1;}else{high=mid;}}
    return low;
}
pub fn Lookup lookup(ref Catalog c,usize module_index,ref u8[] spelling){var usize probes=0;return lookup_measured(c,module_index,spelling,ref mut probes);}
pub fn Lookup lookup_measured(ref Catalog c,usize module_index,ref u8[] spelling,ref mut usize probes){
    if(module_index>=module_count(c)){return Lookup{.code=9,.item=0};}
    var bytes::Buffer canonical=bytes::empty();let u32 code=scalar_modules::normalize(spelling,ref mut canonical);if(code!=0){return Lookup{.code,.item=0};}
    let name=bytes::view(ref canonical);var usize split=0;while(split<name.len && name[split]!=58){split+=1;}
    var usize target=module_index;var usize start=0;
    if(split<name.len){if(split+2>=name.len || name[split+1]!=58){return Lookup{.code=9,.item=0};}var usize found=0;var usize n=first(c,module_index,true,probes);
        while(n<v::len<Import>(ref c.imports)){*probes=*probes+1;let i=*v::at<Import>(ref c.imports,n);if(i.module_index!=module_index){break;}if(same(c,i.start,i.end,name[..split])){found=destination(c,i,probes);break;}n+=1;}
        if(found==0){return Lookup{.code=4,.item=0};}target=found-1;start=split+2;
    }
    var usize n=first(c,target,false,probes)+1;while(n<=declaration_count(c)){*probes=*probes+1;let d=declaration_at(c,n);if(d.module_index!=target){break;}if(same(c,d.start,d.end,name[start..])){return Lookup{.code=if(split<name.len && !d.public){6}else{0},.item=n};}n+=1;}
    return Lookup{.code=5,.item=0};
}
pub fn Result<Catalog,Problem> build(ref v::Vec<syntax::Parsed> sources){
    let owners=v::empty<usize>();let bindings=v::empty<Binding>();return build_inputs(sources,ref owners,ref bindings,false);
}
pub fn Result<Catalog,Problem> build_scoped(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<Binding> bindings){return build_inputs(sources,owners,bindings,true);}
fn Result<Catalog,Problem> build_inputs(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<Binding> bindings,bool scoped){
    var bytes::Buffer names=bytes::empty();var bytes::Buffer source_bytes=bytes::empty();
    var v::Vec<Module> modules=v::empty<Module>();var v::Vec<Declaration> declarations=v::empty<Declaration>();var v::Vec<Import> imports=v::empty<Import>();
    if(v::len<syntax::Parsed>(sources)==0 || v::len<syntax::Parsed>(sources)>limits::source_files()){return .err(fail(10,0,0,0));}
    if(scoped && (v::len<usize>(owners)!=v::len<syntax::Parsed>(sources) || v::len<Binding>(bindings)>4096)){return .err(fail(9,0,0,0));}var usize next_binding=0;
    var usize module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){
        let p=v::at<syntax::Parsed>(sources,module_index);if(syntax::problem_count(p)!=0 || syntax::declaration_count(p)==0){return .err(fail(9,module_index,0,0));}
        let first=syntax::declaration_at(p,0);if(first.kind!=1){return .err(fail(9,module_index,first.start,first.end));}
        var bytes::Buffer path=bytes::empty();let u32 status=scalar_modules::normalize(syntax::source_view(p)[first.name_start..first.name_end],ref mut path);if(status!=0){return .err(fail(status,module_index,first.name_start,first.name_end));}
        if(sdk::reserved(bytes::view(ref path)) && !sdk::authentic(bytes::view(ref path),syntax::source_view(p))){return .err(fail(6801,module_index,first.name_start,first.name_end));}
        let usize owner=if(scoped){*v::at<usize>(owners,module_index)}else{0};if(owner>=limits::source_files()){return .err(fail(10,module_index,first.name_start,first.name_end));}
        var usize n=0;while(n<v::len<Module>(ref modules)){let m=*v::at<Module>(ref modules,n);if(m.owner==owner && spelling_same(ref names,m.start,m.end,bytes::view(ref path))){return .err(fail(1,module_index,first.name_start,first.name_end));}n+=1;}
        let usize start=bytes::len(ref names);let usize source_start=bytes::len(ref source_bytes);if(syntax::source_view(p).len>limits::source_bytes()){return .err(fail(10,module_index,0,syntax::source_view(p).len));}
        if(source_start>limits::source_set_bytes() || syntax::source_view(p).len>limits::source_set_bytes()-source_start){return .err(fail(10,module_index,0,syntax::source_view(p).len));}
        if(!bytes::append(ref mut source_bytes,syntax::source_view(p)).is_ok || !append(ref mut names,bytes::view(ref path)) || !v::push<Module>(ref mut modules,Module{.owner,.start,.end=bytes::len(ref names),.source_start,.source_end=bytes::len(ref source_bytes),.digest=hash::digest(syntax::source_view(p)),.vector_sdk=bytes::equal(bytes::view(ref path),b"std::vec")}).is_ok){return .err(fail(11,module_index,first.start,first.end));}
        var usize index=1;var usize count=0;while(index<syntax::declaration_count(p)){
            let d=syntax::declaration_at(p,index);if(d.kind==8){index+=1;continue;}if(d.kind!=2 && d.kind!=3 && d.kind!=4 && d.kind!=6 && d.kind!=7 && d.kind!=9 && d.kind!=11 || d.owner_end!=0){return .err(fail(9,module_index,d.start,d.end));}
            let name=syntax::source_view(p)[d.name_start..d.name_end];if(bytes::equal(name,b"source_location")){return .err(fail(4401,module_index,d.name_start,d.name_end));}
            n=0;while(n<v::len<Declaration>(ref declarations)){let old=*v::at<Declaration>(ref declarations,n);if(old.module_index==module_index && spelling_same(ref names,old.start,old.end,name)){return .err(fail(2,module_index,d.name_start,d.name_end));}n+=1;}
            n=0;while(n<v::len<Import>(ref imports)){let old=*v::at<Import>(ref imports,n);if(old.module_index==module_index && spelling_same(ref names,old.start,old.end,name)){return .err(fail(if(d.kind==2){7}else{2},module_index,d.name_start,d.name_end));}n+=1;}
            count+=1;if(count>128 || v::len<Declaration>(ref declarations)>=4096 || v::len<Import>(ref imports)>=4096){return .err(fail(10,module_index,d.start,d.end));}
            let usize name_start=bytes::len(ref names);if(!append(ref mut names,name)){return .err(fail(11,module_index,d.name_start,d.name_end));}let usize name_end=bytes::len(ref names);
            if(d.kind==2){if(d.public || d.attribute_count!=0){return .err(fail(9,module_index,d.start,d.end));}var bytes::Buffer target=bytes::empty();let u32 code=scalar_modules::normalize(syntax::source_view(p)[d.start..d.end],ref mut target);if(code!=0){return .err(fail(code,module_index,d.start,d.end));}var usize provider=0;if(scoped){if(next_binding>=v::len<Binding>(bindings)){return .err(fail(9,module_index,d.start,d.end));}let b=*v::at<Binding>(bindings,next_binding);if(b.source!=module_index || b.declaration!=index+1 || b.target>=v::len<syntax::Parsed>(sources)){return .err(fail(9,module_index,d.start,d.end));}provider=b.target+1;next_binding+=1;}let usize path_start=bytes::len(ref names);if(!append(ref mut names,bytes::view(ref target)) || !v::push<Import>(ref mut imports,Import{.module_index,.target=provider,.start=name_start,.end=name_end,.path_start,.path_end=bytes::len(ref names),.source_start=d.start,.source_end=d.end}).is_ok){return .err(fail(11,module_index,d.start,d.end));}}
            else{if(!v::push<Declaration>(ref mut declarations,Declaration{.kind=d.kind,.module_index,.declaration=index+1,.public=d.public,.start=name_start,.end=name_end,.source_start=d.start,.source_end=d.end}).is_ok){return .err(fail(11,module_index,d.start,d.end));}}
            index+=1;
        }module_index+=1;
    }
    if(scoped && next_binding!=v::len<Binding>(bindings)){return .err(fail(9,0,0,0));}let Catalog c=Catalog{.names,.sources=source_bytes,.modules,.declarations,.imports};
    var usize n=0;var usize probes=0;while(n<v::len<Import>(ref c.imports)){let i=*v::at<Import>(ref c.imports,n);if(destination(ref c,i,ref mut probes)==0){return .err(fail(3,i.module_index,i.source_start,i.source_end));}n+=1;}
    return .ok(c);
}
