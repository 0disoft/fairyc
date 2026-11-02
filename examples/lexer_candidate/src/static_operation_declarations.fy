module static_operation_declarations;
import syntax as syntax;
import scanner as scanner;
import generic_transport_scalars as transport;
import std::bytes as bytes;

// Declaration IDs are one-based. No provider body bypasses ordinary checking.
pub struct Info{u32 kind;u32 code;usize operation;u32 scalar;usize parameters;}
fn Info empty(){return Info{.kind=0,.code=0,.operation=0,.scalar=0,.parameters=0};}
fn Info invalid(Info info){var out=info;out.code=4501;return out;}
fn bool direct(ref syntax::Parsed p,usize root,ref u8[] name){return root!=0 && syntax::type_at(p,root).kind==1 && bytes::equal(syntax::type_text(p,root),name);}
pub fn u32 scalar(ref u8[] name){if(bytes::equal(name,b"i32")){return 1;}if(bytes::equal(name,b"i64")){return 2;}if(bytes::equal(name,b"u8")){return 3;}if(bytes::equal(name,b"u32")){return 4;}if(bytes::equal(name,b"usize")){return 5;}if(bytes::equal(name,b"bool")){return 6;}return transport::named(name);}
fn usize marker(ref syntax::Parsed p,usize declaration,ref u8[] name){let d=syntax::declaration_at(p,declaration-1);var usize i=0;var usize found=0;while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],name)){if(found!=0){return 4097;}found=d.attribute_start+i+1;}i+=1;}return found;}
pub fn Info operation(ref syntax::Parsed p,usize declaration){
    var out=empty();if(declaration==0 || declaration>syntax::declaration_count(p)){return invalid(out);}let index=marker(p,declaration,b"interface");if(index==0){return out;}out.kind=1;out.operation=declaration;if(index==4097){return invalid(out);}let d=syntax::declaration_at(p,declaration-1);let a=syntax::attribute_at(p,index-1);
    if(d.kind!=3 || d.attribute_count!=1 || a.argument_count!=0 || d.unsafe_end!=0 || d.owner_end!=0 || syntax::generic_parameter_count(p,declaration-1)!=1 || d.body==0){return invalid(out);}
    let parameter=syntax::generic_parameter_at(p,declaration-1,0);let name=syntax::source_view(p)[parameter.start..parameter.end];if(!direct(p,d.result_type,name)){return invalid(out);}let body=syntax::statement_at(p,d.body);if(body.kind!=1){return invalid(out);}var usize tokens=0;var usize cursor=0;while(cursor<syntax::source_token_count(p)){let t=scanner::token_info(syntax::source_stream(p),cursor);if(t.kind!=2 && t.start>=body.start && t.end<=body.end && t.start<t.end){tokens+=1;}cursor+=1;}if(tokens!=2){return invalid(out);}
    var usize i=d.entry_start;while(i<d.entry_start+d.entry_count){let e=syntax::entry_at(p,i);if(e.kind==1){if(!direct(p,e.type_root,name)){return invalid(out);}out.parameters+=1;}else{return invalid(out);}i+=1;}if(out.parameters>4){return invalid(out);}return out;
}
pub fn Info provider(ref syntax::Parsed p,usize declaration){
    var out=empty();if(declaration==0 || declaration>syntax::declaration_count(p)){return invalid(out);}let index=marker(p,declaration,b"implements");if(index==0){return out;}out.kind=2;if(index==4097){return invalid(out);}let d=syntax::declaration_at(p,declaration-1);let a=syntax::attribute_at(p,index-1);
    if(d.kind!=3 || d.attribute_count!=1 || a.argument_count!=2 || d.generic_parameters!=0 || d.unsafe_end!=0 || d.owner_end!=0 || d.body==0){return invalid(out);}
    let op=syntax::attribute_argument_at(p,a.argument_start);let arg=syntax::attribute_argument_at(p,a.argument_start+1);let raw=syntax::source_view(p);let name=raw[op.start..op.end];let type=raw[arg.start..arg.end];out.scalar=scalar(type);if(out.scalar==0){return invalid(out);}var usize i=1;
    while(i<syntax::declaration_count(p)){let target=syntax::declaration_at(p,i);if(bytes::equal(raw[target.name_start..target.name_end],name)){if(out.operation!=0){return invalid(out);}out.operation=i+1;}i+=1;}if(out.operation==0){return invalid(out);}let target=operation(p,out.operation);if(target.kind!=1 || target.code!=0 || !direct(p,d.result_type,type)){return invalid(out);}
    i=d.entry_start;while(i<d.entry_start+d.entry_count){let e=syntax::entry_at(p,i);if(e.kind==1){if(!direct(p,e.type_root,type)){return invalid(out);}out.parameters+=1;}i+=1;}if(out.parameters!=target.parameters){return invalid(out);}return out;
}
pub fn u32 validate(ref syntax::Parsed p,ref mut usize work){var usize i=1;while(i<syntax::declaration_count(p)){if(*work>=1048576){return 4503;};*work=*work+1;let op=operation(p,i+1);if(op.code!=0){return op.code;}let a=provider(p,i+1);if(a.code!=0){return a.code;}if(a.kind==2){var usize j=1;while(j<i){if(*work>=1048576){return 4503;};*work=*work+1;let b=provider(p,j+1);if(b.kind==2 && b.operation==a.operation && b.scalar==a.scalar){return 4501;}j+=1;}}i+=1;}return 0;}
pub fn Result<usize,u32> select(ref syntax::Parsed p,usize declaration,u32 type,ref mut usize work){let valid=validate(p,work);if(valid!=0){return .err(valid);}let op=operation(p,declaration);if(op.code!=0 || op.kind!=1 || !transport::valid(type)){return .err(4501);}var usize i=1;while(i<syntax::declaration_count(p)){if(*work>=1048576){return .err(4503);};*work=*work+1;let candidate=provider(p,i+1);if(candidate.kind==2 && candidate.operation==declaration && candidate.scalar==type){return .ok(i+1);}i+=1;}return .err(4502);}
