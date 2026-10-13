module module_source_artifact_rebind;
import module_source_artifact_model as model;
import generic_package_bindings as bindings;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import project_cache_encoding as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Recompute source hashes, coordinates and import providers from current input.
// Module/owner numbering may change. This is provenance, not Flow admission.
// Existing binding errors: 9 plumbing, 10 work, 11 allocation, 12 stale source.
pub fn Result<bindings::Bindings,u32> rebind(ref model::Snapshot saved,ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref v::Vec<catalog::Binding> imports,ref coordinates::Coordinates table,ref mut usize work){
    let shape=model::check(saved,work);if(shape!=0){return .err(if(shape==1002){10}else if(shape==1003){11}else{9});}
    let current=switch(bindings::build(sources,owners,imports,table,work)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    if(model::count(saved)!=bindings::count(ref current) || model::import_count(saved)!=bindings::import_count(ref current)){return .err(9);}var usize[512] mapped=[0;512];var bool[512] seen=[false;512];var usize i=0;
    while(i<model::count(saved)){let old=model::module_at(saved,i);let ns=model::text(saved,old.namespace);let path=model::text(saved,old.path);var usize found=bindings::count(ref current);var usize j=0;
        while(j<bindings::count(ref current)){if(!budget::charge(work,1)){return .err(10);}let now=bindings::module_at(ref current,j);if(bytes::equal(ns,bindings::text(ref current,now.namespace)) && bytes::equal(path,bindings::text(ref current,now.path))){if(found!=bindings::count(ref current)){return .err(9);}found=j;}j+=1;}
        if(found==bindings::count(ref current) || seen[found]){return .err(9);}seen[found]=true;mapped[i]=found+1;let now=bindings::module_at(ref current,found);if(!budget::charge(work,9)){return .err(10);}if(old.source_size!=now.source_size){return .err(12);}j=0;while(j<8){if(old.digest[j]!=now.digest[j]){return .err(12);}j+=1;}if(old.count!=now.count){return .err(9);}i+=1;
    }
    i=0;while(i<model::count(saved)){let old=model::module_at(saved,i);let now=bindings::module_at(ref current,mapped[i]-1);var usize j=0;while(j<old.count){if(!budget::charge(work,1)){return .err(10);}let a=model::import_at(saved,old.first+j);let b=bindings::import_at(ref current,now.first+j);if(mapped[a.target]!=b.target+1 || a.declaration!=b.declaration || !bytes::equal(model::text(saved,a.alias),bindings::text(ref current,b.alias)) || !bytes::equal(model::text(saved,a.binding),bindings::text(ref current,b.binding))){return .err(9);}j+=1;}i+=1;}return .ok(current);
}
