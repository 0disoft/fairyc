module module_unit_value_index;
import module_unit_value_model as m;
import module_types as types;
import scalar_views as views;
import std::vec as v;

pub fn usize element(ref v::Vec<types::Type> graph,ref m::Atoms atoms,usize receiver){
    let t=types::at(graph,receiver);if(t.kind==16){return if(types::at(graph,t.first).kind==16){0}else{t.first};}if(t.kind==17 || t.kind==18){return t.first;}if(t.kind==10 || t.kind==11){return m::atom(atoms,3);}if(t.kind==15 && views::extended(t.scalar)){return m::atom(atoms,views::element(t.scalar));}return 0;
}
pub fn m::Place place(ref v::Vec<types::Type> graph,m::Expression receiver,m::Expression selected){
    let t=types::at(graph,receiver.type);var p=receiver.place;p.kind=4;p.projection=selected.source_index;
    let bool view=t.kind==10 || t.kind==11 || t.kind==15 || t.kind==17 || t.kind==18;p.indirect=p.indirect || view;if(view){p.mutable=t.kind==11 || t.kind==15 && views::mutable(t.scalar) || t.kind==18;}
    // A second index into a copied row is readable but not a named address root.
    p.borrowable=p.root!=0 && receiver.place.borrowable && (receiver.place.kind==1 || receiver.place.kind==2);return p;
}
