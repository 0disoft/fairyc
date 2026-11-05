module project_pack;
import project_pack_overview as overview;
import project_pack_projection as projection;
import project_pack_query as query;
import project_pack_around as neighborhood;
import project_catalog_json as w;
import project_atlas_current as current;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_write as framing;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;

fn Result<bytes::Buffer,u32> label(ref u8[] prefix,u32 version,ref u8[] suffix,ref mut usize meter){var out=bytes::empty();discard w::raw(ref mut out,prefix,meter)?;discard w::number(ref mut out,version as usize,meter)?;discard w::raw(ref mut out,suffix,meter)?;return .ok(out);}
// The encoder serializes its logical body once, hashes those exact bytes and
// checks complete envelope length before allocating/copying the final buffer.
fn Result<bytes::Buffer,u32> envelope(ref u8[] payload,u32 version,usize maximum,ref mut usize meter){
    if(payload.len>maximum){return .err(3);}let schema=label(b"fairyc.context-pack/v",version,b"",meter)?;let encoding=label(b"fairyc.context-pack-json/v",version,b"",meter)?;
    let algorithm=label(b"fairyc.context-pack-checksum/v",version,b"-sha256",meter)?;var digest=label(b"fyctx",version,b"-sha256-",meter)?;
    var sha::State state=sha::empty();let tags=b"\x00\x01\x02\x03";var u8[2] tag=[0,tags[version as usize]];
    if(!work::charge(meter,payload.len+1) || !sha::update(ref mut state,b"fairyc.context-pack-checksum") || !sha::update(ref mut state,ref tag) || !framing::hash_number(ref mut state,bytes::len(ref schema)) || !sha::update(ref mut state,bytes::view(ref schema)) || !framing::hash_number(ref mut state,bytes::len(ref encoding)) || !sha::update(ref mut state,bytes::view(ref encoding)) || !framing::hash_number(ref mut state,payload.len) || !sha::update(ref mut state,payload)){return .err(3);}
    if(!sha::append_hex(ref state,ref mut digest)){return .err(2);}var header=bytes::empty();discard w::raw(ref mut header,b"{",meter)?;
    discard w::pair(ref mut header,b"pack_version",bytes::view(ref schema),meter)?;discard w::raw(ref mut header,b",",meter)?;discard w::pair(ref mut header,b"encoding",bytes::view(ref encoding),meter)?;
    discard w::raw(ref mut header,b",",meter)?;discard w::pair(ref mut header,b"checksum_algorithm",bytes::view(ref algorithm),meter)?;discard w::raw(ref mut header,b",",meter)?;discard w::pair(ref mut header,b"checksum",bytes::view(ref digest),meter)?;
    discard w::raw(ref mut header,b",\"pack\":",meter)?;let n=bytes::len(ref header);if(n>maximum || payload.len>maximum-n || maximum-n-payload.len<1){return .err(3);}
    var out=header;discard w::raw(ref mut out,payload,meter)?;discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
// Current-admitted maps only. No decoded pack or caller-mutated query result
// supplies compiler facts, and no map family is converted or relabeled.
pub fn Result<bytes::Buffer,u32> encode_overview(ref current::Ready ready,u32 version,usize records,usize selection,usize output,ref mut usize meter){
    let p=current::partition(ready);if(version<1 || version>3 || version!=validate::version(p) || output==0){return .err(9);}
    let selected=overview::select(ready,records,selection,meter)?;let t=validate::tree(p);let h=validate::header(p);var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;
    discard w::pair(ref mut out,b"map_version",wire::string(t,h,b"logical_schema"),meter)?;discard w::raw(ref mut out,b",\"package_identity\":",meter)?;discard projection::identity(ref mut out,p,meter)?;
    discard w::raw(ref mut out,b",\"compatibility\":",meter)?;discard projection::compatibility(ref mut out,p,meter)?;
    discard w::raw(ref mut out,b",\"query\":{\"kind\":\"overview\",",meter)?;discard w::pair(ref mut out,b"subject",wire::string(t,h,b"package_instance"),meter)?;
    discard w::raw(ref mut out,b",\"direction\":\"\",\"relations\":[]},\"budget\":{\"max_records\":",meter)?;discard w::number(ref mut out,records,meter)?;
    discard w::raw(ref mut out,b",\"max_relations\":0,\"max_depth\":0,\"max_selection_bytes\":",meter)?;discard w::number(ref mut out,selection,meter)?;
    discard w::raw(ref mut out,b"},\"usage\":{\"records\":",meter)?;discard w::number(ref mut out,overview::records(ref selected),meter)?;
    discard w::raw(ref mut out,b",\"relations\":0,\"depth\":0,\"selection_bytes\":",meter)?;discard w::number(ref mut out,overview::selection(ref selected),meter)?;
    discard w::raw(ref mut out,b"},\"completeness\":[",meter)?;discard w::raw(ref mut out,overview::completeness(ref selected),meter)?;
    discard w::raw(ref mut out,b"],\"facts\":{\"overview\":{\"package\":",meter)?;discard w::raw(ref mut out,overview::package(ref selected),meter)?;
    discard w::raw(ref mut out,b",\"modules\":[",meter)?;discard w::raw(ref mut out,overview::modules(ref selected),meter)?;
    discard w::raw(ref mut out,b"]},\"show\":null,\"around\":null},\"relations\":[],\"documentation\":[],\"source_excerpts\":[],\"omissions\":[",meter)?;
    discard w::raw(ref mut out,overview::omissions(ref selected),meter)?;discard w::raw(ref mut out,b"],\"warnings\":[]}",meter)?;
    return envelope(bytes::view(ref out),version,output,meter);
}
pub fn Result<bytes::Buffer,u32> encode_show(ref current::Ready ready,u32 version,ref u8[] symbol,query::Policy policy,usize output,ref mut usize meter){
    let p=current::partition(ready);if(version<1 || version>3 || version!=validate::version(p) || output==0){return .err(9);}
    let selected=query::show(ready,symbol,policy,meter)?;return encode_selected(p,version,policy,output,ref selected,false,meter);
}
pub fn Result<bytes::Buffer,u32> encode_around(ref current::Ready ready,u32 version,ref u8[] symbol,query::Policy policy,usize output,ref mut usize meter){
    let p=current::partition(ready);if(version<1 || version>3 || version!=validate::version(p) || output==0){return .err(9);}
    let selected=neighborhood::select(ready,symbol,policy,meter)?;return encode_selected(p,version,policy,output,ref selected,true,meter);
}
fn Result<bytes::Buffer,u32> encode_selected(ref validate::Partition p,u32 version,query::Policy policy,usize output,ref query::Selection selected,bool around,ref mut usize meter){
    let t=validate::tree(p);let h=validate::header(p);let usage=query::usage(selected);var out=bytes::empty();
    discard w::raw(ref mut out,b"{",meter)?;discard w::pair(ref mut out,b"map_version",wire::string(t,h,b"logical_schema"),meter)?;
    discard w::raw(ref mut out,b",\"package_identity\":",meter)?;discard projection::identity(ref mut out,p,meter)?;
    discard w::raw(ref mut out,b",\"compatibility\":",meter)?;discard projection::compatibility(ref mut out,p,meter)?;
    discard w::raw(ref mut out,b",\"query\":{",meter)?;discard w::pair(ref mut out,b"kind",if(around){b"around"}else{b"show"},meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"subject",query::subject(selected),meter)?;
    discard w::raw(ref mut out,b",",meter)?;let ref u8[] direction=if(policy.direction==0){b"outgoing"}else if(policy.direction==1){b"incoming"}else{b"both"};discard w::pair(ref mut out,b"direction",direction,meter)?;
    discard w::raw(ref mut out,b",\"relations\":",meter)?;discard query::kinds(ref mut out,policy.mask,meter)?;discard w::raw(ref mut out,b"},\"budget\":{\"max_records\":",meter)?;discard w::number(ref mut out,policy.records,meter)?;
    discard w::raw(ref mut out,b",\"max_relations\":",meter)?;discard w::number(ref mut out,policy.relations,meter)?;discard w::raw(ref mut out,b",\"max_depth\":",meter)?;discard w::number(ref mut out,policy.depth,meter)?;
    discard w::raw(ref mut out,b",\"max_selection_bytes\":",meter)?;discard w::number(ref mut out,policy.selection,meter)?;discard w::raw(ref mut out,b"},\"usage\":{\"records\":",meter)?;discard w::number(ref mut out,usage.records,meter)?;
    discard w::raw(ref mut out,b",\"relations\":",meter)?;discard w::number(ref mut out,usage.relations,meter)?;discard w::raw(ref mut out,b",\"depth\":",meter)?;discard w::number(ref mut out,usage.depth,meter)?;
    discard w::raw(ref mut out,b",\"selection_bytes\":",meter)?;discard w::number(ref mut out,usage.selection,meter)?;discard w::raw(ref mut out,b"},\"completeness\":[",meter)?;discard w::raw(ref mut out,query::completeness(selected),meter)?;
    if(around){discard w::raw(ref mut out,b"],\"facts\":{\"overview\":null,\"show\":null,\"around\":{\"nodes\":[",meter)?;discard w::raw(ref mut out,query::facts(selected),meter)?;discard w::raw(ref mut out,b"]}},\"relations\":[",meter)?;}
    else{discard w::raw(ref mut out,b"],\"facts\":{\"overview\":null,\"show\":{\"subject\":",meter)?;discard w::raw(ref mut out,query::facts(selected),meter)?;discard w::raw(ref mut out,b"},\"around\":null},\"relations\":[",meter)?;}
    discard w::raw(ref mut out,query::relations(selected),meter)?;
    discard w::raw(ref mut out,b"],\"documentation\":[],\"source_excerpts\":[],\"omissions\":[",meter)?;discard w::raw(ref mut out,query::omissions(selected),meter)?;
    discard w::raw(ref mut out,b"],\"warnings\":[]}",meter)?;return envelope(bytes::view(ref out),version,output,meter);
}
