module tests::linux_package_file;
import linux_package_file as package;
import linux_rooted_file as rooted;
@link_name("package_file_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("package_file_probe_counts") unsafe extern C fn u32 counts();
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
@test @effects(filesystem) fn void checks_package_metadata_without_weakening_generic_files(){
    let root=opened();let file=switch(rooted::open_read(ref root,b"input")){case .ok(p):{p}case .err(_):{assert(false);return;}};
    unsafe{mode(0);}let info=switch(package::observe(ref file)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(info.device==71 && info.identity==123 && info.size==3 && info.modified_seconds==91 && info.modified_nanos==42 && info.mode==32768 && unsafe{counts()}==100);
    var u32 value=1;while(value<=5){unsafe{mode(value);}let problem=switch(package::observe(ref file)){case .ok(_):{assert(false);return;}case .err(p):{p}};
        assert(problem.kind==if(value==1){6}else if(value==2){3}else{5});
        assert(problem.os_code==if(value==1){31}else if(value==2){5}else{0});assert(unsafe{counts()}==100);
        assert(rooted::observe(ref file).is_ok);value+=1;
    }
}
@test @effects(filesystem) fn void preserves_exact_conditional_removal_and_partial_errors(){
    let root=opened();let info=rooted::Metadata{.device=71,.identity=123,.size=3,.modified_seconds=91,.modified_nanos=42,.mode=32768};
    unsafe{mode(0);}assert(!package::erase_if(ref root,b"../x",info).is_ok && unsafe{counts()}==0);
    var invalid=info;invalid.identity=0;assert(!package::erase_if(ref root,b"input",invalid).is_ok && unsafe{counts()}==0);
    invalid=info;invalid.modified_nanos=1000000000;assert(!package::erase_if(ref root,b"input",invalid).is_ok && unsafe{counts()}==0);
    invalid=info;invalid.mode=16384;assert(!package::erase_if(ref root,b"input",invalid).is_ok && unsafe{counts()}==0);
    assert(switch(package::erase_if(ref root,b"input",info)){case .ok(p):{p}case .err(_):{false}} && unsafe{counts()}==1);
    unsafe{mode(6);}assert(!switch(package::erase_if(ref root,b"input",info)){case .ok(p):{p}case .err(_):{true}} && unsafe{counts()}==1);
    var u32 value=7;while(value<=9){unsafe{mode(value);}let problem=switch(package::erase_if(ref root,b"input",info)){case .ok(_):{assert(false);return;}case .err(p):{p}};
        assert(problem.kind==if(value==7){4}else if(value==8){5}else{3});
        assert(problem.os_code==if(value==9){5}else{0});assert(problem.close_code==if(value==8){0}else{4});assert(problem.removed==(value!=8) && unsafe{counts()}==1);value+=1;
    }
}
