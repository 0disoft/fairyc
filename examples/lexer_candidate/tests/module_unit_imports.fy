module tests::module_unit_imports;
import module_unit_c_emit as emit;
import module_unit_c_sdk as sdk;
import module_unit_c_region as lifecycle;
import c_text as w;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_cleanup_mir_model as clean;
import module_unit_values as values;
import module_unit_moves as moves;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

import module_types as types;
import module_unit_generics as generics;
import module_unit_interfaces as interfaces;
fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@test @effects(console) fn void root_only_transitive_imports_keep_owned_identity(){
    let p={var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){let i32 n=41;let ref i32 borrowed=p::borrow(ref n);assert(p::go(*borrowed)==42);return 0;}")).is_ok);
        assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module unused;fn i32 uncalled(){return true;}")).is_ok);
        assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;import leaf as l;pub fn ref i32 borrow(ref i32 n){return n;}pub fn i32 go(i32 n){return helper(n);}fn i32 helper(i32 n){return l::add(n);}fn i32 uncalled(){return true;}")).is_ok);
        assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module leaf;fn T id<T>(T n){return n;}pub fn i32 add(i32 n){return id<i32>(n)+1;}")).is_ok);
        let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}};
        assert(closure::passes(ref ready)>=3);let owned=interfaces::specializations(closure::imported(ref ready));assert(generics::count(owned)==5);var usize ordinary=0;var usize template=0;var usize i=1;while(i<=generics::count(owned)){let f=generics::at(owned,i);assert(f.module_index>=2 && f.module_index<4);if(f.argument_count==0){ordinary+=1;}else{template+=1;}i+=1;}assert(ordinary==4 && template==1);
        let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}};let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(_):{assert(false);return;}};
        switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);assert(false);return;}}
    };assert(ir::count(ref p)==6);var usize entry=0;var usize i=1;while(i<=ir::count(ref p)){let f=clean::frame_at(ir::cleanup_ir(ref p),i);assert(f.module_index!=1);if(f.module_index==0){entry=i;}i+=1;}
    var bytes::Buffer out=bytes::empty();let result=emit::render(ref p,entry,ref mut out);if(result.code!=0){discard io::write_u32(result.code);}assert(result.code==0 && io::write_bytes(bytes::view(ref out)).is_ok);
}

@test fn void rejected_import_expansion_rolls_back_borrow_types(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){return p::go();}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;fn T id<T>(T n){return n;}pub fn i32 go(){var i32[2] a=[0;2];discard id<ref i32>(ref a[0]);return 0;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var v::Vec<types::Type> graph=v::empty<types::Type>();var usize i=1;while(i<=signatures::type_count(ref metadata)){assert(v::push<types::Type>(ref mut graph,signatures::type_at(ref metadata,i)).is_ok);i+=1;}let plan=switch(generics::build(ref sources,ref metadata,ref bound,ref mut graph)){case .ok(p):{p}case .err(_):{assert(false);return;}};let usize before=types::count(ref graph);let seeds=v::empty<binding::Coordinate>();assert(switch(generics::extend_imports(ref sources,ref metadata,ref bound,ref seeds,plan,ref mut graph)){case .ok(_):{false}case .err(e):{e.code==4502}});assert(types::count(ref graph)==before);
}
