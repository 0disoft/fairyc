module module_bind;
import candidate_limits as limits;
import module_signatures as signatures;
import module_catalog as catalog;
import syntax as syntax;
import raw_view_intrinsics as raw;
import scanner as scanner;
import c_abi_types as c_types;
import records as records;
import attributes as attributes;
import types as types;
import bodies as bodies;
import body_state as ast;
import std::bytes as bytes;
import std::vec as v;

pub struct Problem {u32 code;usize module_index;usize start;usize end;}
fn Problem fail(u32 code,usize module_index,usize start,usize end){return Problem{.code,.module_index,.start,.end};}
fn usize declaration(ref v::Vec<syntax::Parsed> sources,catalog::Declaration d){var usize base=0;var usize i=0;while(i<d.module_index){base+=syntax::declaration_count(v::at<syntax::Parsed>(sources,i));i+=1;}return base+d.declaration;}
fn bool builtin(ref u8[] name){return bytes::equal(name,b"i32") || bytes::equal(name,b"i64") || bytes::equal(name,b"u8") || bytes::equal(name,b"u32") || bytes::equal(name,b"usize") || bytes::equal(name,b"bool") || bytes::equal(name,b"void") || bytes::equal(name,b"str") || bytes::equal(name,b"CStr") || bytes::equal(name,b"Option") || bytes::equal(name,b"Result") || c_types::valid(c_types::named(name));}
fn bool qualified(ref u8[] name){var usize i=0;while(i<name.len){if(name[i]==58){return true;}i+=1;}return false;}
pub fn Result<syntax::Parsed,Problem> bind(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,syntax::Parsed p){
    let usize count=v::len<syntax::Parsed>(sources);
    if(syntax::module_count(ref p)!=count || signatures::module_count(metadata)!=count){return .err(fail(12,0,0,syntax::source_view(ref p).len));}
    var bytes::Buffer expected=bytes::empty();var usize module_index=0;
    while(module_index<count){let file=v::at<syntax::Parsed>(sources,module_index);
        if(!signatures::source_matches(metadata,module_index,file)){return .err(fail(12,module_index,0,0));}
        if(!bytes::append(ref mut expected,syntax::source_view(file)).is_ok || module_index+1<count && !bytes::append(ref mut expected,b"\n").is_ok){return .err(fail(11,module_index,0,0));}
        module_index+=1;
    }
    if(!bytes::equal(bytes::view(ref expected),syntax::source_view(ref p))){return .err(fail(12,0,0,syntax::source_view(ref p).len));}
    var v::Vec<syntax::Declaration> declarations=v::empty<syntax::Declaration>();var v::Vec<ast::Node> expressions=v::empty<ast::Node>();var v::Vec<ast::Statement> statements=v::empty<ast::Statement>();var v::Vec<bodies::Entry> entries=v::empty<bodies::Entry>();var v::Vec<types::Node> types=v::empty<types::Node>();var v::Vec<syntax::Problem> problems=v::empty<syntax::Problem>();var v::Vec<records::Field> fields=v::empty<records::Field>();var v::Vec<attributes::Attribute> attributes=v::empty<attributes::Attribute>();var v::Vec<attributes::Argument> arguments=v::empty<attributes::Argument>();
    var usize i=0;while(i<syntax::declaration_count(ref p)){if(!v::push<syntax::Declaration>(ref mut declarations,syntax::declaration_at(ref p,i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    i=1;while(i<=syntax::type_count(ref p)){var n=syntax::type_at(ref p,i);if(n.kind==1 && !builtin(syntax::type_text(ref p,i))){let usize owner=syntax::module_index(ref p,n.start);let found=signatures::declaration_lookup(metadata,owner,syntax::type_text(ref p,i));if(found.code!=0){return .err(fail(found.code,owner,n.start,n.end));}let d=signatures::declaration_at(metadata,found.item);if(d.kind!=4 && d.kind!=6 && d.kind!=11){return .err(fail(9,owner,n.start,n.end));}n.left=declaration(sources,d);}
        if(!v::push<types::Node>(ref mut types,n).is_ok){return .err(fail(11,0,n.start,n.end));}i+=1;}
    i=1;while(i<=syntax::expression_count(ref p)){let n=syntax::expression_at(ref p,i);var ast::Node node=ast::Node{.kind=n.kind,.start=n.start,.end=n.end,.token=n.token,.left=n.left,.right=n.right};
        if(n.kind==2 && raw::named(syntax::expression_text(ref p,i))==0 && qualified(syntax::expression_text(ref p,i))){let usize owner=syntax::module_index(ref p,n.start);let found=signatures::declaration_lookup(metadata,owner,syntax::expression_text(ref p,i));if(found.code!=0){return .err(fail(found.code,owner,n.start,n.end));}node.right=declaration(sources,signatures::declaration_at(metadata,found.item));}
        if(!v::push<ast::Node>(ref mut expressions,node).is_ok){return .err(fail(11,0,n.start,n.end));}i+=1;}
    i=1;while(i<=syntax::statement_count(ref p)){let n=syntax::statement_at(ref p,i);if(!v::push<ast::Statement>(ref mut statements,ast::Statement{.kind=n.kind,.start=n.start,.end=n.end,.expression=n.expression,.target=n.target,.first=n.first,.second=n.second,.entry=n.entry,.operator=n.operator}).is_ok){return .err(fail(11,0,n.start,n.end));}i+=1;}
    i=0;while(i<syntax::entry_count(ref p)){let n=syntax::entry_at(ref p,i);if(!v::push<bodies::Entry>(ref mut entries,bodies::Entry{.kind=n.kind,.start=n.start,.end=n.end,.name_start=n.name_start,.name_end=n.name_end,.type_start=n.type_start,.type_end=n.type_end,.type_root=n.type_root,.expression=n.expression,.id=i+1}).is_ok){return .err(fail(11,0,n.start,n.end));}i+=1;}
    i=0;while(i<syntax::field_count(ref p)){if(!v::push<records::Field>(ref mut fields,syntax::field_at(ref p,i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    i=0;while(i<syntax::attribute_count(ref p)){if(!v::push<attributes::Attribute>(ref mut attributes,syntax::attribute_at(ref p,i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    i=0;while(i<syntax::attribute_argument_count(ref p)){if(!v::push<attributes::Argument>(ref mut arguments,syntax::attribute_argument_at(ref p,i)).is_ok){return .err(fail(11,0,0,0));}i+=1;}
    var bytes::Buffer input=bytes::empty();if(!bytes::append(ref mut input,syntax::source_view(ref p)).is_ok){return .err(fail(11,0,0,0));}let stream=switch(scanner::collect(input,false,limits::program_bytes(),1048576,256,16777216)){case .ok(s):{s}case .err(_):{return .err(fail(11,0,0,0));}};
    return .ok(syntax::assemble(stream,declarations,problems,expressions,entries,statements,types,fields,attributes,arguments));
}
