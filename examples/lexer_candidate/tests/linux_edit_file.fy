module tests::linux_edit_file;
import linux_edit_file as editing;
import linux_rooted_file as rooted;
import std::bytes as bytes;
@link_name("edit_probe_mode") unsafe extern C fn void mode(u32 value,u32 root_mode);
@link_name("edit_probe_calls") unsafe extern C fn u32 calls();
@link_name("rooted_probe_counts") unsafe extern C fn u32 counts();
@link_name("rooted_probe_written") unsafe extern C fn usize written();
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
@effects(filesystem) fn rooted::Directory child(ref rooted::Directory root){return switch(rooted::directory(root,b"child")){case .ok(p):{p}case .err(_):{assert(false);return child(root);}};}
@effects(filesystem) fn editing::File created(ref rooted::Directory root){return switch(editing::create(root,original())){case .ok(p):{p}case .err(_):{assert(false);return created(root);}};}
fn rooted::Metadata original(){return rooted::Metadata{.device=71,.identity=123,.size=3,.modified_seconds=91,.modified_nanos=42,.mode=33184};}
@test @effects(filesystem) fn void preserves_preparation_publication_and_consuming_close(){
    unsafe{mode(0,0);}let root=opened();let stage=child(ref root);
    var file=created(ref root);
    assert(!editing::publish(ref mut file,ref stage,ref root,b"target").is_ok && unsafe{calls()}==0);
    assert(editing::write_all(ref mut file,b"n\0\xff").is_ok && unsafe{written()}==3);
    assert(editing::prepare(ref mut file).is_ok && unsafe{calls()}==100);
    assert(!editing::prepare(ref mut file).is_ok && !editing::publish(ref mut file,ref stage,ref root,b"../x").is_ok && unsafe{calls()}==100);
    assert(editing::publish(ref mut file,ref stage,ref root,b"target").is_ok && editing::published(ref file) && unsafe{calls()}==101);
    assert(!editing::write_all(ref mut file,b"x").is_ok && !editing::publish(ref mut file,ref stage,ref root,b"target").is_ok && unsafe{calls()}==101);
    assert(editing::close(file).is_ok);assert(rooted::close_directory(stage).is_ok);assert(rooted::close_directory(root).is_ok);
    assert(unsafe{counts()}==303);
}
@test @effects(filesystem) fn void bounds_writes_and_invalidates_preparation(){
    unsafe{mode(0,0);}let root=opened();var file=created(ref root);
    var data=bytes::empty();assert(bytes::resize_zeroed(ref mut data,editing::maximum()-1).is_ok);
    assert(editing::write_all(ref mut file,bytes::view(ref data)).is_ok && unsafe{written()}==editing::maximum()-1);
    assert(editing::prepare(ref mut file).is_ok && unsafe{calls()}==100);
    assert(editing::write_all(ref mut file,b"x").is_ok && unsafe{written()}==editing::maximum());
    assert(!editing::write_all(ref mut file,b"x").is_ok && unsafe{written()}==editing::maximum());
    assert(editing::prepare(ref mut file).is_ok && unsafe{calls()}==200);
    assert(editing::close(file).is_ok);assert(rooted::close_directory(root).is_ok);
}
@test @effects(filesystem) fn void rejects_metadata_and_preserves_native_failure_phases(){
    unsafe{mode(0,0);}let root=opened();let stage=child(ref root);
    let readonly=rooted::Metadata{.device=71,.identity=123,.size=3,.modified_seconds=91,.modified_nanos=42,.mode=33024};
    let malformed=rooted::Metadata{.device=71,.identity=123,.size=3,.modified_seconds=91,.modified_nanos=1000000000,.mode=33184};
    let before=unsafe{counts()};
    assert(!editing::writable(readonly) && !editing::create(ref root,readonly).is_ok && unsafe{calls()}==0);
    assert(!editing::writable(malformed) && !editing::create(ref root,malformed).is_ok && unsafe{calls()}==0);
    assert(unsafe{counts()}==before);
    var u32 value=1;while(value<=4){
        unsafe{mode(value,0);}
        {var file=created(ref root);
            if(value==1){let problem=switch(editing::prepare(ref mut file)){case .ok(_):{assert(false);return;}case .err(p):{p}};assert(problem.stage==6 && problem.os_code==5 && !problem.published);assert(!editing::prepare(ref mut file).is_ok && !editing::write_all(ref mut file,b"x").is_ok && unsafe{calls()}==100);}
            else{assert(editing::prepare(ref mut file).is_ok);let problem=switch(editing::publish(ref mut file,ref stage,ref root,b"target")){case .ok(_):{assert(false);return;}case .err(p):{p}};
                assert(problem.stage==if(value==4){9}else{7} && problem.os_code==if(value==2){116}else if(value==3){5}else{0} && problem.cleanup_code==if(value==2){0}else{13});
                assert(problem.published==(value==4) && editing::published(ref file)==(value==4));
                assert(!editing::publish(ref mut file,ref stage,ref root,b"target").is_ok && !editing::write_all(ref mut file,b"x").is_ok && unsafe{calls()}==101);
            }
        }
        assert(unsafe{counts()}==101);value+=1;
    }
    unsafe{mode(0,0);}assert(rooted::close_directory(stage).is_ok);assert(rooted::close_directory(root).is_ok);
}
@test @effects(filesystem) fn void retains_publication_on_consuming_close_error(){
    unsafe{mode(0,0);}let root=opened();let stage=child(ref root);var file=created(ref root);
    assert(editing::prepare(ref mut file).is_ok && editing::publish(ref mut file,ref stage,ref root,b"target").is_ok);
    unsafe{mode(0,1);}
    switch(editing::close(file)){case .ok(_):{assert(false);}case .err(p):{assert(p.stage==8 && p.os_code==4 && p.published);}};
    assert(unsafe{counts()}==1);unsafe{mode(0,0);}assert(rooted::close_directory(stage).is_ok);assert(rooted::close_directory(root).is_ok);
}
@test @effects(filesystem) fn void preserves_partial_writes_and_never_retries_close(){
    unsafe{mode(0,0);}let root=opened();
    unsafe{mode(0,12);}
    {var file=created(ref root);let p=switch(editing::write_all(ref mut file,b"0123456789")){case .ok(_):{assert(false);return;}case .err(p):{p}};assert(p.stage==4 && p.os_code==5 && p.completed==7);assert(!editing::prepare(ref mut file).is_ok && !editing::write_all(ref mut file,b"x").is_ok && unsafe{calls()}==0);}
    assert(unsafe{counts()}==101);
    unsafe{mode(0,0);}var file=created(ref root);unsafe{mode(0,1);}
    switch(editing::close(file)){case .ok(_):{assert(false);}case .err(p):{assert(p.stage==8 && p.os_code==4 && !p.published);}};
    assert(unsafe{counts()}==1);unsafe{mode(0,0);}assert(rooted::close_directory(root).is_ok);
}
