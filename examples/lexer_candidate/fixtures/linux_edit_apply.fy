module tests::linux_edit_apply;
import linux_edit_apply as applying;
import linux_rooted_file as rooted;
import project_edit_preview as preview;
import project_edit_options as options;
@link_name("edit_apply_probe") unsafe extern C fn void mode(u32 value);
@link_name("edit_apply_verify") unsafe extern C fn void verify(u32 published);
@link_name("remote_fs_probe_live") unsafe extern C fn u32 live();
@effects(filesystem) fn rooted::Directory opened(){return switch(rooted::open(b"/work")){case .ok(p):{p}case .err(_):{assert(false);return opened();}};}
@effects(filesystem,environment) fn void check(u32 scenario,u32 stage,u32 code,u32 cleanup_code,bool published){
    unsafe{mode(scenario);}{let root=opened();let o=options::test_options();let proposal=preview::test_proposal();
        switch(applying::apply(ref root,b"/work",ref o,ref proposal)){
            case .ok(_):{assert(stage==0 && published);}
            case .err(p):{assert(stage!=0 && p.stage==stage && p.code==code && p.cleanup_code==cleanup_code && p.published==published);}
        };assert(unsafe{live()}==1);unsafe{verify(if(published){1}else{0});}
    }assert(unsafe{live()}==0);
}
@test @effects(filesystem,environment) fn void publishes_once_with_private_staging_and_preserved_old_alias(){check(0,0,0,0,true);check(2,0,0,0,true);}
@test @effects(filesystem,environment) fn void refuses_lock_stage_and_anonymous_creation_failures(){check(1,1,17,0,false);check(3,4,3,0,false);check(4,4,4,0,false);check(5,4,95,0,false);check(24,4,17,66,false);check(25,1,17,66,false);}
@test @effects(filesystem,environment) fn void preserves_source_on_write_prepare_and_validation_failures(){check(6,5,5,0,false);check(7,5,5,0,false);check(8,2,5,0,false);check(9,6,5,0,false);check(10,6,5,0,false);check(13,3,4,0,false);check(14,3,5,0,false);check(26,3,4,0,false);check(27,3,5,54,false);}
@test @effects(filesystem,environment) fn void refuses_byte_metadata_drift_and_never_replays_publication(){check(11,7,5,0,false);check(12,7,5,0,false);check(15,8,4,0,false);check(22,8,5,13,false);}
@test @effects(filesystem,environment) fn void retains_known_publication_through_every_cleanup_error(){check(16,9,55,0,true);check(17,9,39,0,true);check(18,9,62,0,true);check(19,10,63,64,true);check(20,9,61,62,true);check(21,8,13,13,true);check(23,10,43,0,true);}
