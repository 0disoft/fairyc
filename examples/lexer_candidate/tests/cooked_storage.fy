module tests::cooked_storage;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import syntax as syntax;
import module_ast_join as join;

fn bytes::Buffer copied(ref u8[] data){return switch(bytes::copy(data)){case .ok(b):{b}case .err(_):{assert(false);copied(b"")}};}
fn void equal_streams(ref scanner::Stream a,ref scanner::Stream b){
    assert(bytes::equal(scanner::source_view(a),scanner::source_view(b)) && scanner::token_count(a)==scanner::token_count(b) && scanner::diagnostic_count(a)==scanner::diagnostic_count(b));
    var usize i=0;while(i<scanner::token_count(a)){let x=scanner::token_info(a,i);let y=scanner::token_info(b,i);assert(x.kind==y.kind && x.trivia==y.trivia && x.start==y.start && x.end==y.end && bytes::equal(scanner::token_cooked(a,i),scanner::token_cooked(b,i)));i+=1;}
    i=0;while(i<scanner::diagnostic_count(a)){let x=scanner::diagnostic_info(a,i);let y=scanner::diagnostic_info(b,i);assert(x.code==y.code && x.start==y.start && x.end==y.end && x.exit_repair==y.exit_repair && bytes::equal(scanner::diagnostic_token(a,i),scanner::diagnostic_token(b,i)));i+=1;}
}
@test fn void joined_pool_preserves_collisions_kinds_and_snapshot(){
    let input=b"b\"\\x00\\x00\" b\"\\x05\\xFA\" b\"\\x00\\x00\" b\"A\" \"\\x41\" c\"A\" 65 0x41 /// doc\r\nb\"\\x41\" /* open";
    let a=switch(scanner::collect(copied(input),false,1048576,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};
    let b=switch(scanner::collect_joined(copied(input))){case .ok(s):{s}case .err(_):{assert(false);return;}};equal_streams(ref a,ref b);
    assert(scanner::storage(ref b).cooked_bytes<scanner::storage(ref a).cooked_bytes);var usize work=0;
    let snapshot=switch(scanner::copy(ref b,ref mut work)){case .ok(s):{s}case .err(_):{assert(false);return;}};equal_streams(ref b,ref snapshot);assert(scanner::storage(ref b).cooked_bytes==scanner::storage(ref snapshot).cooked_bytes);
    work=1048576;switch(scanner::copy(ref b,ref mut work)){case .ok(_):{assert(false);}case .err(c):{assert(c==10);}};
}
@test fn void repeated_cooked_data_exceeds_logical_cap_without_extra_storage(){
    var bytes::Buffer input=bytes::empty();let payload=b"abcdefghijklmnopqrstuvwxyz012345";var usize i=0;
    while(i<3000){assert(bytes::append(ref mut input,b"b\"").is_ok && bytes::append(ref mut input,payload).is_ok && bytes::append(ref mut input,b"\" ").is_ok);i+=1;}
    switch(scanner::collect(copied(bytes::view(ref input)),false,1048576,1048576,256,16777216)){case .ok(_):{assert(false);}case .err(c):{assert(c==4);}};
    let s=switch(scanner::collect_joined(input)){case .ok(s):{s}case .err(_):{assert(false);return;}};
    assert(scanner::diagnostic_count(ref s)==0 && scanner::token_count(ref s)==6001 && scanner::storage(ref s).cooked_bytes==64);
    i=0;while(i<scanner::token_count(ref s)){let t=scanner::token_info(ref s,i);if(t.kind==8){assert(bytes::equal(scanner::token_cooked(ref s,i),b"6162636465666768696a6b6c6d6e6f707172737475767778797a303132333435"));}i+=1;}
}
@test fn void unique_cooked_capacity_remains_65536_bytes(){
    var bytes::Buffer input=bytes::empty();var usize i=0;
    while(i<257){if(i==256){let s=switch(scanner::collect_joined(copied(bytes::view(ref input)))){case .ok(s):{s}case .err(_):{assert(false);return;}};assert(scanner::storage(ref s).cooked_bytes==65536 && scanner::diagnostic_count(ref s)==0);}
        assert(bytes::append(ref mut input,b"b\"").is_ok && bytes::append_usize(ref mut input,1000+i).is_ok);
        var usize n=0;while(n<124){assert(bytes::append(ref mut input,b"x").is_ok);n+=1;}assert(bytes::append(ref mut input,b"\" ").is_ok);i+=1;
    }
    switch(scanner::collect_joined(input)){case .ok(_):{assert(false);}case .err(c):{assert(c==4);}};
}
@test fn void full_cache_keeps_uncached_values_and_bounds_search_work(){
    var bytes::Buffer input=bytes::empty();var usize i=0;
    while(i<5000){if(i==300){var bytes::Buffer sample=copied(bytes::view(ref input));assert(bytes::append(ref mut sample,b"b\"1299\" ").is_ok);
            let s=switch(scanner::collect_joined(sample)){case .ok(s):{s}case .err(_):{assert(false);return;}};assert(scanner::diagnostic_count(ref s)==0 && scanner::storage(ref s).cooked_bytes==2400 && scanner::token_count(ref s)==603);
        }
        assert(bytes::append(ref mut input,b"b\"").is_ok && bytes::append_usize(ref mut input,1000+i).is_ok && bytes::append(ref mut input,b"\" ").is_ok);i+=1;
    }
    let original=switch(scanner::collect(copied(bytes::view(ref input)),false,1048576,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};assert(scanner::diagnostic_count(ref original)==0 && scanner::storage(ref original).cooked_bytes==40000);
    let bounded=switch(scanner::collect_joined(input)){case .ok(s):{s}case .err(_):{assert(false);return;}};
    assert(scanner::diagnostic_count(ref bounded)==1 && scanner::diagnostic_info(ref bounded,0).code==10 && scanner::token_count(ref bounded)<scanner::token_count(ref original));
}
@test fn void joined_work_exhaustion_never_publishes_partial_arenas(){
    var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();var usize module_index=0;
    while(module_index<5){var bytes::Buffer input=bytes::empty();assert(bytes::append(ref mut input,b"module user").is_ok && bytes::append_usize(ref mut input,module_index).is_ok && bytes::append(ref mut input,b";fn void f(){").is_ok);
        var usize i=0;while(i<1000){assert(bytes::append(ref mut input,b"discard b\"").is_ok && bytes::append_usize(ref mut input,1000+module_index*1000+i).is_ok && bytes::append(ref mut input,b"\";").is_ok);i+=1;}assert(bytes::append(ref mut input,b"}").is_ok);
        let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return;}};assert(scanner::diagnostic_count(ref stream)==0);
        let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(syntax::problem_count(ref p)==0);assert(v::push<syntax::Parsed>(ref mut files,p).is_ok);module_index+=1;
    }
    switch(join::join(ref files)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10 && e.module_index==0 && e.start==0 && e.end==0);}};
}
