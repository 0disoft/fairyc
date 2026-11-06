module tests::linux_edit_fs;
import linux_edit_fs as fs;
import linux_rooted_file as rooted;
import linux_directory as directory;
import std::bytes as bytes;
@link_name("edit_source_probe") unsafe extern C fn void mode(u32 scenario,usize size,usize actual,u32 flags);
@link_name("edit_source_probe_live") unsafe extern C fn u32 live();
@link_name("edit_source_probe_calls") unsafe extern C fn u32 calls();
@link_name("edit_source_probe_reads") unsafe extern C fn u32 reads();
@link_name("edit_source_probe_siblings") unsafe extern C fn void siblings(usize count);
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
resource Contents{bytes::Buffer data;rooted::Metadata info;cleanup{}}
fn ref u8[] view(ref Contents p){return bytes::view(ref p.data);}
fn rooted::Metadata metadata(ref Contents p){return p.info;}
@effects(filesystem) fn Contents loaded(ref rooted::Directory root,ref u8[] path,usize maximum){var info=fs::empty_metadata();let data=switch(fs::read_observed(root,path,maximum,ref mut info)){case .ok(p):{p}case .err(_):{assert(false);return loaded(root,path,maximum);}};return Contents{.data,.info};}
@effects(filesystem) fn fs::Problem problem(ref rooted::Directory root,ref u8[] path,usize maximum){var info=fs::empty_metadata();return switch(fs::read_observed(root,path,maximum,ref mut info)){case .ok(_):{assert(false);return problem(root,path,maximum);}case .err(p):{p}};}
@test @effects(filesystem) fn void preserves_binary_metadata_hard_links_and_exact_spelling(){
    unsafe{mode(0,3,3,0);}{let root=opened();let p=loaded(ref root,b"input",3);assert(bytes::equal(view(ref p),b"a\0\xff"));let m=metadata(ref p);assert(m.device==71 && m.identity==123 && m.size==3 && m.modified_seconds==91 && m.modified_nanos==42 && m.mode==33184);let data=switch(fs::read(ref root,b"input",3)){case .ok(q):{q}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref data),b"a\0\xff"));assert(unsafe{live()}==1);}assert(unsafe{live()}==0);
    // Mode 1 has multiple regular hard links. The package-file observer is
    // deliberately absent from this probe, so it cannot authorize these reads.
    unsafe{mode(1,3,3,0);}{let root=opened();let p=loaded(ref root,b"tree/branch/item",3);assert(bytes::equal(view(ref p),b"a\0\xff"));assert(unsafe{live()}==1);assert(problem(ref root,b"Input",3).kind==5);}assert(unsafe{live()}==0);
    unsafe{mode(40,3,3,0);}{let root=opened();let p=loaded(ref root,b"input",3);assert((metadata(ref p).mode&128)==0);}assert(unsafe{live()}==0);
    unsafe{mode(42,3,3,0);}{let root=opened();let p=loaded(ref root,b"input",3);assert(bytes::equal(view(ref p),b"a\0\xff"));}assert(unsafe{live()}==0);
    unsafe{mode(51,3,3,0);}{let root=opened();let p=loaded(ref root,b"input",3);assert(bytes::equal(view(ref p),b"a\0\xff"));}assert(unsafe{live()}==0);
    unsafe{mode(0,0,0,0);}{let root=opened();let p=loaded(ref root,b"input",0);assert(view(ref p).len==0);}assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void rejects_paths_before_io_and_bounds_exact_bytes(){
    unsafe{mode(0,3,3,0);}{let root=opened();let count=unsafe{calls()};
        assert(problem(ref root,b"",3).kind==4 && problem(ref root,b"/input",3).kind==4 && problem(ref root,b"tree//item",3).kind==4 && problem(ref root,b"tree/../input",3).kind==4);
        assert(problem(ref root,b"input/",3).kind==4 && problem(ref root,b"tree\\input",3).kind==4 && problem(ref root,b"tree/con.txt",3).kind==4 && problem(ref root,b"tree/file?",3).kind==4 && problem(ref root,b"tree/file.",3).kind==4 && problem(ref root,b"tree/file ",3).kind==4);
        assert(problem(ref root,b"input",8388609).kind==3 && unsafe{calls()}==count);
        var long=bytes::empty();discard bytes::append(ref mut long,b"tree/");var usize i=0;while(i<256){discard bytes::append_byte(ref mut long,97);i+=1;}assert(problem(ref root,bytes::view(ref long),3).kind==4 && unsafe{calls()}==count);
        discard bytes::truncate(ref mut long,0);i=0;while(i<2048){discard bytes::append(ref mut long,b"a/");i+=1;}assert(problem(ref root,bytes::view(ref long),3).kind==4 && unsafe{calls()}==count);
        assert(problem(ref root,b"input",2).kind==3 && unsafe{reads()}==0);
    }assert(unsafe{live()}==0);
    var usize size=8388607;while(size<=8388609){unsafe{mode(0,size,size,0);}{let root=opened();if(size<=8388608){let p=loaded(ref root,b"input",8388608);assert(view(ref p).len==size && metadata(ref p).size==size);}else{assert(problem(ref root,b"input",8388608).kind==3 && unsafe{reads()}==0);}}assert(unsafe{live()}==0);size+=1;}
    unsafe{mode(0,3,2,0);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.kind==3 && p.completed==2);}assert(unsafe{live()}==0);
    unsafe{mode(0,3,4,0);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.kind==3 && p.completed==4);}assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void detects_open_and_read_metadata_drift_and_native_failures(){
    var u32 scenario=10;while(scenario<=25){if(scenario==16){scenario=20;}unsafe{mode(scenario,3,3,0);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.kind==6 && p.stage==if(scenario<20){5}else{7});if(scenario<20){assert(unsafe{reads()}==0);}}assert(unsafe{live()}==0);scenario+=1;}
    unsafe{mode(43,3,3,0);}{let root=opened();assert(problem(ref root,b"input",3).kind==4 && unsafe{reads()}==0);}assert(unsafe{live()}==0);
    unsafe{mode(44,3,3,0);}{let root=opened();assert(problem(ref root,b"tree/item",3).kind==4 && unsafe{reads()}==0);}assert(unsafe{live()}==0);
    var u32 flags=16;while(flags<=128){unsafe{mode(0,3,3,flags);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.kind==1 && p.os_code==if(flags==128){13}else{5});}assert(unsafe{live()}==0);flags*=2;}
    unsafe{mode(45,3,3,0);}{let root=opened();assert(problem(ref root,b"input",3).kind==7);}assert(unsafe{live()}==0);
    var u32 malformed=46;while(malformed<=48){unsafe{mode(malformed,3,3,0);}{let root=opened();assert(problem(ref root,b"input",3).kind==7);}assert(unsafe{live()}==0);malformed+=1;}
}
@test @effects(filesystem) fn void preserves_close_causes_and_releases_every_owned_handle_once(){
    unsafe{mode(0,3,3,1);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.kind==1 && p.stage==8 && p.os_code==61);}assert(unsafe{live()}==0);
    unsafe{mode(0,3,3,4);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.kind==1 && p.stage==2 && p.os_code==73 && unsafe{reads()}==0);}assert(unsafe{live()}==0);
    unsafe{mode(0,3,3,68);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.os_code==5 && p.close_code==73);}assert(unsafe{live()}==0);
    unsafe{mode(30,3,3,9);}{let root=opened();let p=problem(ref root,b"input",3);assert(p.stage==6 && p.os_code==5 && p.completed==2 && p.close_code==61);}assert(unsafe{live()}==0);
    unsafe{mode(0,3,3,2);}{let root=opened();let p=problem(ref root,b"tree/item",3);assert(p.stage==9 && p.os_code==62);}assert(unsafe{live()}==0);
    unsafe{mode(30,3,3,11);}{let root=opened();let p=problem(ref root,b"tree/item",3);assert(p.os_code==5 && p.close_code==61 && p.directory_close_code==62);}assert(unsafe{live()}==0);
    unsafe{mode(0,3,3,2);}{let root=opened();let p=problem(ref root,b"tree/branch/item",3);assert(p.stage==9 && p.os_code==62 && p.directory_close_code==62);}assert(unsafe{live()}==0);
    unsafe{mode(0,3,3,2);}{let root=opened();let p=problem(ref root,b"tree/missing/item",3);assert(p.kind==5 && p.directory_close_code==62);}assert(unsafe{live()}==0);
    unsafe{mode(0,3,3,2);}{let root=opened();var info=rooted::Metadata{.device=9,.identity=8,.size=7,.modified_seconds=6,.modified_nanos=5,.mode=4};let expected=info;assert(!fs::read_observed(ref root,b"tree/item",3,ref mut info).is_ok && fs::same(info,expected));}assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void shares_entry_and_work_limits_without_allocating_a_listing(){
    var usize count=99999;while(count<=100001){unsafe{mode(0,3,3,0);siblings(count);}{let root=opened();if(count<=100000){let p=loaded(ref root,b"input",3);assert(view(ref p).len==3);}else{assert(problem(ref root,b"input",3).kind==3 && unsafe{reads()}==0);}}assert(unsafe{live()}==0);count+=1;}
    unsafe{mode(0,3,3,0);}{let root=opened();var info=fs::empty_metadata();var budget=directory::EditBudget{.entries=99999,.work=0};let u32 first=switch(fs::read_with_budget(ref root,b"tree/item",3,ref mut budget,ref mut info)){case .ok(_):{0}case .err(p):{p.kind}};assert(first==3 && budget.entries==100000 && unsafe{live()}==1 && unsafe{reads()}==0);}
    assert(unsafe{live()}==0);
    unsafe{mode(0,3,3,0);}{let root=opened();let before=unsafe{calls()};var info=fs::empty_metadata();var budget=directory::EditBudget{.entries=0,.work=directory::edit_work_limit()};assert(!fs::read_with_budget(ref root,b"input",3,ref mut budget,ref mut info).is_ok && unsafe{calls()}==before);budget.work=directory::edit_work_limit()-1;assert(!fs::read_with_budget(ref root,b"input",3,ref mut budget,ref mut info).is_ok && unsafe{live()}==1);budget.entries=100001;budget.work=0;let current=unsafe{calls()};assert(!fs::read_with_budget(ref root,b"input",3,ref mut budget,ref mut info).is_ok && unsafe{calls()}==current);}
    assert(unsafe{live()}==0);
    unsafe{mode(49,3,3,0);}{let root=opened();assert(problem(ref root,b"input",3).kind==3);}assert(unsafe{live()}==0);
    unsafe{mode(50,3,3,0);}{let root=opened();assert(problem(ref root,b"input",3).kind==3);}assert(unsafe{live()}==0);
}
