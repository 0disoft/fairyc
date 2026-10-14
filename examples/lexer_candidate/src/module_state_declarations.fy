module module_state_declarations;
import module_state_layout as state;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import literal_types as literals;
import std::bytes as bytes;
import std::vec as v;

fn catalog::Problem fail(u32 code,usize module_index,usize start,usize end){return catalog::Problem{.code,.module_index,.start,.end};}
fn u32 codec(ref syntax::Parsed p,usize id){if(id==0 || id>syntax::type_count(p) || syntax::type_at(p,id).kind!=1){return 0;}let text=syntax::type_text(p,id);return if(bytes::equal(text,b"bool")){1}else if(bytes::equal(text,b"u32")){2}else if(bytes::equal(text,b"u64")){3}else if(bytes::equal(text,b"bytes20")){4}else if(bytes::equal(text,b"bytes32")){5}else{0};}
fn catalog::Problem declaration(ref syntax::Parsed p,usize module_index,usize index,usize owner,ref mut bytes::Buffer names,ref mut v::Vec<state::Layout> layouts,ref mut v::Vec<state::Field> fields,ref mut usize work){
    let d=syntax::declaration_at(p,index);if(d.public || d.owner_end!=0 || d.attribute_count==0){return fail(7201,module_index,d.start,d.end);}var usize uid_start=0;var usize uid_end=0;var usize a=0;
    while(a<d.attribute_count){let attribute=syntax::attribute_at(p,d.attribute_start+a);if(a!=0 || attribute.argument_count!=1 || !bytes::equal(syntax::source_view(p)[attribute.name_start..attribute.name_end],b"contract_uid")){return fail(7201,module_index,attribute.start,attribute.end);}let argument=syntax::attribute_argument_at(p,attribute.argument_start);let raw=syntax::attribute_argument_text(p,attribute.argument_start);if(argument.kind!=3 || raw.len<2 || raw[0]!=34){return fail(7201,module_index,attribute.start,attribute.end);}var bytes::Buffer decoded=bytes::empty();let decode=literals::decode(ref mut decoded,scanner::token_cooked(syntax::source_stream(p),argument.token));if(decode!=0){return fail(if(decode==3){11}else{7201},module_index,attribute.start,attribute.end);}let uid=bytes::view(ref decoded);if(!state::valid_name(uid,true)){return fail(7201,module_index,if(uid.len==0){attribute.start}else{d.start},if(uid.len==0){attribute.end}else{d.end});}uid_start=bytes::len(names);if(!state::append(names,uid,work)){return fail(if(*work>=1048576){10}else{11},module_index,d.start,d.end);}uid_end=bytes::len(names);a+=1;}
    let layout=v::len<state::Layout>(layouts);let first=v::len<state::Field>(fields);if(!v::push<state::Layout>(layouts,state::Layout{.owner,.module_index,.declaration=index+1,.start=d.start,.end=d.end,.uid_start,.uid_end,.field_start=first,.field_count=d.field_count,.hash=[0;8]}).is_ok){return fail(11,module_index,d.start,d.end);}var usize i=0;
    while(i<d.field_count){let field=syntax::field_at(p,d.field_start+i);let name=syntax::source_view(p)[field.name_start..field.name_end];if(!state::charge(work,1)){return fail(10,module_index,field.start,field.end);}var usize j=0;while(j<i){if(!state::charge(work,1)){return fail(10,module_index,field.start,field.end);}if(bytes::equal(name,bytes::view(names)[v::at<state::Field>(fields,first+j).name_start..v::at<state::Field>(fields,first+j).name_end])){return fail(7202,module_index,field.name_start,field.name_end);}j+=1;}
        let value=codec(p,field.type_root);if(value==0){let t=syntax::type_at(p,field.type_root);return fail(7203,module_index,t.start,t.end);}let u32 key=if(field.kind==2){codec(p,field.key_type)}else{0};if(field.kind==2 && key==0){if(field.key_type==0){return fail(7203,module_index,field.start,field.end);}let t=syntax::type_at(p,field.key_type);return fail(7203,module_index,t.start,t.end);}if(field.kind!=1 && field.kind!=2 || !state::valid_name(name,false)){return fail(7203,module_index,d.start,d.end);}let name_start=bytes::len(names);if(!state::append(names,name,work) || !v::push<state::Field>(fields,state::Field{.kind=field.kind,.key,.value,.name_start,.name_end=bytes::len(names),.start=field.start,.end=field.end,.prefix=[0;8]}).is_ok){return fail(if(*work>=1048576){10}else{11},module_index,field.start,field.end);}
        let prefix=switch(state::field_prefix(bytes::view(names)[uid_start..uid_end],name,field.kind,key,value,i,work)){case .ok(hash):{hash}case .err(code):{return fail(code,module_index,field.start,field.end);}};let slot=v::at_mut<state::Field>(fields,first+i);slot.prefix=prefix;i+=1;
    }let hash=switch(state::layout_digest(bytes::view(names)[uid_start..uid_end],names,fields,first,d.field_count,work)){case .ok(hash):{hash}case .err(code):{return fail(code,module_index,d.start,d.end);}};let slot=v::at_mut<state::Layout>(layouts,layout);slot.hash=hash;return fail(0,0,0,0);
}
struct Owner{usize module_index;usize declaration;usize package;}
pub fn Result<state::Plan,catalog::Problem> build(ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog c){
    var usize work=0;return build_metered(sources,c,ref mut work);
}
// Preserve the independent 1MiB declaration history while allowing an artifact
// consumer to account for its actual work in the shared project ledger.
pub fn Result<state::Plan,catalog::Problem> build_metered(ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog c,ref mut usize work){
    if(!state::charge(work,0)){return .err(fail(10,0,0,0));}var v::Vec<Owner> owners=v::empty<Owner>();var usize module_index=0;
    while(module_index<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,module_index);if(!catalog::source_matches(c,module_index,p)){return .err(fail(9,module_index,0,0));}var usize i=0;while(i<syntax::declaration_count(p)){if(!state::charge(work,1)){return .err(fail(10,module_index,0,0));}let d=syntax::declaration_at(p,i);if(d.kind==10){let package=catalog::module_owner(c,module_index);var usize j=0;while(j<v::len<Owner>(ref owners)){if(!state::charge(work,1)){return .err(fail(10,module_index,d.start,d.end));}if(v::at<Owner>(ref owners,j).package==package){return .err(fail(7201,module_index,d.start,d.end));}j+=1;}if(!v::push<Owner>(ref mut owners,Owner{.module_index,.declaration=i,.package}).is_ok){return .err(fail(11,module_index,d.start,d.end));}}i+=1;}module_index+=1;}
    var bytes::Buffer names=bytes::empty();var v::Vec<state::Layout> layouts=v::empty<state::Layout>();var v::Vec<state::Field> fields=v::empty<state::Field>();var usize i=0;
    while(i<v::len<Owner>(ref owners)){let owner=*v::at<Owner>(ref owners,i);let problem=declaration(v::at<syntax::Parsed>(sources,owner.module_index),owner.module_index,owner.declaration,owner.package,ref mut names,ref mut layouts,ref mut fields,work);if(problem.code!=0){return .err(problem);}i+=1;}
    return .ok(state::assemble(names,layouts,fields));
}
