module sdk_scopes;
import sdk_authority as sdk;
import candidate_limits as limits;
import syntax as syntax;
import pointer_types as pointers;
import type_state as scalar;
import std::bytes as bytes;
import std::vec as v;

// Exact source units, owned independently of Parsed. These records grant no
// authority until the consumer rehashes the retained bytes against bundled pins.
pub struct Scope {usize sdk_id;usize start;usize end;}
pub fn ref u8[] name(usize id){
    let ref u8[] pool=b"std::alloc\0std::bytes\0std::fs\0std::io\0std::num\0std::process\0std::text\0std::vec\0";
    if(id==0 || id>8){return pool[..0];}var usize start=0;var usize at=1;
    while(at<id){while(pool[start]!=0){start+=1;}start+=1;at+=1;}
    var usize end=start;while(pool[end]!=0){end+=1;}return pool[start..end];
}
pub fn bool authentic(ref u8[] source,Scope s){return s.sdk_id>=1 && s.sdk_id<=8 && s.start<=s.end && s.end<=source.len && s.end-s.start<=limits::source_bytes() && sdk::authentic(name(s.sdk_id),source[s.start..s.end]);}
fn Scope scope(ref syntax::Parsed p,syntax::Declaration d){let bool single=syntax::module_count(p)==1;return Scope{.sdk_id=sdk::module_id(syntax::source_view(p)[d.name_start..d.name_end]),.start=if(single){0}else{d.start},.end=if(single){syntax::source_view(p).len}else{d.end}};}
pub fn Result<v::Vec<Scope>,u32> collect(ref syntax::Parsed p){
    var v::Vec<Scope> result=v::empty<Scope>();var usize i=0;
    while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==1){let s=scope(p,d);
        if(s.sdk_id!=0){if(!authentic(syntax::source_view(p),s)){return .err(6801);}if(v::len<Scope>(ref result)>=8 || !v::push<Scope>(ref mut result,s).is_ok){return .err(2);}}
    }i+=1;}return .ok(result);
}
pub fn bool valid(ref u8[] source,ref v::Vec<Scope> scopes){
    if(v::len<Scope>(scopes)>8){return false;}var usize i=0;
    while(i<v::len<Scope>(scopes)){let s=*v::at<Scope>(scopes,i);if(!authentic(source,s)){return false;}
        var usize j=0;while(j<i){let old=*v::at<Scope>(scopes,j);if(old.sdk_id==s.sdk_id || !(old.end<=s.start || s.end<=old.start)){return false;}j+=1;}i+=1;
    }return true;
}
fn usize first(ref u8[] source,ref u8[] needle){var usize i=0;while(i<=source.len && needle.len<=source.len-i){if(bytes::equal(source[i..i+needle.len],needle)){return i+1;}i+=1;}return 0;}
fn usize rule(ref u8[] symbol){return if(bytes::equal(symbol,b"fy_std_process_arg_count_v0")){1}else if(bytes::equal(symbol,b"fy_std_process_arg_v0")){2}else if(bytes::equal(symbol,b"fy_private_vector_initialize_v0")){3}else if(bytes::equal(symbol,b"fy_private_vector_release_empty_v0")){4}else if(bytes::equal(symbol,b"fy_private_vector_reserve_v0")){5}else if(bytes::equal(symbol,b"fy_private_vector_prepare_push_v0")){6}else if(bytes::equal(symbol,b"fy_private_vector_commit_push_v0")){7}else if(bytes::equal(symbol,b"fy_private_vector_pop_slot_v0")){8}else{0};}
fn ref u8[] item(ref u8[] pool,usize index){if(index==0 || index>8){return pool[..0];}var usize start=0;var usize i=1;while(i<index){while(pool[start]!=0){start+=1;}start+=1;i+=1;}var usize end=start;while(pool[end]!=0){end+=1;}return pool[start..end];}
fn ref u8[] function_name(usize code){return item(b"process_arg_count_raw\0process_arg_raw\0initialize_raw\0release_raw\0reserve_raw\0prepare_raw\0commit_raw\0pop_slot_raw\0",code);}
fn ref u8[] declaration_prefix(usize code){return item(b"unsafe extern C fn usize process_arg_count_raw\0unsafe extern C fn raw_process_args_result process_arg_raw\0unsafe extern C fn u32 initialize_raw\0unsafe extern C fn u32 release_raw\0unsafe extern C fn u32 reserve_raw\0unsafe extern C fn u32 prepare_raw\0unsafe extern C fn u32 commit_raw\0unsafe extern C fn ptr mut void pop_slot_raw\0",code);}
pub fn usize foreign_rule(ref u8[] source,Scope s,usize start,usize end,usize link_start,usize link_end){
    if((s.sdk_id!=6 && s.sdk_id!=8) || !authentic(source,s) || start<s.start || end> s.end || start>=end || link_start<s.start || link_end>s.end || link_start>=link_end){return 0;}
    let usize code=rule(source[link_start..link_end]);if(code==0){return 0;}let raw=source[s.start..s.end];
    if(s.sdk_id!=if(code<=2){6}else{8}){return 0;}
    // Vec cleanup calls pop/release before their declarations. A matching call
    // spelling cannot stand in for the exact authenticated extern declaration.
    let prefix=declaration_prefix(code);let usize declaration=first(raw,prefix);let usize link=first(raw,source[link_start..link_end]);
    return if(declaration!=0 && link!=0 && start==s.start+declaration-1+prefix.len-function_name(code).len && end-start==function_name(code).len && link_start==s.start+link-1){code}else{0};
}
pub fn usize foreign_owned(ref u8[] source,ref v::Vec<Scope> scopes,usize start,usize end,usize link_start,usize link_end){
    var usize i=0;while(i<v::len<Scope>(scopes)){let usize code=foreign_rule(source,*v::at<Scope>(scopes,i),start,end,link_start,link_end);if(code!=0){return code;}i+=1;}return 0;
}
pub fn bool result_owned(ref u8[] source,ref v::Vec<Scope> scopes,usize start,usize end){
    var usize i=0;while(i<v::len<Scope>(scopes)){let s=*v::at<Scope>(scopes,i);if(s.sdk_id==6 && authentic(source,s)){let usize at=first(source[s.start..s.end],b"raw_process_args_result");if(at!=0 && start==s.start+at-1 && end-start==b"raw_process_args_result".len){return true;}}i+=1;}return false;
}
pub fn usize parameter_count(usize code){return if(code==2 || code==3 || code==6 || code==7){3}else if(code==4){1}else if(code==5 || code==8){2}else{0};}
pub fn u32 parameter(usize code,usize index){
    if(index>=parameter_count(code)){return 0;}
    if(code==2){return if(index==1){pointers::make(3,true)}else{5};}
    if(code>=3 && code<=8){return if(index==0 || code==6 && index==2 || code==7 && index==1){pointers::make(7,true)}else{5};}
    return 0;
}
pub fn u32 scalar_result(usize code){return if(code==1){5}else if(code>=3 && code<=7){4}else if(code==8){pointers::make(7,true)}else{0};}
pub fn bool parsed_foreign(ref syntax::Parsed p,usize index,usize link_start,usize link_end){
    let d=syntax::declaration_at(p,index);var Scope owner=Scope{.sdk_id=0,.start=0,.end=0};var usize i=0;
    while(i<syntax::declaration_count(p)){let unit=syntax::declaration_at(p,i);if(unit.kind==1 && unit.start<=d.start){owner=scope(p,unit);}i+=1;}
    let usize code=foreign_rule(syntax::source_view(p),owner,d.name_start,d.name_end,link_start,link_end);
    if(code==0){return false;}
    let n=syntax::type_at(p,d.result_type);
    if(code==2){if(n.kind!=1 || !bytes::equal(syntax::type_text(p,d.result_type),b"raw_process_args_result")){return false;}}
    else{let u32 type=if(n.kind==1){scalar::primitive(p,d.result_type)}else{pointers::resolve(p,d.result_type)};if(type!=scalar_result(code)){return false;}}
    var usize count=0;i=d.entry_start;while(i<d.entry_start+d.entry_count){let e=syntax::entry_at(p,i);if(e.kind==1){let u32 type=if(syntax::type_at(p,e.type_root).kind==1){scalar::primitive(p,e.type_root)}else{pointers::resolve(p,e.type_root)};if(type!=parameter(code,count)){return false;}count+=1;}i+=1;}
    return count==parameter_count(code);
}
fn ref u8[] storage_name(u32 code){let ref u8[] pool=b"storage_size\0storage_alignment\0storage_take\0storage_put\0storage_element_address\0storage_borrow\0storage_borrow_mut\0";if(code==0 || code>7){return pool[..0];}var usize start=0;var u32 i=1;while(i<code){while(pool[start]!=0){start+=1;}start+=1;i+=1;}var usize end=start;while(pool[end]!=0){end+=1;}return pool[start..end];}
fn bool storage_span(ref u8[] source,Scope s,usize start,usize end,u32 code){
    let name=storage_name(code);return name.len!=0 && s.sdk_id==8 && authentic(source,s) && start>=s.start && start<end && end<=s.end && name.len<end-start && bytes::equal(source[start..start+name.len],name) && source[start+name.len]==60;
}
pub fn bool storage_owned(ref u8[] source,ref v::Vec<Scope> scopes,usize start,usize end,u32 code){var usize i=0;while(i<v::len<Scope>(scopes)){if(storage_span(source,*v::at<Scope>(scopes,i),start,end,code)){return true;}i+=1;}return false;}
pub fn bool parsed_storage(ref syntax::Parsed p,usize start,usize end,u32 code){var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if(d.kind==1 && storage_span(syntax::source_view(p),scope(p,d),start,end,code)){return true;}i+=1;}return false;}
