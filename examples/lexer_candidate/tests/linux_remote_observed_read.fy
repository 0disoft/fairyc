module tests::linux_remote_observed_read;
import linux_remote_fs as fs;
import linux_rooted_file as rooted;
import std::bytes as bytes;
@link_name("remote_fs_probe_mode") unsafe extern C fn void mode(u32 value);
@link_name("remote_fs_probe_live") unsafe extern C fn u32 live();
@link_name("remote_fs_probe_reads") unsafe extern C fn u32 reads();
@link_name("remote_fs_probe_mutations") unsafe extern C fn u32 mutations();
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
// Independent fact for the model's original binary input; no product observation
// supplies the expected identity or masks a generation change.
fn rooted::Metadata original(){return rooted::Metadata{.device=71,.identity=101,.size=3,.modified_seconds=91,.modified_nanos=42,.mode=32768};}
@effects(filesystem) fn u32 problem(ref rooted::Directory root,ref u8[] name,rooted::Metadata expected,usize maximum){return switch(fs::read_observed(root,name,expected,maximum)){case .ok(_):{0}case .err(p):{p}};}
@test @effects(filesystem) fn void matches_every_observed_field_before_reading(){
    var u32 field=0;while(field<6){unsafe{mode(1);}{let root=opened();var expected=original();
        if(field==0){expected.device+=1;}else if(field==1){expected.identity+=1;}else if(field==2){expected.size-=1;}else if(field==3){expected.modified_seconds+=1;}else if(field==4){expected.modified_nanos+=1;}else{expected.mode+=128;}
        assert(problem(ref root,b"input",expected,3)==4 && unsafe{reads()}==0 && unsafe{mutations()}==0);
    }assert(unsafe{live()}==0);field+=1;}
    unsafe{mode(1);}{let root=opened();let data=switch(fs::read_observed(ref root,b"input",original(),3)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(bytes::equal(bytes::view(ref data),b"a\0\xff") && unsafe{reads()}==2 && unsafe{mutations()}==0);
    }assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void rejects_invalid_facts_paths_and_bounds_without_reading(){
    unsafe{mode(1);}{let root=opened();var expected=original();expected.identity=0;assert(problem(ref root,b"input",expected,3)==4);
        expected=original();expected.mode=16384;assert(problem(ref root,b"input",expected,3)==4);
        expected=original();expected.modified_nanos=1000000000;assert(problem(ref root,b"input",expected,3)==4);
        assert(problem(ref root,b"tree/item",original(),3)==4 && problem(ref root,b"Input",original(),3)==4 && problem(ref root,b"missing",original(),3)==5);
        assert(problem(ref root,b"input",original(),2)==3 && problem(ref root,b"input",original(),16777217)==3);
        assert(unsafe{reads()}==0 && unsafe{mutations()}==0);
    }assert(unsafe{live()}==0);
}
@test @effects(filesystem) fn void retains_complete_read_and_single_link_checks(){
    var u32 value=2;while(value<=6){unsafe{mode(value);}{let root=opened();let code=problem(ref root,b"input",original(),3);
        assert(code==if(value<=3){3}else if(value==6){1}else{4});
        if(value==4){assert(unsafe{reads()}==0);}assert(unsafe{mutations()}==0);
    }assert(unsafe{live()}==0);value+=1;}
}
