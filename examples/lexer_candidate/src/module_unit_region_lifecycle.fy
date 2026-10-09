module module_unit_region_lifecycle;
import module_unit_region_model as r;
import module_unit_region_walk as walk;
import module_unit_value_model as m;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import std::vec as v;

pub fn Result<r::Plan,m::Problem> build(ref syntax::Parsed source,m::Frame frame,ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref v::Vec<m::Local> locals,ref v::Vec<m::Expression> expressions,ref mut usize work){
    var bool needed=false;var usize i=0;while(i<v::len<m::Local>(locals)){if(!m::charge(work,1)){return .err(m::fail(frame,10,0,0,0,0));}needed=needed || v::at<m::Local>(locals,i).kind==6;i+=1;}i=0;while(!needed && i<v::len<m::Expression>(expressions)){if(!m::charge(work,1)){return .err(m::fail(frame,10,0,0,0,0));}needed=v::at<m::Expression>(expressions,i).builtin.kind==300;i+=1;}if(!needed){return .ok(r::empty());}
    if(frame.declaration==0 || frame.declaration>syntax::declaration_count(source)){return .err(m::fail(frame,12,0,0,0,0));}let d=syntax::declaration_at(source,frame.declaration-1);var r::Context c=r::Context{.frame,.work=*work,.groups=0};var v::Vec<r::Scope> regions=v::empty<r::Scope>();var v::Vec<r::Placement> placements=v::empty<r::Placement>();var v::Vec<r::Exit> exits=v::empty<r::Exit>();var v::Vec<usize> active=v::empty<usize>();var v::Vec<r::Loop> loops=v::empty<r::Loop>();let outcomes=switch(walk::block(source,imported,graph,locals,expressions,if(frame.kind==2){d.cleanup_body}else{d.body},0,ref mut active,ref mut loops,ref mut regions,ref mut placements,ref mut exits,ref mut c)){case .ok(n):{n}case .err(p):{return .err(p);}};if(v::len<usize>(ref active)!=0 || v::len<r::Loop>(ref loops)!=0 || (outcomes&12)!=0){return .err(m::fail(frame,12,d.start,d.end,0,0));};*work=c.work;return .ok(r::assemble(regions,placements,exits));
}
