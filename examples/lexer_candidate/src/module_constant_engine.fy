module module_constant_engine;
import module_unit_value_model as m;
import module_constant_arithmetic as arithmetic;
import module_const_eval_declarations as const_eval;
import module_signatures as signatures;
import syntax as syntax;
import type_state as scalars;
import std::bytes as bytes;
import std::vec as v;

pub struct Value{u32 scalar;usize bits;}
struct Entry{usize module_index;usize declaration;usize start;usize end;Value value;}
pub resource Plan{v::Vec<Entry> entries;usize work;cleanup{}}
struct Counter{usize owner;usize steps;}
fn m::Problem fail(usize module_index,u32 code,usize start,usize end){return m::Problem{.code,.module_index,.start,.end,.expected=0,.actual=0};}
fn m::Problem ok(){return fail(0,0,0,0);}
fn bool admitted(u32 scalar){return scalar==1 || scalar==2 || scalar==3 || scalar==4 || scalar==6;}
fn bool charge(ref mut usize work){return m::charge(work,1);}
fn Result<usize,m::Problem> find(ref v::Vec<syntax::Parsed> sources,ref v::Vec<Entry> entries,usize module_index,ref u8[] name,usize start,usize end,ref mut usize work){
    let source=v::at<syntax::Parsed>(sources,module_index);var usize i=0;while(i<v::len<Entry>(entries)){if(!charge(work)){return .err(fail(module_index,4304,start,end));}let entry=*v::at<Entry>(entries,i);if(entry.module_index==module_index){let d=syntax::declaration_at(source,entry.declaration-1);if(bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],name)){return .ok(i+1);}}i+=1;}
    if(bytes::equal(name,b"source_location")){return .err(fail(module_index,4401,start,end));}
    i=1;while(i<syntax::declaration_count(source)){if(!charge(work)){return .err(fail(module_index,4304,start,end));}let d=syntax::declaration_at(source,i);if(bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],name)){return .err(fail(module_index,4302,start,end));}i+=1;}return .err(fail(module_index,101,start,end));
}
// Complete dependency validation precedes evaluation, including dead branches.
fn m::Problem visit(ref v::Vec<syntax::Parsed> sources,ref v::Vec<Entry> entries,usize entry,ref mut v::Vec<u32> states,ref mut usize work){
    let row=*v::at<Entry>(entries,entry-1);let source=v::at<syntax::Parsed>(sources,row.module_index);let d=syntax::declaration_at(source,row.declaration-1);let state=*v::at<u32>(states,entry-1);
    if(state==2){return ok();}if(state==1){return fail(row.module_index,4304,d.name_start,d.name_end);}{let cell=v::at_mut<u32>(states,entry-1);*cell=1;}
    let p=validate(sources,entries,row.module_index,d.expression,states,work,0);if(p.code!=0){return p;}{let cell=v::at_mut<u32>(states,entry-1);*cell=2;}return ok();
}
fn m::Problem validate(ref v::Vec<syntax::Parsed> sources,ref v::Vec<Entry> entries,usize module_index,usize id,ref mut v::Vec<u32> states,ref mut usize work,usize depth){
    let source=v::at<syntax::Parsed>(sources,module_index);if(id==0 || id>syntax::expression_count(source)){return fail(module_index,4302,0,0);}let e=syntax::expression_at(source,id);if(depth>=64 || !charge(work)){return fail(module_index,4304,e.start,e.end);}let raw=syntax::expression_text(source,id);
    if(e.kind==1){return if(syntax::literal_kind(source,id)==5 || bytes::equal(raw,b"true") || bytes::equal(raw,b"false")){ok()}else{fail(module_index,4302,e.start,e.end)};}
    if(e.kind==2){let target=switch(find(sources,entries,module_index,raw,e.start,e.end,work)){case .ok(n):{n}case .err(p):{return p;}};return visit(sources,entries,target,states,work);}
    if(e.kind==3){return validate(sources,entries,module_index,e.left,states,work,depth+1);}
    if(e.kind==4 && (bytes::equal(raw,b"+") || bytes::equal(raw,b"-") || bytes::equal(raw,b"!") || bytes::equal(raw,b"~"))){return validate(sources,entries,module_index,e.left,states,work,depth+1);}
    if(e.kind==5 && (scalars::operation(raw) || scalars::comparison(raw) || bytes::equal(raw,b"&&") || bytes::equal(raw,b"||"))){let p=validate(sources,entries,module_index,e.left,states,work,depth+1);return if(p.code!=0){p}else{validate(sources,entries,module_index,e.right,states,work,depth+1)};}
    return fail(module_index,4302,e.start,e.end);
}
fn m::Problem step(usize module_index,syntax::Expression e,ref v::Vec<usize> owners,ref mut v::Vec<Counter> counters,ref mut usize work){
    if(!charge(work)){return fail(module_index,4304,e.start,e.end);}let usize owner=if(v::len<usize>(owners)==0){0}else{*v::at<usize>(owners,module_index)};var usize i=0;
    while(i<v::len<Counter>(counters)){if(!charge(work)){return fail(module_index,4304,e.start,e.end);}let current=*v::at<Counter>(counters,i);if(current.owner==owner){if(current.steps>=65536){return fail(module_index,4304,e.start,e.end);}let cell=v::at_mut<Counter>(counters,i);cell.steps+=1;return ok();}i+=1;}
    if(!v::push<Counter>(counters,Counter{.owner,.steps=1}).is_ok){return fail(module_index,11,e.start,e.end);}return ok();
}
fn usize literal_bits(ref u8[] raw){var usize base=10;var usize index=0;if(raw.len>=2 && raw[0]==48){if(raw[1]==120 || raw[1]==88){base=16;index=2;}else if(raw[1]==98 || raw[1]==66){base=2;index=2;}}var usize bits=0;while(index<raw.len){let byte=raw[index];index+=1;if(byte==95){continue;}let usize digit=if(byte>=48 && byte<=57){(byte-48) as usize}else if(byte>=65 && byte<=70){(byte-55) as usize}else{(byte-87) as usize};bits=bits*base+digit;}return bits;}
fn Result<bool,m::Problem> signed_minimum(ref syntax::Parsed source,usize module_index,usize id,u32 scalar,ref mut usize work){
    if(scalar!=1 && scalar!=2){return .ok(false);}var usize child=id;var usize depth=0;
    while(child!=0 && syntax::expression_at(source,child).kind==3){let e=syntax::expression_at(source,child);if(depth>=64 || !charge(work)){return .err(fail(module_index,4304,e.start,e.end));}child=e.left;depth+=1;}
    if(child==0 || syntax::expression_at(source,child).kind!=1 || syntax::literal_kind(source,child)!=5){return .ok(false);}let raw=syntax::expression_text(source,child);return .ok(scalars::fits(raw,scalar,true) && literal_bits(raw)==if(scalar==1){2147483648}else{9223372036854775808});
}
fn Result<bool,m::Problem> flexible(ref syntax::Parsed source,usize module_index,usize id,ref mut usize work,usize depth){
    let e=syntax::expression_at(source,id);if(depth>=64 || !charge(work)){return .err(fail(module_index,4304,e.start,e.end));}
    if(e.kind==1){return .ok(syntax::literal_kind(source,id)==5);}let raw=syntax::expression_text(source,id);
    if(e.kind==3 || e.kind==4 && (bytes::equal(raw,b"+") || bytes::equal(raw,b"-") || bytes::equal(raw,b"~"))){return flexible(source,module_index,e.left,work,depth+1);}
    if(e.kind==5 && scalars::operation(raw)){let left=switch(flexible(source,module_index,e.left,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};if(!left){return .ok(false);}return flexible(source,module_index,e.right,work,depth+1);}return .ok(false);
}
// All branches are type checked even when evaluation will short circuit.
fn Result<u32,m::Problem> typecheck(ref v::Vec<syntax::Parsed> sources,ref v::Vec<Entry> entries,usize module_index,usize id,u32 expected,ref mut usize work,usize depth){
    let source=v::at<syntax::Parsed>(sources,module_index);let e=syntax::expression_at(source,id);if(depth>=64 || !charge(work)){return .err(fail(module_index,4304,e.start,e.end));}let raw=syntax::expression_text(source,id);
    if(e.kind==1){if(bytes::equal(raw,b"true") || bytes::equal(raw,b"false")){return .ok(6);}let u32 scalar=if(admitted(expected) && expected!=6){expected}else{1};return if(scalars::fits(raw,scalar,false)){.ok(scalar)}else{.err(fail(module_index,1013,e.start,e.end))};}
    if(e.kind==2){let target=switch(find(sources,entries,module_index,raw,e.start,e.end,work)){case .ok(n):{n}case .err(p):{return .err(p);}};return .ok(v::at<Entry>(entries,target-1).value.scalar);}
    if(e.kind==3){return typecheck(sources,entries,module_index,e.left,expected,work,depth+1);}
    if(e.kind==4){if(bytes::equal(raw,b"-")){let minimum=switch(signed_minimum(source,module_index,e.left,expected,work)){case .ok(v):{v}case .err(p):{return .err(p);}};if(minimum){return .ok(expected);}}
        let scalar=switch(typecheck(sources,entries,module_index,e.left,expected,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};if(bytes::equal(raw,b"!")){let child=syntax::expression_at(source,e.left);return if(scalar==6){.ok(6)}else{.err(fail(module_index,1007,child.start,child.end))};}let valid=if(bytes::equal(raw,b"~")){scalar!=6}else{scalar==1 || scalar==2};return if(valid){.ok(scalar)}else{.err(fail(module_index,1006,e.start,e.end))};
    }
    if(e.kind==5){var u32 left=0;var u32 right=0;let logical=bytes::equal(raw,b"&&") || bytes::equal(raw,b"||");let a=switch(flexible(source,module_index,e.left,work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};let b=switch(flexible(source,module_index,e.right,work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};
        if(!logical && a && !b){right=switch(typecheck(sources,entries,module_index,e.right,expected,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};left=switch(typecheck(sources,entries,module_index,e.left,right,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};}else{left=switch(typecheck(sources,entries,module_index,e.left,expected,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};right=switch(typecheck(sources,entries,module_index,e.right,left,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};}
        if(logical){let bad=syntax::expression_at(source,if(left!=6){e.left}else{e.right});return if(left==6 && right==6){.ok(6)}else{.err(fail(module_index,1007,bad.start,bad.end))};}if(left!=right){let bad=syntax::expression_at(source,e.right);return .err(fail(module_index,1006,bad.start,bad.end));}if(scalars::comparison(raw)){return if(left!=6 || scalars::equal(raw)){.ok(6)}else{.err(fail(module_index,1006,e.start,e.end))};}return if(left!=6){.ok(left)}else{.err(fail(module_index,1006,e.start,e.end))};
    }return .err(fail(module_index,4302,e.start,e.end));
}
fn Result<Value,m::Problem> constant(ref v::Vec<syntax::Parsed> sources,ref mut v::Vec<Entry> entries,usize entry,ref mut v::Vec<u32> states,ref v::Vec<usize> owners,ref mut v::Vec<Counter> counters,ref mut usize work){
    let row=*v::at<Entry>(entries,entry-1);if(*v::at<u32>(states,entry-1)==2){return .ok(row.value);}let source=v::at<syntax::Parsed>(sources,row.module_index);let d=syntax::declaration_at(source,row.declaration-1);
    if(*v::at<u32>(states,entry-1)==1){return .err(fail(row.module_index,4304,d.name_start,d.name_end));}{let cell=v::at_mut<u32>(states,entry-1);*cell=1;}
    let value=switch(evaluate(sources,entries,row.module_index,d.expression,row.value.scalar,states,owners,counters,work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};
    if(value.scalar!=row.value.scalar){let e=syntax::expression_at(source,d.expression);return .err(fail(row.module_index,1006,e.start,e.end));}{let cell=v::at_mut<Entry>(entries,entry-1);cell.value=value;}{let cell=v::at_mut<u32>(states,entry-1);*cell=2;}return .ok(value);
}
fn Result<Value,m::Problem> evaluate(ref v::Vec<syntax::Parsed> sources,ref mut v::Vec<Entry> entries,usize module_index,usize id,u32 scalar,ref mut v::Vec<u32> states,ref v::Vec<usize> owners,ref mut v::Vec<Counter> counters,ref mut usize work,usize depth){
    let source=v::at<syntax::Parsed>(sources,module_index);let e=syntax::expression_at(source,id);if(depth>=64){return .err(fail(module_index,4304,e.start,e.end));}let charged=step(module_index,e,owners,counters,work);if(charged.code!=0){return .err(charged);}let raw=syntax::expression_text(source,id);
    if(e.kind==1){if(bytes::equal(raw,b"true") || bytes::equal(raw,b"false")){return .ok(Value{.scalar=6,.bits=if(bytes::equal(raw,b"true")){1}else{0}});}let u32 selected=if(admitted(scalar) && scalar!=6){scalar}else{1};if(!scalars::fits(raw,selected,false)){return .err(fail(module_index,1013,e.start,e.end));}return .ok(Value{.scalar=selected,.bits=literal_bits(raw)});}
    if(e.kind==2){let target=switch(find(sources,entries,module_index,raw,e.start,e.end,work)){case .ok(n):{n}case .err(p):{return .err(p);}};return constant(sources,entries,target,states,owners,counters,work);}
    if(e.kind==3){return evaluate(sources,entries,module_index,e.left,scalar,states,owners,counters,work,depth+1);}
    if(e.kind==4){if(bytes::equal(raw,b"-")){let minimum=switch(signed_minimum(source,module_index,e.left,scalar,work)){case .ok(v):{v}case .err(p):{return .err(p);}};if(minimum){let child=syntax::expression_at(source,e.left);let p=step(module_index,child,owners,counters,work);if(p.code!=0){return .err(p);}return .ok(Value{.scalar,.bits=if(scalar==1){2147483648}else{9223372036854775808}});}}
        let value=switch(evaluate(sources,entries,module_index,e.left,scalar,states,owners,counters,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};let computed=arithmetic::prefix(value.scalar,raw,value.bits);return if(computed.code==0){.ok(Value{.scalar=computed.scalar,.bits=computed.bits})}else{.err(fail(module_index,computed.code,e.start,e.end))};
    }
    if(e.kind==5){var Value left=Value{.scalar=0,.bits=0};var Value right=left;let logical=bytes::equal(raw,b"&&") || bytes::equal(raw,b"||");let numeric=scalars::operation(raw) || scalars::comparison(raw);let a=switch(flexible(source,module_index,e.left,work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};let b=switch(flexible(source,module_index,e.right,work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};
        if(numeric && a && !b){right=switch(evaluate(sources,entries,module_index,e.right,scalar,states,owners,counters,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};left=switch(evaluate(sources,entries,module_index,e.left,right.scalar,states,owners,counters,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};}
        else{left=switch(evaluate(sources,entries,module_index,e.left,scalar,states,owners,counters,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};if(logical){if(left.scalar!=6){let child=syntax::expression_at(source,e.left);return .err(fail(module_index,1007,child.start,child.end));}if(bytes::equal(raw,b"&&") && left.bits==0 || bytes::equal(raw,b"||") && left.bits==1){return .ok(left);}}right=switch(evaluate(sources,entries,module_index,e.right,left.scalar,states,owners,counters,work,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};}
        if(left.scalar!=right.scalar){let child=syntax::expression_at(source,e.right);return .err(fail(module_index,1006,child.start,child.end));}let computed=arithmetic::binary(left.scalar,raw,left.bits,right.bits);return if(computed.code==0){.ok(Value{.scalar=computed.scalar,.bits=computed.bits})}else{.err(fail(module_index,computed.code,e.start,e.end))};
    }return .err(fail(module_index,4302,e.start,e.end));
}
// owners are package-instance coordinates, not module-name spellings.
pub fn Result<Plan,m::Problem> build_owned(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners){
    if(v::len<usize>(owners)!=0 && v::len<usize>(owners)!=v::len<syntax::Parsed>(sources)){return .err(fail(0,12,0,0));}var v::Vec<Entry> entries=v::empty<Entry>();var v::Vec<u32> states=v::empty<u32>();var v::Vec<Counter> counters=v::empty<Counter>();var usize work=0;var usize module_index=0;
    while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);var usize i=0;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(!charge(ref mut work)){return .err(fail(module_index,4304,d.start,d.end));}let admission=const_eval::read(source,i);if(admission.code!=0){return .err(fail(module_index,admission.code,admission.start,admission.end));}if(admission.annotated){let expression=syntax::expression_at(source,admission.expression);return .err(fail(module_index,4302,expression.start,expression.end));}
        if(d.kind==7){let u32 scalar=scalars::primitive(source,d.result_type);if(d.public || d.attribute_count!=0 || d.result_type==0 || syntax::type_at(source,d.result_type).kind!=1 || !admitted(scalar)){return .err(fail(module_index,4301,d.start,d.end));}if(v::len<Entry>(ref entries)>=1024){return .err(fail(module_index,4304,d.start,d.end));}if(!v::push<Entry>(ref mut entries,Entry{.module_index,.declaration=i+1,.start=d.start,.end=d.end,.value=Value{.scalar,.bits=0}}).is_ok || !v::push<u32>(ref mut states,0).is_ok){return .err(fail(module_index,11,d.start,d.end));}}
        if(d.kind==8 && (d.public || d.attribute_count!=0)){return .err(fail(module_index,4301,d.start,d.end));}i+=1;
    }module_index+=1;}
    var usize i=1;while(i<=v::len<Entry>(ref entries)){let p=visit(sources,ref entries,i,ref mut states,ref mut work);if(p.code!=0){return .err(p);}i+=1;}
    module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);i=1;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(d.kind==8){let p=validate(sources,ref entries,module_index,d.expression,ref mut states,ref mut work,0);if(p.code!=0){return .err(p);}}i+=1;}module_index+=1;}
    i=1;while(i<=v::len<Entry>(ref entries)){let row=*v::at<Entry>(ref entries,i-1);let source=v::at<syntax::Parsed>(sources,row.module_index);let d=syntax::declaration_at(source,row.declaration-1);let scalar=switch(typecheck(sources,ref entries,row.module_index,d.expression,row.value.scalar,ref mut work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};if(scalar!=row.value.scalar){let e=syntax::expression_at(source,d.expression);return .err(fail(row.module_index,1006,e.start,e.end));}i+=1;}
    module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);i=1;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(d.kind==8){let scalar=switch(typecheck(sources,ref entries,module_index,d.expression,6,ref mut work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};if(scalar!=6){let e=syntax::expression_at(source,d.expression);return .err(fail(module_index,1007,e.start,e.end));}}i+=1;}module_index+=1;}
    i=0;while(i<v::len<u32>(ref states)){let cell=v::at_mut<u32>(ref mut states,i);*cell=0;i+=1;}i=1;while(i<=v::len<Entry>(ref entries)){let value=switch(constant(sources,ref mut entries,i,ref mut states,owners,ref mut counters,ref mut work)){case .ok(v):{v}case .err(p):{return .err(p);}};discard value;i+=1;}
    module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);i=1;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(d.kind==8){let value=switch(evaluate(sources,ref mut entries,module_index,d.expression,6,ref mut states,owners,ref mut counters,ref mut work,0)){case .ok(v):{v}case .err(p):{return .err(p);}};if(value.scalar!=6){let e=syntax::expression_at(source,d.expression);return .err(fail(module_index,1007,e.start,e.end));}if(value.bits==0){return .err(fail(module_index,4305,d.start,d.end));}}i+=1;}module_index+=1;}
    return .ok(Plan{.entries,.work});
}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources){let owners=v::empty<usize>();return build_owned(sources,ref owners);}
pub fn Result<Plan,m::Problem> build_project(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){var v::Vec<usize> owners=v::empty<usize>();var usize i=0;if(signatures::module_count(metadata)!=v::len<syntax::Parsed>(sources)){return .err(fail(0,12,0,0));}while(i<v::len<syntax::Parsed>(sources)){if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return .err(fail(i,12,0,0));}if(!v::push<usize>(ref mut owners,signatures::module_owner(metadata,i)).is_ok){return .err(fail(i,11,0,0));}i+=1;}return build_owned(sources,ref owners);}
pub fn Result<Value,m::Problem> get(ref Plan p,usize module_index,usize declaration,ref mut usize work){var usize i=0;while(i<v::len<Entry>(ref p.entries)){if(!charge(work)){return .err(fail(module_index,4304,0,0));}let e=*v::at<Entry>(ref p.entries,i);if(e.module_index==module_index && e.declaration==declaration){return .ok(e.value);}i+=1;}return .err(fail(module_index,4301,0,0));}
pub fn usize work(ref Plan p){return p.work;}
