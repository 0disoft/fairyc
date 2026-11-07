module module_unit_value_index;
import module_unit_value_model as m;
import module_types as types;
import scalar_views as views;
import std::vec as v;

// For-in only: ordinary source indexing of Vec descriptors stays closed.
// SDK authentication owns kind 13; only shared descriptors and scalar items
// are admitted here, never resource or nested-vector elements.
pub fn usize vector_iteration_element(ref v::Vec<types::Type> graph,usize receiver){
    if(receiver==0 || receiver>types::count(graph)){return 0;}let reference=types::at(graph,receiver);if(reference.kind!=7 || reference.first==0 || reference.first>types::count(graph)){return 0;}
    let vector=types::at(graph,reference.first);if(vector.kind!=13 || vector.first==0 || vector.first>types::count(graph)){return 0;}let element=types::at(graph,vector.first);
    return if(element.kind==1 && element.scalar>=1 && element.scalar<=6){vector.first}else{0};
}

pub fn usize element(ref v::Vec<types::Type> graph,ref m::Atoms atoms,usize receiver){
    let t=types::at(graph,receiver);if(t.kind==16){return if(types::at(graph,t.first).kind==16){0}else{t.first};}if(t.kind==17 || t.kind==18){return t.first;}if(t.kind==10 || t.kind==11){return m::atom(atoms,3);}if(t.kind==15 && views::extended(t.scalar)){return m::atom(atoms,views::element(t.scalar));}return 0;
}
pub fn m::Place place(ref v::Vec<types::Type> graph,m::Expression receiver,m::Expression selected){
    let t=types::at(graph,receiver.type);var p=receiver.place;p.kind=4;p.projection=selected.source_index;
    let bool view=t.kind==10 || t.kind==11 || t.kind==15 || t.kind==17 || t.kind==18;p.indirect=p.indirect || view;if(view){p.mutable=t.kind==11 || t.kind==15 && views::mutable(t.scalar) || t.kind==18;}
    // A second index into a copied row is readable but not a named address root.
    p.borrowable=p.root!=0 && receiver.place.borrowable && (receiver.place.kind==1 || receiver.place.kind==2);return p;
}
