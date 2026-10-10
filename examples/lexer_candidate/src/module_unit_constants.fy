module module_unit_constants;
import module_unit_value_model as m;
import syntax as syntax;
import type_state as scalars;
import std::bytes as bytes;
import std::vec as v;

// First owned-unit slice: private typed scalar literals, grouping and prefixes.
// Unsupported constant expressions are rejected, including unused declarations.
pub struct Value {u32 scalar;usize bits;}
pub struct Entry {usize module_index;usize declaration;usize start;usize end;Value value;}
pub resource Plan {v::Vec<Entry> entries;usize work;cleanup{}}
fn m::Problem fail(usize module_index,u32 code,usize start,usize end){return m::Problem{.code,.module_index,.start,.end,.expected=0,.actual=0};}
fn bool admitted(u32 scalar){return scalar==1 || scalar==3 || scalar==4 || scalar==6;}
fn Result<Value,m::Problem> literal(ref syntax::Parsed source,usize module_index,usize id,u32 scalar,bool minimum,ref mut usize steps,usize depth){
    if(id==0 || id>syntax::expression_count(source)){return .err(fail(module_index,4302,0,0));}let e=syntax::expression_at(source,id);
    if(depth>=64 || *steps>=65536){return .err(fail(module_index,4304,e.start,e.end));};*steps=*steps+1;
    let raw=syntax::expression_text(source,id);
    if(e.kind==3){return literal(source,module_index,e.left,scalar,minimum,steps,depth+1);}
    if(e.kind==4){
        if(bytes::equal(raw,b"-") && scalar==1){var usize child=e.left;var usize groups=0;while(syntax::expression_at(source,child).kind==3){if(groups+depth>=64 || *steps>=65536){return .err(fail(module_index,4304,e.start,e.end));};*steps=*steps+1;child=syntax::expression_at(source,child).left;groups+=1;}if(syntax::expression_at(source,child).kind!=1){return .err(fail(module_index,4302,e.start,e.end));}let value=switch(literal(source,module_index,e.left,scalar,true,steps,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};if(value.bits>2147483648){return .err(fail(module_index,4303,e.start,e.end));}return .ok(Value{.scalar,.bits=if(value.bits==0){0}else{4294967296-value.bits}});}
        if(bytes::equal(raw,b"+") && scalar==1){return literal(source,module_index,e.left,scalar,false,steps,depth+1);}
        if(bytes::equal(raw,b"!") && scalar==6 || bytes::equal(raw,b"~") && scalar!=6){let value=switch(literal(source,module_index,e.left,scalar,false,steps,depth+1)){case .ok(v):{v}case .err(p):{return .err(p);}};return .ok(Value{.scalar,.bits=if(scalar==6){1-value.bits}else{value.bits ^ if(scalar==3){255}else{4294967295}}});}
        return .err(fail(module_index,4302,e.start,e.end));
    }
    if(e.kind!=1){return .err(fail(module_index,4302,e.start,e.end));}
    if(scalar==6){if(!bytes::equal(raw,b"true") && !bytes::equal(raw,b"false")){return .err(fail(module_index,1006,e.start,e.end));}return .ok(Value{.scalar,.bits=if(bytes::equal(raw,b"true")){1}else{0}});}
    if(syntax::literal_kind(source,id)!=5){return .err(fail(module_index,1006,e.start,e.end));}
    if(!scalars::fits(raw,scalar,minimum)){return .err(fail(module_index,1013,e.start,e.end));}
    var usize base=10;var usize index=0;if(raw.len>=2 && raw[0]==48){if(raw[1]==120 || raw[1]==88){base=16;index=2;}else if(raw[1]==98 || raw[1]==66){base=2;index=2;}}
    var usize bits=0;while(index<raw.len){let byte=raw[index];index+=1;if(byte==95){continue;}let usize digit=if(byte>=48 && byte<=57){(byte-48) as usize}else if(byte>=65 && byte<=70){(byte-55) as usize}else{(byte-87) as usize};bits=bits*base+digit;}
    return .ok(Value{.scalar,.bits});
}
pub fn Result<Plan,m::Problem> build(ref v::Vec<syntax::Parsed> sources){
    var v::Vec<Entry> entries=v::empty<Entry>();var usize work=0;var usize steps=0;var usize module_index=0;
    while(module_index<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,module_index);var usize i=0;while(i<syntax::declaration_count(source)){let d=syntax::declaration_at(source,i);if(!m::charge(ref mut work,1)){return .err(fail(module_index,4304,d.start,d.end));}
        var usize a=0;while(a<d.attribute_count){let attribute=syntax::attribute_at(source,d.attribute_start+a);if(bytes::equal(syntax::source_view(source)[attribute.name_start..attribute.name_end],b"const_eval")){return .err(fail(module_index,4302,attribute.start,attribute.end));}a+=1;}
        if(d.kind==7){let u32 scalar=scalars::primitive(source,d.result_type);if(d.public || d.attribute_count!=0 || d.result_type==0 || syntax::type_at(source,d.result_type).kind!=1 || !admitted(scalar)){return .err(fail(module_index,4301,d.start,d.end));}
            let value=switch(literal(source,module_index,d.expression,scalar,false,ref mut steps,0)){case .ok(v):{v}case .err(p):{return .err(p);}};if(v::len<Entry>(ref entries)>=1024){return .err(fail(module_index,4304,d.start,d.end));}if(!v::push<Entry>(ref mut entries,Entry{.module_index,.declaration=i+1,.start=d.start,.end=d.end,.value}).is_ok){return .err(fail(module_index,11,d.start,d.end));}
        }else if(d.kind==8){if(d.public || d.attribute_count!=0){return .err(fail(module_index,4301,d.start,d.end));}let value=switch(literal(source,module_index,d.expression,6,false,ref mut steps,0)){case .ok(v):{v}case .err(p):{return .err(p);}};if(value.bits==0){return .err(fail(module_index,4305,d.start,d.end));}}
        i+=1;
    }module_index+=1;}if(!m::charge(ref mut work,steps)){return .err(fail(0,4304,0,0));}return .ok(Plan{.entries,.work});
}
pub fn Result<Value,m::Problem> get(ref Plan p,usize module_index,usize declaration,ref mut usize work){var usize i=0;while(i<v::len<Entry>(ref p.entries)){if(!m::charge(work,1)){return .err(fail(module_index,4304,0,0));}let e=*v::at<Entry>(ref p.entries,i);if(e.module_index==module_index && e.declaration==declaration){return .ok(e.value);}i+=1;}return .err(fail(module_index,4301,0,0));}
pub fn usize work(ref Plan p){return p.work;}
