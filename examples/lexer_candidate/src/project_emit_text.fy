module project_emit_text;
import project_emit as emit;
import project_check_text as output;
import project_check_status as status;
import project_compiler_input as input;
import syntax as syntax;
import std::bytes as bytes;

pub fn i32 exit_code(ref emit::Error e){let phase=emit::phase(e);if(phase==1){return status::error(emit::frontend_error(e));}if(phase==2){return if(emit::entry_error(e).kind<=3){2}else{4};}if(phase==3){return if(emit::emission_error(e).code==1004){2}else{4};}return 4;}
pub fn bool error(ref emit::Error e,ref mut bytes::Buffer out){let phase=emit::phase(e);if(phase==1){return output::error(emit::frontend_error(e),out);}if(phase==4){return output::append(out,b"fairyc: internal compiler processing failed\n");}
    let sources=emit::sources(e);let selected=if(phase==2){emit::entry_error(e).module_index}else{emit::emission_error(e).module_index};let start=if(phase==2){emit::entry_error(e).start}else{emit::emission_error(e).start};let end=if(phase==2){emit::entry_error(e).end}else{emit::emission_error(e).end};
    if(selected<input::count(sources) && start<end && end<=syntax::source_view(input::source(sources,selected)).len){let original=input::origin(sources,selected);if(!output::prefix(out,output::package_path(emit::workspace(e),original.owner),input::file(sources,selected),syntax::source_view(input::source(sources,selected)),start)){return false;}}
    if(phase==2){return output::append(out,if(emit::entry_error(e).kind<=3){b"fairyc: candidate workspace root entry main::main must have type fn() -> i32\n"}else{b"fairyc: internal entry selection failed\n"});}
    return output::append(out,b"fairyc: candidate workspace C emission failed (code ") && output::number(out,emit::emission_error(e).code as usize) && output::append(out,b")\n");
}
