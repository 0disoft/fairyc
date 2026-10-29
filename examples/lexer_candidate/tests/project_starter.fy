module tests::project_starter;
import project_starter as starter;
import std::bytes as bytes;
@link_name("starter_template_check") unsafe extern C fn void check(usize index,ptr u8 data,usize size);
@test fn void matches_all_reference_starter_bytes(){var usize i=0;while(i<5){let body=switch(starter::body(i,b"hello-fairy",b"example.hello-fairy",b"fairyc-language/20")){case .ok(p):{p}case .err(_):{assert(false);return;}};unsafe{check(i,bytes::view(ref body) as ptr u8,bytes::len(ref body));};i+=1;}assert(!starter::body(5,b"x",b"x.y",b"fairyc-language/20").is_ok);assert(bytes::equal(starter::leaf(1),b"src") && bytes::equal(starter::leaf(7),b"tests/main.fy") && starter::leaf(8).len==0);}
