module tests::linux_package_directory;
import linux_package_directory as package;
import linux_rooted_file as rooted;
import std::bytes as bytes;
@link_name("package_directory_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("package_directory_probe_calls") unsafe extern C fn u32 calls();
@link_name("rooted_probe_counts") unsafe extern C fn u32 counts();
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
@effects(filesystem) fn package::Directory created(ref rooted::Directory root){return switch(package::create(root,b"stage")){case .ok(p):{p}case .err(_):{assert(false);return created(root);}};}
@test @effects(filesystem) fn void validates_input_and_transfers_owned_handles(){
    unsafe{mode(0);}
    {
        let root=opened();
        assert(!package::create(ref root,b"").is_ok && !package::create(ref root,b"..").is_ok && !package::edit(ref root,b"a/b").is_ok && !package::create(ref root,b"\xff").is_ok);
        var u8[256] long=[65;256];let ref u8[] view=ref long;
        assert(!package::create(ref root,view).is_ok && unsafe{calls()}==0);
        let stage=created(ref root);
        assert(rooted::directory_handle(package::directory(ref stage))==2 && unsafe{calls()}==100 && unsafe{counts()}==100);
    }
    assert(unsafe{counts()}==103);
    unsafe{mode(0);}
    {let root=opened();let stage=switch(package::edit(ref root,b"stage")){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(rooted::directory_handle(package::directory(ref stage))==2 && unsafe{calls()}==100);}
    assert(unsafe{counts()}==103);
    var u32 value=11;while(value<=13){
        unsafe{mode(value);}
        {let root=opened();let problem=switch(package::create(ref root,b"stage")){case .ok(_):{assert(false);return;}case .err(p):{p}};assert(problem.kind==5 && problem.os_code==0 && problem.cleanup_code==0);}
        assert(unsafe{counts()}==if(value==13){101}else{102});value+=1;
    }
}
@test @effects(filesystem) fn void preserves_native_errors_and_reclaims_partial_or_invalid_outputs(){
    var u32 value=1;
    while(value<=7){
        unsafe{mode(value);}
        {let root=opened();let problem=switch(package::create(ref root,b"stage")){case .ok(_):{assert(false);return;}case .err(p):{p}};
            assert(problem.kind==if(value<=2){3}else{5});
            assert(problem.os_code==if(value==1){17}else if(value==2){5}else{0});
            assert(problem.cleanup_code==if(value==7){5}else{0});
            assert(unsafe{calls()}==100);
        }
        assert(unsafe{counts()}==if(value==1){101}else if(value==2 || value==7){103}else{102});
        value+=1;
    }
    unsafe{mode(8);}
    {let root=opened();let problem=switch(package::create(ref root,b"stage")){case .ok(_):{assert(false);return;}case .err(p):{p}};assert(problem.kind==3 && problem.os_code==5 && problem.cleanup_code==4);}
    assert(unsafe{counts()}==103);
}
@test @effects(filesystem) fn void publication_and_removal_freeze_the_old_name(){
    unsafe{mode(0);}
    {let root=opened();var stage=created(ref root);
        assert(!package::publish(ref mut stage,ref root,b"../x").is_ok && unsafe{calls()}==100);
        assert(package::publish(ref mut stage,ref root,b"package").is_ok && unsafe{calls()}==110);
        let repeated=switch(package::publish(ref mut stage,ref root,b"package")){case .ok(_):{assert(false);return;}case .err(p):{p}};
        assert(repeated.kind==6 && !package::erase(ref mut stage).is_ok && unsafe{calls()}==110);
    }assert(unsafe{counts()}==103);
    unsafe{mode(0);}
    {let root=opened();var stage=created(ref root);assert(package::erase(ref mut stage).is_ok && unsafe{calls()}==101);assert(!package::erase(ref mut stage).is_ok && !package::publish(ref mut stage,ref root,b"package").is_ok && unsafe{calls()}==101);}
    assert(unsafe{counts()}==103);
}
@test @effects(filesystem) fn void failed_publication_retains_cleanup_authority_without_retry(){
    unsafe{mode(9);}
    {let root=opened();var stage=created(ref root);let problem=switch(package::publish(ref mut stage,ref root,b"package")){case .ok(_):{assert(false);return;}case .err(p):{p}};
        assert(problem.kind==3 && problem.os_code==4 && unsafe{calls()}==110);
        assert(package::erase(ref mut stage).is_ok && unsafe{calls()}==111);
    }assert(unsafe{counts()}==103);
    unsafe{mode(10);}
    {let root=opened();var stage=created(ref root);let problem=switch(package::erase(ref mut stage)){case .ok(_):{assert(false);return;}case .err(p):{p}};assert(problem.kind==3 && problem.os_code==39 && unsafe{calls()}==101);}
    assert(unsafe{counts()}==103);
}

@test @effects(filesystem) fn void duplicates_owned_directory_without_losing_or_closing_borrowed_handles(){
    unsafe{mode(0);}
    {let root=opened();let owned={let staged=created(ref root);switch(package::duplicate(ref staged)){case .ok(p):{p}case .err(_):{assert(false);return;}}};
        assert(rooted::directory_handle(ref owned)==9 && unsafe{counts()}==102);
        assert(rooted::close_directory(owned).is_ok && unsafe{counts()}==103);
    }assert(unsafe{counts()}==104);
    var u32 value=14;while(value<=18){unsafe{mode(value);}
        {let root=opened();let staged=created(ref root);let problem=switch(package::duplicate(ref staged)){case .ok(_):{assert(false);return;}case .err(p):{p}};
            assert(problem.kind==if(value==18){3}else{5});assert(problem.os_code==if(value==18){5}else{0});assert(problem.cleanup_code==if(value==18){4}else{0});
        }assert(unsafe{counts()}==if(value==18){104}else{103});value+=1;
    }
}
