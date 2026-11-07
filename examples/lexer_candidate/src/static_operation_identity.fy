module static_operation_identity;
import static_operation_declarations as operations;
import static_operation_group_identity as groups;
import generic_source_function_json as functions;
import generic_source_json as json;
import generic_identity as identity;
import project_manifest_values as lexical;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Provider{u32 scalar;usize declaration;usize start;usize end;}
pub resource Metadata{bytes::Buffer digest;bytes::Buffer bound;bytes::Buffer names;v::Vec<Provider> providers;cleanup{}}
pub fn usize count(ref Metadata m){return v::len<Provider>(ref m.providers);}
pub fn Provider at(ref Metadata m,usize i){return *v::at<Provider>(ref m.providers,i);}
pub fn ref u8[] definition(ref Metadata m){return bytes::view(ref m.digest);}
pub fn ref u8[] bound_definition(ref Metadata m){return bytes::view(ref m.bound);}
pub fn ref u8[] provider_definition(ref Metadata m,usize i){let row=at(m,i);return bytes::view(ref m.names)[row.start..row.end];}
// Provider metadata belongs only to this operation. Grouped definitions include
// all siblings; ungrouped definitions retain their original AST bytes/domain.
pub fn Result<Metadata,u32> build(ref syntax::Parsed p,usize declaration,ref u8[] base,ref mut usize work){
    var v::Vec<Provider> providers=v::empty<Provider>();var bytes::Buffer names=bytes::empty();var usize i=1;
    while(i<syntax::declaration_count(p)){let info=operations::provider(p,i+1);if(info.code!=0){return .err(info.code);}if(info.kind==2 && info.operation==declaration){
        let raw=switch(functions::encode(p,i,work)){case .ok(b):{b}case .err(code):{return .err(code);}};
        let digest=switch(identity::digest_text(b"source-static-provider-v1",bytes::view(ref raw),work)){case .ok(b):{b}case .err(code):{return .err(code);}};
        let start=bytes::len(ref names);if(!bytes::append(ref mut names,bytes::view(ref digest)).is_ok){return .err(11);}let row=Provider{.scalar=info.scalar,.declaration=i+1,.start,.end=bytes::len(ref names)};if(v::len<Provider>(ref providers)>=13 || !v::push<Provider>(ref mut providers,row).is_ok){return .err(11);}var usize slot=v::len<Provider>(ref providers)-1;
        while(slot>0){let previous=*v::at<Provider>(ref providers,slot-1);if(!lexical::before(syntax::type_text(p,syntax::declaration_at(p,row.declaration-1).result_type),syntax::type_text(p,syntax::declaration_at(p,previous.declaration-1).result_type))){break;}{let changed=v::at_mut<Provider>(ref mut providers,slot);*changed=previous;}slot-=1;}{let changed=v::at_mut<Provider>(ref mut providers,slot);*changed=row;}
    }i+=1;}
    let digest=if(operations::group(p,declaration).len!=0){switch(groups::definition(p,declaration,work)){case .ok(b):{b}case .err(code):{return .err(code);}}}else{
    var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer raw=bytes::empty();if(!json::raw(ref mut raw,b"[",ref mut ledger) || !functions::value(p,declaration-1,ref mut raw,ref mut ledger)){return .err(ledger.code);}i=0;
    while(i<v::len<Provider>(ref providers)){let row=*v::at<Provider>(ref providers,i);if(!json::raw(ref mut raw,b",",ref mut ledger) || !functions::value(p,row.declaration-1,ref mut raw,ref mut ledger)){return .err(ledger.code);}i+=1;}if(!json::raw(ref mut raw,b"]",ref mut ledger)){return .err(ledger.code);};*work=ledger.used;
    switch(identity::digest_text(b"source-static-operation-v1",bytes::view(ref raw),work)){case .ok(b):{b}case .err(code):{return .err(code);}}};var bytes::Buffer joined=bytes::empty();if(!bytes::append(ref mut joined,base).is_ok || !bytes::append(ref mut joined,bytes::view(ref digest)).is_ok){return .err(11);}
    let bound=switch(identity::digest_text(b"source-static-bound-operation-v1",bytes::view(ref joined),work)){case .ok(b):{b}case .err(code):{return .err(code);}};return .ok(Metadata{.digest,.bound,.names,.providers});
}
