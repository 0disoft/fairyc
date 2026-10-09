module tests::module_unit_plan;
import module_unit_plan as plan;
import module_units as units;
import module_unit_bind as binding;
import module_unit_interfaces as interfaces;
import module_unit_generics as generics;
import module_signatures as signatures;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn void suffix(ref mut bytes::Buffer text,usize n){let digits=b"0123456789";if(n>=10){assert(bytes::append_byte(text,digits[n/10]).is_ok);}assert(bytes::append_byte(text,digits[n%10]).is_ok);}
fn syntax::Parsed fixture(ref u8[] header,usize count){var bytes::Buffer text=bytes::empty();assert(bytes::append(ref mut text,header).is_ok);var usize i=0;while(i<count){assert(bytes::append(ref mut text,b"fn T f").is_ok);suffix(ref mut text,i);assert(bytes::append(ref mut text,b"<T>(T n){return n;}").is_ok);i+=1;}assert(bytes::append(ref mut text,b"fn i32 run(){").is_ok);i=0;while(i<count){assert(bytes::append(ref mut text,b"discard f").is_ok);suffix(ref mut text,i);assert(bytes::append(ref mut text,b"<i32>(1);").is_ok);i+=1;}assert(bytes::append(ref mut text,b"return 0;}").is_ok);return parsed(bytes::view(ref text));}
@test fn void specialization_pressure_refines_ast_ranges_without_losing_coordinates(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,fixture(b"module first;",35)).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,fixture(b"module second;",35)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};let initial=switch(units::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(units::count(ref initial)==1);
    let p=switch(plan::build(ref sources,ref metadata)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(plan::count(ref p)==2 && plan::file_count(ref p)==2 && plan::splits(ref p)==1 && plan::attempts(ref p)==3 && plan::source_bytes(ref p)==units::source_bytes(ref initial));
    var usize i=0;var usize total=0;while(i<plan::count(ref p)){let u=plan::at(ref p,i);assert(u.first==i && u.stop==i+1 && u.usage.bytes==syntax::source_view(v::at<syntax::Parsed>(ref sources,i)).len);let bound=switch(binding::bind(ref sources,ref metadata,u.first,u.stop)){case .ok(b):{b}case .err(_):{assert(false);return;}};let imported=switch(interfaces::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};let g=interfaces::specializations(ref imported);assert(generics::count(g)==35);var usize j=1;while(j<=generics::count(g)){assert(generics::at(g,j).module_index==i);j+=1;}total+=u.usage.bytes;i+=1;}
    assert(total+1==plan::source_bytes(ref p) && plan::work(ref p)<1048576 && plan::binding_work(ref p)<1048576 && plan::interface_work(ref p)<1048576);
}
@test fn void irreducible_or_noncapacity_errors_remain_original_failures(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,fixture(b"module main;",65)).is_ok);let metadata=switch(signatures::build(ref sources)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(plan::build(ref sources,ref metadata)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4503 && p.module_index==0 && p.end>p.start);}};
    var v::Vec<syntax::Parsed> bad=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut bad,parsed(b"module first;fn i32 run(){return 0;}")).is_ok);let text=b"module second;fn i32 value(i32 n){return n;}fn i32 run(){return value<i32>(1);}";assert(v::push<syntax::Parsed>(ref mut bad,parsed(text)).is_ok);let m=switch(signatures::build(ref bad)){case .ok(m):{m}case .err(_):{assert(false);return;}};switch(plan::build(ref bad,ref m)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4501 && p.module_index==1 && bytes::equal(text[p.start..p.end],b"value<i32>"));}};
}
