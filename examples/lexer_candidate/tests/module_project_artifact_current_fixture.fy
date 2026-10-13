module tests::module_project_artifact_current_fixture;
import tests::module_identity_artifact_fixture as fixture;
import module_project_artifact_encode as encoder;
import module_project_artifact_decode as decoder;
import module_project_artifact_model as model;
import module_program_artifact_wire as framing;
import candidate_artifact_wire as wire;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
pub resource Current{v::Vec<syntax::Parsed> sources;v::Vec<usize> owners;v::Vec<catalog::Binding> imports;coordinates::Coordinates coordinates;cleanup{}}
pub fn ref v::Vec<syntax::Parsed> sources(ref Current p){return ref p.sources;}
pub fn ref v::Vec<usize> owners(ref Current p){return ref p.owners;}
pub fn ref v::Vec<catalog::Binding> imports(ref Current p){return ref p.imports;}
pub fn ref coordinates::Coordinates coordinate_table(ref Current p){return ref p.coordinates;}
pub fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
fn syntax::Parsed parsed(bytes::Buffer source){let s=switch(scanner::collect(source,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(filesystem,console) pub fn wire::Payload packet(){let data=fixture::data();let loaded=fixture::loaded(fixture::workspace(ref data));let p=fixture::compiled(ref loaded);var usize work=0;return switch(encoder::encode(ref p,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return packet();}};}
@effects(console) pub fn model::Restored restored(ref u8[] raw){var usize work=0;return switch(decoder::decode(raw,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return restored(raw);}};}
@effects(filesystem) pub fn Current current(usize mode){let data=fixture::data();let loaded=fixture::loaded(fixture::workspace(ref data));let originals=fixture::inputs(ref loaded);let count=input::count(originals);var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var usize i=0;
    while(i<count){let old=if(mode==1){count-1-i}else{i};var raw=copied(syntax::source_view(input::source(originals,old)));if(mode==2 && i==0){assert(bytes::append_byte(ref mut raw,10).is_ok);}assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok && v::push<usize>(ref mut owners,*v::at<usize>(input::owners(originals),old)+if(mode==1){7}else{0}).is_ok);i+=1;}
    var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();i=0;while(i<v::len<catalog::Binding>(input::bindings(originals))){var row=*v::at<catalog::Binding>(input::bindings(originals),i);if(mode==1){row.source=count-1-row.source;row.target=count-1-row.target;}if(mode==3 && i==0){row.target=(row.target+1)%count;}assert(v::push<catalog::Binding>(ref mut imports,row).is_ok);i+=1;}
    // Resolved bindings follow current module/declaration order, as the loader does.
    i=1;while(i<v::len<catalog::Binding>(ref imports)){let row=*v::at<catalog::Binding>(ref imports,i);var usize at=i;while(at>0){let old=*v::at<catalog::Binding>(ref imports,at-1);if(old.source<row.source || old.source==row.source && old.declaration<row.declaration){break;}{let cell=v::at_mut<catalog::Binding>(ref mut imports,at);*cell=old;}at-=1;}{let cell=v::at_mut<catalog::Binding>(ref mut imports,at);*cell=row;}i+=1;}
    let table=fixture::coordinate_table(ref loaded);var bytes::Buffer names=bytes::empty();var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();i=0;while(i<coordinates::coordinate_count(table)){let row=coordinates::coordinate_at(table,i);let start=bytes::len(ref names);assert(bytes::append(ref mut names,coordinates::coordinate_text(table,i)).is_ok);if(mode==4 && i==0){let old=bytes::view(ref names)[start+13];bytes::view_mut(ref mut names)[start+13]=if(old==97){98}else{97};}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=row.owner+if(mode==1){7}else{0},.start,.end=bytes::len(ref names)}).is_ok);i+=1;}
    return Current{.sources,.owners,.imports,.coordinates=coordinates::assemble(names,rows)};
}
pub fn framing::Span identity_span(ref u8[] raw){var usize work=0;var c=switch(framing::cursor_marker(raw,b"FYPRJ000",ref mut work)){case .ok(c):{c}case .err(_):{assert(false);return identity_span(raw);}};discard framing::member_from(raw,ref mut c,framing::byte_limit(),b"FYPROG00",ref mut work);discard framing::member_from(raw,ref mut c,8388608,b"FYPROV00",ref mut work);let span=framing::member_from(raw,ref mut c,8388608,b"FYIDS000",ref mut work);assert(framing::end(raw,ref mut c,ref mut work)==0);return span;}
