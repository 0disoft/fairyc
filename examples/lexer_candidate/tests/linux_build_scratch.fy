module tests::linux_build_scratch;
import linux_build_scratch as scratch;
import std::bytes as bytes;
@link_name("build_scratch_mode") unsafe extern C fn void mode(u32 value);
@link_name("build_scratch_closes") unsafe extern C fn u32 closes();
@test @effects(filesystem) fn void owns_exact_path_and_consuming_cleanup(){
    unsafe{mode(0);}let directory=switch(scratch::create(b"/tmp///")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(bytes::equal(scratch::path(ref directory),b"/tmp/scratch"));assert(scratch::close(directory).is_ok);assert(unsafe{closes()}==1);
    unsafe{mode(1);}let failed=switch(scratch::create(b"/tmp")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    switch(scratch::close(failed)){case .ok(_):{assert(false);}case .err(e):{assert(e==5);}};assert(unsafe{closes()}==1);
    unsafe{mode(2);}assert(!scratch::create(b"/tmp").is_ok);assert(unsafe{closes()}==1);
    unsafe{mode(3);}assert(!scratch::create(b"/tmp").is_ok);assert(unsafe{closes()}==1);
}
