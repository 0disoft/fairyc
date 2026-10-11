module project_lock_read_file;
import project_source_file as files;
import project_lock_reader as reader;
import project_lock_document as document;
import std::bytes as bytes;

// Stage 1 preserves bounded regular-file/path errors and OS codes;
// stage 2 preserves strict reader stage/code/offset/package/edge information.
pub struct Problem{u32 stage;u32 code;u32 os_code;u32 reader_stage;usize start;usize end;usize node;usize edge;}
@effects(filesystem)
pub fn Result<document::Document,Problem> read(ref u8[] root,ref mut usize meter){
    let bytes::Buffer input=switch(files::read(root,b"fairy.lock",8388608,meter)){case .ok(value):{value}case .err(p):{return .err(Problem{.stage=1,.code=p.code,.os_code=p.os_code,.reader_stage=0,.start=p.start,.end=p.end,.node=0,.edge=0});}};
    return switch(reader::parse(input,meter)){case .ok(value):{.ok(value)}case .err(p):{.err(Problem{.stage=2,.code=p.code,.os_code=0,.reader_stage=p.stage,.start=p.start,.end=p.end,.node=p.node,.edge=p.edge})}};
}
