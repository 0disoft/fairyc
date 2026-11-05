module project_pack_snapshot;
import project_pack_projection as projection;
import project_catalog_json as w;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;

// Checked map wire records and the detached Go snapshot structs deliberately
// use different field names. Do not forward raw wire JSON into a query node.
pub fn Result<bool,u32> key(ref mut bytes::Buffer out,ref json::Tree t,usize at,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;
    discard projection::fields(out,t,at,b"PackageInstance=package_instance|Module=module|Kind=kind|Name=name",meter)?;
    return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> span(ref mut bytes::Buffer out,ref json::Tree t,usize at,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;discard projection::fields(out,t,at,b"PackageInstance=package_instance|Source=source",meter)?;
    discard w::raw(out,b",\"StartByte\":",meter)?;discard w::number(out,wire::integer(t,at,b"start_byte"),meter)?;
    discard w::raw(out,b",\"EndByte\":",meter)?;discard w::number(out,wire::integer(t,at,b"end_byte"),meter)?;return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> type(ref mut bytes::Buffer out,ref json::Tree t,usize at,usize depth,ref mut usize meter){
    if(depth>=64 || !work::charge(meter,1)){return .err(3);}
    discard w::raw(out,b"{",meter)?;discard projection::fields(out,t,at,b"Kind=kind|Name=name|PackageCoordinate=package_coordinate",meter)?;
    discard w::raw(out,b",\"Mutable\":",meter)?;discard w::boolean(out,wire::boolean(t,at,b"mutable"),meter)?;
    discard w::raw(out,b",\"Length\":",meter)?;discard w::number(out,wire::integer(t,at,b"length"),meter)?;
    discard w::raw(out,b",\"Arguments\":",meter)?;discard types(out,t,wire::field(t,at,b"arguments"),depth+1,meter)?;return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> types(ref mut bytes::Buffer out,ref json::Tree t,usize at,usize depth,ref mut usize meter){
    let n=wire::count(t,at);if(n==0){return w::raw(out,b"null",meter);}discard w::raw(out,b"[",meter)?;
    var usize i=0;var usize row=at+1;while(i<n){if(i>0){discard w::raw(out,b",",meter)?;}discard type(out,t,row,depth,meter)?;row=json::next(t,row);if(json::token(t,row).kind==6){row+=1;}i+=1;}
    return w::raw(out,b"]",meter);
}
fn Result<bool,u32> field(ref mut bytes::Buffer out,ref json::Tree t,usize at,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;discard projection::fields(out,t,at,b"Name=name",meter)?;
    discard w::raw(out,b",\"Type\":",meter)?;discard type(out,t,wire::field(t,at,b"type"),0,meter)?;
    discard w::raw(out,b",\"Span\":",meter)?;discard span(out,t,wire::field(t,at,b"span"),meter)?;return w::raw(out,b"}",meter);
}
// shape 0 = field, 1 = parameter, 2 = variant. Empty snapshot slices are nil;
// nonempty order is already authenticated by the owning current-map reader.
fn Result<bool,u32> sequence(ref mut bytes::Buffer out,ref json::Tree t,usize at,u32 shape,ref mut usize meter){
    let n=wire::count(t,at);if(n==0){return w::raw(out,b"null",meter);}discard w::raw(out,b"[",meter)?;
    var usize i=0;var usize row=at+1;while(i<n){if(!work::charge(meter,1)){return .err(3);}if(i>0){discard w::raw(out,b",",meter)?;}
        if(shape==0){discard field(out,t,row,meter)?;}
        else if(shape==1){
            discard w::raw(out,b"{",meter)?;discard projection::fields(out,t,row,b"Name=name",meter)?;
            discard w::raw(out,b",\"Type\":",meter)?;discard type(out,t,wire::field(t,row,b"type"),0,meter)?;
            discard w::raw(out,b",\"Mutable\":",meter)?;discard w::boolean(out,wire::boolean(t,row,b"mutable"),meter)?;
            discard w::raw(out,b",\"Span\":",meter)?;discard span(out,t,wire::field(t,row,b"span"),meter)?;discard w::raw(out,b"}",meter)?;
        }else{
            discard w::raw(out,b"{",meter)?;discard projection::fields(out,t,row,b"Name=name",meter)?;
            discard w::raw(out,b",\"Span\":",meter)?;discard span(out,t,wire::field(t,row,b"span"),meter)?;
            discard w::raw(out,b",\"Fields\":",meter)?;discard sequence(out,t,wire::field(t,row,b"fields"),0,meter)?;discard w::raw(out,b"}",meter)?;
        }
        row=json::next(t,row);if(json::token(t,row).kind==6){row+=1;}i+=1;
    }return w::raw(out,b"]",meter);
}
pub fn Result<bool,u32> declaration(ref mut bytes::Buffer out,ref json::Tree t,usize at,ref mut usize meter){
    discard w::raw(out,b"{\"Key\":",meter)?;discard key(out,t,wire::field(t,at,b"key"),meter)?;
    discard w::raw(out,b",\"Span\":",meter)?;discard span(out,t,wire::field(t,at,b"span"),meter)?;
    discard w::raw(out,b",",meter)?;discard projection::fields(out,t,at,b"Provenance=provenance",meter)?;
    discard w::raw(out,b",\"Public\":",meter)?;discard w::boolean(out,wire::boolean(t,at,b"public"),meter)?;
    discard w::raw(out,b",\"Unsafe\":",meter)?;discard w::boolean(out,wire::boolean(t,at,b"unsafe"),meter)?;
    discard w::raw(out,b",\"Foreign\":",meter)?;discard w::boolean(out,wire::boolean(t,at,b"foreign"),meter)?;
    discard w::raw(out,b",",meter)?;discard projection::fields(out,t,at,b"ABI=abi|LinkName=link_name",meter)?;
    discard w::raw(out,b",\"Type\":",meter)?;discard type(out,t,wire::field(t,at,b"type"),0,meter)?;
    discard w::raw(out,b",\"Owner\":",meter)?;let owner=wire::field(t,at,b"owner");if(owner>=json::count(t) || json::token(t,owner).kind!=1){discard w::raw(out,b"null",meter)?;}else{discard type(out,t,owner,0,meter)?;}
    discard w::raw(out,b",",meter)?;discard projection::fields(out,t,at,b"Receiver=receiver",meter)?;
    discard w::raw(out,b",\"Parameters\":",meter)?;discard sequence(out,t,wire::field(t,at,b"parameters"),1,meter)?;
    discard w::raw(out,b",\"Effects\":",meter)?;discard w::strings(out,t,wire::field(t,at,b"effects"),true,meter)?;
    discard w::raw(out,b",\"Fields\":",meter)?;discard sequence(out,t,wire::field(t,at,b"fields"),0,meter)?;
    discard w::raw(out,b",\"Variants\":",meter)?;discard sequence(out,t,wire::field(t,at,b"variants"),2,meter)?;
    discard w::raw(out,b",\"ReprC\":",meter)?;discard w::boolean(out,wire::boolean(t,at,b"repr_c"),meter)?;
    discard w::raw(out,b",",meter)?;discard projection::fields(out,t,at,b"DropKind=drop_kind",meter)?;
    discard w::raw(out,b",\"DropEffects\":",meter)?;discard w::strings(out,t,wire::field(t,at,b"drop_effects"),true,meter)?;return w::raw(out,b"}",meter);
}
