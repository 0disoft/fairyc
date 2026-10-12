module tests::module_unit_mutable_call_places;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_unit_values as values;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn void unsupported(ref u8[] text){
    let data=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let tokens=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let source=switch(syntax::parse(tokens,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref source)==0);
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,source).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(values::build(ref sources,ref metadata,ref ready)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==2 && bytes::equal(text[e.start..e.end],b"ref identity(alias).value"));}};
}
@test fn void temporary_mutable_call_fields_are_not_admitted_as_borrow_places(){
    unsupported(b"module main;struct Cell{i32 value;}fn ref mut Cell identity(ref mut Cell cell){return cell;}fn i32 main(){var Cell cell=Cell{.value=0};let ref mut Cell alias=ref mut cell;let ref i32 view=ref identity(alias).value;alias.value=1;return *view;}");
    unsupported(b"module main;struct Cell{i32 value;}fn ref mut Cell identity(ref mut Cell cell){return cell;}fn i32 main(){var Cell cell=Cell{.value=0};let ref mut Cell alias=ref mut cell;let ref i32 view=ref identity(alias).value;let i32 saved=*view;alias.value=1;return saved;}");
}
