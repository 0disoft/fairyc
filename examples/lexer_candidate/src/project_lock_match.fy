module project_lock_match;
import project_lock_document as document;
import project_lock_json as json;
import project_module_visibility as visibility;
import project_path as work;
import std::bytes as bytes;

// Stages: 1 fresh-workspace projection, 2 actual marshal, 3 expected marshal,
// 4 comparison work bound. False means a valid lock is stale, never a read error.
pub struct Problem{u32 stage;u32 code;usize node;}
pub fn Result<bool,Problem> matches(ref document::Document actual,ref visibility::Resolved workspace,ref mut usize meter){
    let expected=switch(document::project(workspace,meter)){case .ok(value):{value}case .err(p):{return .err(Problem{.stage=1,.code=p.code,.node=p.node});}};
    let observed=switch(json::marshal(actual,meter)){case .ok(value):{value}case .err(code):{return .err(Problem{.stage=2,.code,.node=0});}};
    let current=switch(json::marshal(ref expected,meter)){case .ok(value):{value}case .err(code):{return .err(Problem{.stage=3,.code,.node=0});}};
    if(!work::charge(meter,bytes::len(ref observed)+bytes::len(ref current)+1)){return .err(Problem{.stage=4,.code=1,.node=0});}
    return .ok(bytes::equal(bytes::view(ref observed),bytes::view(ref current)));
}
