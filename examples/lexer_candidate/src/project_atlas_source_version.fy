module project_atlas_source_version;
import project_compiler_input as input;
import project_compiler_core as compiler;
import module_project_identified_ir as identified;
import module_signatures as signatures;
import module_public_type_text as coordinates;
import project_path as work;
import syntax as syntax;

// Reauthenticate the full exact source closure against Complete before choosing
// a map family. Filtering one package/symbol cannot hide unsupported templates
// in another package, including unused generic aggregates and SDK templates.
pub fn Result<bool,u32> admit(ref input::Input source,ref coordinates::Coordinates table,ref compiler::Complete complete,ref signatures::Program metadata,u32 version,ref mut usize meter){
    if(version<1 || version>3){return .err(101);}
    let code=identified::current(compiler::checked_program(complete),input::sources(source),metadata,table,meter);
    if(code!=0){return .err(if(code==10){3}else{7});}
    var usize m=0;while(m<input::count(source)){let p=input::source(source,m);var usize i=0;
        while(i<syntax::declaration_count(p)){if(!work::charge(meter,1)){return .err(3);}let d=syntax::declaration_at(p,i);
            if(d.generic_parameters!=0 && (version==1 || version==2 && d.kind!=3)){return .err(101);}i+=1;
        }m+=1;
    }return .ok(true);
}
