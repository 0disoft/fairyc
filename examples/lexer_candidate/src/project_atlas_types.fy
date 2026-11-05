module project_atlas_types;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import module_unit_cleanup_mir_model as clean;
import c_abi_types as c;
import generic_transport_scalars as scalars;
import scalar_views as views;
import pointer_types as pointers;
import project_atlas_write as w;
import project_path as work;
import syntax as syntax;
import std::bytes as bytes;
import module_types as graph;
import std::vec as v;
import project_atlas_sdk_vector as vector;

// Consume the factory's verified, source-bound type graph and exact package
// coordinates. Neither transient type IDs nor import aliases leave this port.
// 2 allocation; 3 shared work/depth; 7 invalid/unsupported type or coordinate.
pub fn Result<bytes::Buffer,u32> module_name(ref u8[] path,ref mut usize meter){
    if(path.len<4 || !bytes::equal(path[path.len-3..],b".fy")){return .err(7);}
    var out=bytes::empty();var usize i=0;
    while(i<path.len-3){discard w::raw(ref mut out,if(path[i]==47){b"::"}else{path[i..i+1]},meter)?;i+=1;}
    return .ok(out);
}
fn ref u8[] scalar(u32 id){
    if(id==1){return b"i32";}if(id==2){return b"i64";}if(id==3){return b"u8";}
    if(id==4){return b"u32";}if(id==5){return b"usize";}if(id==6){return b"bool";}return b"void";
}
pub fn Result<bool,u32> head(ref mut bytes::Buffer out,ref u8[] kind,ref u8[] name,ref u8[] coordinate,bool mutable,usize length,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;discard w::pair(out,b"kind",kind,meter)?;discard w::raw(out,b",",meter)?;
    discard w::pair(out,b"name",name,meter)?;discard w::raw(out,b",",meter)?;discard w::pair(out,b"package_coordinate",coordinate,meter)?;
    discard w::raw(out,if(mutable){b",\"mutable\":true,\"length\":"}else{b",\"mutable\":false,\"length\":"},meter)?;
    discard w::number(out,length,meter)?;return w::raw(out,b",\"arguments\":[",meter);
}
pub fn Result<bool,u32> builtin(ref mut bytes::Buffer out,ref u8[] name,ref mut usize meter){discard head(out,b"builtin",name,b"",false,0,meter)?;return w::raw(out,b"]}",meter);}
fn Result<bool,u32> scalar_type(ref mut bytes::Buffer out,u32 id,ref mut usize meter){
    if(c::valid(id)){
        let ref u8[] names=b"bool|char|double|float|int|long|longlong|schar|short|size_t|ssize_t|uchar|uint|ulong|ulonglong|ushort|";
        var u32 selected=128;var usize start=0;var usize end=0;
        while(end<names.len){if(names[end]==124){if(selected==id){break;}selected+=1;start=end+1;}end+=1;}
        var name=bytes::empty();discard w::raw(ref mut name,b"c::",meter)?;discard w::raw(ref mut name,names[start..end],meter)?;
        return builtin(out,bytes::view(ref name),meter);
    }
    if(scalars::extended(id)){return builtin(out,scalars::name(id),meter);}
    if(id==0 || id>7){return .err(7);}
    return builtin(out,scalar(id),meter);
}
fn Result<bool,u32> nominal_head(ref input::Input source,ref coordinates::Coordinates table,usize module_index,usize declaration,u32 kind,ref mut bytes::Buffer out,ref mut usize meter){
    if(module_index>=input::count(source) || declaration==0){return .err(7);}
    let parsed=input::source(source,module_index);if(declaration>syntax::declaration_count(parsed)){return .err(7);}
    let d=syntax::declaration_at(parsed,declaration-1);
    if(d.kind!=if(kind==2 || kind==20){4}else if(kind==3 || kind==13 || kind==21){6}else{11} || (d.generic_parameters!=0)!=(kind==13 || kind==20 || kind==21)){return .err(7);}
    if(kind==13 && !vector::declaration(source,module_index,declaration,meter)?){return .err(7);}
    let module_name=module_name(input::file(source,module_index),meter)?;var name=module_name;
    discard w::raw(ref mut name,b"::",meter)?;discard w::raw(ref mut name,syntax::declaration_name(parsed,declaration),meter)?;
    if(coordinates::coordinate_count(table)>512){return .err(7);}
    var usize found=coordinates::coordinate_count(table);var usize i=0;let owner=input::origin(source,module_index).owner;
    while(i<coordinates::coordinate_count(table)){
        if(!work::charge(meter,1)){return .err(3);}
        if(coordinates::coordinate_at(table,i).owner==owner){if(found!=coordinates::coordinate_count(table)){return .err(7);}found=i;}i+=1;
    }
    if(found==coordinates::coordinate_count(table)){return .err(7);}
    let coordinate=coordinates::coordinate_text(table,found);if(!coordinates::valid_coordinate(coordinate)){return .err(7);}
    return head(out,b"nominal",bytes::view(ref name),coordinate,false,0,meter);
}
fn Result<bool,u32> pointer_scalar(ref mut bytes::Buffer out,u32 id,usize depth,ref mut usize meter){
    if(depth>=64 || !work::charge(meter,1)){return .err(3);}
    if(id==1048591){if(depth+1>=64){return .err(3);}discard head(out,b"pointer",b"",b"",true,0,meter)?;discard builtin(out,b"void",meter)?;return w::raw(out,b"]}",meter);}
    return scalar_type(out,id,meter);
}
fn Result<bool,u32> walk(ref input::Input source,ref coordinates::Coordinates table,ref v::Vec<graph::Type> p,usize id,usize depth,ref mut bytes::Buffer out,ref mut usize meter){
    if(depth>=64 || !work::charge(meter,1)){return .err(3);}
    if(id==0 || id>graph::count(p)){return .err(7);}let t=graph::at(p,id);
    if(t.kind==1){return scalar_type(out,t.scalar,meter);}
    if(t.kind>=2 && t.kind<=4){discard nominal_head(source,table,t.module_index,t.declaration,t.kind,out,meter)?;return w::raw(out,b"]}",meter);}
    if(t.kind==13){
        if(t.scalar!=0 || t.second!=0 || t.first==0 || t.first>=id){return .err(7);}
        discard nominal_head(source,table,t.module_index,t.declaration,t.kind,out,meter)?;
        discard walk(source,table,p,t.first,depth+1,out,meter)?;return w::raw(out,b"]}",meter);
    }
    if(t.kind==20 || t.kind==21){
        discard nominal_head(source,table,t.module_index,t.declaration,t.kind,out,meter)?;let count=syntax::generic_parameter_count(input::source(source,t.module_index),t.declaration-1);
        if(t.scalar!=0 || t.second!=0 || count==0 || count>8){return .err(7);}var usize next=t.first;var usize i=0;
        while(next!=0){if(i>=count || next>=id || !work::charge(meter,1)){return .err(if(i>=count || next>=id){7}else{3});}let tuple=graph::at(p,next);if(tuple.kind!=22 || tuple.scalar!=0 || tuple.module_index!=0 || tuple.declaration!=0 || tuple.first==0 || tuple.first>=next || tuple.second>=next){return .err(7);}let argument=graph::at(p,tuple.first);if(argument.kind!=1 || !scalars::valid(argument.scalar)){return .err(7);}if(i>0){discard w::raw(out,b",",meter)?;}discard walk(source,table,p,tuple.first,depth+1,out,meter)?;next=tuple.second;i+=1;}
        if(i!=count){return .err(7);}return w::raw(out,b"]}",meter);
    }
    if(t.kind==19){return builtin(out,b"Region",meter);}
    if(t.kind==9 || t.kind==14){
        if(depth+1>=64 || !work::charge(meter,1)){return .err(3);}
        discard head(out,b"borrow",b"",b"",false,0,meter)?;
        discard builtin(out,if(t.kind==9){b"str"}else{b"CStr"},meter)?;return w::raw(out,b"]}",meter);
    }
    if(t.kind==10 || t.kind==11 || t.kind==15){
        let u32 tag=if(t.kind==10){10}else if(t.kind==11){11}else{t.scalar};if(!views::valid(tag)){return .err(7);}
        if(depth+1>=64 || !work::charge(meter,1)){return .err(3);}
        discard head(out,b"borrow",b"",b"",views::mutable(tag),0,meter)?;
        var name=bytes::empty();discard w::raw(ref mut name,b"[",meter)?;
        discard w::raw(ref mut name,scalar(views::element(tag)),meter)?;discard w::raw(ref mut name,b"]",meter)?;
        discard builtin(out,bytes::view(ref name),meter)?;return w::raw(out,b"]}",meter);
    }
    if(t.kind==12){
        if(t.first==0 && !pointers::valid(t.scalar)){return .err(7);}
        discard head(out,b"pointer",b"",b"",if(t.first==0){pointers::mutable(t.scalar)}else{t.scalar==1},0,meter)?;
        if(t.first!=0){discard walk(source,table,p,t.first,depth+1,out,meter)?;}
        else{discard pointer_scalar(out,pointers::inner(t.scalar),depth+1,meter)?;}
        return w::raw(out,b"]}",meter);
    }
    if(t.kind==17 || t.kind==18){
        discard head(out,b"borrow",b"",b"",t.kind==18,0,meter)?;discard head(out,b"slice",b"",b"",false,0,meter)?;
        discard walk(source,table,p,t.first,depth+2,out,meter)?;return w::raw(out,b"]}]}",meter);
    }
    if(t.kind==5 || t.kind==6 || t.kind==7 || t.kind==8 || t.kind==16){
        discard head(out,if(t.kind==5){b"option"}else if(t.kind==6){b"result"}else if(t.kind==16){b"fixed-array"}else{b"borrow"},b"",b"",t.kind==8,if(t.kind==16){t.second}else{0},meter)?;
        discard walk(source,table,p,t.first,depth+1,out,meter)?;
        if(t.kind==6){discard w::raw(out,b",",meter)?;discard walk(source,table,p,t.second,depth+1,out,meter)?;}
        return w::raw(out,b"]}",meter);
    }
    // SDK specialization/template facts need their own source identities.
    return .err(7);
}
pub fn Result<bytes::Buffer,u32> render(ref input::Input source,ref coordinates::Coordinates table,ref clean::Program p,usize id,ref mut usize meter){
    return render_records(source,table,clean::type_records(p),id,meter);
}
// The caller owns the current complete-source signature/field resolution.
// Detached type records alone grant no cache, source or declaration authority.
pub fn Result<bytes::Buffer,u32> render_records(ref input::Input source,ref coordinates::Coordinates table,ref v::Vec<graph::Type> p,usize id,ref mut usize meter){
    if(graph::count(p)>4096){return .err(7);}
    var out=bytes::empty();discard walk(source,table,p,id,0,ref mut out,meter)?;return .ok(out);
}
