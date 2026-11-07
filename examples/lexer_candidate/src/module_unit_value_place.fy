module module_unit_value_place;
import module_unit_value_model as m;
import module_types as types;
import std::vec as v;
import module_unit_value_index as indexes;
import module_unit_value_view as views;
import module_unit_interfaces as interfaces;
import pointer_types as pointers;

pub fn m::Place field(ref v::Vec<types::Type> graph,m::Expression receiver,m::Expression selected){
    if(selected.field.id==0 || receiver.type==0){return m::no_place();}let t=types::at(graph,receiver.type);var p=receiver.place;
    if(p.kind==0 && t.kind!=7 && t.kind!=8){return m::no_place();}p.kind=2;p.projection=selected.source_index;p.indirect=p.indirect || t.kind==7 || t.kind==8;
    // ADR 0391 permits shared-reference call projections. ADRs 0445/0446
    // restrict mutable-reference field places to named roots; grouping keeps
    // that root place, while a returned temporary reference has no named place.
    if(t.kind==7 || t.kind==8){let named_mutable=receiver.place.kind==1 && receiver.place.root!=0;p.mutable=t.kind==8 && named_mutable;p.borrowable=t.kind==7 || named_mutable;}let owner=types::at(graph,selected.field.owner);p.resource_field=p.resource_field || owner.kind==3 || owner.kind==13 || owner.kind==21;return p;
}
pub fn bool referent(types::Type t){return t.kind==1 && t.scalar>=1 && t.scalar<=6 || t.kind==2 || t.kind==3 || t.kind==4 || t.kind==5 || t.kind==6 || t.kind==13 || t.kind==20 || t.kind==21 || t.kind==12 && t.first==0 && t.scalar==1048591;}
pub fn bool raw_mutable(types::Type t){return t.kind==12 && if(t.first==0){pointers::mutable(t.scalar)}else{t.scalar==1};}
pub fn bool opaque(types::Type t){return t.kind==12 && t.first==0 && pointers::valid(t.scalar) && pointers::inner(t.scalar)==7;}
// The caller also checks lexical unsafe provenance and direct local syntax for
// void erasure. A safe view yields its data address, never its carrier address.
pub fn bool raw_cast(types::Type source,types::Type target,types::Type referent){
    if(target.kind!=12){return false;}let exclusive=raw_mutable(target);
    if(source.kind==7 || source.kind==8){if(exclusive && source.kind!=8){return false;}if(opaque(target)){return true;}return if(target.first!=0){source.first==target.first && referent.kind==2}else{pointers::valid(target.scalar) && (referent.kind==1 && referent.scalar==pointers::inner(target.scalar) || referent.kind==12 && referent.first==0 && referent.scalar==1048591 && target.scalar==1048601)};}
    if(!views::view(source) && source.kind!=9 && source.kind!=14){return false;}if(exclusive && !views::mutable(source)){return false;}
    return opaque(target) || target.first==0 && pointers::valid(target.scalar) && pointers::inner(target.scalar)==3 && (source.kind==9 || source.kind==10 || source.kind==11 || source.kind==14);
}
fn m::Problem borrow_problem(m::Frame frame,m::Expression expression,m::Expression target,bool mutable){
    let bool named=target.kind==2 || target.place.kind==2 || target.place.kind==4;let bool addressable=named && target.place.borrowable && target.place.kind!=0;
    if(mutable && (!addressable || !target.place.mutable || target.place.resource_field)){return m::fail(frame,1008,target.start,target.end,0,target.type);}
    if(!addressable){return m::fail(frame,2,expression.start,expression.end,0,target.type);}return m::fail(frame,0,0,0,0,0);
}
fn Result<usize,m::Problem> canonical_borrow(m::Frame frame,m::Expression expression,usize target,bool mutable,ref mut v::Vec<types::Type> graph,ref mut usize work){
    if(!m::charge(work,types::count(graph)+1)){return .err(m::fail(frame,10,expression.start,expression.end,0,0));}let usize id=types::canonical(graph,types::Type{.kind=if(mutable){8}else{7},.scalar=0,.module_index=0,.declaration=0,.first=target,.second=0});return if(id==0){.err(m::fail(frame,if(types::count(graph)>=4096){10}else{11},expression.start,expression.end,0,0))}else{.ok(id)};
}
pub fn Result<usize,m::Problem> borrow(m::Frame frame,m::Expression expression,m::Expression target,bool mutable,ref mut v::Vec<types::Type> graph,ref mut usize work){
    let p=borrow_problem(frame,expression,target,mutable);if(p.code!=0){return .err(p);}return canonical_borrow(frame,expression,target.type,mutable,graph,work);
}
// Values validates the imported prefix once, then only appends canonical types.
// Preserve validation/error order and the ordinary path for every cache miss.
pub fn Result<usize,m::Problem> borrow_cached(m::Frame frame,m::Expression expression,m::Expression target,bool mutable,ref interfaces::Interfaces imported,ref mut v::Vec<types::Type> graph,ref mut usize work){
    let p=borrow_problem(frame,expression,target,mutable);if(p.code!=0){return .err(p);}
    if(target.type!=0 && target.type<=interfaces::type_count(imported)){if(!m::charge(work,1)){return .err(m::fail(frame,10,expression.start,expression.end,0,0));}let id=interfaces::borrow_type(imported,target.type,mutable);
        if(id!=0 && id<=types::count(graph)){let t=types::at(graph,id);if(t.kind==if(mutable){8}else{7} && t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==target.type && t.second==0){return .ok(id);}}}
    return canonical_borrow(frame,expression,target.type,mutable,graph,work);
}
pub fn bool same(m::Place a,m::Place b){return a.kind==b.kind && a.root==b.root && a.scope==b.scope && a.projection==b.projection && a.mutable==b.mutable && a.indirect==b.indirect && a.resource_field==b.resource_field && a.borrowable==b.borrowable;}
pub fn m::Place projected(ref v::Vec<types::Type> graph,m::Expression child,m::Expression expression){
    if(expression.kind==9){return indexes::place(graph,child,expression);}if(expression.kind==3){return child.place;}if(expression.kind==8){return field(graph,child,expression);}var p=child.place;
    if(expression.kind==26){p.kind=0;p.projection=expression.source_index;p.mutable=views::mutable(types::at(graph,expression.type));p.indirect=true;p.borrowable=false;return p;}
    if(expression.kind==21 || expression.kind==22){p.kind=0;p.mutable=expression.kind==22;p.indirect=p.indirect || views::view(types::at(graph,expression.type));p.borrowable=false;return p;}
    if(expression.kind==4 && expression.place.kind==3){p.kind=3;p.projection=expression.source_index;p.mutable=types::at(graph,child.type).kind==8;p.indirect=true;p.borrowable=false;return p;}return expression.place;
}
