module generic_package_context;
import generic_package_bindings as bindings;
import generic_source_module_json as modules;
import generic_source_function_json as functions;
import generic_source_aggregate_json as aggregates;
import generic_source_item_json as items;
import generic_source_observations as observations;
import generic_source_json as json;
import generic_identity as identity;
import module_public_interface_write as binary;
import project_path as bookkeeping;
import sha256 as hash;
import sha256_stream as sha;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Independent scopes: 16MiB graph bookkeeping, 1MiB per declaration/sidecar
// writer, 1GiB cumulative encoding work, and 96MiB streamed hash input. Neither
// the complete graph nor complete module JSON is retained between fragments.
pub struct Budget{usize work;usize encoded_work;usize hashed_bytes;u32 code;}
pub fn Budget empty(){return Budget{.work=0,.encoded_work=0,.hashed_bytes=0,.code=0};}
pub fn usize encoding_limit(){return 1073741824;}
pub fn usize hash_limit(){return 100663296;}
pub struct Stamp{usize size;u32[8] digest;}
pub resource Context{bytes::Buffer digest;v::Vec<Stamp> sources;usize json_bytes;usize peak_chunk_bytes;usize peak_chunk_work;cleanup{}}
pub fn ref u8[] digest(ref Context p){return bytes::view(ref p.digest);}
pub fn usize count(ref Context p){return v::len<Stamp>(ref p.sources);}
pub fn usize json_bytes(ref Context p){return p.json_bytes;}
pub fn usize peak_chunk_bytes(ref Context p){return p.peak_chunk_bytes;}
pub fn usize peak_chunk_work(ref Context p){return p.peak_chunk_work;}
fn bool reject(ref mut Budget b,u32 code){if(b.code==0){b.code=code;}return false;}
fn bool valid(ref mut Budget b){if(b.code!=0){return false;}return if(b.work<=16777216 && b.encoded_work<=encoding_limit() && b.hashed_bytes<=hash_limit()){true}else{reject(b,10)};}
fn bool charge(ref mut Budget b,usize cost){if(b.code!=0){return false;}return if(bookkeeping::charge(ref mut b.work,cost)){true}else{reject(b,10)};}
fn bool encoded(ref mut Budget b,usize cost){if(b.code!=0){return false;}if(b.encoded_work>encoding_limit() || cost>encoding_limit()-b.encoded_work){return reject(b,10);}b.encoded_work+=cost;return true;}
fn bool update(ref mut sha::State state,ref u8[] value,ref mut Budget b){if(b.code!=0){return false;}if(b.hashed_bytes>hash_limit() || value.len>hash_limit()-b.hashed_bytes){return reject(b,10);}if(!sha::update(state,value)){return reject(b,10);}b.hashed_bytes+=value.len;return true;}
fn bool begin(ref mut sha::State state,ref u8[] domain,usize length,ref mut Budget b){var bytes::Buffer header=bytes::empty();var usize work=0;var u32 code=identity::text(ref mut header,b"fairyc.genericplan.digest.v0",ref mut work);if(code==0){code=identity::text(ref mut header,domain,ref mut work);}if(code==0){code=binary::number(ref mut header,length,ref mut work);}if(code!=0){return reject(b,code);}return charge(b,work) && length<=hash_limit()-bytes::len(ref header) && update(state,bytes::view(ref header),b);}
fn Result<bytes::Buffer,u32> finish(ref sha::State state,ref mut Budget b){if(!charge(b,65)){return .err(b.code);}var bytes::Buffer out=bytes::empty();if(!sha::append_hex(state,ref mut out)){discard reject(b,11);return .err(b.code);}return .ok(out);}
fn Result<bool,u32> less(ref u8[] a,ref u8[] z,ref mut Budget b){var usize i=0;while(i<a.len && i<z.len){if(!charge(b,1)){return .err(b.code);}if(a[i]!=z[i]){return .ok(a[i]<z[i]);}i+=1;}return .ok(a.len<z.len);}
fn Result<bool,u32> before(ref bindings::Bindings table,usize a,usize z,bool imports,ref mut Budget b){if(!charge(b,1)){return .err(b.code);}if(imports){return less(bindings::text(table,bindings::import_at(table,a).alias),bindings::text(table,bindings::import_at(table,z).alias),b);}let left=bindings::module_at(table,a);let right=bindings::module_at(table,z);if(left.owner!=right.owner){return less(bindings::text(table,left.namespace),bindings::text(table,right.namespace),b);}return less(bindings::text(table,left.path),bindings::text(table,right.path),b);}
fn Result<v::Vec<usize>,u32> order(ref bindings::Bindings table,usize first,usize count,bool imports,ref mut Budget b){var v::Vec<usize> rows=v::empty<usize>();var usize i=0;while(i<count){if(!charge(b,1)){return .err(b.code);}let id=first+i;if(!v::push<usize>(ref mut rows,id).is_ok){discard reject(b,11);return .err(b.code);}var usize at=i;while(at>0){let previous=*v::at<usize>(ref rows,at-1);let earlier=switch(before(table,id,previous,imports,b)){case .ok(value):{value}case .err(code):{return .err(code);}};if(!earlier){break;}let slot=v::at_mut<usize>(ref mut rows,at);*slot=previous;at-=1;}let slot=v::at_mut<usize>(ref mut rows,at);*slot=id;i+=1;}return .ok(rows);}
fn bool matches(bindings::Module m,ref syntax::Parsed p,ref mut Budget b){let raw=syntax::source_view(p);if(!charge(b,raw.len+1)){return false;}if(m.source_size!=raw.len){return reject(b,12);}let actual=hash::digest(raw);var usize i=0;while(i<8){if(m.digest[i]!=actual[i]){return reject(b,12);}i+=1;}return true;}

struct Stream{sha::State state;usize length;usize peak_bytes;usize peak_work;bool hashing;}
fn Stream stream(bool hashing){return Stream{.state=sha::empty(),.length=0,.peak_bytes=0,.peak_work=0,.hashing};}
fn bool piece(ref mut Stream s,ref bytes::Buffer data,json::Ledger ledger,ref mut Budget b){if(!encoded(b,ledger.used)){return false;}if(ledger.code!=0){return reject(b,ledger.code);}let raw=bytes::view(data);if(s.length>hash_limit() || raw.len>hash_limit()-s.length){return reject(b,10);}if(s.hashing && !update(ref mut s.state,raw,b)){return false;}s.length+=raw.len;if(raw.len>s.peak_bytes){s.peak_bytes=raw.len;}if(ledger.used>s.peak_work){s.peak_work=ledger.used;}return true;}
fn bool raw(ref mut Stream s,ref u8[] value,ref mut Budget b){var json::Ledger ledger=json::empty();var bytes::Buffer data=bytes::empty();let ok=json::raw(ref mut data,value,ref mut ledger);return piece(s,ref data,ledger,b) && ok;}
fn bool quoted(ref mut Stream s,ref u8[] value,ref mut Budget b){var json::Ledger ledger=json::empty();var bytes::Buffer data=bytes::empty();let ok=json::quoted(ref mut data,value,ref mut ledger);return piece(s,ref data,ledger,b) && ok;}
fn bool declaration_list(ref syntax::Parsed p,u32 kind,ref mut Stream s,ref mut Budget b){var usize count=0;var usize at=1;while(at<syntax::declaration_count(p)){if(!charge(b,1)){return false;}if(syntax::declaration_at(p,at).kind==kind){count+=1;}at+=1;}if(count==0){return raw(s,b"null",b);}if(!raw(s,b"[",b)){return false;}var usize written=0;at=1;
    while(at<syntax::declaration_count(p)){if(!charge(b,1)){return false;}let d=syntax::declaration_at(p,at);if(d.kind==kind){if(written!=0 && !raw(s,b",",b)){return false;}var json::Ledger ledger=json::empty();var bytes::Buffer data=bytes::empty();let ok=if(kind==3){functions::value(p,at,ref mut data,ref mut ledger)}else if(kind==4 || kind==6 || kind==11){aggregates::value(p,at,ref mut data,ref mut ledger)}else{items::value(p,at,ref mut data,ref mut ledger)};if(!piece(s,ref data,ledger,b) || !ok){return false;}written+=1;}at+=1;}return raw(s,b"]",b);
}
fn bool module_value(ref syntax::Parsed p,ref mut Stream s,ref mut Budget b){var json::Ledger ledger=json::empty();var bytes::Buffer header=bytes::empty();let ok=modules::header(p,ref mut header,ref mut ledger);if(!piece(s,ref header,ledger,b) || !ok){return false;}
    if(!raw(s,b",\"Imports\":",b) || !declaration_list(p,2,s,b) || !raw(s,b",\"Constants\":",b) || !declaration_list(p,7,s,b) || !raw(s,b",\"StaticAssertions\":",b) || !declaration_list(p,8,s,b) || !raw(s,b",\"States\":",b) || !declaration_list(p,10,s,b) || !raw(s,b",\"Structs\":",b) || !declaration_list(p,4,s,b) || !raw(s,b",\"Resources\":",b) || !declaration_list(p,6,s,b) || !raw(s,b",\"Enums\":",b) || !declaration_list(p,11,s,b) || !raw(s,b",\"ForeignFunctions\":",b) || !declaration_list(p,9,s,b) || !raw(s,b",\"Functions\":",b) || !declaration_list(p,3,s,b) || !raw(s,b",\"Bindings\":",b)){return false;}
    ledger=json::empty();var bytes::Buffer binding_values=bytes::empty();let bound=observations::bindings(p,ref mut binding_values,ref mut ledger);if(!piece(s,ref binding_values,ledger,b) || !bound || !raw(s,b",\"Literals\":",b)){return false;}ledger=json::empty();var bytes::Buffer literal_values=bytes::empty();let literals=observations::literals(p,ref mut literal_values,ref mut ledger);return piece(s,ref literal_values,ledger,b) && literals && raw(s,b"}",b);
}
fn bool entry(ref bindings::Bindings table,usize at,ref syntax::Parsed p,ref mut Stream s,ref mut Budget b){let m=bindings::module_at(table,at);let imports=switch(order(table,m.first,m.count,true,b)){case .ok(value):{value}case .err(_):{return false;}};if(!raw(s,b"{\"Owner\":",b) || !quoted(s,bindings::text(table,m.namespace),b) || !raw(s,b",\"Module\":",b) || !module_value(p,s,b) || !raw(s,b",\"Imports\":{",b)){return false;}var usize i=0;
    while(i<m.count){if(i!=0 && !raw(s,b",",b)){return false;}let row=bindings::import_at(table,*v::at<usize>(ref imports,i));let provider=bindings::module_at(table,row.target);if(!quoted(s,bindings::text(table,row.alias),b) || !raw(s,b":",b) || !quoted(s,bindings::text(table,provider.namespace),b)){return false;}i+=1;}return raw(s,b"}}",b);
}
fn bool graph_value(ref v::Vec<syntax::Parsed> sources,ref bindings::Bindings table,ref v::Vec<usize> sorted,ref mut Stream s,ref mut Budget b){if(!raw(s,b"[",b)){return false;}var usize i=0;while(i<v::len<usize>(sorted)){if(i!=0 && !raw(s,b",",b)){return false;}let at=*v::at<usize>(sorted,i);if(!entry(table,at,v::at<syntax::Parsed>(sources,at),s,b)){return false;}i+=1;}return raw(s,b"]",b);}
pub fn Result<Context,u32> compute(ref v::Vec<syntax::Parsed> sources,ref bindings::Bindings table,ref mut Budget b){if(!valid(b)){return .err(b.code);}let count=bindings::count(table);if(count==0 || count>512 || count!=v::len<syntax::Parsed>(sources)){discard reject(b,9);return .err(b.code);}let sorted=switch(order(table,0,count,false,b)){case .ok(value):{value}case .err(code):{return .err(code);}};var v::Vec<Stamp> stamps=v::empty<Stamp>();var usize i=0;
    while(i<count){let m=bindings::module_at(table,i);if(!matches(m,v::at<syntax::Parsed>(sources,i),b)){return .err(b.code);}if(!v::push<Stamp>(ref mut stamps,Stamp{.size=m.source_size,.digest=m.digest}).is_ok){discard reject(b,11);return .err(b.code);}i+=1;}
    var Stream first=stream(false);if(!graph_value(sources,table,ref sorted,ref mut first,b)){return .err(b.code);}var Stream second=stream(true);if(!begin(ref mut second.state,b"source-nominal-closed-graph-v0",first.length,b)){if(b.code==0){discard reject(b,10);}return .err(b.code);}if(!graph_value(sources,table,ref sorted,ref mut second,b)){return .err(b.code);}if(first.length!=second.length){discard reject(b,9);return .err(b.code);}let digest=switch(finish(ref second.state,b)){case .ok(value):{value}case .err(code):{return .err(code);}};return .ok(Context{.digest,.sources=stamps,.json_bytes=first.length,.peak_chunk_bytes=first.peak_bytes,.peak_chunk_work=first.peak_work});
}
// Reuse the closed context after its workspace/binding table has been destroyed.
pub fn Result<bytes::Buffer,u32> definition(ref Context context,usize at,ref syntax::Parsed source,usize declaration,ref mut Budget b){if(!valid(b)){return .err(b.code);}if(at>=count(context) || declaration>=syntax::declaration_count(source)){discard reject(b,9);return .err(b.code);}let stamp=*v::at<Stamp>(ref context.sources,at);let source_bytes=syntax::source_view(source);if(!charge(b,source_bytes.len+1)){return .err(b.code);}let actual=hash::digest(source_bytes);if(source_bytes.len!=stamp.size){discard reject(b,12);return .err(b.code);}var usize i=0;while(i<8){if(stamp.digest[i]!=actual[i]){discard reject(b,12);return .err(b.code);}i+=1;}let d=syntax::declaration_at(source,declaration);if((d.kind!=4 && d.kind!=6) || d.generic_parameters!=0 || d.name_start>=d.name_end || d.name_end>source_bytes.len){discard reject(b,9);return .err(b.code);}var Stream first=stream(false);if(!module_value(source,ref mut first,b)){return .err(b.code);}let name=source_bytes[d.name_start..d.name_end];let length=digest(context).len+first.length+name.len+2;var Stream second=stream(true);
    if(!begin(ref mut second.state,b"source-nominal-package-v0",length,b) || !raw(ref mut second,digest(context),b) || !raw(ref mut second,b"\n",b) || !module_value(source,ref mut second,b) || !raw(ref mut second,b"\n",b) || !raw(ref mut second,name,b)){if(b.code==0){discard reject(b,10);}return .err(b.code);}if(second.length!=length){discard reject(b,9);return .err(b.code);}return finish(ref second.state,b);
}
