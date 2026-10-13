module generic_signature_diagnostics;
import generic_source_plan as plan;
import generic_source_plan_input as input;
import generic_source_catalog as catalog;
import generic_concrete_types as concrete;
import generic_package_bindings as bindings;
import project_path as budget;
import project_manifest_values as lexical;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Source-only evidence. No unsupported scalar is interned into executable types.
// Names are detached from both the plan and original source owners.
pub struct Diagnostic{usize module_index;usize start;usize end;usize name_start;usize name_end;}
pub resource Reports{bytes::Buffer names;v::Vec<Diagnostic> rows;cleanup{}}
pub fn usize count(ref Reports p){return v::len<Diagnostic>(ref p.rows);}
pub fn Diagnostic at(ref Reports p,usize index){assert(index<count(p));return *v::at<Diagnostic>(ref p.rows,index);}
pub fn ref u8[] name(ref Reports p,usize index){let d=at(p,index);return bytes::view(ref p.names)[d.name_start..d.name_end];}
pub fn Reports empty(){return Reports{.names=bytes::empty(),.rows=v::empty<Diagnostic>()};}
fn bool unsupported(ref u8[] name){return bytes::equal(name,b"i8") || bytes::equal(name,b"i16") || bytes::equal(name,b"isize") || bytes::equal(name,b"u16") || bytes::equal(name,b"u64") || bytes::equal(name,b"f32") || bytes::equal(name,b"f64");}
fn Result<usize,u32> parameter(ref catalog::Catalog templates,catalog::Template row,ref u8[] name,ref mut usize work){var usize i=0;while(i<row.count){let p=catalog::parameter_at(templates,row.first+i);let text=catalog::text(templates,p.name);if(!budget::charge(work,name.len+text.len+1)){return .err(10);}if(bytes::equal(name,text)){return .ok(i+1);}i+=1;}return .ok(0);}
fn Result<bool,u32> contains(ref plan::Plan original,plan::Node selected,ref syntax::Parsed p,usize root,usize depth,ref mut usize work){
    if(!budget::charge(work,1)){return .err(10);}if(root==0){return .ok(false);}if(root>syntax::type_count(p)){return .err(9);}if(depth>=64){return .err(10);}let n=syntax::type_at(p,root);
    if(n.kind==1){let templates=input::templates(plan::input(original));let row=catalog::template_at(templates,selected.template_index);let text=syntax::type_text(p,root);let param=switch(parameter(templates,row,text,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
        if(param!=0){if(param>selected.count){return .err(9);}let types=input::types(plan::input(original));let id=selected.arguments[param-1];if(id==0 || id>concrete::count(types)){return .err(9);}let t=concrete::at(types,id-1);return .ok(t.kind==1 && t.scalar==0);}return .ok(unsupported(text));
    }
    let left=switch(contains(original,selected,p,n.left,depth+1,work)){case .ok(value):{value}case .err(code):{return .err(code);}};if(left){return .ok(true);}
    // Array lengths are expressions, not signature type arguments.
    if(n.kind==8){return .ok(false);}return contains(original,selected,p,n.right,depth+1,work);
}
fn Result<i32,u32> compare(ref bytes::Buffer names,Diagnostic old,usize module_index,usize start,usize end,ref u8[] text,ref mut usize work){if(!budget::charge(work,text.len+old.name_end-old.name_start+1)){return .err(10);}if(module_index!=old.module_index){return .ok(if(module_index<old.module_index){-1}else{1});}if(start!=old.start){return .ok(if(start<old.start){-1}else{1});}if(end!=old.end){return .ok(if(end<old.end){-1}else{1});}let previous=bytes::view(names)[old.name_start..old.name_end];return .ok(if(bytes::equal(text,previous)){0}else if(lexical::before(text,previous)){-1}else{1});}
fn u32 append(ref mut bytes::Buffer names,ref mut v::Vec<Diagnostic> rows,usize module_index,usize start,usize end,ref u8[] text,ref mut usize work){
    var usize i=0;while(i<v::len<Diagnostic>(rows)){let old=*v::at<Diagnostic>(rows,i);let order=switch(compare(names,old,module_index,start,end,text,work)){case .ok(value):{value}case .err(code):{return code;}};if(order==0){return 0;}if(order<0){break;}i+=1;}
    if(v::len<Diagnostic>(rows)>=4096 || text.len>1048576 || bytes::len(names)>1048576-text.len || !budget::charge(work,text.len+1)){return 10;}let first=bytes::len(names);if(!bytes::append(names,text).is_ok){return 11;}let d=Diagnostic{.module_index,.start,.end,.name_start=first,.name_end=bytes::len(names)};if(!v::push<Diagnostic>(rows,d).is_ok){return 11;}var usize at=v::len<Diagnostic>(rows)-1;while(at>i){if(!budget::charge(work,1)){return 10;}let old=*v::at<Diagnostic>(rows,at-1);{let slot=v::at_mut<Diagnostic>(rows,at);*slot=old;}at-=1;}{let slot=v::at_mut<Diagnostic>(rows,i);*slot=d;}return 0;
}
fn u32 signature(ref plan::Plan original,plan::Node selected,ref syntax::Parsed p,usize root,ref mut bytes::Buffer names,ref mut v::Vec<Diagnostic> rows,ref mut usize work){
    let rejected=switch(contains(original,selected,p,root,0,work)){case .ok(value):{value}case .err(code):{return code;}};if(!rejected){return 0;}let n=syntax::type_at(p,root);let templates=input::templates(plan::input(original));let row=catalog::template_at(templates,selected.template_index);let text=syntax::type_text(p,root);
    if(n.kind==1){let param=switch(parameter(templates,row,text,work)){case .ok(value):{value}case .err(code):{return code;}};if(param!=0){let types=input::types(plan::input(original));let t=concrete::at(types,selected.arguments[param-1]-1);return append(names,rows,row.module_index,n.start,n.end,concrete::text(types,t.name),work);}}
    // Go substitutes children but retains the written Text of a composite type.
    return append(names,rows,row.module_index,n.start,n.end,text,work);
}
// Reports use original input module order, then span and feature, with duplicate
// instances compacted. A CLI can order modules by its retained physical paths.
// This projects function signatures only; body diagnostics remain a separate phase.
pub fn Result<Reports,u32> collect(ref plan::Plan original,ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}let templates=input::templates(plan::input(original));let packages=catalog::packages(templates);if(bindings::count(packages)!=v::len<syntax::Parsed>(sources)){return .err(9);}var usize i=0;
    while(i<v::len<syntax::Parsed>(sources)){let code=catalog::current(packages,i,v::at<syntax::Parsed>(sources,i),work);if(code!=0){return .err(code);}i+=1;}
    var bytes::Buffer names=bytes::empty();var v::Vec<Diagnostic> rows=v::empty<Diagnostic>();i=0;while(i<plan::count(original)){if(!budget::charge(work,1)){return .err(10);}let selected=plan::at(original,i);if(selected.state!=2 || selected.template_index>=catalog::count(templates)){return .err(9);}let row=catalog::template_at(templates,selected.template_index);if(row.kind==3){let p=v::at<syntax::Parsed>(sources,row.module_index);let d=syntax::declaration_at(p,row.declaration);var u32 code=signature(original,selected,p,d.result_type,ref mut names,ref mut rows,work);if(code!=0){return .err(code);}var usize j=d.entry_start;
        while(j<d.entry_start+d.entry_count){if(!budget::charge(work,1)){return .err(10);}let e=syntax::entry_at(p,j);if(e.kind==1){code=signature(original,selected,p,e.type_root,ref mut names,ref mut rows,work);if(code!=0){return .err(code);}}j+=1;}}
        i+=1;
    }return .ok(Reports{.names,.rows});
}
