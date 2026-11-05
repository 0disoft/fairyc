module project_calls_catalog;
import project_calls_options as options;
import project_calls_order as order;
import project_catalog_json as w;
import project_atlas_join as join;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_write as framing;
import project_json_tree as json;
import project_path as work;
import sha256_stream as sha;
import std::bytes as bytes;
import std::vec as v;

fn Result<bool,u32> context(ref mut bytes::Buffer out,ref validate::Partition p,ref mut usize meter){
    let t=validate::tree(p);let h=validate::header(p);discard w::raw(out,b"{",meter)?;
    let generic=wire::string(t,h,b"generic_interface_hash");if(generic.len>0){discard w::pair(out,b"generic_interface_hash",generic,meter)?;discard w::raw(out,b",",meter)?;}
    let names=b"logical_schema|compiler_build|language|package_uid|package_instance|package_coordinate|package_source|interface_hash|generation";var usize start=0;
    while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let key=names[start..end];if(start>0){discard w::raw(out,b",",meter)?;}
        discard w::pair(out,key,wire::string(t,h,if(bytes::equal(key,b"generation")){b"generation_status"}else{key}),meter)?;start=end+1;
    }
    discard w::raw(out,b",\"map_omissions\":",meter)?;discard w::strings(out,t,wire::field(t,h,b"omissions"),false,meter)?;
    discard w::raw(out,b",\"completeness\":[",meter)?;var usize i=0;var bool first=true;
    while(i<validate::completeness_count(p)){if(!work::charge(meter,1)){return .err(3);}let row=validate::completeness(p,i);let relation=wire::string(t,row,b"relation");
        if(bytes::equal(relation,b"calls") || validate::version(p)>1 && (bytes::equal(relation,b"generic-specializations") || bytes::equal(relation,b"generic-template-bodies"))){
            if(!first){discard w::raw(out,b",",meter)?;}first=false;
            discard w::raw(out,b"{",meter)?;discard w::pair(out,b"Relation",relation,meter)?;discard w::raw(out,b",",meter)?;
            discard w::pair(out,b"Status",wire::string(t,row,b"status"),meter)?;discard w::raw(out,b",\"Resolved\":",meter)?;discard w::number(out,wire::integer(t,row,b"resolved"),meter)?;
            discard w::raw(out,b",\"External\":",meter)?;discard w::number(out,wire::integer(t,row,b"external"),meter)?;discard w::raw(out,b",\"Unknown\":",meter)?;discard w::number(out,wire::integer(t,row,b"unknown"),meter)?;
            discard w::raw(out,b",\"Reasons\":",meter)?;discard w::strings(out,t,wire::field(t,row,b"reasons"),true,meter)?;discard w::raw(out,b"}",meter)?;
        }i+=1;
    }return w::raw(out,b"]}",meter);
}
fn Result<bool,u32> symbol(ref mut bytes::Buffer out,ref join::Graph g,join::Symbol s,ref u8[] implementation,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;discard w::pair(out,b"package_instance",join::instance(g,s.package),meter)?;discard w::raw(out,b",",meter)?;
    discard w::pair(out,b"symbol",join::visible(g,s),meter)?;if(implementation.len>0){discard w::raw(out,b",",meter)?;discard w::pair(out,b"specialization",implementation,meter)?;}return w::raw(out,b"}",meter);
}
fn Result<bool,u32> call(ref mut bytes::Buffer out,ref join::Graph g,join::Edge e,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;if(e.resource_body){discard w::raw(out,b"\"source_body\":\"cleanup\",",meter)?;}
    discard w::raw(out,b"\"source\":",meter)?;discard symbol(out,g,e.source,join::source_specialization(g,e),meter)?;
    discard w::raw(out,b",\"target\":",meter)?;discard symbol(out,g,e.target,join::target_specialization(g,e),meter)?;
    let p=join::partition(g,e.source.package);let t=validate::tree(p);let span=wire::field(t,e.call,b"span");discard w::raw(out,b",\"span\":{",meter)?;
    discard w::pair(out,b"package_instance",wire::string(t,span,b"package_instance"),meter)?;discard w::raw(out,b",",meter)?;discard w::pair(out,b"source",wire::string(t,span,b"source"),meter)?;
    discard w::raw(out,b",\"start_byte\":",meter)?;discard w::number(out,wire::integer(t,span,b"start_byte"),meter)?;discard w::raw(out,b",\"end_byte\":",meter)?;discard w::number(out,wire::integer(t,span,b"end_byte"),meter)?;
    // Join admission proves resolved exact calls from compiler-observed bodies.
    return w::raw(out,b"},\"certainty\":\"exact\",\"provenance\":\"compiler-observed\"}",meter);
}
fn Result<bytes::Buffer,u32> label(ref u8[] prefix,u32 version,ref u8[] suffix,ref mut usize meter){var out=bytes::empty();discard w::raw(ref mut out,prefix,meter)?;discard w::number(ref mut out,version as usize,meter)?;discard w::raw(ref mut out,suffix,meter)?;return .ok(out);}
fn Result<bytes::Buffer,u32> envelope(ref u8[] payload,u32 version,usize maximum,ref mut usize meter){
    let schema=label(b"fairyc.workspace-call-catalog/v",version,b"",meter)?;let encoding=label(b"fairyc.workspace-call-catalog-json/v",version,b"",meter)?;
    let algorithm=label(b"fairyc.workspace-call-catalog-checksum/v",version,b"-sha256",meter)?;var digest=label(b"fywcall",version,b"-sha256-",meter)?;
    var sha::State state=sha::empty();let tags=b"\x00\x01\x02";var u8[2] tag=[0,tags[version as usize]];
    if(!work::charge(meter,payload.len+1) || !sha::update(ref mut state,b"fairyc.workspace-call-catalog-checksum") || !sha::update(ref mut state,ref tag) || !framing::hash_number(ref mut state,bytes::len(ref schema)) || !sha::update(ref mut state,bytes::view(ref schema)) || !framing::hash_number(ref mut state,bytes::len(ref encoding)) || !sha::update(ref mut state,bytes::view(ref encoding)) || !framing::hash_number(ref mut state,payload.len) || !sha::update(ref mut state,payload)){return .err(3);}
    if(!sha::append_hex(ref state,ref mut digest)){return .err(2);}var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;
    discard w::pair(ref mut out,b"catalog_version",bytes::view(ref schema),meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"encoding",bytes::view(ref encoding),meter)?;
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"checksum_algorithm",bytes::view(ref algorithm),meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"checksum",bytes::view(ref digest),meter)?;
    discard w::raw(ref mut out,b",\"catalog\":",meter)?;discard w::raw(ref mut out,payload,meter)?;discard w::raw(ref mut out,b"}",meter)?;if(bytes::len(ref out)>maximum){return .err(3);}return .ok(out);
}
// Current Graph authority is mandatory. Never convert/relabel map versions or
// echo caller bounds that differ from the actual join's admission budget.
pub fn Result<bytes::Buffer,u32> encode(ref join::Graph g,ref options::Options request,ref mut usize meter){
    if(join::package_count(g)==0 || validate::version(join::partition(g,0))!=options::map_version(request) || join::max_packages(g)!=options::packages(request) || join::max_calls(g)!=options::calls(request)){return .err(9);}
    var usize selected=join::package_count(g);var usize matches=0;var usize i=0;
    while(i<join::package_count(g)){if(!work::charge(meter,1)){return .err(3);}let p=join::partition(g,i);if(bytes::equal(wire::string(validate::tree(p),validate::header(p),b"package_uid"),options::uid(request))){selected=i;matches+=1;}i+=1;}
    if(matches!=1){return .err(9);}let subject=join::resolve(g,join::instance(g,selected),options::symbol(request),meter)?;
    var rows=v::empty<usize>();var usize cross=0;i=0;
    while(i<join::edge_count(g)){if(!work::charge(meter,1)){return .err(3);}let e=join::edge(g,i);if(e.source.package!=e.target.package){cross+=1;}
        if(options::outgoing(request) && join::equal(e.source,subject) || options::incoming(request) && join::equal(e.target,subject)){if(!v::push<usize>(ref mut rows,i).is_ok){return .err(2);}}i+=1;
    }
    let ordered=order::sort(g,rows,meter)?;var calls=bytes::empty();var usize count=0;var usize previous=0;i=0;
    while(i<v::len<usize>(ref ordered)){let at=*v::at<usize>(ref ordered,i);let e=join::edge(g,at);
        if(count==0 || order::compare(g,join::edge(g,previous),e,meter)?!=0){if(count>0){discard w::raw(ref mut calls,b",",meter)?;}discard call(ref mut calls,g,e,meter)?;previous=at;count+=1;}i+=1;
    }
    var out=bytes::empty();discard w::raw(ref mut out,b"{\"context\":",meter)?;discard context(ref mut out,join::partition(g,selected),meter)?;
    discard w::raw(ref mut out,b",\"subject\":",meter)?;discard symbol(ref mut out,g,subject,b"",meter)?;discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"direction",options::direction(request),meter)?;
    discard w::raw(ref mut out,b",\"budget\":{\"max_packages\":",meter)?;discard w::number(ref mut out,join::max_packages(g),meter)?;discard w::raw(ref mut out,b",\"max_calls\":",meter)?;discard w::number(ref mut out,join::max_calls(g),meter)?;
    discard w::raw(ref mut out,b"},\"usage\":{\"packages\":",meter)?;discard w::number(ref mut out,join::package_count(g),meter)?;discard w::raw(ref mut out,b",\"calls\":",meter)?;discard w::number(ref mut out,join::edge_count(g),meter)?;
    discard w::raw(ref mut out,b",\"cross_package_calls\":",meter)?;discard w::number(ref mut out,cross,meter)?;discard w::raw(ref mut out,b",\"selected_calls\":",meter)?;discard w::number(ref mut out,count,meter)?;
    discard w::raw(ref mut out,b"},\"calls\":[",meter)?;discard w::raw(ref mut out,bytes::view(ref calls),meter)?;discard w::raw(ref mut out,b"],\"complete\":true}",meter)?;
    return envelope(bytes::view(ref out),options::version(request),options::output(request),meter);
}
