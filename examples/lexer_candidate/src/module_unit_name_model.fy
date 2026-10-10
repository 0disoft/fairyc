module module_unit_name_model;
import module_unit_body_ast as ast;
import module_signatures as signatures;
import module_ast_join as join;
import scanner as scanner;
import raw_view_intrinsics as raw;
import storage_intrinsics as storage;
import std::bytes as bytes;
import std::vec as v;

pub struct Frame {u32 kind;usize id;usize module_index;usize declaration;usize receiver_type;bool unsafe_function;bool sdk_vector;}
// Parameters, let, var, payload patterns, byte iteration, region and receiver.
pub struct Symbol {u32 kind;usize type;usize entry;usize pattern;usize scope;usize start;usize end;}
// kind 1 local; 2 canonical provider declaration; 3 closed builtin;
// 4 operation with its own lexical unsafe context (not its operand's context).
pub struct Reference {usize expression;usize source_index;u32 kind;usize symbol;usize module_index;usize declaration;u32 declaration_kind;u32 builtin;usize start;usize end;bool unsafe_context;}
pub struct Context {Frame frame;usize[128] locals;usize active;usize scope;usize next_scope;usize unsafe_depth;usize work;}
pub resource Resolved {Frame frame;v::Vec<Symbol> symbols;v::Vec<Reference> references;v::Vec<usize> uses;v::Vec<usize> entries;v::Vec<usize> patterns;usize work;cleanup{}}
pub fn join::Problem fail(ref Context c,u32 code,usize start,usize end){return join::Problem{.code,.module_index=c.frame.module_index,.start,.end};}
pub fn bool charge(ref mut Context c,usize amount){if(c.work>1048576 || amount>1048576-c.work){return false;}c.work+=amount;return true;}
pub fn Context context(Frame frame){return Context{.frame,.locals=[0;128],.active=0,.scope=1,.next_scope=2,.unsafe_depth=if(frame.unsafe_function){1}else{0},.work=0};}
pub fn ref u8[] spelling(ref ast::Ast a,ref scanner::Stream source,usize id){let n=ast::expression_at(a,id);return scanner::source_view(source)[n.start..n.end];}
pub fn bool simple(ref u8[] name){if(name.len==0){return false;}for(let u8 ch in name){if(!((ch>=65 && ch<=90) || (ch>=97 && ch<=122) || (ch>=48 && ch<=57) || ch==95)){return false;}}return true;}
fn bool named(ref scanner::Stream source,Symbol s,ref u8[] name){return if(s.kind==10){bytes::equal(name,b"this")}else{bytes::equal(scanner::source_view(source)[s.start..s.end],name)};}
pub fn Result<usize,join::Problem> add(ref scanner::Stream source,ref mut Context c,ref mut v::Vec<Symbol> symbols,Symbol item){
    let name=scanner::source_view(source)[item.start..item.end];if(bytes::equal(name,b"source_location")){return .err(fail(c,4401,item.start,item.end));}if(item.kind!=10 && !simple(name)){return .err(fail(c,3,item.start,item.end));}var usize i=c.active;
    while(i>0){i-=1;if(!charge(c,1)){return .err(fail(c,10,item.start,item.end));}let s=*v::at<Symbol>(symbols,c.locals[i]-1);if(s.scope==c.scope && (if(item.kind==10){s.kind==10}else{named(source,s,name)})){return .err(fail(c,2,item.start,item.end));}}
    if(c.active>=128 || v::len<Symbol>(symbols)>=16384){return .err(fail(c,10,item.start,item.end));}if(!v::push<Symbol>(symbols,item).is_ok){return .err(fail(c,11,item.start,item.end));}let usize id=v::len<Symbol>(symbols);let usize slot=c.active;c.locals[slot]=id;c.active+=1;return .ok(id);
}
pub fn join::Problem reference(ref ast::Ast a,ref scanner::Stream source,ref signatures::Program metadata,usize expression,ref mut Context c,ref v::Vec<Symbol> symbols,ref mut v::Vec<Reference> references,ref mut v::Vec<usize> uses){
    let n=ast::expression_at(a,expression);let name=spelling(a,source,expression);var Reference r=Reference{.expression,.source_index=ast::expression_origin(a,expression),.kind=0,.symbol=0,.module_index=0,.declaration=0,.declaration_kind=0,.builtin=0,.start=n.start,.end=n.end,.unsafe_context=c.unsafe_depth!=0};var usize i=c.active;
    if(simple(name)){while(i>0){i-=1;if(!charge(c,1)){return fail(c,10,n.start,n.end);}let s=*v::at<Symbol>(symbols,c.locals[i]-1);if(named(source,s,name)){r.kind=1;r.symbol=c.locals[i];break;}}}
    if(r.kind==0){let u32 raw_kind=raw::named(name);let u32 storage_kind=if(c.frame.sdk_vector){storage::named(name)}else{0};
        if(raw_kind!=0 || storage_kind!=0 || bytes::equal(name,b"source_location")){if(bytes::equal(name,b"source_location")){var bool called=false;var usize parent=expression+1;while(parent<=ast::expression_count(a)){if(!charge(c,1)){return fail(c,10,n.start,n.end);}let candidate=ast::expression_at(a,parent);if(candidate.kind==6 && candidate.left==expression){called=true;break;}parent+=1;}if(!called){return fail(c,4401,n.start,n.end);}}r.kind=3;r.builtin=if(storage_kind!=0){100+storage_kind}else if(raw_kind!=0){raw_kind}else{200};}
        else{var usize probes=0;let found=signatures::declaration_lookup_measured(metadata,c.frame.module_index,name,ref mut probes);if(!charge(c,probes)){return fail(c,10,n.start,n.end);}if(found.code!=0){return fail(c,if(found.code==5){1}else{found.code},n.start,n.end);}let d=signatures::declaration_at(metadata,found.item);r.kind=2;r.module_index=d.module_index;r.declaration=d.declaration;r.declaration_kind=d.kind;}
    }
    if(v::len<Reference>(references)>=16384){return fail(c,10,n.start,n.end);}if(!v::push<Reference>(references,r).is_ok){return fail(c,11,n.start,n.end);}let slot=v::at_mut<usize>(uses,expression-1);*slot=v::len<Reference>(references);return fail(c,0,0,0);
}
pub fn Resolved assemble(Frame frame,v::Vec<Symbol> symbols,v::Vec<Reference> references,v::Vec<usize> uses,v::Vec<usize> entries,v::Vec<usize> patterns,usize work){return Resolved{.frame,.symbols,.references,.uses,.entries,.patterns,.work};}
pub fn join::Problem operation(ref ast::Ast a,usize expression,ref mut Context c,ref mut v::Vec<Reference> references,ref mut v::Vec<usize> uses){
    let n=ast::expression_at(a,expression);if(n.kind!=23 || !charge(c,1) || v::len<Reference>(references)>=16384){return fail(c,10,n.start,n.end);}
    if(!v::push<Reference>(references,Reference{.expression,.source_index=ast::expression_origin(a,expression),.kind=4,.symbol=0,.module_index=0,.declaration=0,.declaration_kind=0,.builtin=0,.start=n.start,.end=n.end,.unsafe_context=c.unsafe_depth!=0}).is_ok){return fail(c,11,n.start,n.end);}
    let slot=v::at_mut<usize>(uses,expression-1);*slot=v::len<Reference>(references);return fail(c,0,0,0);
}
pub fn usize local_count(ref Resolved p){return v::len<Symbol>(ref p.symbols);}
pub fn usize reference_count(ref Resolved p){return v::len<Reference>(ref p.references);}
pub fn usize work(ref Resolved p){return p.work;}
pub fn Symbol symbol_at(ref Resolved p,usize id){return *v::at<Symbol>(ref p.symbols,id-1);}
pub fn usize use_count(ref Resolved p){return v::len<usize>(ref p.uses);}
pub fn usize use_at(ref Resolved p,usize expression){return *v::at<usize>(ref p.uses,expression-1);}
pub fn Reference reference_at(ref Resolved p,usize id){return *v::at<Reference>(ref p.references,id-1);}
pub fn usize entry_symbol(ref Resolved p,usize entry){return *v::at<usize>(ref p.entries,entry-1);}
pub fn usize pattern_symbol(ref Resolved p,usize expression){return *v::at<usize>(ref p.patterns,expression-1);}
