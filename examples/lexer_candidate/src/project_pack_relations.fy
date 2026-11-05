module project_pack_relations;
import project_pack_snapshot as snapshot;
import project_pack_projection as projection;
import project_pack_options as options;
import project_catalog_json as w;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Token coordinates stay owned by the exact-current map. Sorting and query
// selection use owned indices; neither edits the map or merges dependency maps.
pub struct Relation{usize source;usize implementation;bool resource_body;usize kind;usize item;usize body;usize span;usize target;}
pub fn ref u8[] source(ref validate::Partition p,Relation r){return wire::string(validate::tree(p),r.source,b"visible");}
pub fn ref u8[] specialization(ref validate::Partition p,Relation r){return wire::string(validate::tree(p),r.implementation,b"id");}
fn Result<bool,u32> body(ref mut v::Vec<Relation> rows,ref validate::Partition p,usize at,usize implementation,bool resource_body,ref mut usize meter){
    let t=validate::tree(p);let source=validate::symbol_for_key(p,t,wire::field(t,at,b"function"),meter)?;if(source>=json::count(t)){return .err(9);}
    let names=b"allocation_sites|borrows|calls|drop_sites|trap_categories|move_sites|return_error_sites|unsafe_sites";var usize start=0;var usize category=0;
    while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let list=wire::field(t,at,names[start..end]);let n=wire::count(t,list);var usize i=0;var usize item=list+1;
        while(i<n){if(v::len<Relation>(rows)>=65536 || !work::charge(meter,1)){return .err(3);}
            let usize kind=if(category==0){0}else if(category==1){if(bytes::equal(wire::string(t,item,b"kind"),b"mutable")){1}else{2}}else if(category==2){if(bytes::equal(wire::string(t,item,b"target_kind"),b"external")){4}else{3}}else if(category==3){5}else if(category==4){6}else if(category==5){7}else if(category==6){8}else{9};
            let span=if(kind==6){json::count(t)}else if(category==1 || category==2){wire::field(t,item,b"span")}else{item};
            let target=if(category==2){validate::symbol_for_key(p,t,wire::field(t,item,b"target"),meter)?}else{json::count(t)};
            if(!v::push<Relation>(rows,Relation{.source,.implementation,.resource_body,.kind,.item,.body=at,.span,.target}).is_ok){return .err(2);}
            item=json::next(t,item);if(json::token(t,item).kind==6){item+=1;}i+=1;
        }category+=1;start=end+1;
    }return .ok(true);
}
pub fn Result<v::Vec<Relation>,u32> build(ref current::Ready ready,ref mut usize meter){
    let p=current::partition(ready);let t=validate::tree(p);var rows=v::empty<Relation>();var usize i=0;
    while(i<validate::function_count(p)){discard body(ref mut rows,p,validate::function(p,i),0,false,meter)?;i+=1;}
    i=0;while(i<validate::specialization_count(p)){if(!work::charge(meter,1)){return .err(3);}let row=validate::specialization(p,i);let f=wire::field(t,row,b"function");let cleanup_body=wire::field(t,row,b"cleanup");
        if(f<json::count(t)){discard body(ref mut rows,p,f,row,false,meter)?;}else if(cleanup_body<json::count(t)){discard body(ref mut rows,p,cleanup_body,row,true,meter)?;}i+=1;
    }
    let expected=wire::integer(t,wire::field(t,validate::header(p),b"counts"),b"relations");if(v::len<Relation>(ref rows)!=expected){return .err(9);}return .ok(rows);
}
pub fn Result<bytes::Buffer,u32> encode(ref validate::Partition p,Relation r,ref mut usize meter){
    let t=validate::tree(p);var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;
    if(r.implementation!=0){discard w::pair(ref mut out,b"source_specialization",specialization(p,r),meter)?;discard w::raw(ref mut out,b",",meter)?;}
    if(r.resource_body){discard w::raw(ref mut out,b"\"source_body\":\"cleanup\",",meter)?;}
    discard w::pair(ref mut out,b"source",source(p,r),meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"kind",options::relation(r.kind),meter)?;
    if(r.kind==3 || r.kind==4){
        discard w::raw(ref mut out,b",\"target\":{",meter)?;let concrete=wire::string(t,r.item,b"specialization");if(concrete.len>0){discard w::pair(ref mut out,b"specialization",concrete,meter)?;discard w::raw(ref mut out,b",",meter)?;}
        if(r.target<json::count(t)){discard w::pair(ref mut out,b"symbol",wire::string(t,r.target,b"visible"),meter)?;discard w::raw(ref mut out,b",",meter)?;}
        discard projection::fields(ref mut out,t,wire::field(t,r.item,b"target"),b"package_instance=package_instance|module=module|kind=kind|name=name",meter)?;discard w::raw(ref mut out,b"}",meter)?;
    }
    if(r.span<json::count(t)){discard w::raw(ref mut out,b",\"span\":",meter)?;discard snapshot::span(ref mut out,t,r.span,meter)?;}
    let certainty=if(r.kind==3 || r.kind==4){wire::string(t,r.item,b"certainty")}else{b""};if(certainty.len>0){discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"certainty",certainty,meter)?;}
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"provenance",wire::string(t,r.body,b"provenance"),meter)?;
    if(r.kind==6){let detail=json::value(t,r.item);if(detail.len>0){discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"detail",detail,meter)?;}}
    if(r.kind==3 || r.kind==4){let abi=wire::string(t,r.item,b"abi");let link=wire::string(t,r.item,b"link_name");if(abi.len>0){discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"abi",abi,meter)?;}if(link.len>0){discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"link_name",link,meter)?;}}
    if((r.kind==1 || r.kind==2) && wire::boolean(t,r.item,b"escapes")){discard w::raw(ref mut out,b",\"escapes\":true",meter)?;}
    discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
