module tests::linux_remote_fs;
import linux_remote_fs as fs;
import linux_rooted_file as rooted;
import linux_package_directory as lease;
import std::bytes as bytes;
@link_name("remote_fs_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("remote_fs_probe_live") unsafe extern C fn u32 live();
@link_name("remote_fs_probe_mutations") unsafe extern C fn u32 mutations();
@link_name("remote_fs_probe_reads") unsafe extern C fn u32 reads();
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
@effects(filesystem) fn u32 read_problem(ref rooted::Directory root,ref u8[] path,usize maximum){return switch(fs::read(root,path,maximum)){case .ok(_):{0}case .err(p):{p}};}
@effects(filesystem) fn bytes::Buffer read(ref rooted::Directory root,ref u8[] path,usize maximum){return switch(fs::read(root,path,maximum)){case .ok(p):{p}case .err(_):{assert(false);return read(root,path,maximum);}};}
@effects(filesystem) fn lease::Directory stage(ref rooted::Directory root){return switch(fs::create(root,b"stage")){case .ok(p):{p}case .err(_):{assert(false);return stage(root);}};}
@test @effects(filesystem) fn void reads_exact_binary_files_and_refuses_aliases_links_and_changes(){
    unsafe{mode(1);}{let root=opened();let contents=read(ref root,b"input",3);assert(bytes::equal(bytes::view(ref contents),b"a\0\xff"));let nested=read(ref root,b"tree/item",3);assert(bytes::equal(bytes::view(ref nested),b"a\0\xff"));assert(read_problem(ref root,b"input",2)==3 && read_problem(ref root,b"Input",3)==4 && read_problem(ref root,b"missing",3)==5 && read_problem(ref root,b"tree",3)==4);assert(unsafe{mutations()}==0);}assert(unsafe{live()}==0);
    var u32 value=2;while(value<=6){unsafe{mode(value);}{let root=opened();let code=read_problem(ref root,b"input",3);assert(code==if(value<=3){3}else if(value<=5){4}else{1});if(value==4){assert(unsafe{reads()}==0);}assert(unsafe{mutations()}==0);}assert(unsafe{live()}==0);value+=1;}
    unsafe{mode(11);}{let root=opened();assert(read_problem(ref root,b"input",3)==4 && unsafe{reads()}==0);}assert(unsafe{live()}==0);
    unsafe{mode(12);}{let root=opened();assert(read_problem(ref root,b"input",3)==4 && unsafe{reads()}==0);}assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void writes_syncs_and_publishes_complete_private_trees(){
    unsafe{mode(0);}{let root=opened();var staged=stage(ref root);
        assert(fs::write(lease::directory(ref staged),b"nested/data",b"a\0\xff")==0);
        assert(fs::write(lease::directory(ref staged),b"nested/empty",b"")==0);
        assert(fs::publish(ref mut staged,ref root,b"vendor")==0);
        assert(fs::publish(ref mut staged,ref root,b"another")!=0);
        let contents=read(ref root,b"vendor/nested/data",3);assert(bytes::equal(bytes::view(ref contents),b"a\0\xff"));
        let empty=read(ref root,b"vendor/nested/empty",0);assert(bytes::len(ref empty)==0);
        assert(fs::write(ref root,b"vendor/nested/data",b"new")==4);
        assert(!fs::create(ref root,b"vendor").is_ok && fs::erase(ref mut staged)!=0);
    }assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void rejects_paths_before_mutation_and_bounds_stage_cleanup(){
    unsafe{mode(0);}{let root=opened();assert(fs::write(ref root,b"ok/..",b"x")==4 && fs::write(ref root,b"a//b",b"x")==4 && fs::write(ref root,b"a/",b"x")==4 && fs::write(ref root,b"/a",b"x")==4 && fs::write(ref root,b"a\\b",b"x")==4 && fs::write(ref root,b"a:b",b"x")==4);
        var path=bytes::empty();var usize i=0;while(i<33){if(i>0){discard bytes::append_byte(ref mut path,47);}discard bytes::append_byte(ref mut path,97);i+=1;}
        assert(!fs::descend(ref root,bytes::view(ref path),true).is_ok && read_problem(ref root,b"input",16777217)==3 && unsafe{mutations()}==0);
        var staged=stage(ref root);assert(fs::write(lease::directory(ref staged),b"nested/data",b"a\0\xff")==0);
        var usize remaining=2;assert(fs::clear(lease::directory(ref staged),33,ref mut remaining)==3 && remaining==2);
        remaining=0;assert(fs::clear(lease::directory(ref staged),0,ref mut remaining)==3);
        remaining=2;assert(fs::clear(lease::directory(ref staged),0,ref mut remaining)==0 && remaining==0 && fs::erase(ref mut staged)==0);
    }assert(unsafe{live()}==0);
    unsafe{mode(13);}{let root=opened();var staged=stage(ref root);assert(fs::write(lease::directory(ref staged),b"data",b"a\0\xff")==0);var usize remaining=1;assert(fs::clear(lease::directory(ref staged),0,ref mut remaining)==1);let preserved=read(lease::directory(ref staged),b"data",3);assert(bytes::equal(bytes::view(ref preserved),b"a\0\xff") && fs::erase(ref mut staged)==1);}assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void failures_release_anonymous_files_without_publishing_names(){
    var u32 value=7;while(value<=9){unsafe{mode(value);}{let root=opened();assert(fs::write(ref root,b"data",b"a\0\xff")==if(value==8){4}else{1});let found=switch(fs::kind(ref root,b"data")){case .ok(p):{p}case .err(_):{99}};assert(found==0 && unsafe{mutations()}==0);}assert(unsafe{live()}==0);value+=1;}
}
