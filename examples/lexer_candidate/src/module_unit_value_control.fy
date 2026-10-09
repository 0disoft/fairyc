module module_unit_value_control;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_interfaces as interfaces;
import module_types as types;
import module_unit_value_view as views;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Selection {m::Problem problem;usize type;}
// A single block forwards a descriptor's owner; multi-arm origin joins remain
// separate from value typing. Scalar and owned tails do not create places.
pub fn m::Place block_origin(ref syntax::Parsed source,usize body,ref v::Vec<types::Type> graph,ref v::Vec<m::Expression> expressions){
    let b=syntax::statement_at(source,body);let usize id=m::index(expressions,b.expression);if(id==0){return m::no_place();}let tail=*v::at<m::Expression>(expressions,id-1);if(tail.type==0){return m::no_place();}let t=types::at(graph,tail.type);
    if(t.kind!=7 && t.kind!=8 && t.kind!=9 && t.kind!=14 && !views::view(t)){return m::no_place();}var p=tail.place;p.kind=0;p.borrowable=false;return p;
}
// Statement consumers validate body expressions separately. A block's type is
// its tail or void, even when a return/assert terminates one of its statements.
pub fn Selection block(ref syntax::Parsed source,m::Frame frame,usize body,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    if(body==0 || body>syntax::statement_count(source)){return Selection{.problem=m::fail(frame,12,0,0,0,0),.type=0};}let b=syntax::statement_at(source,body);
    if(depth>=64 || !m::charge(work,1)){return Selection{.problem=m::fail(frame,10,b.start,b.end,0,0),.type=0};}if(b.kind!=1){return Selection{.problem=m::fail(frame,12,b.start,b.end,0,0),.type=0};}
    if(b.expression==0){return Selection{.problem=m::fail(frame,0,0,0,0,0),.type=m::atom(atoms,7)};}let usize child=m::index(expressions,b.expression);if(child==0){return Selection{.problem=m::fail(frame,12,b.start,b.end,0,0),.type=0};}
    let p=scalar::evaluate(source,frame,child,expected,imported,graph,atoms,expressions,work,depth+1);let tail=*v::at<m::Expression>(expressions,child-1);if(p.code!=0){return Selection{.problem=p,.type=0};}if(tail.state==2){return Selection{.problem=m::fail(frame,2,tail.start,tail.end,0,0),.type=0};}return Selection{.problem=p,.type=tail.type};
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);var usize type=0;
    if(e.kind==12 || e.kind==31){let selected=block(source,frame,e.left,expected,imported,graph,atoms,expressions,work,depth+1);if(selected.problem.code!=0){return selected.problem;}type=selected.type;}
    else{let usize condition=m::index(expressions,e.left);if(condition==0 || e.right==0){return m::fail(frame,12,e.start,e.end,0,0);}let p=scalar::evaluate(source,frame,condition,m::atom(atoms,6),imported,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let c=*v::at<m::Expression>(expressions,condition-1);if(c.state==2 || c.type!=0 && c.type!=m::atom(atoms,6)){return m::fail(frame,1007,c.start,c.end,m::atom(atoms,6),c.type);}
        let branches=syntax::expression_at(source,e.right);if(branches.kind!=14){return m::fail(frame,12,e.start,e.end,0,0);}let then=block(source,frame,branches.left,expected,imported,graph,atoms,expressions,work,depth+1);if(then.problem.code!=0){return then.problem;}
        if(branches.right==0){if(c.type!=0 && then.type!=0){type=m::atom(atoms,7);}}
        else{let usize alternative=m::index(expressions,branches.right);if(alternative==0){return m::fail(frame,12,e.start,e.end,0,0);}let q=scalar::evaluate(source,frame,alternative,expected,imported,graph,atoms,expressions,work,depth+1);if(q.code!=0){return q;}let other=*v::at<m::Expression>(expressions,alternative-1);if(other.state==2 || then.type!=0 && other.type!=0 && then.type!=other.type){return m::fail(frame,1006,other.start,other.end,then.type,other.type);}if(c.type!=0 && then.type!=0 && other.type!=0){type=then.type;}}
    }
    if(type!=0){let origin=if(e.kind==12 || e.kind==31){block_origin(source,e.left,graph,expressions)}else{m::no_place()};let slot=v::at_mut<m::Expression>(expressions,id-1);slot.type=type;slot.expected=expected;slot.state=1;slot.place=origin;}return m::fail(frame,0,0,0,0,0);
}

// ADR0213 for-header restrictions are checked structurally and charged to the
// same body work budget; while/assert conditions keep their ordinary consumer.
pub fn m::Problem header(ref syntax::Parsed source,m::Frame frame,usize id,ref mut usize work,usize depth){
    let e=syntax::expression_at(source,id);if(depth>=64 || !m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}if(e.kind==1 || e.kind==2){return m::fail(frame,0,0,0,0,0);}if(e.kind==3 || e.kind==4){let op=syntax::expression_text(source,id);if(e.kind==4 && !bytes::equal(op,b"+") && !bytes::equal(op,b"-") && !bytes::equal(op,b"!") && !bytes::equal(op,b"~")){return m::fail(frame,2,e.start,e.end,0,0);}return header(source,frame,e.left,work,depth+1);}if(e.kind==5){let p=header(source,frame,e.left,work,depth+1);return if(p.code!=0){p}else{header(source,frame,e.right,work,depth+1)};}return m::fail(frame,2,e.start,e.end,0,0);
}
