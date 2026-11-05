module project_catalog;
import project_catalog_options as options;
import project_catalog_projection as projection;
import project_catalog_json as w;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_model as model;
import project_atlas_wire as wire;
import project_atlas_write as framing;
import project_json_tree as json;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;

struct Selection{usize total;usize start;usize skipped;}
fn Result<bool,u32> matches(ref validate::Partition p,usize index,ref options::Options request,ref mut usize meter){
    let t=validate::tree(p);let key=wire::field(t,validate::symbol(p,index),b"declaration");
    let module_name=wire::string(t,key,b"module");let name=wire::string(t,key,b"name");
    if(!work::charge(meter,module_name.len+name.len+options::module_name(request).len+options::name(request).len+32)){return .err(3);}
    if(options::module_name(request).len>0 && !bytes::equal(module_name,options::module_name(request)) || options::name(request).len>0 && !bytes::equal(name,options::name(request))){return .ok(false);}
    var usize i=0;while(i<5){if(options::has_kind(request,i) && bytes::equal(wire::string(t,key,b"kind"),options::kind(i))){return .ok(true);}i+=1;}return .ok(false);
}
fn Result<Selection,u32> selection(ref validate::Partition p,ref options::Options request,ref mut usize meter){
    let t=validate::tree(p);let h=validate::header(p);
    if(validate::version(p)!=options::version(request) || !bytes::equal(wire::string(t,h,b"package_uid"),options::uid(request))){return .err(9);}
    if(options::module_name(request).len>0){let modules=model::rows(t,wire::field(t,validate::root(p),b"modules"),4096,meter)?;
        let at=model::find(t,ref modules,b"path",options::module_name(request),meter);if(*meter>=16777216){return .err(3);}if(at==json::count(t)){return .err(9);}
    }
    // Semantic admission already proves strict full-ID order and a one-to-one
    // original declaration association. Preserve it in two linear passes.
    var usize total=0;var usize start=0;var usize skipped=0;var usize i=0;var bool found=options::after(request).len==0;
    while(i<validate::symbol_count(p)){if(matches(p,i,request,meter)?){total+=1;
        if(options::after(request).len>0 && bytes::equal(wire::string(t,validate::symbol(p,i),b"full"),options::after(request))){found=true;start=i+1;skipped=total;}
    }i+=1;}
    if(!found){return .err(9);}return .ok(Selection{.total,.start,.skipped});
}
fn Result<bytes::Buffer,u32> label(ref u8[] prefix,u32 version,ref u8[] suffix,ref mut usize meter){var out=bytes::empty();discard w::raw(ref mut out,prefix,meter)?;discard w::number(ref mut out,version as usize,meter)?;discard w::raw(ref mut out,suffix,meter)?;return .ok(out);}
fn Result<bytes::Buffer,u32> envelope(ref u8[] payload,u32 version,usize maximum,ref mut usize meter){
    let schema=label(b"fairyc.symbol-catalog/v",version,b"",meter)?;let encoding=label(b"fairyc.symbol-catalog-json/v",version,b"",meter)?;
    let algorithm=label(b"fairyc.symbol-catalog-checksum/v",version,b"-sha256",meter)?;var digest=label(b"fyscat",version,b"-sha256-",meter)?;
    var sha::State state=sha::empty();let tags=b"\x00\x01\x02";var u8[2] tag=[0,tags[(version-1) as usize]];
    if(!work::charge(meter,payload.len+1) || !sha::update(ref mut state,b"fairyc.symbol-catalog-checksum") || !sha::update(ref mut state,ref tag) || !framing::hash_number(ref mut state,bytes::len(ref schema)) || !sha::update(ref mut state,bytes::view(ref schema)) || !framing::hash_number(ref mut state,bytes::len(ref encoding)) || !sha::update(ref mut state,bytes::view(ref encoding)) || !framing::hash_number(ref mut state,payload.len) || !sha::update(ref mut state,payload)){return .err(3);}
    if(!sha::append_hex(ref state,ref mut digest)){return .err(2);}var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;
    discard w::pair(ref mut out,b"catalog_version",bytes::view(ref schema),meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"encoding",bytes::view(ref encoding),meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"checksum_algorithm",bytes::view(ref algorithm),meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"checksum",bytes::view(ref digest),meter)?;discard w::raw(ref mut out,b",\"catalog\":",meter)?;
    discard w::raw(ref mut out,payload,meter)?;discard w::raw(ref mut out,b"}",meter)?;if(bytes::len(ref out)>maximum){return .err(3);}return .ok(out);
}
// Output only. No map conversion, workspace/cache reads or publication. Ready
// carries independently authenticated current-source facts for the exact schema.
pub fn Result<bytes::Buffer,u32> encode(ref current::Ready ready,ref options::Options request,ref mut usize meter){
    let p=current::partition(ready);let selected=selection(p,request,meter)?;var mandatory=bytes::empty();
    discard w::raw(ref mut mandatory,b"{\"context\":",meter)?;discard projection::context(ref mut mandatory,p,false,meter)?;
    discard projection::filters(ref mut mandatory,request,selected.total,meter)?;discard w::raw(ref mut mandatory,b"}",meter)?;
    var usize used=bytes::len(ref mandatory)+1;if(used>options::selection(request)){return .err(3);}
    var rows=bytes::empty();var usize count=0;var usize i=selected.start;var bool limited=false;var bool symbol_limit=false;var last=bytes::empty();
    while(i<validate::symbol_count(p)){if(matches(p,i,request,meter)?){let row=projection::entry(p,i,meter)?;let cost=bytes::len(ref row)+1;
        if(count>=options::symbols(request) || cost>options::selection(request)-used){if(count==0){return .err(3);}limited=true;symbol_limit=count>=options::symbols(request);break;}
        if(count>0){discard w::raw(ref mut rows,b",",meter)?;}discard w::raw(ref mut rows,bytes::view(ref row),meter)?;
        discard bytes::truncate(ref mut last,0);discard w::raw(ref mut last,wire::string(validate::tree(p),validate::symbol(p,i),b"full"),meter)?;
        count+=1;used+=cost;
    }i+=1;}
    var out=bytes::empty();discard w::raw(ref mut out,b"{\"context\":",meter)?;discard projection::context(ref mut out,p,true,meter)?;
    discard projection::filters(ref mut out,request,selected.total,meter)?;discard w::raw(ref mut out,b",\"symbols\":[",meter)?;
    discard w::raw(ref mut out,bytes::view(ref rows),meter)?;discard w::raw(ref mut out,b"]",meter)?;
    if(limited){discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"next_after",bytes::view(ref last),meter)?;}
    discard w::raw(ref mut out,b",\"complete\":",meter)?;discard w::boolean(ref mut out,!limited,meter)?;
    discard w::raw(ref mut out,b",\"budget\":{\"max_symbols\":",meter)?;discard w::number(ref mut out,options::symbols(request),meter)?;
    discard w::raw(ref mut out,b",\"max_selection_bytes\":",meter)?;discard w::number(ref mut out,options::selection(request),meter)?;
    discard w::raw(ref mut out,b"},\"usage\":{\"symbols\":",meter)?;discard w::number(ref mut out,count,meter)?;
    discard w::raw(ref mut out,b",\"selection_bytes\":",meter)?;discard w::number(ref mut out,used,meter)?;
    discard w::raw(ref mut out,b"},\"omissions\":[",meter)?;
    if(limited){discard w::raw(ref mut out,b"{\"code\":",meter)?;discard w::string(ref mut out,if(symbol_limit){b"symbol-budget"}else{b"selection-byte-budget"},meter)?;
        discard w::raw(ref mut out,b",\"section\":\"symbols\",\"count\":",meter)?;discard w::number(ref mut out,selected.total-selected.skipped-count,meter)?;discard w::raw(ref mut out,b",\"count_known\":true}",meter)?;
    }discard w::raw(ref mut out,b"]}",meter)?;return envelope(bytes::view(ref out),options::version(request),options::output(request),meter);
}
