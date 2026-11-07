module static_operation_group_identity;
import static_operation_declarations as operations;
import generic_source_function_json as functions;
import generic_source_json as json;
import generic_identity as identity;
import project_manifest_values as lexical;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Admission owns group size and tuple completeness. Keep original written ASTs,
// including annotations and spans, in operation-name then scalar-name order.
fn u32 members(ref syntax::Parsed p,usize root,ref u8[] label,ref mut v::Vec<usize> rows){
    let owner=syntax::module_index(p,syntax::declaration_at(p,root-1).start);var usize i=1;while(i<syntax::declaration_count(p)){let declaration=i+1;if(syntax::module_index(p,syntax::declaration_at(p,i).start)==owner && bytes::equal(operations::group(p,declaration),label)){
        if(v::len<usize>(rows)>=16 || !v::push<usize>(rows,declaration).is_ok){return 11;}var usize slot=v::len<usize>(rows)-1;
        while(slot>0){let previous=*v::at<usize>(rows,slot-1);if(!lexical::before(syntax::declaration_name(p,declaration),syntax::declaration_name(p,previous))){break;}{let changed=v::at_mut<usize>(rows,slot);*changed=previous;}slot-=1;}{let changed=v::at_mut<usize>(rows,slot);*changed=declaration;}
    }i+=1;}return 0;
}
fn u32 providers(ref syntax::Parsed p,usize declaration,ref mut bytes::Buffer raw,ref mut json::Ledger ledger){
    var v::Vec<usize> rows=v::empty<usize>();var usize i=1;while(i<syntax::declaration_count(p)){let info=operations::provider(p,i+1);if(info.code!=0){return info.code;}if(info.kind==2 && info.operation==declaration){
        if(v::len<usize>(ref rows)>=13 || !v::push<usize>(ref mut rows,i+1).is_ok){return 11;}var usize slot=v::len<usize>(ref rows)-1;
        while(slot>0){let previous=*v::at<usize>(ref rows,slot-1);if(!lexical::before(syntax::type_text(p,syntax::declaration_at(p,i).result_type),syntax::type_text(p,syntax::declaration_at(p,previous-1).result_type))){break;}{let changed=v::at_mut<usize>(ref mut rows,slot);*changed=previous;}slot-=1;}{let changed=v::at_mut<usize>(ref mut rows,slot);*changed=i+1;}
    }i+=1;}i=0;while(i<v::len<usize>(ref rows)){if(!json::raw(raw,b",",ledger) || !functions::value(p,*v::at<usize>(ref rows,i)-1,raw,ledger)){return ledger.code;}i+=1;}return 0;
}
pub fn Result<bytes::Buffer,u32> definition(ref syntax::Parsed p,usize declaration,ref mut usize work){
    var v::Vec<usize> rows=v::empty<usize>();let code=members(p,declaration,operations::group(p,declaration),ref mut rows);if(code!=0){return .err(code);}
    var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer raw=bytes::empty();if(!json::raw(ref mut raw,b"[",ref mut ledger)){return .err(ledger.code);}var usize i=0;
    while(i<v::len<usize>(ref rows)){if(i!=0 && !json::raw(ref mut raw,b",",ref mut ledger)){return .err(ledger.code);}let member=*v::at<usize>(ref rows,i);if(!functions::value(p,member-1,ref mut raw,ref mut ledger)){return .err(ledger.code);}let code=providers(p,member,ref mut raw,ref mut ledger);if(code!=0){return .err(code);}i+=1;}
    if(!json::raw(ref mut raw,b"]",ref mut ledger)){return .err(ledger.code);};*work=ledger.used;return identity::digest_text(b"source-static-operation-group-v1",bytes::view(ref raw),work);
}
