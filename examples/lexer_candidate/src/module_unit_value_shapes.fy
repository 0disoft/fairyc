module module_unit_value_shapes;
import module_unit_value_model as m;
import module_unit_interfaces as interfaces;
import module_types as types;
import std::vec as v;

fn Result<bool,m::Problem> walk(ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,m::Frame frame,m::Expression e,usize id,bool allow_arrays,ref mut v::Vec<usize> path,usize depth,ref mut usize work){
    if(depth>=64 || !m::charge(work,depth+1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}let t=types::at(graph,id);if(t.kind==1){return .ok(t.scalar>=1 && t.scalar<=6);}if(t.kind!=2){return .ok(false);}var usize i=0;while(i<depth){if(*v::at<usize>(path,i)==id){return .ok(false);}i+=1;}let slot=v::at_mut<usize>(path,depth);*slot=id;
    if(!m::charge(work,interfaces::shape_count(imported))){return .err(m::fail(frame,10,e.start,e.end,0,0));}i=0;while(i<interfaces::shape_count(imported)){let shape=interfaces::shape_at(imported,i);if(shape.type==id){if(shape.field_count==0){return .ok(false);}var usize j=0;while(j<shape.field_count){if(!m::charge(work,1)){return .err(m::fail(frame,10,e.start,e.end,0,0));}let f=interfaces::field_at(imported,shape.field_start+j);if(f.type==0){return .ok(false);}let child=types::at(graph,f.type);var usize element=f.type;var bool arrays=allow_arrays;if(child.kind==16){if(!allow_arrays || child.second<1 || child.second>4096){return .ok(false);}element=child.first;arrays=false;}let valid=switch(walk(imported,graph,frame,e,element,arrays,path,depth+1,work)){case .ok(b):{b}case .err(p):{return .err(p);}};if(!valid){return .ok(false);}j+=1;}return .ok(true);}i+=1;}return .ok(false);
}
fn Result<bool,m::Problem> validate(ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,m::Frame frame,m::Expression e,usize id,bool allow_arrays,ref mut usize work){var v::Vec<usize> path=v::empty<usize>();var usize i=0;while(i<64){if(!v::push<usize>(ref mut path,0).is_ok){return .err(m::fail(frame,11,e.start,e.end,0,0));}i+=1;}return walk(imported,graph,frame,e,id,allow_arrays,ref mut path,0,work);}

pub fn Result<bool,m::Problem> element(ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,m::Frame frame,m::Expression e,usize id,ref mut usize work){return validate(imported,graph,frame,e,id,true,work);}
pub fn Result<bool,m::Problem> inline_element(ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,m::Frame frame,m::Expression e,usize id,ref mut usize work){return validate(imported,graph,frame,e,id,false,work);}
