module module_constant_model;
import module_unit_value_model as m;
import syntax as syntax;
import type_state as scalars;
import std::bytes as bytes;
import std::vec as v;

pub struct Value{u32 scalar;usize bits;}
pub struct Entry{usize module_index;usize declaration;usize start;usize end;usize expression;bool callable;usize parameter_start;usize parameter_count;Value value;}
pub struct Parameter{usize start;usize end;u32 scalar;}
pub struct Counter{usize owner;usize steps;}
pub fn m::Problem fail(usize module_index,u32 code,usize start,usize end){return m::Problem{.code,.module_index,.start,.end,.expected=0,.actual=0};}
pub fn m::Problem ok(){return fail(0,0,0,0);}
pub fn bool admitted(u32 scalar){return scalar==1 || scalar==2 || scalar==3 || scalar==4 || scalar==6;}
pub fn bool charge(ref mut usize work){return m::charge(work,1);}
pub fn Result<usize,m::Problem> find(ref v::Vec<syntax::Parsed> sources,ref v::Vec<Entry> entries,usize module_index,ref u8[] name,usize start,usize end,ref mut usize work){
    let source=v::at<syntax::Parsed>(sources,module_index);var usize i=0;while(i<v::len<Entry>(entries)){if(!charge(work)){return .err(fail(module_index,4304,start,end));}let entry=*v::at<Entry>(entries,i);if(entry.module_index==module_index){let d=syntax::declaration_at(source,entry.declaration-1);if(bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],name)){return .ok(i+1);}}i+=1;}
    if(bytes::equal(name,b"source_location")){return .err(fail(module_index,4401,start,end));}
    i=1;while(i<syntax::declaration_count(source)){if(!charge(work)){return .err(fail(module_index,4304,start,end));}let d=syntax::declaration_at(source,i);if(bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],name)){return .err(fail(module_index,4302,start,end));}i+=1;}return .err(fail(module_index,101,start,end));
}
pub fn usize literal_bits(ref u8[] raw){var usize base=10;var usize index=0;if(raw.len>=2 && raw[0]==48){if(raw[1]==120 || raw[1]==88){base=16;index=2;}else if(raw[1]==98 || raw[1]==66){base=2;index=2;}}var usize bits=0;while(index<raw.len){let byte=raw[index];index+=1;if(byte==95){continue;}let usize digit=if(byte>=48 && byte<=57){(byte-48) as usize}else if(byte>=65 && byte<=70){(byte-55) as usize}else{(byte-87) as usize};bits=bits*base+digit;}return bits;}
pub fn Result<bool,m::Problem> signed_minimum(ref syntax::Parsed source,usize module_index,usize id,u32 scalar,ref mut usize work){
    if(scalar!=1 && scalar!=2){return .ok(false);}var usize child=id;var usize depth=0;
    while(child!=0 && syntax::expression_at(source,child).kind==3){let e=syntax::expression_at(source,child);if(depth>=64 || !charge(work)){return .err(fail(module_index,4304,e.start,e.end));}child=e.left;depth+=1;}
    if(child==0 || syntax::expression_at(source,child).kind!=1 || syntax::literal_kind(source,child)!=5){return .ok(false);}let raw=syntax::expression_text(source,child);return .ok(scalars::fits(raw,scalar,true) && literal_bits(raw)==if(scalar==1){2147483648}else{9223372036854775808});
}
pub fn Result<bool,m::Problem> flexible(ref syntax::Parsed source,usize module_index,usize id,ref mut usize work,usize depth){
    let e=syntax::expression_at(source,id);if(depth>=64 || !charge(work)){return .err(fail(module_index,4304,e.start,e.end));}
    if(e.kind==1){return .ok(syntax::literal_kind(source,id)==5);}let raw=syntax::expression_text(source,id);
    if(e.kind==3 || e.kind==4 && (bytes::equal(raw,b"+") || bytes::equal(raw,b"-") || bytes::equal(raw,b"~"))){return flexible(source,module_index,e.left,work,depth+1);}
    if(e.kind==5 && scalars::operation(raw)){let left=switch(flexible(source,module_index,e.left,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};if(!left){return .ok(false);}return flexible(source,module_index,e.right,work,depth+1);}
    if(e.kind==12){let tail=switch(tail(source,module_index,e.left)){case .ok(n):{n}case .err(p):{return .err(p);}};return flexible(source,module_index,tail,work,depth+1);}
    if(e.kind==13){let branch=syntax::expression_at(source,e.right);let child=switch(tail(source,module_index,branch.left)){case .ok(n):{n}case .err(p):{return .err(p);}};let left=switch(flexible(source,module_index,child,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};if(!left){return .ok(false);}return flexible(source,module_index,branch.right,work,depth+1);}
    return .ok(false);
}

pub fn Result<usize,m::Problem> parameter(ref syntax::Parsed source,ref v::Vec<Entry> entries,ref v::Vec<Parameter> parameters,usize context,ref u8[] name,usize start,usize end,ref mut usize work){
    if(context==0){return .ok(0);}let row=*v::at<Entry>(entries,context-1);var usize i=0;
    while(i<row.parameter_count){if(!charge(work)){return .err(fail(row.module_index,4304,start,end));}let p=*v::at<Parameter>(parameters,row.parameter_start+i);if(bytes::equal(syntax::source_view(source)[p.start..p.end],name)){return .ok(row.parameter_start+i+1);}i+=1;}return .ok(0);
}
pub fn Result<usize,m::Problem> callee(ref v::Vec<syntax::Parsed> sources,ref v::Vec<Entry> entries,ref v::Vec<Parameter> parameters,usize module_index,usize id,usize context,ref mut usize work){
    let source=v::at<syntax::Parsed>(sources,module_index);let e=syntax::expression_at(source,id);let path=syntax::expression_at(source,e.left);
    if(path.kind!=2){return .err(fail(module_index,4302,e.start,e.end));}let raw=syntax::expression_text(source,e.left);
    let binding=switch(parameter(source,entries,parameters,context,raw,path.start,path.end,work)){case .ok(n):{n}case .err(p):{return .err(p);}};
    if(binding!=0 || bytes::equal(raw,b"source_location")){return .err(fail(module_index,4302,e.start,e.end));}
    let target=switch(find(sources,entries,module_index,raw,path.start,path.end,work)){case .ok(n):{n}case .err(p):{return .err(if(p.code==4302){fail(module_index,4302,e.start,e.end)}else{p});}};
    let row=*v::at<Entry>(entries,target-1);if(!row.callable){return .err(fail(module_index,4302,e.start,e.end));}
    if(syntax::call_argument_count(source,id)!=row.parameter_count){return .err(fail(module_index,1010,e.start,e.end));}
    var usize i=0;while(i<row.parameter_count){if(!charge(work)){return .err(fail(module_index,4304,e.start,e.end));}let argument=syntax::call_argument_entry(source,id,i);if(syntax::call_argument_labeled(source,argument)){let p=*v::at<Parameter>(parameters,row.parameter_start+i);if(!bytes::equal(syntax::expression_text(source,argument),syntax::source_view(source)[p.start..p.end])){let a=syntax::expression_at(source,argument);return .err(fail(module_index,1014,a.start,a.end));}}i+=1;}return .ok(target);
}
pub fn Result<usize,m::Problem> tail(ref syntax::Parsed source,usize module_index,usize block){
    let b=syntax::statement_at(source,block);return if(b.kind==1 && b.second==0 && b.expression!=0){.ok(b.expression)}else{.err(fail(module_index,4302,b.start,b.end))};
}
