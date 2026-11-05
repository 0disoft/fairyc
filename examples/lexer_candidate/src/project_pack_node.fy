module project_pack_node;
import project_pack_snapshot as snapshot;
import project_pack_projection as projection;
import project_pack_options as options;
import project_catalog_json as w;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;

struct Counts{usize[10] values;}
fn Result<bool,u32> add_counts(ref mut Counts counts,ref json::Tree t,usize body,ref mut usize meter){
    let names=b"allocation_sites|borrows|calls|drop_sites|trap_categories|move_sites|return_error_sites|unsafe_sites";var usize start=0;var usize category=0;
    while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let rows=wire::field(t,body,names[start..end]);let n=wire::count(t,rows);
        if(!work::charge(meter,n+1)){return .err(3);}
        if(category==1 || category==2){var usize i=0;var usize at=rows+1;while(i<n){
            let usize k=if(category==1){if(bytes::equal(wire::string(t,at,b"kind"),b"mutable")){1}else{2}}else{if(bytes::equal(wire::string(t,at,b"target_kind"),b"external")){4}else{3}};
            counts.values[k]+=1;at=json::next(t,at);if(json::token(t,at).kind==6){at+=1;}i+=1;
        }}else{let usize k=if(category==0){0}else if(category==3){5}else if(category==4){6}else if(category==5){7}else if(category==6){8}else{9};counts.values[k]+=n;}
        category+=1;start=end+1;
    }return .ok(true);
}
fn Result<bool,u32> facts(ref mut bytes::Buffer out,ref json::Tree t,usize body,ref mut usize meter){
    var Counts counts=Counts{.values=[0;10]};discard add_counts(ref mut counts,t,body,meter)?;
    discard w::raw(out,b"{",meter)?;discard w::pair(out,b"provenance",wire::string(t,body,b"provenance"),meter)?;
    discard w::raw(out,b",\"relations\":",meter)?;var bool first=true;var usize i=0;
    while(i<10){if(counts.values[i]>0){if(first){discard w::raw(out,b"[",meter)?;}else{discard w::raw(out,b",",meter)?;}first=false;
        discard w::raw(out,b"{",meter)?;discard w::pair(out,b"kind",options::relation(i),meter)?;discard w::raw(out,b",\"count\":",meter)?;discard w::number(out,counts.values[i],meter)?;discard w::raw(out,b"}",meter)?;
    }i+=1;}discard w::raw(out,if(first){b"null"}else{b"]"},meter)?;return w::raw(out,b"}",meter);
}
pub fn Result<bytes::Buffer,u32> encode(ref current::Ready ready,usize index,ref mut usize meter){
    let p=current::partition(ready);if(index>=validate::symbol_count(p)){return .err(9);}let t=validate::tree(p);let symbol=validate::symbol(p,index);let declaration=validate::symbol_declaration(p,index);
    var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;let binders=wire::field(t,declaration,b"type_parameters");
    if(binders<json::count(t) && wire::count(t,binders)>0){discard w::raw(ref mut out,b"\"type_parameters\":",meter)?;discard w::strings(ref mut out,t,binders,false,meter)?;discard w::raw(ref mut out,b",",meter)?;}
    var usize n=0;var usize i=0;while(i<validate::specialization_count(p)){if(!work::charge(meter,1)){return .err(3);}let row=validate::specialization(p,i);if(model::keys(t,wire::field(t,row,b"declaration"),wire::field(t,declaration,b"key"),meter)==0){n+=1;}i+=1;}
    if(n>0){discard w::raw(ref mut out,b"\"specializations\":[",meter)?;var usize written=0;i=0;
        while(i<validate::specialization_count(p)){if(!work::charge(meter,1)){return .err(3);}let row=validate::specialization(p,i);i+=1;if(model::keys(t,wire::field(t,row,b"declaration"),wire::field(t,declaration,b"key"),meter)!=0){continue;}
            if(written>0){discard w::raw(ref mut out,b",",meter)?;}written+=1;discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"id",wire::string(t,row,b"id"),meter)?;
            discard w::raw(ref mut out,b",\"type_arguments\":",meter)?;discard snapshot::types(ref mut out,t,wire::field(t,row,b"type_arguments"),0,meter)?;
            let body=wire::field(t,row,b"function");if(body<json::count(t)){discard w::raw(ref mut out,b",\"function\":",meter)?;discard facts(ref mut out,t,body,meter)?;}
            let aggregate=wire::field(t,row,b"aggregate");if(aggregate<json::count(t)){discard w::raw(ref mut out,b",\"aggregate\":",meter)?;discard snapshot::declaration(ref mut out,t,aggregate,meter)?;}
            let cleanup_body=wire::field(t,row,b"cleanup");if(cleanup_body<json::count(t)){discard w::raw(ref mut out,b",\"cleanup\":",meter)?;discard facts(ref mut out,t,cleanup_body,meter)?;}discard w::raw(ref mut out,b"}",meter)?;
        }discard w::raw(ref mut out,b"],",meter)?;
    }
    discard w::raw(ref mut out,b"\"symbol\":{\"Declaration\":",meter)?;discard snapshot::key(ref mut out,t,wire::field(t,symbol,b"declaration"),meter)?;
    discard w::raw(ref mut out,b",",meter)?;discard projection::fields(ref mut out,t,symbol,b"Visible=visible|Full=full",meter)?;
    discard w::raw(ref mut out,b"},\"declaration\":",meter)?;discard snapshot::declaration(ref mut out,t,declaration,meter)?;
    let body=validate::symbol_function(p,index);if(body<json::count(t)){discard w::raw(ref mut out,b",\"function\":",meter)?;discard facts(ref mut out,t,body,meter)?;}
    discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
