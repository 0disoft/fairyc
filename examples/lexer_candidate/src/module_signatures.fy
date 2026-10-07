module module_signatures;
import module_catalog as catalog;
import module_types as types;
import syntax as syntax;
import effect_declarations as effects;
import foreign_declarations as foreign;
import std::bytes as bytes;
import std::vec as v;

// Owned source-set signatures; table indices are never the single-file scalar
// type IDs used by the older module checker. That checker remains closed until
// body typing and nominal layout ownership can consume this exact graph.
pub struct Signature {usize module_index;usize declaration;bool public;usize result;usize parameter_start;usize parameter_count;u32 effects;bool must_use;bool unsafe_call;bool foreign;usize link_start;usize link_end;}
pub resource Program {catalog::Catalog declarations;types::Graph graph;v::Vec<Signature> signatures;v::Vec<usize> parameters;bytes::Buffer links;cleanup{}}
pub fn usize count(ref Program p){return v::len<Signature>(ref p.signatures);}
pub fn Signature at(ref Program p,usize id){assert(id>0 && id<=count(p));return *v::at<Signature>(ref p.signatures,id-1);}
pub fn usize parameter_at(ref Program p,usize index){return *v::at<usize>(ref p.parameters,index);}
pub fn types::Type type_at(ref Program p,usize id){return types::graph_at(ref p.graph,id);}
pub fn ref u8[] linker_name(ref Program p,usize id){let s=at(p,id);return bytes::view(ref p.links)[s.link_start..s.link_end];}
pub fn catalog::Lookup lookup(ref Program p,usize module_index,ref u8[] spelling){let found=catalog::lookup(ref p.declarations,module_index,spelling);if(found.code!=0){return found;}let d=catalog::declaration_at(ref p.declarations,found.item);var usize i=1;while(i<=count(p)){let s=at(p,i);if(s.module_index==d.module_index && s.declaration==d.declaration){return catalog::Lookup{.code=0,.item=i};}i+=1;}return catalog::Lookup{.code=9,.item=0};}
fn catalog::Problem fail(u32 code,usize module_index,usize start,usize end){return catalog::Problem{.code,.module_index,.start,.end};}
pub fn Result<Program,catalog::Problem> build(ref v::Vec<syntax::Parsed> sources){
    let c=switch(catalog::build(sources)){case .ok(c):{c}case .err(p):{return .err(p);}};
    var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<Signature> signatures=v::empty<Signature>();var v::Vec<usize> parameters=v::empty<usize>();var bytes::Buffer links=bytes::empty();
    var usize module_index=0;while(module_index<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,module_index);var usize index=1;
        while(index<syntax::declaration_count(p)){let d=syntax::declaration_at(p,index);if(d.kind==3 || d.kind==9){
            if(d.owner_end!=0 || d.generic_parameters!=0){return .err(fail(4501,module_index,d.start,d.end));}
            let contract=effects::read(p,index);if(contract.code!=0){return .err(fail(contract.code,module_index,contract.start,contract.end));}
            var usize link_start=0;var usize link_end=0;
            if(d.kind==9){let ffi=foreign::read(p,index);if(ffi.code!=0){return .err(fail(ffi.code,module_index,ffi.start,ffi.end));}link_start=bytes::len(ref links);let name=syntax::source_view(p)[ffi.link_start..ffi.link_end];if(name.len>1048576-bytes::len(ref links) || !bytes::append(ref mut links,name).is_ok){return .err(fail(11,module_index,d.start,d.end));}link_end=bytes::len(ref links);}
            let result=types::resolve(ref c,p,module_index,d.result_type,ref mut graph);if(result.problem.code!=0){return .err(fail(result.problem.code,module_index,result.problem.start,result.problem.end));}
            let result_type=types::at(ref graph,result.type);if(contract.must_use && (result_type.kind==1 && result_type.scalar==7 || result_type.kind!=1 && result_type.kind!=2 && result_type.kind!=4 && result_type.kind!=12)){return .err(fail(4201,module_index,contract.must_start,contract.must_end));}
            let usize parameter_start=v::len<usize>(ref parameters);var usize e=d.entry_start;
            while(e<d.entry_start+d.entry_count){let entry=syntax::entry_at(p,e);if(entry.kind==1){let type=types::resolve(ref c,p,module_index,entry.type_root,ref mut graph);if(type.problem.code!=0){return .err(fail(type.problem.code,module_index,type.problem.start,type.problem.end));}let t=types::at(ref graph,type.type);if(t.kind==1 && t.scalar==7){return .err(fail(9,module_index,entry.type_start,entry.type_end));}if(v::len<usize>(ref parameters)>=32768 || !v::push<usize>(ref mut parameters,type.type).is_ok){return .err(fail(10,module_index,entry.start,entry.end));}}e+=1;}
            if(v::len<usize>(ref parameters)-parameter_start>128 || v::len<Signature>(ref signatures)>=4096){return .err(fail(10,module_index,d.start,d.end));}
            let usize parameter_count=v::len<usize>(ref parameters)-parameter_start;
            if(d.kind==9){var usize previous=0;while(previous<v::len<Signature>(ref signatures)){let old=*v::at<Signature>(ref signatures,previous);if(old.foreign && bytes::equal(bytes::view(ref links)[old.link_start..old.link_end],bytes::view(ref links)[link_start..link_end])){if(old.result!=result.type || old.parameter_count!=parameter_count){return .err(fail(5106,module_index,d.start,d.end));}var usize n=0;while(n<parameter_count){if(*v::at<usize>(ref parameters,old.parameter_start+n)!=*v::at<usize>(ref parameters,parameter_start+n)){return .err(fail(5106,module_index,d.start,d.end));}n+=1;}}previous+=1;}}
            if(!v::push<Signature>(ref mut signatures,Signature{.module_index,.declaration=index+1,.public=d.public,.result=result.type,.parameter_start,.parameter_count,.effects=contract.bits,.must_use=contract.must_use,.unsafe_call=d.unsafe_end!=0,.foreign=d.kind==9,.link_start,.link_end}).is_ok){return .err(fail(11,module_index,d.start,d.end));}
        }index+=1;}module_index+=1;
    }
    let owned=switch(types::assemble(ref c,graph)){case .ok(g):{g}case .err(code):{return .err(fail(code,0,0,0));}};
    return .ok(Program{.declarations=c,.graph=owned,.signatures,.parameters,.links});
}
