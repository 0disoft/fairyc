module module_unit_value_assignment;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_value_shapes as shapes;
import module_unit_value_view as views;
import module_unit_value_control as control;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn m::Problem ok(m::Frame f){return m::fail(f,0,0,0,0,0);}
fn Result<usize,m::Problem> ungroup(m::Frame f,usize id,ref v::Vec<m::Expression> expressions,ref mut usize work){var usize next=id;var usize depth=0;while(next!=0 && v::at<m::Expression>(expressions,next-1).kind==3){let e=*v::at<m::Expression>(expressions,next-1);if(depth>=64 || !m::charge(work,1)){return .err(m::fail(f,10,e.start,e.end,0,0));}next=m::index(expressions,e.left);depth+=1;}return if(next==0){.err(m::fail(f,12,0,0,0,0))}else{.ok(next)};}
fn Result<bool,m::Problem> plain(m::Frame f,m::Expression e,usize type,bool stored,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref mut usize work){if(!stored && types::at(graph,type).kind!=1 && types::at(graph,type).kind!=2){return .ok(false);}return if(stored){shapes::stored(imported,graph,f,e,type,work)}else{shapes::element(imported,graph,f,e,type,work)};}
fn m::Problem writable(ref syntax::Parsed source,m::Frame f,m::Expression e,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref v::Vec<m::Expression> expressions,ref mut usize work){
    if(e.kind==4 && bytes::equal(syntax::expression_text(source,e.source_index),b"*")){let child=switch(ungroup(f,m::index(expressions,e.left),expressions,work)){case .ok(c):{c}case .err(p):{return p;}};let valid=switch(plain(f,e,e.type,false,imported,graph,work)){case .ok(v):{v}case .err(p):{return p;}};if(v::at<m::Expression>(expressions,child-1).kind!=2 || !valid){return m::fail(f,1008,e.start,e.end,0,e.type);}}
    else if(e.kind!=2 && e.kind!=8 && e.kind!=9){return m::fail(f,1008,e.start,e.end,0,e.type);}
    if(e.place.kind==0 || !e.place.mutable){return m::fail(f,1008,e.start,e.end,0,e.type);}
    if(e.kind==8 || e.kind==9){var usize root=m::index(expressions,e.source_index);var usize depth=0;var usize fields=0;
        while(root!=0){let n=*v::at<m::Expression>(expressions,root-1);if(n.kind!=3 && n.kind!=8){break;}if(depth>=64 || !m::charge(work,1)){return m::fail(f,10,e.start,e.end,0,0);}if(n.kind==8){fields+=1;}root=m::index(expressions,n.left);depth+=1;}if(root==0){return m::fail(f,12,e.start,e.end,0,0);}let base=*v::at<m::Expression>(expressions,root-1);
        if(base.kind==9){var exclusive=e;exclusive.kind=22;let p=views::element_borrow(f,exclusive,e,imported,graph,expressions,work);if(p.code!=0){return p;}if(e.kind==8){let root_ok=switch(plain(f,e,base.type,false,imported,graph,work)){case .ok(b):{b}case .err(p):{return p;}};let target_ok=switch(plain(f,e,e.type,false,imported,graph,work)){case .ok(b):{b}case .err(p):{return p;}};if(!root_ok || !target_ok){return m::fail(f,1008,e.start,e.end,0,e.type);}}}
        else if(e.kind==8){if(base.kind!=2){return m::fail(f,1008,e.start,e.end,0,e.type);}if(fields>1){let t=types::at(graph,base.type);let usize type=if(t.kind==7 || t.kind==8){t.first}else{base.type};let root_ok=switch(plain(f,e,type,true,imported,graph,work)){case .ok(b):{b}case .err(p):{return p;}};let target_ok=switch(plain(f,e,e.type,true,imported,graph,work)){case .ok(b):{b}case .err(p):{return p;}};if(!root_ok || !target_ok){return m::fail(f,1008,e.start,e.end,0,e.type);}}}
    }return ok(f);
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame f,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut m::Consumer c,ref mut usize work){
    if(c.target==0 || c.expression==0 || c.operator>=scanner::token_count(syntax::source_stream(source))){return m::fail(f,12,c.start,c.end,0,0);}let p=scalar::evaluate(source,f,c.target,0,imported,graph,atoms,expressions,work,0);if(p.code!=0){return p;}let target=*v::at<m::Expression>(expressions,c.target-1);if(target.type==0){return ok(f);}let q=scalar::evaluate(source,f,c.expression,target.type,imported,graph,atoms,expressions,work,0);if(q.code!=0){return q;}let value=*v::at<m::Expression>(expressions,c.expression-1);let place=writable(source,f,target,imported,graph,expressions,work);if(place.code!=0){return place;}if(value.state==2 || value.type!=0 && value.type!=target.type){return m::fail(f,1006,value.start,value.end,target.type,value.type);}
    let op=scanner::token_raw(syntax::source_stream(source),c.operator);let t=types::at(graph,target.type);if(!bytes::equal(op,b"=") && (t.kind!=1 || t.scalar<1 || t.scalar>5)){return m::fail(f,2,c.start,c.end,0,target.type);}if(c.header!=0){let h=control::header(source,f,value.source_index,work,0);if(h.code==10){return h;}if(target.kind!=2 || t.kind!=1 || t.scalar<1 || t.scalar>6 || h.code!=0){return m::fail(f,2,c.start,value.end,0,target.type);}}if(value.type!=0){c.expected=target.type;c.actual=value.type;c.checked=true;}return ok(f);
}
