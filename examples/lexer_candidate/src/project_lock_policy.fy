module project_lock_policy;
import project_module_visibility as visibility;
import project_lock_document as document;
import project_lock_json as json;
import project_lock_read_file as reading;
import project_lock_write as writing;
import project_source_file as files;
import project_path as work;
import std::bytes as bytes;

pub enum State{Unchanged;Updated;Verified;}
// 1 fresh projection, 2 canonical marshal, 3 locked physical/strict read,
// 4 stale, 5 comparison work, 6 atomic publication. Read details and secondary
// write errors stay available to the CLI; missing never becomes stale.
pub struct Problem{u32 stage;u32 code;u32 os_code;u32 read_stage;u32 reader_stage;usize start;usize end;usize node;usize edge;usize completed;u32 close_code;u32 close_os_code;u32 cleanup_code;u32 cleanup_os_code;}
fn Problem fail(u32 stage,u32 code){return Problem{.stage,.code,.os_code=0,.read_stage=0,.reader_stage=0,.start=0,.end=0,.node=0,.edge=0,.completed=0,.close_code=0,.close_os_code=0,.cleanup_code=0,.cleanup_os_code=0};}
pub fn u32 state_code(State state){return switch(state){case .Unchanged:{1}case .Updated:{2}case .Verified:{3}};}
pub fn ref str label(State state){let code=state_code(state);if(code==1){return "unchanged";}if(code==2){return "updated";}return "verified";}
fn Result<bool,Problem> same(ref u8[] actual,ref u8[] expected,ref mut usize meter){if(!work::charge(meter,actual.len+expected.len+1)){return .err(fail(5,1));}return .ok(bytes::equal(actual,expected));}
@effects(filesystem)
pub fn Result<State,Problem> apply(ref u8[] root,ref visibility::Resolved workspace,bool locked,ref mut usize meter){
    // Always derive the expected lock from fresh resolved inputs, even in locked
    // mode. Lock contents never select sources, SDKs or dependency identities.
    let projected=switch(document::project(workspace,meter)){case .ok(value):{value}case .err(e):{var Problem p=fail(1,e.code);p.node=e.node;return .err(p);}};
    let canonical=switch(json::marshal(ref projected,meter)){case .ok(value):{value}case .err(code):{return .err(fail(2,code));}};
    if(locked){
        let actual=switch(reading::read(root,meter)){case .ok(value):{value}case .err(e):{var Problem p=fail(3,e.code);p.os_code=e.os_code;p.read_stage=e.stage;p.reader_stage=e.reader_stage;p.start=e.start;p.end=e.end;p.node=e.node;p.edge=e.edge;return .err(p);}};
        let observed=switch(json::marshal(ref actual,meter)){case .ok(value):{value}case .err(code):{return .err(fail(2,code));}};
        let current=switch(same(bytes::view(ref observed),bytes::view(ref canonical),meter)){case .ok(value):{value}case .err(p):{return .err(p);}};
        return if(current){.ok(.Verified)}else{.err(fail(4,1))};
    }
    // Ordinary mode compares physical bytes. A noncanonical or invalid old
    // document is replaced by the freshly projected canonical one, as Go does.
    switch(files::read(root,b"fairy.lock",8388608,meter)){
        case .ok(actual):{let current=switch(same(bytes::view(ref actual),bytes::view(ref canonical),meter)){case .ok(value):{value}case .err(p):{return .err(p);}};if(current){return .ok(.Unchanged);}}
        case .err(_):{}
    };
    switch(writing::write(root,canonical,meter)){
        case .ok(_):{return .ok(.Updated);}
        case .err(e):{var Problem p=fail(6,e.code);p.read_stage=e.stage;p.os_code=e.os_code;p.completed=e.completed;p.close_code=e.close_code;p.close_os_code=e.close_os_code;p.cleanup_code=e.cleanup_code;p.cleanup_os_code=e.cleanup_os_code;return .err(p);}
    };
    return .err(fail(6,4));
}
