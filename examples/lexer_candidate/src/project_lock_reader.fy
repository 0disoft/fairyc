module project_lock_reader;
import project_lock_fields as fields;
import project_lock_validate as validate;
import project_lock_document as document;
import std::bytes as bytes;

// Stages: 1 JSON/closed fields (original byte offsets), 2 semantic validation
// (package/edge indices), 3 retained document allocation/work bound.
pub struct Problem{u32 stage;u32 code;usize start;usize end;usize node;usize edge;}
pub fn Result<document::Document,Problem> parse(bytes::Buffer source,ref mut usize meter){
    let draft=switch(fields::parse(source,meter)){case .ok(value):{value}case .err(p):{return .err(Problem{.stage=1,.code=p.code,.start=p.start,.end=p.end,.node=0,.edge=0});}};
    let checked=switch(validate::validate(draft,meter)){case .ok(value):{value}case .err(p):{return .err(Problem{.stage=2,.code=p.code,.start=0,.end=0,.node=p.node,.edge=p.edge});}};
    return switch(document::from_validated(ref checked,meter)){case .ok(value):{.ok(value)}case .err(p):{.err(Problem{.stage=3,.code=p.code,.start=0,.end=0,.node=p.node,.edge=0})}};
}
