module module_unit_value_iteration;
import module_unit_value_model as m;
import module_unit_value_scalar as scalar;
import module_unit_value_index as index;
import module_unit_value_view as views;
import module_unit_interfaces as interfaces;
import module_types as types;
import syntax as syntax;
import std::vec as v;

// Loop headers have no written initializer, but their source still needs value
// evaluation before the origin pass can inspect its canonical type.
pub fn m::Problem resolve(ref syntax::Parsed source,m::Frame frame,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref m::Atoms atoms,ref v::Vec<m::Local> locals,ref mut v::Vec<m::Expression> expressions,ref mut usize work){
    var usize local=0;while(local<v::len<m::Local>(locals)){let item=*v::at<m::Local>(locals,local);if(item.kind==5){var usize statement=1;var bool found=false;while(statement<=syntax::statement_count(source)){if(!m::charge(work,1)){return m::fail(frame,10,item.start,item.end,0,0);}let s=syntax::statement_at(source,statement);if(s.kind==18 && s.target==item.source_index){found=true;let id=m::index(expressions,s.expression);if(id==0){return m::fail(frame,12,s.start,s.end,0,0);}let checked=scalar::evaluate(source,frame,id,0,imported,graph,atoms,expressions,work,0);if(checked.code!=0){return checked;}let e=*v::at<m::Expression>(expressions,id-1);if(e.type!=0 && item.type!=0){let source_type=types::at(graph,e.type);let item_type=types::at(graph,item.type);let borrowed=item_type.kind==7 || item_type.kind==8;let element=if(borrowed){item_type.first}else{item.type};let selected=index::element(graph,atoms,e.type);if(selected==0 || selected!=element || source_type.kind!=16 && !views::view(source_type) || item_type.kind==8 && !(source_type.kind==16 && e.place.mutable || views::mutable(source_type))){return m::fail(frame,2,e.start,e.end,element,selected);}}break;}statement+=1;}if(!found){return m::fail(frame,12,item.start,item.end,0,0);}}local+=1;}return m::fail(frame,0,0,0,0,0);
}
