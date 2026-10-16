module module_unit_value_array;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_value_shapes as shapes;
import module_unit_interfaces as interfaces;
import module_types as types;
import array_types as arrays;
import syntax as syntax;
import std::vec as v;

pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);var usize count=0;var v::Vec<usize> children=v::empty<usize>();var usize element=if(expected!=0 && types::at(graph,expected).kind==16){types::at(graph,expected).first}else{0};
    if(e.kind==29){let usize raw=m::index(expressions,e.right);if(raw==0){return m::fail(frame,12,e.start,e.end,0,0);}let c=*v::at<m::Expression>(expressions,raw-1);let p=scalar::evaluate(source,frame,raw,m::atom(atoms,5),imported,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let token=syntax::expression_text(source,c.source_index);if(!m::charge(work,token.len+1)){return m::fail(frame,10,e.start,e.end,0,0);}count=if(c.constant!=0 && types::at(graph,c.type).kind==1 && types::at(graph,c.type).scalar>=1 && types::at(graph,c.type).scalar<=5 && c.constant_bits>=1 && c.constant_bits<=4096){c.constant_bits}else if(c.kind==1){arrays::count(token)}else{0};if(count==0){return m::fail(frame,2,c.start,c.end,0,0);}let usize child=m::index(expressions,e.left);if(child==0 || !v::push<usize>(ref mut children,child).is_ok){return m::fail(frame,if(child==0){12}else{11},e.start,e.end,0,0);}}
    else{var usize link=e.right;while(link!=0){if(!m::charge(work,1) || count>=4096){return m::fail(frame,10,e.start,e.end,0,0);}let row=syntax::expression_at(source,link);let usize child=m::index(expressions,row.left);if(child==0 || row.kind!=25){return m::fail(frame,12,e.start,e.end,0,0);}if(!v::push<usize>(ref mut children,child).is_ok){return m::fail(frame,11,e.start,e.end,0,0);}count+=1;link=row.right;}if(count==0){return m::fail(frame,2,e.start,e.end,0,0);}}
    var usize i=v::len<usize>(ref children);while(i>0){i-=1;if(!m::charge(work,1)){return m::fail(frame,10,e.start,e.end,0,0);}let usize child=*v::at<usize>(ref children,i);let p=scalar::evaluate(source,frame,child,element,imported,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let c=*v::at<m::Expression>(expressions,child-1);if(c.state==2){return m::fail(frame,1006,c.start,c.end,element,0);}if(c.type==0){return m::fail(frame,0,0,0,0,0);}if(element==0){element=c.type;}if(c.type!=element){return m::fail(frame,1006,c.start,c.end,element,c.type);}}
    let valid=switch(shapes::element(imported,graph,frame,e,element,work)){case .ok(b):{b}case .err(p):{return p;}};if(!valid){return m::fail(frame,2,e.start,e.end,0,element);}if(!m::charge(work,types::count(graph)+1)){return m::fail(frame,10,e.start,e.end,0,0);}let usize type=types::canonical(graph,types::Type{.kind=16,.scalar=0,.module_index=0,.declaration=0,.first=element,.second=count});if(type==0){return m::fail(frame,if(types::count(graph)>=4096){10}else{11},e.start,e.end,0,0);}let slot=v::at_mut<m::Expression>(expressions,id-1);slot.type=type;slot.expected=expected;slot.state=1;return m::fail(frame,0,0,0,0,0);
}
