module module_unit_value_scalar;
import module_unit_value_model as m;
import module_types as types;
import type_state as scalars;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn m::Problem ok(m::Frame frame){return m::fail(frame,0,0,0,0,0);}
fn u32 scalar(ref v::Vec<types::Type> graph,usize id){return if(id!=0 && types::at(graph,id).kind==1){types::at(graph,id).scalar}else{0};}
fn void record(ref mut v::Vec<m::Expression> expressions,usize id,usize expected,usize type,bool minimum){let slot=v::at_mut<m::Expression>(expressions,id-1);slot.expected=expected;slot.type=type;slot.state=1;slot.minimum=minimum;}
fn m::Problem literal(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work){
    let e=*v::at<m::Expression>(expressions,id-1);let token=syntax::expression_text(source,e.source_index);let u32 kind=syntax::literal_kind(source,e.source_index);var usize type=0;
    if(e.state==1 && (!e.flexible || e.expected==expected)){return ok(frame);}if(!m::charge(work,token.len+1)){return m::fail(frame,10,e.start,e.end,0,0);}
    if(bytes::equal(token,b"null")){if(expected==0){return ok(frame);}if(types::at(graph,expected).kind!=12){return m::fail(frame,1006,e.start,e.end,expected,0);}type=expected;}
    else if(kind==5){let u32 target=if(scalars::integer(scalar(graph,expected))){scalar(graph,expected)}else{1};type=m::atom(atoms,target);if(!scalars::fits(token,target,false)){return m::fail(frame,1013,e.start,e.end,type,0);}}
    else if(kind==7 || kind==8 || kind==9){type=if(kind==7){atoms.text}else if(kind==8){atoms.bytes}else{atoms.cstr};}
    else{type=m::atom(atoms,6);}if(type==0){return m::fail(frame,12,e.start,e.end,0,0);}record(expressions,id,expected,type,false);return ok(frame);
}
fn m::Problem unary(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);let op=syntax::expression_text(source,e.source_index);let bool minus=bytes::equal(op,b"-");let bool plus=bytes::equal(op,b"+");let bool complement=bytes::equal(op,b"~");let bool negate=bytes::equal(op,b"!");
    // Borrow/dereference operators need place and ownership consumers.
    if(!minus && !plus && !complement && !negate){return ok(frame);}let usize child=m::index(expressions,e.left);if(child==0){return m::fail(frame,12,e.start,e.end,0,0);}let c=*v::at<m::Expression>(expressions,child-1);
    let u32 signed_type=if(scalar(graph,expected)==2){2}else{1};if(minus && c.kind==1 && syntax::literal_kind(source,c.source_index)==5){let raw=syntax::expression_text(source,c.source_index);if(!m::charge(work,raw.len*2+1)){return m::fail(frame,10,e.start,e.end,0,0);}if(!scalars::fits(raw,signed_type,false) && scalars::fits(raw,signed_type,true)){let usize type=m::atom(atoms,signed_type);record(expressions,child,type,type,false);record(expressions,id,expected,type,true);return ok(frame);}}
    let usize context=if((minus || plus || complement) && scalar(graph,expected)==2 || complement && scalars::integer(scalar(graph,expected))){expected}else{0};let p=evaluate(source,frame,child,context,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let actual=*v::at<m::Expression>(expressions,child-1);
    if(actual.state==2){return m::fail(frame,1006,actual.start,actual.end,0,0);}if(actual.type==0){return ok(frame);}let u32 code=scalar(graph,actual.type);
    if(negate && actual.type!=m::atom(atoms,6)){return m::fail(frame,1007,actual.start,actual.end,m::atom(atoms,6),actual.type);}if((minus || plus) && code!=1 && code!=2){return m::fail(frame,1006,actual.start,actual.end,m::atom(atoms,1),actual.type);}if(complement && !scalars::integer(code)){return m::fail(frame,2,e.start,e.end,0,actual.type);}record(expressions,id,expected,actual.type,false);return ok(frame);
}
fn m::Problem binary(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);let op=syntax::expression_text(source,e.source_index);let bool arithmetic=scalars::operation(op);let bool compare=scalars::comparison(op);let bool logical=bytes::equal(op,b"&&") || bytes::equal(op,b"||");if(!arithmetic && !compare && !logical){return m::fail(frame,2,e.start,e.end,0,0);}
    let usize left=m::index(expressions,e.left);let usize right=m::index(expressions,e.right);if(left==0 || right==0){return m::fail(frame,12,e.start,e.end,0,0);}let lhs=*v::at<m::Expression>(expressions,left-1);let rhs=*v::at<m::Expression>(expressions,right-1);let usize context=if(arithmetic && scalars::integer(scalar(graph,expected))){expected}else{0};
    let bool null_left=compare && lhs.kind==1 && bytes::equal(syntax::expression_text(source,lhs.source_index),b"null");let bool reversed=null_left || (arithmetic || compare) && lhs.flexible && !rhs.flexible;let usize first=if(reversed){right}else{left};let usize second=if(reversed){left}else{right};
    let p=evaluate(source,frame,first,context,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let a=*v::at<m::Expression>(expressions,first-1);if(a.state==2){return m::fail(frame,1006,a.start,a.end,0,0);}if(a.type==0){return ok(frame);}
    // An unknown anchor leaves its flexible peer pending, avoiding a premature
    // i32 range failure before field/cast/call-argument typing supplies context.
    let usize peer=if(logical){0}else if(null_left || !reversed){a.type}else if(scalars::integer(scalar(graph,a.type))){a.type}else{context};let q=evaluate(source,frame,second,peer,graph,atoms,expressions,work,depth+1);if(q.code!=0){return q;}let b=*v::at<m::Expression>(expressions,second-1);if(b.state==2){return m::fail(frame,1006,b.start,b.end,0,0);}if(b.type==0){return ok(frame);}let l=*v::at<m::Expression>(expressions,left-1);let r=*v::at<m::Expression>(expressions,right-1);
    if(logical){if(l.type!=m::atom(atoms,6)){return m::fail(frame,1007,l.start,l.end,m::atom(atoms,6),l.type);}if(r.type!=m::atom(atoms,6)){return m::fail(frame,1007,r.start,r.end,m::atom(atoms,6),r.type);}record(expressions,id,expected,m::atom(atoms,6),false);return ok(frame);}
    if(l.type!=r.type){return m::fail(frame,1006,r.start,r.end,l.type,r.type);}let u32 code=scalar(graph,l.type);if(!scalars::integer(code) && !(scalars::equal(op) && (code==6 || types::at(graph,l.type).kind==12))){return m::fail(frame,2,e.start,e.end,0,l.type);}record(expressions,id,expected,if(compare){m::atom(atoms,6)}else{l.type},false);return ok(frame);
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    if(id==0){return ok(frame);}let e=*v::at<m::Expression>(expressions,id-1);if(depth>=64 || !m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}if(expected>types::count(graph)){return m::fail(frame,12,e.start,e.end,0,expected);}if(e.state==2){return ok(frame);}
    if(e.kind==1){return literal(source,frame,id,expected,graph,atoms,expressions,work);}if(e.state==1 && (!e.flexible || e.expected==expected)){return ok(frame);}
    if(e.kind==3){let usize child=m::index(expressions,e.left);if(child==0){return m::fail(frame,12,e.start,e.end,0,0);}let p=evaluate(source,frame,child,expected,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let c=*v::at<m::Expression>(expressions,child-1);if(c.state==1){record(expressions,id,expected,c.type,false);}else if(c.state==2){let slot=v::at_mut<m::Expression>(expressions,id-1);slot.state=2;}return ok(frame);}
    if(e.kind==4){return unary(source,frame,id,expected,graph,atoms,expressions,work,depth);}if(e.kind==5){return binary(source,frame,id,expected,graph,atoms,expressions,work,depth);}return ok(frame);
}
