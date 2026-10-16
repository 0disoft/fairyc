module tests::project_native_test_options;
import project_test_options as options;
import std::bytes as bytes;
import std::vec as v;

@test fn void execution_options_preserve_arguments_and_deadline(){
    var v::Vec<bytes::Buffer> argv=v::empty<bytes::Buffer>();
    assert(options::test_add(ref mut argv,b"--native"));
    assert(options::test_add(ref mut argv,b"--timeout-seconds"));
    assert(options::test_add(ref mut argv,b"2"));
    assert(options::test_add(ref mut argv,b"--stdin-file"));
    assert(options::test_add(ref mut argv,b"fixtures/input.bin"));
    assert(options::test_add(ref mut argv,b"--"));
    assert(options::test_add(ref mut argv,b""));
    assert(options::test_add(ref mut argv,b"--release"));
    let selected=switch(options::parse_test_options(ref argv)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(options::timeout_ms(ref selected)==2000);
    assert(bytes::equal(options::stdin_file(ref selected),b"fixtures/input.bin"));
    assert(v::len<bytes::Buffer>(options::arguments(ref selected))==2);
    assert(bytes::len(v::at<bytes::Buffer>(options::arguments(ref selected),0))==0);
    assert(bytes::equal(bytes::view(v::at<bytes::Buffer>(options::arguments(ref selected),1)),b"--release"));
    assert(!options::flags(ref selected).release);
}

@test fn void exact_selection_retains_canonical_order(){
    var v::Vec<bytes::Buffer> argv=v::empty<bytes::Buffer>();
    assert(options::test_add(ref mut argv,b"--native"));
    assert(options::test_add(ref mut argv,b"--exact"));
    assert(options::test_add(ref mut argv,b"tests::z"));
    assert(options::test_add(ref mut argv,b"--exact"));
    assert(options::test_add(ref mut argv,b"tests::a"));
    let selected=switch(options::parse_test_options(ref argv)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    var v::Vec<options::TestIdentity> tests=v::empty<options::TestIdentity>();
    var usize i=0;while(i<3){let name=if(i==0){b"tests::z"}else if(i==1){b"tests::skip"}else{b"tests::a"};
        let identity=switch(bytes::copy(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(v::push<options::TestIdentity>(ref mut tests,options::identity(identity,bytes::empty(),bytes::empty(),0,i)).is_ok);i+=1;
    }
    var v::Vec<usize> chosen=v::empty<usize>();
    assert(options::test_choose(ref selected,ref tests,ref mut chosen));
    assert(v::len<usize>(ref chosen)==2);
    assert(*v::at<usize>(ref chosen,0)==2 && *v::at<usize>(ref chosen,1)==0);
    assert(options::timeout_ms(ref selected)==60000);
}
