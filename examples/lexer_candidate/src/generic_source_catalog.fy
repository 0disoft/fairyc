module generic_source_catalog;
import generic_package_bindings as bindings;
import generic_identity as identity;
import project_path as budget;
import syntax as syntax;
import scanner as scanner;
import sdk_authority as sdk;
import sha256 as hash;
import std::bytes as bytes;
import std::vec as v;

// Original source templates, before concrete expansion erases their declarations.
// This catalog is not a genericplan.Declaration: dependencies are collected next.
pub struct Text {usize start;usize end;}
pub struct Parameter {Text name;bool scalar;}
pub struct Template {usize module_index;usize declaration;u32 kind;bool public;Text name;Text id;Text definition;usize first;usize count;usize start;usize end;usize name_start;usize name_end;}
pub resource Catalog {bindings::Bindings bindings;bytes::Buffer names;v::Vec<Template> templates;v::Vec<Parameter> parameters;bool scalar_constraints;cleanup{}}
pub struct Problem {u32 code;usize module_index;usize declaration;}
// Private helper codes: 9 invalid input, 10 work/output, 11 allocation, 12 stale,
// 13 inaccessible target, 4501 source template admission. No public diagnostics.
fn Problem fail(u32 code,usize module_index,usize declaration){return Problem{.code,.module_index,.declaration};}
pub fn usize count(ref Catalog p){return v::len<Template>(ref p.templates);}
pub fn Template template_at(ref Catalog p,usize at){assert(at<count(p));return *v::at<Template>(ref p.templates,at);}
pub fn Parameter parameter_at(ref Catalog p,usize at){assert(at<v::len<Parameter>(ref p.parameters));return *v::at<Parameter>(ref p.parameters,at);}
pub fn ref u8[] text(ref Catalog p,Text t){assert(t.start<=t.end && t.end<=bytes::len(ref p.names));return bytes::view(ref p.names)[t.start..t.end];}
pub fn ref bindings::Bindings packages(ref Catalog p){return ref p.bindings;}
pub fn bool scalar_constraints(ref Catalog p){return p.scalar_constraints;}
fn Result<Text,u32> retain(ref mut bytes::Buffer names,ref u8[] data,ref mut usize work){
    if(data.len>1048576 || bytes::len(names)>1048576-data.len || !budget::charge(work,data.len+1)){return .err(10);}
    let start=bytes::len(names);if(!bytes::append(names,data).is_ok){return .err(11);}return .ok(Text{.start,.end=bytes::len(names)});
}
fn bool builtin(ref u8[] name){return bytes::equal(name,b"bool") || bytes::equal(name,b"u8") || bytes::equal(name,b"i8") || bytes::equal(name,b"i16") || bytes::equal(name,b"u16") || bytes::equal(name,b"i32") || bytes::equal(name,b"i64") || bytes::equal(name,b"u32") || bytes::equal(name,b"u64") || bytes::equal(name,b"isize") || bytes::equal(name,b"usize") || bytes::equal(name,b"f32") || bytes::equal(name,b"f64") || bytes::equal(name,b"void") || bytes::equal(name,b"never") || bytes::equal(name,b"str") || bytes::equal(name,b"Region") || bytes::equal(name,b"Option") || bytes::equal(name,b"Result");}
fn bool generated(ref u8[] name){return name.len>=12 && bytes::equal(name[..12],b"_fy_generic_");}
fn Result<bool,u32> less(ref bytes::Buffer names,Text a,Text b,ref mut usize work){
    let left=bytes::view(names)[a.start..a.end];let right=bytes::view(names)[b.start..b.end];var usize i=0;
    while(i<left.len && i<right.len){if(!budget::charge(work,1)){return .err(10);}if(left[i]!=right[i]){return .ok(left[i]<right[i]);}i+=1;}return .ok(left.len<right.len);
}
fn u32 sorted(ref mut v::Vec<Template> rows,ref bytes::Buffer names,ref mut usize work){
    var usize i=1;while(i<v::len<Template>(rows)){let value=*v::at<Template>(rows,i);var usize at=i;
        while(at>0){if(!budget::charge(work,1)){return 10;}let previous=*v::at<Template>(rows,at-1);let before=switch(less(names,value.id,previous.id,work)){case .ok(value):{value}case .err(code):{return code;}};if(!before){break;}let slot=v::at_mut<Template>(rows,at);*slot=previous;at-=1;}
        let slot=v::at_mut<Template>(rows,at);*slot=value;i+=1;
    }return 0;
}
pub fn u32 current(ref bindings::Bindings p,usize module_index,ref syntax::Parsed source,ref mut usize work){
    if(module_index>=bindings::count(p)){return 9;}let raw=syntax::source_view(source);if(!budget::charge(work,raw.len+1)){return 10;}let m=bindings::module_at(p,module_index);
    if(raw.len!=m.source_size || syntax::problem_count(source)!=0){return 12;}let digest=hash::digest(raw);var usize i=0;while(i<8){if(digest[i]!=m.digest[i]){return 12;}i+=1;}return 0;
}
fn u32 namespace_names(ref syntax::Parsed p,ref mut usize work){
    let raw=syntax::source_view(p);var usize i=1;while(i<syntax::declaration_count(p)){if(!budget::charge(work,1)){return 10;}let d=syntax::declaration_at(p,i);
        // Source state is not in the generic expander's value/type namespace.
        if(d.kind!=2 && d.kind!=3 && d.kind!=4 && d.kind!=6 && d.kind!=7 && d.kind!=9 && d.kind!=11){i+=1;continue;}
        if(d.name_start>=d.name_end || d.name_end>raw.len || !budget::charge(work,d.name_end-d.name_start+1)){return 9;}let name=raw[d.name_start..d.name_end];if(generated(name)){return 4501;}
        var usize j=1;while(j<i){if(!budget::charge(work,1)){return 10;}let old=syntax::declaration_at(p,j);if(old.kind==2 || old.kind==3 || old.kind==4 || old.kind==6 || old.kind==7 || old.kind==9 || old.kind==11){if(old.name_start>=old.name_end || old.name_end>raw.len){return 9;}if(!budget::charge(work,name.len+1)){return 10;}if(bytes::equal(name,raw[old.name_start..old.name_end])){return 4501;}}j+=1;}i+=1;
    }return 0;
}
fn Result<Template,u32> collect_template(ref bindings::Bindings resolved,usize module_index,ref syntax::Parsed p,usize declaration,bool vector,ref mut bytes::Buffer names,ref mut v::Vec<Parameter> parameters,ref mut usize work){
    let d=syntax::declaration_at(p,declaration);let raw=syntax::source_view(p);let name=raw[d.name_start..d.name_end];
    if(d.owner_end!=0 || d.kind==3 && bytes::equal(name,b"source_location") || d.kind!=3 && (!vector || builtin(name) || bytes::equal(name,b"source_location"))){return .err(4501);}
    let count=syntax::generic_parameter_count(p,declaration);if(count==0 || count>8 || v::len<Parameter>(parameters)>32768-count){return .err(4501);}let first=v::len<Parameter>(parameters);var usize i=0;
    while(i<count){if(!budget::charge(work,count+1)){return .err(10);}let n=syntax::generic_parameter_at(p,declaration,i);if(n.start>=n.end || n.end>raw.len){return .err(9);}let parameter=raw[n.start..n.end];if(!budget::charge(work,parameter.len+1)){return .err(10);}if(!scanner::canonical_path(parameter) || builtin(parameter)){return .err(4501);}
        var usize j=first;while(j<v::len<Parameter>(parameters)){let old=*v::at<Parameter>(parameters,j);if(!budget::charge(work,parameter.len+1)){return .err(10);}if(bytes::equal(parameter,bytes::view(names)[old.name.start..old.name.end])){return .err(4501);}j+=1;}
        let saved=switch(retain(names,parameter,work)){case .ok(value):{value}case .err(code):{return .err(code);}};if(!v::push<Parameter>(parameters,Parameter{.name=saved,.scalar=d.kind==3 && !vector}).is_ok){return .err(11);}i+=1;
    }
    let owned_module=bindings::module_at(resolved,module_index);var bytes::Buffer id=bytes::empty();
    if(!bytes::append(ref mut id,bindings::text(resolved,owned_module.namespace)).is_ok || !bytes::append(ref mut id,b"::").is_ok || !bytes::append(ref mut id,bindings::text(resolved,owned_module.path)).is_ok || !bytes::append(ref mut id,b"::").is_ok || !bytes::append(ref mut id,name).is_ok){return .err(11);}
    var usize definition_work=0;let digest=switch(bindings::definition(resolved,module_index,p,declaration,ref mut definition_work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    if(!budget::charge(work,definition_work)){return .err(10);}if(!identity::valid_hex(bytes::view(ref digest),true)){return .err(9);}
    let saved_name=switch(retain(names,name,work)){case .ok(value):{value}case .err(code):{return .err(code);}};let saved_id=switch(retain(names,bytes::view(ref id),work)){case .ok(value):{value}case .err(code):{return .err(code);}};let definition=switch(retain(names,bytes::view(ref digest),work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    return .ok(Template{.module_index,.declaration,.kind=d.kind,.public=d.public,.name=saved_name,.id=saved_id,.definition,.first,.count,.start=d.start,.end=d.end,.name_start=d.name_start,.name_end=d.name_end});
}
// Uses the current source language policy: user functions are scalar-only;
// aggregate/nominal exceptions require the exact authenticated SDK Vec bytes.
pub fn Result<Catalog,Problem> build(bindings::Bindings resolved,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    if(bindings::count(ref resolved)!=v::len<syntax::Parsed>(sources)){return .err(fail(9,0,0));}var usize m=0;
    while(m<v::len<syntax::Parsed>(sources)){let code=current(ref resolved,m,v::at<syntax::Parsed>(sources,m),work);if(code!=0){return .err(fail(code,m,0));}m+=1;}
    var bytes::Buffer names=bytes::empty();var v::Vec<Template> rows=v::empty<Template>();var v::Vec<Parameter> parameters=v::empty<Parameter>();m=0;
    while(m<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,m);let owned_module=bindings::module_at(ref resolved,m);let path=bindings::text(ref resolved,owned_module.path);let raw=syntax::source_view(p);
        if(!budget::charge(work,path.len+raw.len+1)){return .err(fail(10,m,0));}let authentic=sdk::authentic(path,raw);if(sdk::reserved(path) && !authentic){return .err(fail(4501,m,0));}let vector=authentic && bytes::equal(path,b"std::vec");
        let valid=namespace_names(p,work);if(valid!=0){return .err(fail(valid,m,0));}var usize d_index=1;
        while(d_index<syntax::declaration_count(p)){if(!budget::charge(work,1)){return .err(fail(10,m,d_index));}let d=syntax::declaration_at(p,d_index);
            if(d.generic_parameters!=0){if((d.kind!=3 && d.kind!=4 && d.kind!=6) || v::len<Template>(ref rows)>=4096){return .err(fail(4501,m,d_index));}
                let row=switch(collect_template(ref resolved,m,p,d_index,vector,ref mut names,ref mut parameters,work)){case .ok(value):{value}case .err(code):{return .err(fail(code,m,d_index));}};if(!v::push<Template>(ref mut rows,row).is_ok){return .err(fail(11,m,d_index));}
            }d_index+=1;
        }m+=1;
    }let code=sorted(ref mut rows,ref names,work);if(code!=0){return .err(fail(code,0,0));}return .ok(Catalog{.bindings=resolved,.names,.templates=rows,.parameters,.scalar_constraints=true});
}
// Go's ordinary package preparation has no explicit parameter abilities;
// its authenticated Vec lane adds scalar abilities to user templates. Keep
// source admission (Parameter.scalar) separate from that serialized fact.
pub fn Result<Catalog,Problem> build_project(bindings::Bindings resolved,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    var out=switch(build(resolved,sources,work)){case .ok(p):{p}case .err(e):{return .err(e);}};var bool vector=false;var usize i=0;
    while(i<bindings::count(ref out.bindings)){let m=bindings::module_at(ref out.bindings,i);let path=bindings::text(ref out.bindings,m.path);if(!budget::charge(work,path.len+1)){return .err(fail(10,i,0));}if(bytes::equal(path,b"std::vec")){vector=true;}i+=1;}
    out.scalar_constraints=vector;return .ok(out);
}
// Missing targets are ok(0), so nongeneric named types need no fabricated edge.
// kind 3 selects functions; kind 0 selects structs/resources. Indices are 1-based.
pub fn Result<usize,u32> lookup(ref Catalog p,usize module_index,ref u8[] path,u32 kind,ref mut usize work){
    if(!budget::charge(work,path.len+1)){return .err(10);}if(module_index>=bindings::count(ref p.bindings) || (kind!=0 && kind!=3) || !scanner::canonical_path(path)){return .err(9);}
    var usize split=path.len;var usize i=0;while(i<path.len){if(path[i]==58){if(split!=path.len){return .ok(0);}split=i;i+=1;}i+=1;}
    var usize target=module_index;if(split!=path.len){let owned_module=bindings::module_at(ref p.bindings,module_index);target=bindings::count(ref p.bindings);i=0;
        while(i<owned_module.count){if(!budget::charge(work,split+1)){return .err(10);}let row=bindings::import_at(ref p.bindings,owned_module.first+i);if(bytes::equal(bindings::text(ref p.bindings,row.alias),path[..split])){target=row.target;break;}i+=1;}if(target==bindings::count(ref p.bindings)){return .ok(0);}
    }let name=if(split==path.len){path}else{path[split+2..]};i=0;
    while(i<count(p)){if(!budget::charge(work,1)){return .err(10);}let row=template_at(p,i);if(row.module_index==target && (kind==3 && row.kind==3 || kind==0 && row.kind!=3)){if(!budget::charge(work,name.len+1)){return .err(10);}if(bytes::equal(name,text(p,row.name))){if(target!=module_index && !row.public){return .err(13);}return .ok(i+1);}}i+=1;}
    return .ok(0);
}
