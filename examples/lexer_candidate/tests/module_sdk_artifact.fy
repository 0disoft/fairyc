module tests::module_sdk_artifact;
import module_sdk_artifact as codec;
import module_unit_sdk_sources as owned;
import candidate_artifact_wire as wire;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;

resource Fixture{bytes::Buffer golden;bytes::Buffer alloc;bytes::Buffer io;cleanup{}}
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
fn bytes::Buffer field(ref u8[] raw,ref mut wire::Cursor cursor,ref mut usize work){let n=wire::count_from(raw,cursor,65536,1,work);assert(wire::status(cursor)==0);let data=copied(raw[cursor.offset..cursor.offset+n]);cursor.offset+=n;return data;}
@effects(filesystem) fn Fixture fixture(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref path;
    let data=switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);return fixture();}};
    let raw=bytes::view(ref data);var usize work=0;var cursor=switch(wire::cursor(raw,b"FYSGOLD0",ref mut work)){case .ok(c):{c}case .err(_):{assert(false);return fixture();}};
    let golden=field(raw,ref mut cursor,ref mut work);let alloc=field(raw,ref mut cursor,ref mut work);let io=field(raw,ref mut cursor,ref mut work);assert(wire::end(raw,ref mut cursor,ref mut work)==0);return Fixture{.golden,.alloc,.io};
}
fn owned::Sources sample(ref Fixture p,usize history){var v::Vec<owned::Unit> units=v::empty<owned::Unit>();let first=bytes::len(ref p.alloc);let total=first+bytes::len(ref p.io);
    assert(v::push<owned::Unit>(ref mut units,owned::Unit{.module_index=1,.sdk_id=1,.start=0,.end=first}).is_ok);
    assert(v::push<owned::Unit>(ref mut units,owned::Unit{.module_index=3,.sdk_id=4,.start=first,.end=total}).is_ok);
    var source=copied(bytes::view(ref p.alloc));assert(bytes::append(ref mut source,bytes::view(ref p.io)).is_ok);
    return switch(owned::assemble(4,units,source,history)){case .ok(p):{p}case .err(_):{assert(false);return sample(p,history);}};
}
@effects(console) fn wire::Payload encoded(ref owned::Sources p,ref mut usize work){return switch(codec::encode(p,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return encoded(p,work);}};}
@effects(console) fn owned::Sources decoded(ref u8[] raw,ref mut usize work){return switch(codec::decode(raw,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return decoded(raw,work);}};}
fn u32 rejected(ref u8[] raw){var usize work=0;return switch(codec::decode(raw,ref mut work)){case .ok(_):{0}case .err(code):{code}};}
@effects(filesystem,console) fn owned::Sources detached(){let p=fixture();let original=sample(ref p,123);var usize work=0;let payload=encoded(ref original,ref mut work);work=0;return decoded(wire::content(ref payload),ref mut work);}

@test @effects(filesystem,console)
fn void sdk_binary_matches_golden_and_reauthenticates_detached_bytes(){let fixture=fixture();let p=sample(ref fixture,123);var usize work=0;let data=encoded(ref p,ref mut work);assert(bytes::equal(wire::content(ref data),bytes::view(ref fixture.golden)));
    let restored=detached();assert(owned::module_count(ref restored)==4 && owned::count(ref restored)==2 && owned::work(ref restored)==0);
    assert(owned::at(ref restored,0).module_index==1 && owned::at(ref restored,0).sdk_id==1 && owned::at(ref restored,0).start==0 && owned::at(ref restored,0).end==bytes::len(ref fixture.alloc));
    assert(owned::at(ref restored,1).module_index==3 && owned::at(ref restored,1).sdk_id==4 && owned::at(ref restored,1).start==bytes::len(ref fixture.alloc) && owned::at(ref restored,1).end==owned::data(ref restored).len);
    assert(bytes::equal(owned::source_at(ref restored,0),bytes::view(ref fixture.alloc)) && bytes::equal(owned::source_at(ref restored,1),bytes::view(ref fixture.io)));
    var usize proof=0;assert(owned::check(ref restored,ref mut proof).code==0 && proof==owned::data(ref restored).len+2);
    work=0;let again=encoded(ref restored,ref mut work);assert(bytes::equal(wire::content(ref data),wire::content(ref again)));
    let empty=owned::empty();work=0;let none=encoded(ref empty,ref mut work);assert(wire::content(ref none).len==32);work=0;let blank=decoded(wire::content(ref none),ref mut work);assert(owned::count(ref blank)==0 && owned::module_count(ref blank)==0 && owned::work(ref blank)==0);
    let users=switch(owned::assemble(512,v::empty<owned::Unit>(),bytes::empty(),0)){case .ok(p):{p}case .err(_):{assert(false);owned::empty()}};work=0;let user_data=encoded(ref users,ref mut work);work=0;let user_copy=decoded(wire::content(ref user_data),ref mut work);assert(owned::module_count(ref user_copy)==512 && owned::count(ref user_copy)==0);
}
@test @effects(filesystem,console)
fn void sdk_codec_preserves_exact_work_and_fresh_verification_cost(){let fixture=fixture();let p=sample(ref fixture,123);var usize cost=0;let data=encoded(ref p,ref mut cost);
    assert(cost==9+3*9+2*4*9+3+2*owned::data(ref p).len);
    var usize work=16777216-cost;let fit=encoded(ref p,ref mut work);assert(work==16777216 && bytes::equal(wire::content(ref data),wire::content(ref fit)));
    work=16777216-cost+1;assert(switch(codec::encode(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    cost=0;let restored=decoded(wire::content(ref data),ref mut cost);assert(cost==9+3*9+2*4*9+4+2*owned::data(ref p).len);
    work=16777216-cost;let copy=decoded(wire::content(ref data),ref mut work);assert(work==16777216 && owned::module_count(ref copy)==4);
    work=16777216-cost+1;assert(switch(codec::decode(wire::content(ref data),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    work=16777217;assert(switch(codec::decode(wire::content(ref data),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==16777217);
    work=16777217;assert(switch(codec::encode(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==16777217);
    var usize measured=0;let authentic=switch(owned::assemble_artifact(512,v::empty<owned::Unit>(),bytes::empty(),ref mut measured)){case .ok(p):{p}case .err(_):{assert(false);owned::empty()}};assert(measured==0 && owned::module_count(ref authentic)==512);
}
@test @effects(filesystem)
fn void sdk_codec_rejects_bad_coordinates_duplicate_authority_and_altered_source(){let fixture=fixture();let raw=bytes::view(ref fixture.golden);var usize i=0;while(i<96){assert(rejected(raw[..i])!=0);i+=1;}assert(rejected(raw[..raw.len-1])==1001 && rejected(raw[..raw.len/2])==1001);
    var bytes::Buffer trailing=copied(raw);assert(bytes::append_byte(ref mut trailing,0).is_ok && rejected(bytes::view(ref trailing))==1001);
    var u32 mode=0;while(mode<17){var bytes::Buffer changed=copied(raw);let view=bytes::view_mut(ref mut changed);
        if(mode==0){view[7]=49;}else if(mode==1){view[15]=0;}else if(mode==2){view[8]=1;}else if(mode==3){view[23]=9;}
        else if(mode==4){view[31]=4;}else if(mode==5){view[39]=4;}else if(mode==6){view[47]=1;}else if(mode==7){view[55]=0;}
        else if(mode==8){view[63]=1;}else if(mode==9){view[71]=1;}else if(mode==10){view[79]=0;}else if(mode==11){view[96]=0;}
        else if(mode==12){view[88]=1;}else if(mode==13){view[raw.len-1]=0;}else if(mode==14){view[48]=1;}else if(mode==15){view[32]=1;}else{view[87]=0;}
        assert(rejected(bytes::view(ref changed))==if(mode==2 || mode==3 || mode==12){1002}else{1001});mode+=1;
    }
}
