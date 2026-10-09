module module_unit_value_region;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_value_builtin as builtins;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import std::vec as v;

// Plain Region storage is inert and contains no safe borrow. Nominal shape
// cycles, depth and aggregate traversal remain bounded by the unit budget.
fn u32 plain(ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,usize id,ref mut v::Vec<usize> path,usize depth,ref mut usize work){
    if(depth>=64 || !m::charge(work,1)){return 10;}if(id==0 || id>types::count(graph)){return 12;}let t=types::at(graph,id);if(t.kind==3 || t.kind==13){return 3102;}if(t.kind==1){return if(t.scalar!=7){0}else{2};}if(t.kind==12){return 0;}
    if(t.kind==5 || t.kind==6 || t.kind==16){let wrapper=v::at_mut<usize>(path,depth);*wrapper=id;let first=plain(imported,graph,t.first,path,depth+1,work);if(first!=0 || t.kind!=6){return first;}return plain(imported,graph,t.second,path,depth+1,work);}if(t.kind!=2 && t.kind!=4){return 2;}
    var usize i=0;while(i<depth){if(!m::charge(work,1)){return 10;}if(*v::at<usize>(path,i)==id){return 2;}i+=1;}let slot=v::at_mut<usize>(path,depth);*slot=id;i=0;
    while(i<interfaces::shape_count(imported)){if(!m::charge(work,1)){return 10;}let shape=interfaces::shape_at(imported,i);if(shape.type==id){var usize j=0;while(j<shape.field_count){if(!m::charge(work,1)){return 10;}let f=interfaces::field_at(imported,shape.field_start+j);if(f.kind==0 || f.kind==4){let code=plain(imported,graph,f.type,path,depth+1,work);if(code!=0){return code;}}j+=1;}return 0;}i+=1;}return 12;
}
pub fn m::Problem evaluate(ref syntax::Parsed source,m::Frame frame,usize id,usize expected,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref mut v::Vec<m::Expression> expressions,ref mut usize work,usize depth){
    let e=*v::at<m::Expression>(expressions,id-1);let usize callee=m::index(expressions,e.left);if(callee==0){return m::fail(frame,12,e.start,e.end,0,0);}let field=*v::at<m::Expression>(expressions,callee-1);let usize receiver=m::index(expressions,field.left);if(field.kind!=8 || receiver==0){return m::fail(frame,12,e.start,e.end,0,0);}
    let checked=scalar::evaluate(source,frame,receiver,0,imported,graph,atoms,expressions,work,depth+1);if(checked.code!=0){return checked;}let owner=*v::at<m::Expression>(expressions,receiver-1);if(owner.type==0){return m::fail(frame,0,0,0,0,0);}if(types::at(graph,owner.type).kind!=19){return m::fail(frame,2,field.start,field.end,0,owner.type);}
    if(syntax::call_argument_count(source,e.source_index)!=1){return m::fail(frame,2,e.start,e.end,0,0);}let usize value=m::index(expressions,syntax::call_argument(source,e.source_index,0));if(value==0){return m::fail(frame,12,e.start,e.end,0,0);}let p=scalar::evaluate(source,frame,value,0,imported,graph,atoms,expressions,work,depth+1);if(p.code!=0){return p;}let actual=*v::at<m::Expression>(expressions,value-1);if(actual.type==0){return m::fail(frame,0,0,0,0,0);}if(actual.state==2){return m::fail(frame,2,e.start,e.end,0,0);}
    var v::Vec<usize> path=v::empty<usize>();var usize i=0;while(i<64){if(!v::push<usize>(ref mut path,0).is_ok){return m::fail(frame,11,e.start,e.end,0,0);}i+=1;}let valid=plain(imported,graph,actual.type,ref mut path,0,work);if(valid!=0){return m::fail(frame,valid,e.start,e.end,0,actual.type);}let error=interfaces::region_error(imported);if(error==0 || error>types::count(graph)){return m::fail(frame,2,e.start,e.end,0,0);}
    let borrow=switch(builtins::canonical(frame,e,types::Type{.kind=7,.scalar=0,.module_index=0,.declaration=0,.first=actual.type,.second=0},graph,work)){case .ok(t):{t}case .err(p):{return p;}};let result=switch(builtins::canonical(frame,e,types::Type{.kind=6,.scalar=0,.module_index=0,.declaration=0,.first=borrow,.second=error},graph,work)){case .ok(t):{t}case .err(p):{return p;}};
    var place=owner.place;place.kind=0;place.projection=e.source_index;place.indirect=true;place.mutable=false;place.borrowable=false;builtins::record(expressions,id,expected,result,place);return m::fail(frame,0,0,0,0,0);
}
