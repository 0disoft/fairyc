module module_unit_value_builtin;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_value_view as views;
import module_unit_value_storage as storage;
import module_unit_interfaces as interfaces;
import module_types as types;
import pointer_types as pointers;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn m::Problem ok(m::Frame f){return m::fail(f,0,0,0,0,0);}
pub fn Result<v::Vec<usize>,m::Problem> arguments(ref syntax::Parsed source,m::Frame f,m::Expression e,ref mut usize work){
    var v::Vec<usize> rows=v::empty<usize>();var usize link=e.right;while(link!=0){if(!m::charge(work,1) || v::len<usize>(ref rows)>=1024){return .err(m::fail(f,10,e.start,e.end,0,0));}let row=syntax::expression_at(source,link);if(row.kind!=7){return .err(m::fail(f,12,row.start,row.end,0,0));}if(!v::push<usize>(ref mut rows,link).is_ok){return .err(m::fail(f,11,row.start,row.end,0,0));}link=row.right;}return .ok(rows);
}
pub fn Result<usize,m::Problem> canonical(m::Frame f,m::Expression e,types::Type t,ref mut v::Vec<types::Type> graph,ref mut usize work){if(!m::charge(work,types::count(graph)+1)){return .err(m::fail(f,10,e.start,e.end,0,0));}let usize id=types::canonical(graph,t);return if(id==0){.err(m::fail(f,if(types::count(graph)>=4096){10}else{11},e.start,e.end,0,0))}else{.ok(id)};}
pub fn bool borrowed(types::Type t){return t.kind==7 || t.kind==8 || t.kind==9 || t.kind==14 || views::view(t);}
pub fn bool mutable(types::Type t){return t.kind==8 || views::mutable(t);}
pub fn m::Place projected(m::Expression e,m::Place origin){var p=origin;p.kind=0;p.projection=e.source_index;p.mutable=e.builtin.kind==11;p.indirect=true;p.borrowable=false;return p;}
pub fn void record(ref mut v::Vec<m::Expression> expressions,usize id,usize expected,usize type,m::Place origin){let slot=v::at_mut<m::Expression>(expressions,id-1);slot.state=1;slot.type=type;slot.expected=expected;slot.place=origin;}
fn m::Problem raw(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);let rows=switch(arguments(source,frame,e,work)){case .ok(r):{r}case .err(p):{return p;}};if(v::len<usize>(ref rows)!=3){return m::fail(frame,1010,e.start,e.end,3,v::len<usize>(ref rows));}
    var bool complete=true;var m::Problem unsupported=ok(frame);var m::Place origin=m::no_place();var usize i=0;
    while(i<3){let usize wrapper=m::index(expressions,*v::at<usize>(ref rows,2-i));if(wrapper==0){return m::fail(frame,12,e.start,e.end,0,0);}let row=*v::at<m::Expression>(expressions,wrapper-1);let usize child=m::index(expressions,row.left);if(child==0){return m::fail(frame,12,e.start,e.end,0,0);}let usize context=if(i==2){m::atom(atoms,5)}else{0};let p=scalar::evaluate(source,frame,child,context,imported,graph,atoms,expressions,work,depth+1);if(p.code!=0 && p.code!=2){return p;}if(p.code==2 && unsupported.code==0){unsupported=p;}
        let actual=*v::at<m::Expression>(expressions,child-1);let slot=v::at_mut<m::Expression>(expressions,wrapper-1);slot.expected=context;
        if(actual.state==2){if(unsupported.code==0){unsupported=m::fail(frame,2,actual.start,actual.end,0,0);}}
        else if(actual.type==0){complete=false;if(i==1 && actual.kind==1 && bytes::equal(syntax::expression_text(source,actual.source_index),b"null") && unsupported.code==0){unsupported=m::fail(frame,2,actual.start,actual.end,0,0);}}
        else{let t=types::at(graph,actual.type);if(i==0){if(!borrowed(t) || mutable(t)!=(e.builtin.kind==11)){if(unsupported.code==0){unsupported=m::fail(frame,2,actual.start,actual.end,0,actual.type);}}origin=actual.place;}
            else if(i==1){if(t.kind!=12 || t.first!=0 || (pointers::inner(t.scalar)!=3 && pointers::inner(t.scalar)!=7) || e.builtin.kind==11 && !pointers::mutable(t.scalar)){if(unsupported.code==0){unsupported=m::fail(frame,2,actual.start,actual.end,0,actual.type);}}}
            else if(actual.type!=context){return m::fail(frame,1006,actual.start,actual.end,context,actual.type);}
        }i+=1;
    }
    if(!e.builtin.unsafe_context){return m::fail(frame,5101,e.start,e.end,0,0);}if(unsupported.code!=0){return unsupported;}if(!complete){return ok(frame);}
    let usize type=if(e.builtin.kind==9){atoms.text}else if(e.builtin.kind==10){atoms.bytes}else if(e.builtin.kind==12){atoms.cstr}else{switch(canonical(frame,e,types::Type{.kind=11,.scalar=0,.module_index=0,.declaration=0,.first=0,.second=0},graph,work)){case .ok(t):{t}case .err(p):{return p;}}};
    record(expressions,id,expected,type,projected(e,origin));return ok(frame);
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);if(e.builtin.kind==200){let rows=switch(arguments(source,frame,e,work)){case .ok(r):{r}case .err(p):{return p;}};if(v::len<usize>(ref rows)!=0){return m::fail(frame,1010,e.start,e.end,0,v::len<usize>(ref rows));}record(expressions,id,expected,atoms.text,m::no_place());return ok(frame);}
    if(e.builtin.kind>=9 && e.builtin.kind<=12){return raw(source,frame,id,expected,imported,graph,atoms,expressions,work,depth);}if(e.builtin.kind>=101 && e.builtin.kind<=107){return storage::evaluate(source,frame,id,expected,imported,graph,atoms,expressions,work,depth);}return ok(frame);
}
