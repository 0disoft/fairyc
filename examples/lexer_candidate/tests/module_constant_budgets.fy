module tests::module_constant_budgets;
import module_unit_constants as constants;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(bytes::Buffer source){let stream=switch(scanner::collect(source,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}
@test
fn void expression_allowance_is_shared_within_a_package_and_separate_between_packages(){
    var bytes::Buffer expression=bytes::empty();assert(bytes::append(ref mut expression,b"true").is_ok);var usize level=0;
    while(level<11){var bytes::Buffer next=bytes::empty();assert(bytes::append(ref mut next,b"(").is_ok && bytes::append(ref mut next,bytes::view(ref expression)).is_ok && bytes::append(ref mut next,b"&&").is_ok && bytes::append(ref mut next,bytes::view(ref expression)).is_ok && bytes::append(ref mut next,b")").is_ok);expression=next;level+=1;}
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var u8 name=97;var usize index=0;
    while(index<12){var bytes::Buffer source=bytes::empty();assert(bytes::append(ref mut source,b"module ").is_ok && bytes::append_byte(ref mut source,name).is_ok && bytes::append(ref mut source,b";static_assert(").is_ok && bytes::append(ref mut source,bytes::view(ref expression)).is_ok && bytes::append(ref mut source,b");").is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(source)).is_ok && v::push<usize>(ref mut owners,if(index<6){0}else{1}).is_ok);name+=1;index+=1;}
    // Every declaration has the same expression workload; only checked owner
    // coordinates change which package consumes the 65,536-node allowance.
    let distinct=switch(constants::build_owned(ref sources,ref owners)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(constants::work(ref distinct)>0);
    index=0;while(index<v::len<usize>(ref owners)){let cell=v::at_mut<usize>(ref mut owners,index);*cell=0;index+=1;}
    switch(constants::build_owned(ref sources,ref owners)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==4304 && p.module_index<12 && p.start<p.end);}};
}
