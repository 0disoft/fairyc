module module_unit_value_storage;
import module_unit_value_model as m;
import module_unit_value_builtin as builtins;
import module_unit_value_scalar as scalar;
import module_unit_value_shapes as shapes;
import module_unit_interfaces as interfaces;
import module_types as types;
import pointer_types as pointers;
import scalar_views as views;
import syntax as syntax;
import std::vec as v;

fn m::Problem ok(m::Frame f){return m::fail(f,0,0,0,0,0);}
pub fn bool borrow(m::Builtin b){return b.kind==106 || b.kind==107;}
pub fn m::Place projected(m::Expression e,m::Place owner){var p=owner;p.kind=0;p.projection=e.source_index;p.mutable=e.builtin.kind==107;p.indirect=true;p.borrowable=false;return p;}
fn Result<usize,m::Problem> result(m::Frame f,m::Expression e,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut usize work){
    let b=e.builtin;if(b.element==0 || b.element>types::count(graph) || b.kind<101 || b.kind>107){return .err(m::fail(f,12,e.start,e.end,0,b.element));}let t=types::at(graph,b.element);
    if(b.slice){if(!borrow(b)){return .err(m::fail(f,2,e.start,e.end,0,b.element));}let bool exclusive=b.kind==107;
        if(t.kind==2){let plain=switch(shapes::element(imported,graph,f,e,b.element,work)){case .ok(p):{p}case .err(p):{return .err(p);}};if(!plain){return .err(m::fail(f,2,e.start,e.end,0,b.element));}return builtins::canonical(f,e,types::Type{.kind=if(exclusive){18}else{17},.scalar=0,.module_index=0,.declaration=0,.first=b.element,.second=0},graph,work);}
        if(t.kind!=1 || t.scalar<1 || t.scalar>6){return .err(m::fail(f,2,e.start,e.end,0,b.element));}let u32 code=views::make(t.scalar,exclusive);return builtins::canonical(f,e,types::Type{.kind=if(code==10){10}else if(code==11){11}else{15},.scalar=if(views::extended(code)){code}else{0},.module_index=0,.declaration=0,.first=0,.second=0},graph,work);
    }
    if(!(t.kind==1 && t.scalar>=1 && t.scalar<=6 || t.kind>=2 && t.kind<=4)){return .err(m::fail(f,2,e.start,e.end,0,b.element));}
    if(b.kind<=102){return .ok(m::atom(atoms,5));}if(b.kind==103){return .ok(b.element);}if(b.kind==104){return .ok(m::atom(atoms,7));}
    if(b.kind==105){return builtins::canonical(f,e,types::Type{.kind=12,.scalar=pointers::make(7,true),.module_index=0,.declaration=0,.first=0,.second=0},graph,work);}
    return builtins::canonical(f,e,types::Type{.kind=if(b.kind==107){8}else{7},.scalar=0,.module_index=0,.declaration=0,.first=b.element,.second=0},graph,work);
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame f,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);let b=e.builtin;let rows=switch(builtins::arguments(source,f,e,work)){case .ok(r):{r}case .err(p):{return p;}};let usize count=if(b.kind<=102){0}else if(b.kind==103){1}else if(b.slice){3}else{2};
    if(v::len<usize>(ref rows)!=count){return m::fail(f,2,e.start,e.end,count,v::len<usize>(ref rows));}
    let type=switch(result(f,e,imported,graph,atoms,work)){case .ok(t):{t}case .err(p):{return p;}};
    var usize pointer=0;if(count!=0 && !borrow(b)){pointer=switch(builtins::canonical(f,e,types::Type{.kind=12,.scalar=pointers::make(7,true),.module_index=0,.declaration=0,.first=0,.second=0},graph,work)){case .ok(t):{t}case .err(p):{return p;}};}
    var bool complete=true;var m::Place origin=m::no_place();var usize i=0;
    while(i<count){let usize wrapper=m::index(expressions,*v::at<usize>(ref rows,count-1-i));if(wrapper==0){return m::fail(f,12,e.start,e.end,0,0);}let row=*v::at<m::Expression>(expressions,wrapper-1);let usize child=m::index(expressions,row.left);if(child==0){return m::fail(f,12,e.start,e.end,0,0);}
        let usize context=if(borrow(b)){if(i==2){m::atom(atoms,5)}else{0}}else if(i==0){pointer}else if(b.kind==105){m::atom(atoms,5)}else{b.element};let p=scalar::evaluate(source,f,child,context,imported,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let actual=*v::at<m::Expression>(expressions,child-1);let slot=v::at_mut<m::Expression>(expressions,wrapper-1);slot.expected=context;
        if(actual.state==2){return m::fail(f,2,e.start,e.end,0,0);}if(actual.type==0){complete=false;}else if(borrow(b) && i<2){let t=types::at(graph,actual.type);let bool exclusive=b.kind==107;
            if(i==0){if((t.kind!=7 && t.kind!=8) || (t.kind==8)!=exclusive){return m::fail(f,2,e.start,e.end,0,actual.type);}origin=actual.place;}
            else if(t.kind!=12 || t.first!=0 || pointers::inner(t.scalar)!=7 || exclusive && !pointers::mutable(t.scalar)){return m::fail(f,2,e.start,e.end,0,actual.type);}
        }else if(actual.type!=context){return m::fail(f,2,e.start,e.end,context,actual.type);}i+=1;
    }
    if(b.kind>102 && !b.unsafe_context){return m::fail(f,5101,e.start,e.end,0,0);}if(!complete){return ok(f);}builtins::record(expressions,id,expected,type,if(borrow(b)){projected(e,origin)}else{m::no_place()});return ok(f);
}
