module module_unit_value_walk;
import module_unit_value_model as m;
import module_unit_name_model as names;
import module_types as types;
import type_state as scalars;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn m::Problem literal(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref v::Vec<types::Type> graph,ref mut v::Vec<m::Expression> expressions,ref mut usize work){
    let e=*v::at<m::Expression>(expressions,id-1);let token=syntax::expression_text(source,e.source_index);let u32 kind=syntax::literal_kind(source,e.source_index);var usize type=0;
    if(e.state==1 && (expected==0 || e.type==expected)){return m::fail(frame,0,0,0,0,0);}
    if(!m::charge(work,types::count(graph)+1)){return m::fail(frame,10,e.start,e.end,0,0);}
    if(bytes::equal(token,b"null")){if(expected==0){return m::fail(frame,0,0,0,0,0);}if(types::at(graph,expected).kind!=12){return m::fail(frame,1006,e.start,e.end,expected,0);}type=expected;}
    else if(kind==5){let u32 scalar=if(expected!=0 && types::at(graph,expected).kind==1 && scalars::integer(types::at(graph,expected).scalar)){types::at(graph,expected).scalar}else{1};type=m::find(graph,1,scalar);if(!scalars::fits(token,scalar,false)){return m::fail(frame,1008,e.start,e.end,type,0);}}
    else if(kind==7 || kind==8 || kind==9){type=m::find(graph,if(kind==7){9}else if(kind==8){10}else{14},0);}
    else{type=m::find(graph,1,6);}
    if(type==0){return m::fail(frame,12,e.start,e.end,0,0);}if(expected!=0 && type!=expected){return m::fail(frame,1006,e.start,e.end,expected,type);}let slot=v::at_mut<m::Expression>(expressions,id-1);slot.expected=expected;slot.type=type;slot.state=1;return m::fail(frame,0,0,0,0,0);
}
// Contextualize only a direct literal through parentheses. Operators, call
// arguments and control-flow joins have their own later contextual consumers.
fn m::Problem hint(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref v::Vec<types::Type> graph,ref mut v::Vec<m::Expression> expressions,ref mut usize work){
    var usize at=id;var usize depth=0;var usize[64] parents=[0;64];
    while(at!=0){let e=*v::at<m::Expression>(expressions,at-1);if(depth>=64 || !m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}if(e.kind==1){let p=literal(source,frame,at,expected,graph,expressions,work);if(p.code!=0){return p;}break;}if(e.kind!=3){break;}parents[depth]=at;depth+=1;at=m::index(expressions,e.left);}
    let usize type=if(at==0){0}else{v::at<m::Expression>(expressions,at-1).type};let bool nonvalue=at!=0 && v::at<m::Expression>(expressions,at-1).state==2;
    while(depth>0){depth-=1;if(type!=0){let slot=v::at_mut<m::Expression>(expressions,parents[depth]-1);slot.expected=expected;slot.type=type;slot.state=1;}else if(nonvalue){let slot=v::at_mut<m::Expression>(expressions,parents[depth]-1);slot.state=2;}}
    return m::fail(frame,0,0,0,0,0);
}
pub fn Result<m::Values,m::Problem> resolve(ref syntax::Parsed source,ref names::Resolved resolved,m::Frame frame,v::Vec<m::Expression> expressions_input,v::Vec<m::Local> locals_input,v::Vec<m::Return> returns_input,ref v::Vec<types::Type> graph,usize initial_work){
    var v::Vec<m::Expression> expressions=expressions_input;var v::Vec<m::Local> locals=locals_input;var v::Vec<m::Return> returns=returns_input;var usize work=initial_work;var bool changed=true;var usize rounds=0;
    while(changed && rounds<64){changed=false;rounds+=1;var usize i=0;
        while(i<v::len<m::Local>(ref locals)){let local=*v::at<m::Local>(ref locals,i);if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,local.start,local.end,0,0));}let usize id=m::index(ref expressions,local.initializer);if(id!=0){let p=hint(source,frame,id,local.expected,graph,ref mut expressions,ref mut work);if(p.code!=0){return .err(p);}let e=*v::at<m::Expression>(ref expressions,id-1);if(e.state==2){return .err(m::fail(frame,1006,e.start,e.end,local.expected,0));}if(e.type!=0){if(types::at(graph,e.type).kind==1 && types::at(graph,e.type).scalar==7 || local.expected!=0 && local.expected!=e.type){return .err(m::fail(frame,1006,e.start,e.end,local.expected,e.type));}if(local.type==0){let slot=v::at_mut<m::Local>(ref mut locals,i);slot.type=e.type;changed=true;}}}i+=1;}
        i=1;while(i<=v::len<m::Expression>(ref expressions)){let e=*v::at<m::Expression>(ref expressions,i-1);if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}var usize type=0;
            if(e.kind==2 && e.state==0){let usize use=names::use_at(resolved,i);if(use!=0){let r=names::reference_at(resolved,use);if(r.kind==1){type=v::at<m::Local>(ref locals,r.symbol-1).type;}}}
            else if(e.kind==3 && e.state==0){let usize child=m::index(ref expressions,e.left);if(child!=0){type=v::at<m::Expression>(ref expressions,child-1).type;}}
            if(e.type==0 && type!=0){let slot=v::at_mut<m::Expression>(ref mut expressions,i-1);slot.type=type;slot.state=1;changed=true;}i+=1;
        }
        i=0;while(i<v::len<m::Return>(ref returns)){let r=*v::at<m::Return>(ref returns,i);if(!m::charge(ref mut work,1)){return .err(m::fail(frame,10,r.start,r.end,0,0));}let usize id=m::index(ref expressions,r.expression);var usize actual=0;
            if(r.expression==0){actual=m::find(graph,1,7);}else if(id!=0){let p=hint(source,frame,id,r.expected,graph,ref mut expressions,ref mut work);if(p.code!=0){return .err(p);}let e=*v::at<m::Expression>(ref expressions,id-1);if(e.state==2){return .err(m::fail(frame,1006,e.start,e.end,r.expected,0));}actual=e.type;}
            if(actual!=0){if(actual!=r.expected){let usize start=if(id==0){r.start}else{v::at<m::Expression>(ref expressions,id-1).start};let usize end=if(id==0){r.end}else{v::at<m::Expression>(ref expressions,id-1).end};return .err(m::fail(frame,1006,start,end,r.expected,actual));}let slot=v::at_mut<m::Return>(ref mut returns,i);slot.actual=actual;slot.checked=true;}i+=1;
        }
    }if(changed){return .err(m::fail(frame,10,0,0,0,0));}var usize unresolved=0;var usize inferred=0;var usize checked_returns=0;var usize i=0;while(i<v::len<m::Expression>(ref expressions)){if(v::at<m::Expression>(ref expressions,i).state==0){unresolved+=1;}i+=1;}i=0;while(i<v::len<m::Local>(ref locals)){let s=*v::at<m::Local>(ref locals,i);if(s.expected==0 && s.type!=0){inferred+=1;}i+=1;}i=0;while(i<v::len<m::Return>(ref returns)){if(v::at<m::Return>(ref returns,i).checked){checked_returns+=1;}i+=1;}
    return .ok(m::assemble(frame,expressions,locals,returns,unresolved,inferred,checked_returns,work));
}
